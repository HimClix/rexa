package engine

import (
	"unicode"

	"github.com/himclix/rexa/compiler"
	"github.com/himclix/rexa/internal/bitset"
	"github.com/himclix/rexa/internal/runeutil"
)

const (
	defaultCacheCapacity = 10000 // max DFA states cached
	maxCacheResets       = 5    // if we reset this many times, give up on DFA
)

// LazyDFA builds DFA states on the fly from the NFA program.
// Each DFA state is a set of NFA instruction PCs (the epsilon closure).
// States are cached; if the cache overflows it's flushed and rebuilt.
type LazyDFA struct {
	prog         *compiler.Program
	cache        map[string]*dfaState
	cacheResets  int
	capacity     int
	abandoned    bool
	start        *dfaState
	closureBuf   []int
	resultBuf    []int
	keyBuf       []byte
	closureSeen  *bitset.BitSet
	transNFA     []int
	arena        []int
	arenaPos     int
	hasEndAnchor bool // pattern contains $ or \z — need end-of-input check
}

type dfaState struct {
	nfaStates    []int              // sorted NFA PCs in this DFA state
	isMatch      bool               // true if any NFA state is InstMatch
	needsEndCheck bool              // true if match is only valid at end of input
	ascii        [128]*dfaState     // fast ASCII transition table
	next         map[rune]*dfaState // overflow for non-ASCII
	deadEnd      bool
	asciiDone    [2]uint64
}

func NewLazyDFA(prog *compiler.Program, capacity int) *LazyDFA {
	if capacity <= 0 {
		capacity = defaultCacheCapacity
	}
	hasEnd := false
	for _, inst := range prog.Insts {
		if inst.Op == compiler.InstEndLine || inst.Op == compiler.InstEndText {
			hasEnd = true
			break
		}
	}
	return &LazyDFA{
		prog:         prog,
		cache:        make(map[string]*dfaState),
		capacity:     capacity,
		closureSeen:  bitset.New(len(prog.Insts)),
		hasEndAnchor: hasEnd,
	}
}

func (d *LazyDFA) Abandoned() bool {
	return d.abandoned
}

func (d *LazyDFA) SearchBool(prog *compiler.Program, input Input, pos int) (matched bool, abandoned bool) {
	inp := &input
	if prog.AnchoredStart && pos == 0 {
		m, _, _ := d.matchAt(inp, 0)
		return m, d.abandoned
	}
	for startPos := pos; startPos <= inp.Length(); startPos++ {
		m, _, _ := d.matchAt(inp, startPos)
		if d.abandoned {
			return false, true
		}
		if m {
			return true, false
		}
	}
	return false, false
}

// SearchInto finds first match, writing result into dst. Returns (matched, abandoned).
func (d *LazyDFA) SearchInto(prog *compiler.Program, input *Input, pos int, dst *MatchResult, caps []CaptureSlot) (bool, bool) {
	if prog.AnchoredStart && pos == 0 {
		m, s, e := d.matchAt(input, 0)
		if d.abandoned {
			return false, true
		}
		if m {
			caps[0] = CaptureSlot{Start: s, End: e}
			dst.Matched = true
			dst.Captures = caps[:1]
			return true, false
		}
		return false, false
	}
	for startPos := pos; startPos <= input.Length(); startPos++ {
		m, s, e := d.matchAt(input, startPos)
		if d.abandoned {
			return false, true
		}
		if m {
			caps[0] = CaptureSlot{Start: s, End: e}
			dst.Matched = true
			dst.Captures = caps[:1]
			return true, false
		}
	}
	return false, false
}

func (d *LazyDFA) Search(prog *compiler.Program, input *Input, pos int) *MatchResult {
	if prog.AnchoredStart && pos == 0 {
		matched, start, end := d.matchAt(input, 0)
		if d.abandoned {
			return nil
		}
		if matched {
			return &MatchResult{Matched: true, Captures: []CaptureSlot{{Start: start, End: end}}}
		}
		return &MatchResult{Matched: false}
	}
	for startPos := pos; startPos <= input.Length(); startPos++ {
		matched, start, end := d.matchAt(input, startPos)
		if d.abandoned {
			return nil
		}
		if matched {
			return &MatchResult{Matched: true, Captures: []CaptureSlot{{Start: start, End: end}}}
		}
	}
	return &MatchResult{Matched: false}
}

func (d *LazyDFA) Match(prog *compiler.Program, input *Input, pos int) *MatchResult {
	matched, start, end := d.matchAt(input, pos)
	if d.abandoned {
		return nil
	}
	if matched {
		return &MatchResult{Matched: true, Captures: []CaptureSlot{{Start: start, End: end}}}
	}
	return &MatchResult{Matched: false}
}

// matchAt returns (matched, matchStart, matchEnd) without allocating.
func (d *LazyDFA) matchAt(input *Input, pos int) (bool, int, int) {
	startState := d.startState()
	if startState == nil {
		return false, 0, 0
	}

	cur := startState
	matchEnd := -1

	if cur.isMatch {
		matchEnd = pos
	}

	for i := pos; i < input.Length(); i++ {
		r, ok := input.RuneAt(i)
		if !ok {
			break
		}

		next := d.transition(cur, r)
		if d.abandoned {
			return false, 0, 0
		}
		if next == nil || next.deadEnd {
			break
		}

		cur = next
		if cur.isMatch {
			matchEnd = i + 1
		}
	}

	if matchEnd >= 0 {
		if d.hasEndAnchor && matchEnd != input.Length() {
			return false, 0, 0
		}
		return true, pos, matchEnd
	}
	return false, 0, 0
}

func (d *LazyDFA) startState() *dfaState {
	if d.start != nil {
		return d.start
	}
	initial := d.epsilonClosure([]int{d.prog.Start})
	if len(initial) == 0 {
		return nil
	}
	d.start = d.getOrCreateState(initial)
	return d.start
}

func (d *LazyDFA) transition(from *dfaState, r rune) *dfaState {
	// ASCII fast path: use fixed-size array (no map lookup)
	if r < 128 {
		idx := int(r)
		word := idx / 64
		bit := uint(idx % 64)
		if from.asciiDone[word]&(1<<bit) != 0 {
			return from.ascii[idx]
		}
		next := d.computeTransition(from, r)
		from.ascii[idx] = next
		from.asciiDone[word] |= 1 << bit
		return next
	}

	// Non-ASCII: use overflow map
	if from.next != nil {
		if next, ok := from.next[r]; ok {
			return next
		}
	}
	next := d.computeTransition(from, r)
	if from.next == nil {
		from.next = make(map[rune]*dfaState)
	}
	from.next[r] = next
	return next
}

var deadEndState = &dfaState{deadEnd: true}

func (d *LazyDFA) computeTransition(from *dfaState, r rune) *dfaState {
	d.transNFA = d.transNFA[:0]
	for _, pc := range from.nfaStates {
		inst := &d.prog.Insts[pc]
		if d.instMatchesRune(inst, r) {
			d.transNFA = append(d.transNFA, inst.Out)
		}
	}

	if len(d.transNFA) == 0 {
		return deadEndState
	}

	closed := d.epsilonClosure(d.transNFA)
	next := d.getOrCreateState(closed)
	if d.abandoned {
		return nil
	}
	return next
}

func (d *LazyDFA) instMatchesRune(inst *compiler.Inst, r rune) bool {
	switch inst.Op {
	case compiler.InstRune:
		if inst.CaseInsensitive {
			return runeutil.EqualFold(r, inst.Rune)
		}
		return r == inst.Rune
	case compiler.InstRuneClass:
		if rt, ok := inst.UnicodeTable.(*unicode.RangeTable); ok && rt != nil {
			matched := unicode.Is(rt, r)
			if inst.Negated {
				return !matched
			}
			return matched
		}
		for _, rg := range inst.Ranges {
			if r >= rg.Lo && r <= rg.Hi {
				return !inst.Negated
			}
		}
		return inst.Negated
	case compiler.InstAnyNotNL:
		return r != '\n'
	case compiler.InstAnyChar:
		return true
	default:
		return false
	}
}

func (d *LazyDFA) epsilonClosure(seeds []int) []int {
	nInst := len(d.prog.Insts)
	d.closureBuf = d.closureBuf[:0]
	d.resultBuf = d.resultBuf[:0]
	stack := d.closureBuf
	d.closureSeen.Clear()
	seen := d.closureSeen
	result := d.resultBuf

	for _, pc := range seeds {
		if pc >= 0 && pc < nInst && !seen.Has(pc) {
			seen.Set(pc)
			stack = append(stack, pc)
		}
	}

	for len(stack) > 0 {
		pc := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		inst := &d.prog.Insts[pc]
		switch inst.Op {
		case compiler.InstJump:
			if inst.Out >= 0 && inst.Out < nInst && !seen.Has(inst.Out) {
				seen.Set(inst.Out)
				stack = append(stack, inst.Out)
			}
		case compiler.InstSplit:
			if inst.Out >= 0 && inst.Out < nInst && !seen.Has(inst.Out) {
				seen.Set(inst.Out)
				stack = append(stack, inst.Out)
			}
			if inst.Out1 >= 0 && inst.Out1 < nInst && !seen.Has(inst.Out1) {
				seen.Set(inst.Out1)
				stack = append(stack, inst.Out1)
			}
		case compiler.InstCapStart, compiler.InstCapEnd:
			if inst.Out >= 0 && inst.Out < nInst && !seen.Has(inst.Out) {
				seen.Set(inst.Out)
				stack = append(stack, inst.Out)
			}
		case compiler.InstBeginLine, compiler.InstEndLine,
			compiler.InstBeginText, compiler.InstEndText,
			compiler.InstWordBoundary, compiler.InstNoWordBoundary:
			// Zero-width assertions: skip in DFA (overapproximate by including successor)
			if inst.Out >= 0 && inst.Out < nInst && !seen.Has(inst.Out) {
				seen.Set(inst.Out)
				stack = append(stack, inst.Out)
			}
		default:
			result = append(result, pc)
		}
	}

	sortInts(result)
	d.closureBuf = stack[:0]
	out := d.arenaAlloc(len(result))
	copy(out, result)
	d.resultBuf = result[:0]
	return out
}

func (d *LazyDFA) getOrCreateState(nfaStates []int) *dfaState {
	key := d.makeKey(nfaStates)
	if s, ok := d.cache[key]; ok {
		return s
	}

	if len(d.cache) >= d.capacity {
		d.cacheResets++
		if d.cacheResets >= maxCacheResets {
			d.abandoned = true
			return nil
		}
		d.cache = make(map[string]*dfaState)
		d.arenaPos = 0
		d.start = nil
	}

	isMatch := false
	for _, pc := range nfaStates {
		if d.prog.Insts[pc].Op == compiler.InstMatch {
			isMatch = true
			break
		}
	}

	s := &dfaState{
		nfaStates: nfaStates,
		isMatch:   isMatch,
	}
	d.cache[key] = s
	return s
}

func (d *LazyDFA) makeKey(pcs []int) string {
	need := len(pcs) * 4
	if cap(d.keyBuf) < need {
		d.keyBuf = make([]byte, need)
	}
	buf := d.keyBuf[:need]
	for i, pc := range pcs {
		buf[i*4] = byte(pc >> 24)
		buf[i*4+1] = byte(pc >> 16)
		buf[i*4+2] = byte(pc >> 8)
		buf[i*4+3] = byte(pc)
	}
	return string(buf)
}

func (d *LazyDFA) arenaAlloc(n int) []int {
	if d.arenaPos+n > len(d.arena) {
		newSize := len(d.arena) * 2
		if newSize < 1024 {
			newSize = 1024
		}
		if newSize < d.arenaPos+n {
			newSize = d.arenaPos + n
		}
		newArena := make([]int, newSize)
		copy(newArena, d.arena[:d.arenaPos])
		d.arena = newArena
	}
	s := d.arena[d.arenaPos : d.arenaPos+n]
	d.arenaPos += n
	return s
}

func sortInts(a []int) {
	// Simple insertion sort — state sets are small (typically <50)
	for i := 1; i < len(a); i++ {
		key := a[i]
		j := i - 1
		for j >= 0 && a[j] > key {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = key
	}
}
