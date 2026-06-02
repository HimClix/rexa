package engine

import (
	"errors"
	"unicode"

	"github.com/himclix/rexa/compiler"
	"github.com/himclix/rexa/internal/runeutil"
)

var ErrBacktrackLimit = errors.New("rexa: backtrack limit exceeded")

type BacktrackEngine struct {
	limit int64
}

func NewBacktrackEngine(limit int64) *BacktrackEngine {
	return &BacktrackEngine{limit: limit}
}

type btFrame struct {
	pc       int
	pos      int
	captures []CaptureSlot
	choice   int // for splits: 0 = first branch tried, 1 = second
}

type btRunner struct {
	prog  *compiler.Program
	input *Input
	stack []btFrame
	steps int64
	limit int64
	err   error
}

func (e *BacktrackEngine) Search(prog *compiler.Program, input *Input, startPos int) *MatchResult {
	for pos := startPos; pos <= input.Length(); pos++ {
		result := e.Match(prog, input, pos)
		if result.Matched {
			return result
		}
		if result.Err != nil {
			return result
		}
	}
	return &MatchResult{Matched: false}
}

func (e *BacktrackEngine) Match(prog *compiler.Program, input *Input, pos int) *MatchResult {
	r := &btRunner{
		prog:  prog,
		input: input,
		limit: e.limit,
	}

	caps := EmptySlots(prog.NumCap + 1)
	caps[0] = CaptureSlot{Start: pos, End: -1}

	r.push(prog.Start, pos, caps)

	for len(r.stack) > 0 {
		if r.limit > 0 {
			r.steps++
			if r.steps > r.limit {
				return &MatchResult{Matched: false, Err: ErrBacktrackLimit}
			}
		}

		frame := r.pop()
		pc := frame.pc
		sp := frame.pos
		caps := frame.captures

		inst := &prog.Insts[pc]

		switch inst.Op {
		case compiler.InstRune:
			ch, ok := input.RuneAt(sp)
			if ok && matchRuneBT(ch, inst.Rune, inst.CaseInsensitive) {
				r.push(inst.Out, sp+1, caps)
			}

		case compiler.InstRuneClass:
			ch, ok := input.RuneAt(sp)
			if ok && matchRuneClassBT(ch, inst) {
				r.push(inst.Out, sp+1, caps)
			}

		case compiler.InstAnyNotNL:
			ch, ok := input.RuneAt(sp)
			if ok && ch != '\n' {
				r.push(inst.Out, sp+1, caps)
			}

		case compiler.InstAnyChar:
			_, ok := input.RuneAt(sp)
			if ok {
				r.push(inst.Out, sp+1, caps)
			}

		case compiler.InstSplit:
			if inst.Greedy {
				r.push(inst.Out1, sp, caps)
				r.push(inst.Out, sp, caps)
			} else {
				r.push(inst.Out, sp, caps)
				r.push(inst.Out1, sp, caps)
			}

		case compiler.InstJump:
			r.push(inst.Out, sp, caps)

		case compiler.InstCapStart:
			newCaps := CopySlots(caps)
			if inst.Cap < len(newCaps) {
				newCaps[inst.Cap].Start = sp
			}
			r.push(inst.Out, sp, newCaps)

		case compiler.InstCapEnd:
			newCaps := CopySlots(caps)
			if inst.Cap < len(newCaps) {
				newCaps[inst.Cap].End = sp
			}
			r.push(inst.Out, sp, newCaps)

		case compiler.InstBackref:
			ref := inst.Ref
			if ref < len(caps) && caps[ref].Start >= 0 && caps[ref].End >= 0 {
				captured := input.Slice(caps[ref].Start, caps[ref].End)
				if matchBackref(input, sp, captured, inst.CaseInsensitive) {
					r.push(inst.Out, sp+len(captured), caps)
				}
			}

		case compiler.InstBeginLine:
			if sp == 0 {
				r.push(inst.Out, sp, caps)
			}

		case compiler.InstEndLine:
			if sp == input.Length() {
				r.push(inst.Out, sp, caps)
			}

		case compiler.InstBeginText:
			if sp == 0 {
				r.push(inst.Out, sp, caps)
			}

		case compiler.InstEndText:
			if sp == input.Length() {
				r.push(inst.Out, sp, caps)
			}

		case compiler.InstWordBoundary:
			if isWordBoundaryBT(input, sp) {
				r.push(inst.Out, sp, caps)
			}

		case compiler.InstNoWordBoundary:
			if !isWordBoundaryBT(input, sp) {
				r.push(inst.Out, sp, caps)
			}

		case compiler.InstMatch:
			finalCaps := CopySlots(caps)
			finalCaps[0].End = sp
			return &MatchResult{Matched: true, Captures: finalCaps}

		case compiler.InstLookaheadStart:
			matched := r.runLookahead(inst.Out, sp, caps)
			if inst.Negated {
				matched = !matched
			}
			if matched {
				r.push(inst.Out1, sp, caps) // continue at Out1, position unchanged
			}

		case compiler.InstLookbehindStart:
			matched := r.runLookbehind(inst.Out, sp, caps)
			if inst.Negated {
				matched = !matched
			}
			if matched {
				r.push(inst.Out1, sp, caps)
			}

		case compiler.InstAtomicStart:
			stackMark := len(r.stack)
			r.push(inst.Out, sp, caps)
			result := r.runUntilMatchOrExhaust(prog, input, stackMark)
			if result != nil && result.Matched {
				return result
			}

		case compiler.InstAtomicEnd:
			r.push(inst.Out, sp, caps)

		case compiler.InstFail:
			// do nothing, backtrack
		}
	}

	return &MatchResult{Matched: false}
}

func (r *btRunner) runLookahead(bodyStart int, pos int, caps []CaptureSlot) bool {
	sub := &btRunner{
		prog:  r.prog,
		input: r.input,
		limit: r.limit - r.steps,
	}
	sub.push(bodyStart, pos, caps)

	for len(sub.stack) > 0 {
		if sub.limit > 0 {
			r.steps++
			sub.steps++
			if sub.steps > sub.limit {
				r.err = ErrBacktrackLimit
				return false
			}
		}

		frame := sub.pop()
		inst := &r.prog.Insts[frame.pc]

		switch inst.Op {
		case compiler.InstMatch:
			return true
		case compiler.InstRune:
			ch, ok := r.input.RuneAt(frame.pos)
			if ok && matchRuneBT(ch, inst.Rune, inst.CaseInsensitive) {
				sub.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstRuneClass:
			ch, ok := r.input.RuneAt(frame.pos)
			if ok && matchRuneClassBT(ch, inst) {
				sub.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstAnyNotNL:
			ch, ok := r.input.RuneAt(frame.pos)
			if ok && ch != '\n' {
				sub.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstAnyChar:
			if _, ok := r.input.RuneAt(frame.pos); ok {
				sub.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstSplit:
			if inst.Greedy {
				sub.push(inst.Out1, frame.pos, frame.captures)
				sub.push(inst.Out, frame.pos, frame.captures)
			} else {
				sub.push(inst.Out, frame.pos, frame.captures)
				sub.push(inst.Out1, frame.pos, frame.captures)
			}
		case compiler.InstJump:
			sub.push(inst.Out, frame.pos, frame.captures)
		case compiler.InstCapStart:
			newCaps := CopySlots(frame.captures)
			if inst.Cap < len(newCaps) {
				newCaps[inst.Cap].Start = frame.pos
			}
			sub.push(inst.Out, frame.pos, newCaps)
		case compiler.InstCapEnd:
			newCaps := CopySlots(frame.captures)
			if inst.Cap < len(newCaps) {
				newCaps[inst.Cap].End = frame.pos
			}
			sub.push(inst.Out, frame.pos, newCaps)
		case compiler.InstBeginLine:
			prevR, prevOk := r.input.RuneAt(frame.pos - 1)
			if frame.pos == 0 || (prevOk && prevR == '\n') {
				sub.push(inst.Out, frame.pos, frame.captures)
			}
		case compiler.InstEndLine:
			ch, ok := r.input.RuneAt(frame.pos)
			if frame.pos == r.input.Length() || (ok && ch == '\n') {
				sub.push(inst.Out, frame.pos, frame.captures)
			}
		case compiler.InstBeginText:
			if frame.pos == 0 {
				sub.push(inst.Out, frame.pos, frame.captures)
			}
		case compiler.InstEndText:
			if frame.pos == r.input.Length() {
				sub.push(inst.Out, frame.pos, frame.captures)
			}
		case compiler.InstWordBoundary:
			if isWordBoundaryBT(r.input, frame.pos) {
				sub.push(inst.Out, frame.pos, frame.captures)
			}
		case compiler.InstNoWordBoundary:
			if !isWordBoundaryBT(r.input, frame.pos) {
				sub.push(inst.Out, frame.pos, frame.captures)
			}
		case compiler.InstBackref:
			ref := inst.Ref
			if ref < len(frame.captures) && frame.captures[ref].Start >= 0 && frame.captures[ref].End >= 0 {
				captured := r.input.Slice(frame.captures[ref].Start, frame.captures[ref].End)
				if matchBackref(r.input, frame.pos, captured, inst.CaseInsensitive) {
					sub.push(inst.Out, frame.pos+len(captured), frame.captures)
				}
			}
		case compiler.InstFail:
			// backtrack
		}
	}
	return false
}

func (r *btRunner) runLookbehind(bodyStart int, pos int, caps []CaptureSlot) bool {
	for tryPos := 0; tryPos <= pos; tryPos++ {
		sub := &btRunner{
			prog:  r.prog,
			input: r.input,
			limit: r.limit - r.steps,
		}
		sub.push(bodyStart, tryPos, caps)

		for len(sub.stack) > 0 {
			if sub.limit > 0 {
				r.steps++
				sub.steps++
				if sub.steps > sub.limit {
					r.err = ErrBacktrackLimit
					return false
				}
			}

			frame := sub.pop()
			inst := &r.prog.Insts[frame.pc]

			if inst.Op == compiler.InstMatch && frame.pos == pos {
				return true
			}

			switch inst.Op {
			case compiler.InstMatch:
				// matched but not at the right position
			case compiler.InstRune:
				ch, ok := r.input.RuneAt(frame.pos)
				if ok && matchRuneBT(ch, inst.Rune, inst.CaseInsensitive) {
					sub.push(inst.Out, frame.pos+1, frame.captures)
				}
			case compiler.InstRuneClass:
				ch, ok := r.input.RuneAt(frame.pos)
				if ok && matchRuneClassBT(ch, inst) {
					sub.push(inst.Out, frame.pos+1, frame.captures)
				}
			case compiler.InstAnyNotNL:
				ch, ok := r.input.RuneAt(frame.pos)
				if ok && ch != '\n' {
					sub.push(inst.Out, frame.pos+1, frame.captures)
				}
			case compiler.InstAnyChar:
				if _, ok := r.input.RuneAt(frame.pos); ok {
					sub.push(inst.Out, frame.pos+1, frame.captures)
				}
			case compiler.InstSplit:
				if inst.Greedy {
					sub.push(inst.Out1, frame.pos, frame.captures)
					sub.push(inst.Out, frame.pos, frame.captures)
				} else {
					sub.push(inst.Out, frame.pos, frame.captures)
					sub.push(inst.Out1, frame.pos, frame.captures)
				}
			case compiler.InstJump:
				sub.push(inst.Out, frame.pos, frame.captures)
			case compiler.InstFail:
				// backtrack
			}
		}
	}
	return false
}

func (r *btRunner) runUntilMatchOrExhaust(prog *compiler.Program, input *Input, stackMark int) *MatchResult {
	for len(r.stack) > stackMark {
		if r.limit > 0 {
			r.steps++
			if r.steps > r.limit {
				return &MatchResult{Matched: false, Err: ErrBacktrackLimit}
			}
		}

		frame := r.pop()
		inst := &prog.Insts[frame.pc]

		switch inst.Op {
		case compiler.InstMatch:
			finalCaps := CopySlots(frame.captures)
			finalCaps[0].End = frame.pos
			r.stack = r.stack[:stackMark] // cut all frames from atomic body
			return &MatchResult{Matched: true, Captures: finalCaps}
		case compiler.InstRune:
			ch, ok := input.RuneAt(frame.pos)
			if ok && matchRuneBT(ch, inst.Rune, inst.CaseInsensitive) {
				r.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstRuneClass:
			ch, ok := input.RuneAt(frame.pos)
			if ok && matchRuneClassBT(ch, inst) {
				r.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstAnyNotNL:
			ch, ok := input.RuneAt(frame.pos)
			if ok && ch != '\n' {
				r.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstAnyChar:
			if _, ok := input.RuneAt(frame.pos); ok {
				r.push(inst.Out, frame.pos+1, frame.captures)
			}
		case compiler.InstSplit:
			if inst.Greedy {
				r.push(inst.Out1, frame.pos, frame.captures)
				r.push(inst.Out, frame.pos, frame.captures)
			} else {
				r.push(inst.Out, frame.pos, frame.captures)
				r.push(inst.Out1, frame.pos, frame.captures)
			}
		case compiler.InstJump:
			r.push(inst.Out, frame.pos, frame.captures)
		case compiler.InstCapStart:
			newCaps := CopySlots(frame.captures)
			if inst.Cap < len(newCaps) {
				newCaps[inst.Cap].Start = frame.pos
			}
			r.push(inst.Out, frame.pos, newCaps)
		case compiler.InstCapEnd:
			newCaps := CopySlots(frame.captures)
			if inst.Cap < len(newCaps) {
				newCaps[inst.Cap].End = frame.pos
			}
			r.push(inst.Out, frame.pos, newCaps)
		case compiler.InstAtomicEnd:
			r.push(inst.Out, frame.pos, frame.captures)
		case compiler.InstFail:
			// backtrack
		}
	}
	return nil
}

func (r *btRunner) push(pc, pos int, caps []CaptureSlot) {
	r.stack = append(r.stack, btFrame{pc: pc, pos: pos, captures: caps})
}

func (r *btRunner) pop() btFrame {
	n := len(r.stack)
	f := r.stack[n-1]
	r.stack = r.stack[:n-1]
	return f
}

func matchRuneBT(input, pattern rune, caseInsensitive bool) bool {
	if input == pattern {
		return true
	}
	if caseInsensitive {
		return runeutil.EqualFold(input, pattern)
	}
	return false
}

func matchRuneClassBT(r rune, inst *compiler.Inst) bool {
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

func matchBackref(input *Input, pos int, captured []rune, caseInsensitive bool) bool {
	for i, r := range captured {
		ch, ok := input.RuneAt(pos + i)
		if !ok {
			return false
		}
		if caseInsensitive {
			if !runeutil.EqualFold(ch, r) {
				return false
			}
		} else if ch != r {
			return false
		}
	}
	return true
}

func isWordBoundaryBT(input *Input, pos int) bool {
	var left, right bool
	if pos > 0 {
		r, ok := input.RuneAt(pos - 1)
		left = ok && runeutil.IsWordChar(r)
	}
	r, ok := input.RuneAt(pos)
	right = ok && runeutil.IsWordChar(r)
	return left != right
}
