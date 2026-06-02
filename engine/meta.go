package engine

import (
	"github.com/himclix/rexa/compiler"
	"github.com/himclix/rexa/internal/bm"
)

type EngineKind int

const (
	EngineLiteral EngineKind = iota
	EngineLazyDFA
	EnginePikeVM
	EngineBacktrack
)

func (k EngineKind) String() string {
	switch k {
	case EngineLiteral:
		return "literal"
	case EngineLazyDFA:
		return "lazydfa"
	case EnginePikeVM:
		return "pikevm"
	case EngineBacktrack:
		return "backtrack"
	default:
		return "unknown"
	}
}

type MetaEngine struct {
	literal      *LiteralEngine
	lazydfa      *LazyDFA
	pikevm       *PikeVM
	backtrack    *BacktrackEngine
	prefilter    *bm.Searcher
	prefix       []rune
	prog         *compiler.Program
	useDFA       bool
	useBacktrack bool
	Used         EngineKind
	// Pre-allocated result slots to avoid per-call allocation
	cachedResult  MatchResult
	cachedCaps    [8]CaptureSlot
	noMatchResult MatchResult
}

func NewMetaEngine(prog *compiler.Program) *MetaEngine {
	return NewMetaEngineWithOptions(prog, 1_000_000, 0)
}

func NewMetaEngineWithOptions(prog *compiler.Program, backtrackLimit int64, cacheCapacity int) *MetaEngine {
	me := &MetaEngine{
		pikevm: NewPikeVM(),
		prog:   prog,
	}

	if prog.IsLiteral && prog.PrefixComplete && prog.LiteralPrefix != "" {
		me.literal = NewLiteralEngine(prog.LiteralPrefix)
	}

	if prog.LiteralPrefix != "" && !prog.PrefixComplete {
		runes := []rune(prog.LiteralPrefix)
		me.prefilter = bm.New(runes)
		me.prefix = runes
	}

	if prog.NeedsBacktrack {
		me.backtrack = NewBacktrackEngine(backtrackLimit)
		me.useBacktrack = true
	} else if !prog.NeedsLookaround && !prog.HasWordBoundary() {
		me.lazydfa = NewLazyDFA(prog, cacheCapacity)
		me.useDFA = true
	}

	return me
}

func (me *MetaEngine) Literal() *LiteralEngine {
	return me.literal
}

func (me *MetaEngine) PikeVM() *PikeVM {
	return me.pikevm
}

func (me *MetaEngine) SearchBool(prog *compiler.Program, input Input, pos int) bool {
	if me.useDFA && !me.lazydfa.Abandoned() {
		me.Used = EngineLazyDFA
		matched, abandoned := me.lazydfa.SearchBool(prog, input, pos)
		if !abandoned {
			return matched
		}
	}
	if me.useBacktrack {
		inp := NewInputString(input.String())
		return me.backtrack.Search(prog, inp, pos).Matched
	}
	return me.pikevm.SearchBool(prog, input, pos)
}

// SearchRange returns match boundaries without allocating MatchResult.
func (me *MetaEngine) SearchRange(prog *compiler.Program, input Input, pos int) (start, end int, ok bool) {
	if me.useBacktrack {
		inp := NewInputString(input.String())
		r := me.backtrack.Search(prog, inp, pos)
		if !r.Matched || len(r.Captures) == 0 {
			return 0, 0, false
		}
		me.Used = EngineBacktrack
		return r.Captures[0].Start, r.Captures[0].End, true
	}
	if me.useDFA && !me.lazydfa.Abandoned() {
		me.Used = EngineLazyDFA
		matched, abandoned := me.lazydfa.SearchBool(prog, input, pos)
		if abandoned {
			goto pikevm
		}
		if !matched {
			return 0, 0, false
		}
		// DFA confirmed match exists — find exact boundaries via DFA
		inp := &input
		for startPos := pos; startPos <= inp.Length(); startPos++ {
			m, s, e := me.lazydfa.matchAt(inp, startPos)
			if me.lazydfa.abandoned {
				goto pikevm
			}
			if m {
				// Re-run Pike VM at this position for correct boundaries
				me.pikevm.initPool(len(prog.Insts))
				result := me.pikevm.matchAt(prog, inp, s)
				if result.Matched && len(result.Captures) > 0 {
					return result.Captures[0].Start, result.Captures[0].End, true
				}
				return s, e, true
			}
		}
		return 0, 0, false
	}
pikevm:
	me.Used = EnginePikeVM
	inp := &input
	result := me.pikevm.Search(prog, inp, pos)
	if !result.Matched || len(result.Captures) == 0 {
		return 0, 0, false
	}
	return result.Captures[0].Start, result.Captures[0].End, true
}

func (me *MetaEngine) Search(prog *compiler.Program, input *Input, pos int) *MatchResult {
	if me.literal != nil {
		me.Used = EngineLiteral
		return me.literal.Search(prog, input, pos)
	}

	if me.useBacktrack {
		me.Used = EngineBacktrack
		return me.backtrack.Search(prog, input, pos)
	}

	// DFA finds match position, Pike VM gives correct boundaries
	if me.useDFA && !me.lazydfa.Abandoned() {
		me.Used = EngineLazyDFA
		matched, abandoned := me.lazydfa.SearchInto(prog, input, pos, &me.cachedResult, me.cachedCaps[:])
		if abandoned {
			// fall through to Pike VM
		} else if !matched {
			me.noMatchResult.Matched = false
			me.noMatchResult.Captures = nil
			return &me.noMatchResult
		} else {
			startPos := me.cachedResult.Captures[0].Start
			return me.pikevm.Match(prog, input, startPos)
		}
	}

	if me.prefilter != nil {
		return me.searchWithPrefilter(prog, input, pos)
	}

	me.Used = EnginePikeVM
	return me.pikevm.Search(prog, input, pos)
}

func (me *MetaEngine) Match(prog *compiler.Program, input *Input, pos int) *MatchResult {
	if me.literal != nil {
		me.Used = EngineLiteral
		return me.literal.Match(prog, input, pos)
	}

	if me.useBacktrack {
		me.Used = EngineBacktrack
		return me.backtrack.Match(prog, input, pos)
	}

	if me.useDFA && !me.lazydfa.Abandoned() {
		me.Used = EngineLazyDFA
		result := me.lazydfa.Match(prog, input, pos)
		if result != nil {
			return result
		}
	}

	me.Used = EnginePikeVM
	return me.pikevm.Match(prog, input, pos)
}

func (me *MetaEngine) searchWithPrefilter(prog *compiler.Program, input *Input, pos int) *MatchResult {
	me.Used = EnginePikeVM

	for pos <= input.Length() {
		idx := me.prefilter.Search(input.Runes(), pos)
		if idx < 0 {
			return &MatchResult{Matched: false}
		}

		result := me.pikevm.Match(prog, input, idx)
		if result.Matched {
			return result
		}

		pos = idx + 1
	}

	return &MatchResult{Matched: false}
}
