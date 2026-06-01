package engine

import "github.com/himclix/rexa/compiler"

type CaptureSlot struct {
	Start int
	End   int
}

func EmptySlots(n int) []CaptureSlot {
	slots := make([]CaptureSlot, n)
	for i := range slots {
		slots[i] = CaptureSlot{Start: -1, End: -1}
	}
	return slots
}

func CopySlots(src []CaptureSlot) []CaptureSlot {
	dst := make([]CaptureSlot, len(src))
	copy(dst, src)
	return dst
}

type MatchResult struct {
	Matched  bool
	Captures []CaptureSlot
	Err      error
}

type Engine interface {
	Match(prog *compiler.Program, input *Input, pos int) *MatchResult
	Search(prog *compiler.Program, input *Input, pos int) *MatchResult
}
