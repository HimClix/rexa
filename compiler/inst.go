package compiler

type OpCode int

const (
	InstRune            OpCode = iota // match single rune
	InstRuneClass                     // match rune against ranges
	InstAnyNotNL                      // match any char except \n
	InstAnyChar                       // match any char (dotall)
	InstSplit                         // NFA split: try Out and Out1
	InstJump                          // unconditional jump to Out
	InstMatch                         // successful match
	InstFail                          // forced failure
	InstCapStart                      // begin capture group
	InstCapEnd                        // end capture group
	InstBeginLine                     // ^
	InstEndLine                       // $
	InstBeginText                     // \A
	InstEndText                       // \z
	InstWordBoundary                  // \b
	InstNoWordBoundary                // \B
	InstLookaheadStart                // begin lookahead
	InstLookaheadEnd                  // end lookahead
	InstLookbehindStart               // begin lookbehind
	InstLookbehindEnd                 // end lookbehind
	InstBackref                       // match captured group content
	InstAtomicStart                   // begin atomic group
	InstAtomicEnd                     // end atomic group
)

type Inst struct {
	Op              OpCode
	Out             int         // primary successor
	Out1            int         // secondary successor (for InstSplit)
	Rune            rune        // for InstRune
	Ranges          []Range     // for InstRuneClass
	UnicodeTable    interface{} // *unicode.RangeTable for Unicode property classes
	Cap             int         // for InstCapStart/End
	Ref             int         // for InstBackref
	Greedy          bool        // for InstSplit: prefer Out (greedy) vs Out1 (lazy)
	Negated         bool        // for InstRuneClass: negated class
	CaseInsensitive bool        // for InstRune/InstRuneClass: (?i) mode
}

type Range struct {
	Lo, Hi rune
}
