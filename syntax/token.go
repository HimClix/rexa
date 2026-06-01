package syntax

type TokenKind int

const (
	TokLiteral   TokenKind = iota // single rune: 'a', '\n', '\x41'
	TokDot                        // '.'
	TokCaret                      // '^'
	TokDollar                     // '$'
	TokStar                       // '*'
	TokPlus                       // '+'
	TokQuestion                   // '?'
	TokPipe                       // '|'
	TokLParen                     // '('
	TokRParen                     // ')'
	TokLBracket                   // '['
	TokRBracket                   // ']'
	TokLBrace                     // '{'
	TokRBrace                     // '}'
	TokBackref                    // \1..\99, \k<name>
	TokGroupOpen                  // (?:, (?=, (?!, (?<=, (?<!, (?<name>, (?P<name>
	TokQuantifier                 // {n}, {n,}, {n,m}
	TokCharClass                  // \d, \w, \s, \D, \W, \S
	TokUnicodeClass               // \p{L}, \P{Lu}
	TokEOF
)

type GroupKind int

const (
	GroupCapture       GroupKind = iota // (
	GroupNonCapture                     // (?:
	GroupNamed                          // (?<name> or (?P<name>
	GroupLookahead                      // (?=
	GroupNegLookahead                   // (?!
	GroupLookbehind                     // (?<=
	GroupNegLookbehind                  // (?<!
	GroupAtomic                        // (?>
	GroupFlags                         // (?i), (?mi-s)
)

type Token struct {
	Kind      TokenKind
	Rune      rune      // for TokLiteral
	Value     string    // raw text of the token
	Min, Max  int       // for TokQuantifier: {Min,Max}, Max == -1 means unbounded
	GroupKind GroupKind  // for TokGroupOpen
	Name      string    // for named groups, named backrefs
	Ref       int       // for TokBackref by number
	Negated   bool      // for TokCharClass/TokUnicodeClass (uppercase variants)
	Pos       int       // byte offset in original pattern
}
