package syntax

type Op int

const (
	OpLiteral        Op = iota // single rune
	OpDot                      // . (any char except \n, or any char if DotAll)
	OpCharClass                // character class [abc], [a-z]
	OpConcat                   // AB
	OpAlternate                // A|B
	OpCapture                  // (...) capturing group
	OpGroup                    // (?:...) non-capturing group
	OpNamedCapture             // (?<name>...) named capturing group
	OpStar                     // A*
	OpPlus                     // A+
	OpQuest                    // A?
	OpRepeat                   // A{n,m}
	OpLazy                     // wraps quantifier for lazy mode
	OpPossessive               // wraps quantifier for possessive mode
	OpLookahead                // (?=A)
	OpNegLookahead             // (?!A)
	OpLookbehind               // (?<=A)
	OpNegLookbehind            // (?<!A)
	OpAtomic                   // (?>A)
	OpBackref                  // \1, \k<name>
	OpBeginLine                // ^
	OpEndLine                  // $
	OpBeginText                // \A
	OpEndText                  // \z
	OpWordBoundary             // \b
	OpNoWordBoundary           // \B
	OpEmpty                    // empty match
)

type CharRange struct {
	Lo, Hi rune
}

type Flags uint32

const (
	FlagCaseInsensitive Flags = 1 << iota
	FlagMultiline
	FlagDotAll
	FlagUngreedy
	FlagUnicode
)

type Node struct {
	Op           Op
	Children     []*Node
	Rune         rune        // for OpLiteral
	Ranges       []CharRange // for OpCharClass (explicit ranges)
	UnicodeTable interface{} // *unicode.RangeTable for Unicode property classes
	Min, Max     int         // for OpRepeat: {min,max}, Max == -1 means unbounded
	Cap          int         // capture group index (0 = whole match)
	Name         string      // for OpNamedCapture, OpBackref by name
	Ref          int         // for OpBackref by number
	Flags        Flags
	Negated      bool // for OpCharClass (negated class)
}

type Tree struct {
	Root     *Node
	NumCap   int
	CapNames map[string]int
}
