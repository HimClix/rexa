package engine

import (
	"unicode"

	"github.com/himclix/rexa/compiler"
	"github.com/himclix/rexa/internal/bitset"
	"github.com/himclix/rexa/internal/runeutil"
)

type PikeVM struct {
	pool *machinePool
}

func NewPikeVM() *PikeVM {
	return &PikeVM{}
}

func (vm *PikeVM) initPool(nInst int) {
	if vm.pool == nil {
		vm.pool = newMachinePool(nInst)
	}
}

func (vm *PikeVM) Search(prog *compiler.Program, input *Input, startPos int) *MatchResult {
	vm.initPool(len(prog.Insts))
	if prog.AnchoredStart && startPos == 0 {
		return vm.matchAt(prog, input, 0)
	}
	for pos := startPos; pos <= input.Length(); pos++ {
		result := vm.matchAt(prog, input, pos)
		if result.Matched {
			return result
		}
	}
	return &MatchResult{Matched: false}
}

func (vm *PikeVM) Match(prog *compiler.Program, input *Input, pos int) *MatchResult {
	vm.initPool(len(prog.Insts))
	return vm.matchAt(prog, input, pos)
}

func (vm *PikeVM) SearchBool(prog *compiler.Program, input Input, startPos int) bool {
	vm.initPool(len(prog.Insts))
	m := vm.pool.get()
	m.input = input
	if prog.AnchoredStart && startPos == 0 {
		result := vm.runBoolMatch(prog, m, 0)
		vm.pool.put(m)
		return result
	}
	for pos := startPos; pos <= m.input.Length(); pos++ {
		if vm.runBoolMatch(prog, m, pos) {
			vm.pool.put(m)
			return true
		}
	}
	vm.pool.put(m)
	return false
}

func (vm *PikeVM) runBoolMatch(prog *compiler.Program, m *machine, pos int) bool {
	input := &m.input
	current := m.lightA[:0]
	next := m.lightB[:0]
	seenCur := m.seenA
	seenNext := m.seenB
	seenCur.Clear()
	seenNext.Clear()

	current = addLight(prog, current, seenCur, prog.Start, pos, pos, input)

	found := false

	for sp := pos; sp <= input.Length(); sp++ {
		r, hasRune := input.RuneAt(sp)

		for _, t := range current {
			inst := &prog.Insts[t.pc]
			switch inst.Op {
			case compiler.InstRune:
				if hasRune && matchRune(r, inst.Rune, inst.CaseInsensitive) {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstRuneClass:
				if hasRune && matchRuneClass(r, inst) {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstAnyNotNL:
				if hasRune && r != '\n' {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstAnyChar:
				if hasRune {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstMatch:
				found = true
			}
		}

		if found && len(next) == 0 {
			break
		}
		if !hasRune {
			break
		}

		current, next = next, current
		next = next[:0]
		seenCur, seenNext = seenNext, seenCur
		seenNext.Clear()
	}

	m.lightA = current[:0]
	m.lightB = next[:0]
	vm.pool.put(m)

	return found
}

func (vm *PikeVM) matchAt(prog *compiler.Program, input *Input, pos int) *MatchResult {
	return vm.matchWithCap(prog, input, pos)
}

func (vm *PikeVM) matchNoCap(prog *compiler.Program, input *Input, pos int) *MatchResult {
	m := vm.pool.get()

	current := m.lightA[:0]
	next := m.lightB[:0]
	seenCur := m.seenA
	seenNext := m.seenB
	seenCur.Clear()
	seenNext.Clear()

	current = addLight(prog, current, seenCur, prog.Start, pos, pos, input)

	matchStart, matchEnd := -1, -1

	for sp := pos; sp <= input.Length(); sp++ {
		r, hasRune := input.RuneAt(sp)

		for _, t := range current {
			inst := &prog.Insts[t.pc]
			switch inst.Op {
			case compiler.InstRune:
				if hasRune && matchRune(r, inst.Rune, inst.CaseInsensitive) {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstRuneClass:
				if hasRune && matchRuneClass(r, inst) {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstAnyNotNL:
				if hasRune && r != '\n' {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstAnyChar:
				if hasRune {
					next = addLight(prog, next, seenNext, inst.Out, t.start, sp+1, input)
				}
			case compiler.InstMatch:
				matchStart = t.start
				matchEnd = sp
			}
		}

		if matchStart >= 0 && len(next) == 0 {
			break
		}
		if !hasRune {
			break
		}

		current, next = next, current
		next = next[:0]
		seenCur, seenNext = seenNext, seenCur
		seenNext.Clear()
	}
	m.lightA = current[:0]
	m.lightB = next[:0]
	vm.pool.put(m)

	if matchStart >= 0 {
		return &MatchResult{
			Matched:  true,
			Captures: []CaptureSlot{{Start: matchStart, End: matchEnd}},
		}
	}
	return &MatchResult{Matched: false}
}

func (vm *PikeVM) matchWithCap(prog *compiler.Program, input *Input, pos int) *MatchResult {
	m := vm.pool.get()

	numCap := prog.NumCap + 1
	initCaps := make([]CaptureSlot, numCap)
	for i := range initCaps {
		initCaps[i] = CaptureSlot{Start: -1, End: -1}
	}
	initCaps[0] = CaptureSlot{Start: pos, End: -1}

	current := m.threadA[:0]
	next := m.threadB[:0]
	seenCur := m.seenA
	seenNext := m.seenB
	seenCur.Clear()
	seenNext.Clear()

	current = addThreadPooled(prog, current, seenCur, prog.Start, initCaps, input, pos)

	var matched *MatchResult

	for sp := pos; sp <= input.Length(); sp++ {
		r, hasRune := input.RuneAt(sp)

		for _, t := range current {
			inst := &prog.Insts[t.pc]

			switch inst.Op {
			case compiler.InstRune:
				if hasRune && matchRune(r, inst.Rune, inst.CaseInsensitive) {
					next = addThreadPooled(prog, next, seenNext, inst.Out, t.captures, input, sp+1)
				}
			case compiler.InstRuneClass:
				if hasRune && matchRuneClass(r, inst) {
					next = addThreadPooled(prog, next, seenNext, inst.Out, t.captures, input, sp+1)
				}
			case compiler.InstAnyNotNL:
				if hasRune && r != '\n' {
					next = addThreadPooled(prog, next, seenNext, inst.Out, t.captures, input, sp+1)
				}
			case compiler.InstAnyChar:
				if hasRune {
					next = addThreadPooled(prog, next, seenNext, inst.Out, t.captures, input, sp+1)
				}
			case compiler.InstMatch:
				caps := CopySlots(t.captures)
				caps[0].End = sp
				matched = &MatchResult{Matched: true, Captures: caps}
				goto stepDone
			}
		}
	stepDone:

		if matched != nil && len(next) == 0 {
			break
		}
		if !hasRune {
			break
		}

		current, next = next, current
		next = next[:0]
		seenCur, seenNext = seenNext, seenCur
		seenNext.Clear()
	}

	m.threadA = current[:0]
	m.threadB = next[:0]
	vm.pool.put(m)

	if matched != nil {
		return matched
	}
	return &MatchResult{Matched: false}
}

func addLight(prog *compiler.Program, threads []lightThread, seen *bitset.BitSet, pc, startPos, curPos int, input *Input) []lightThread {
	if seen.Has(pc) {
		return threads
	}
	seen.Set(pc)
	inst := &prog.Insts[pc]
	switch inst.Op {
	case compiler.InstJump:
		return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
	case compiler.InstSplit:
		threads = addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		threads = addLight(prog, threads, seen, inst.Out1, startPos, curPos, input)
		return threads
	case compiler.InstCapStart, compiler.InstCapEnd:
		return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
	case compiler.InstBeginLine:
		if curPos == 0 {
			return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		}
		return threads
	case compiler.InstEndLine:
		if curPos == input.Length() {
			return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		}
		return threads
	case compiler.InstBeginText:
		if curPos == 0 {
			return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		}
		return threads
	case compiler.InstEndText:
		if curPos == input.Length() {
			return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		}
		return threads
	case compiler.InstWordBoundary:
		if isWordBoundary(input, curPos) {
			return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		}
		return threads
	case compiler.InstNoWordBoundary:
		if !isWordBoundary(input, curPos) {
			return addLight(prog, threads, seen, inst.Out, startPos, curPos, input)
		}
		return threads
	}
	return append(threads, lightThread{pc: pc, start: startPos})
}

func addThreadPooled(prog *compiler.Program, threads []thread, seen *bitset.BitSet, pc int, caps []CaptureSlot, input *Input, pos int) []thread {
	if seen.Has(pc) {
		return threads
	}
	seen.Set(pc)
	inst := &prog.Insts[pc]
	switch inst.Op {
	case compiler.InstJump:
		return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
	case compiler.InstSplit:
		threads = addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		threads = addThreadPooled(prog, threads, seen, inst.Out1, caps, input, pos)
		return threads
	case compiler.InstCapStart:
		newCaps := CopySlots(caps)
		if inst.Cap < len(newCaps) {
			newCaps[inst.Cap].Start = pos
		}
		return addThreadPooled(prog, threads, seen, inst.Out, newCaps, input, pos)
	case compiler.InstCapEnd:
		newCaps := CopySlots(caps)
		if inst.Cap < len(newCaps) {
			newCaps[inst.Cap].End = pos
		}
		return addThreadPooled(prog, threads, seen, inst.Out, newCaps, input, pos)
	case compiler.InstBeginLine:
		if pos == 0 {
			return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		}
		return threads
	case compiler.InstEndLine:
		if pos == input.Length() {
			return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		}
		return threads
	case compiler.InstBeginText:
		if pos == 0 {
			return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		}
		return threads
	case compiler.InstEndText:
		if pos == input.Length() {
			return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		}
		return threads
	case compiler.InstWordBoundary:
		if isWordBoundary(input, pos) {
			return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		}
		return threads
	case compiler.InstNoWordBoundary:
		if !isWordBoundary(input, pos) {
			return addThreadPooled(prog, threads, seen, inst.Out, caps, input, pos)
		}
		return threads
	}
	return append(threads, thread{pc: pc, captures: caps})
}

func matchRune(input, pattern rune, caseInsensitive bool) bool {
	if input == pattern {
		return true
	}
	if caseInsensitive {
		return runeutil.EqualFold(input, pattern)
	}
	return false
}

func matchRuneClass(r rune, inst *compiler.Inst) bool {
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
}

func isWordBoundary(input *Input, pos int) bool {
	var left, right bool
	if pos > 0 {
		r, ok := input.RuneAt(pos - 1)
		left = ok && runeutil.IsWordChar(r)
	}
	r, ok := input.RuneAt(pos)
	right = ok && runeutil.IsWordChar(r)
	return left != right
}
