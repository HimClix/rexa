package engine

import (
	"strings"

	"github.com/himclix/rexa/compiler"
	"github.com/himclix/rexa/internal/bm"
)

type LiteralEngine struct {
	searcher   *bm.Searcher
	pattern    []rune
	patternStr string
	patLen     int
}

func NewLiteralEngine(literal string) *LiteralEngine {
	runes := []rune(literal)
	return &LiteralEngine{
		searcher:   bm.New(runes),
		pattern:    runes,
		patternStr: literal,
		patLen:     len(runes),
	}
}

// SearchString searches directly on a string without creating Input. Zero allocs.
func (e *LiteralEngine) SearchString(s string, pos int) (start, end int, ok bool) {
	idx := strings.Index(s[pos:], e.patternStr)
	if idx < 0 {
		return 0, 0, false
	}
	byteStart := pos + idx
	runeStart := byteStart // correct for ASCII
	runeEnd := runeStart + e.patLen
	return runeStart, runeEnd, true
}

// MatchString checks if the pattern matches at exactly position pos. Zero allocs for ASCII.
func (e *LiteralEngine) MatchString(s string, pos int) (end int, ok bool) {
	if pos+len(e.patternStr) > len(s) {
		return 0, false
	}
	if s[pos:pos+len(e.patternStr)] == e.patternStr {
		return pos + e.patLen, true
	}
	return 0, false
}

func (e *LiteralEngine) Match(prog *compiler.Program, input *Input, pos int) *MatchResult {
	end, ok := e.MatchString(input.String(), pos)
	if !ok {
		return &MatchResult{Matched: false}
	}
	return &MatchResult{
		Matched:  true,
		Captures: []CaptureSlot{{Start: pos, End: end}},
	}
}

func (e *LiteralEngine) Search(prog *compiler.Program, input *Input, pos int) *MatchResult {
	start, end, ok := e.SearchString(input.String(), pos)
	if !ok {
		return &MatchResult{Matched: false}
	}
	return &MatchResult{
		Matched:  true,
		Captures: []CaptureSlot{{Start: start, End: end}},
	}
}
