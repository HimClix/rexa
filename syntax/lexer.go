package syntax

import "unicode/utf8"

type Lexer struct {
	pattern string
	runes   []rune
	pos     int
	tokens  []Token
}

func NewLexer(pattern string) *Lexer {
	return &Lexer{
		pattern: pattern,
		runes:   []rune(pattern),
	}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	for l.pos < len(l.runes) {
		if err := l.scanToken(); err != nil {
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Kind: TokEOF, Pos: len(l.pattern)})
	return l.tokens, nil
}

func (l *Lexer) peek() (rune, bool) {
	if l.pos >= len(l.runes) {
		return 0, false
	}
	return l.runes[l.pos], true
}

func (l *Lexer) advance() rune {
	r := l.runes[l.pos]
	l.pos++
	return r
}

func (l *Lexer) bytePos() int {
	n := 0
	for i := 0; i < l.pos && n < len(l.pattern); i++ {
		_, size := utf8.DecodeRuneInString(l.pattern[n:])
		n += size
	}
	return n
}

func (l *Lexer) emit(tok Token) {
	l.tokens = append(l.tokens, tok)
}

func (l *Lexer) scanToken() error {
	bp := l.bytePos()
	r := l.advance()

	switch r {
	case '\\':
		return l.scanEscape(bp)
	case '.':
		l.emit(Token{Kind: TokDot, Pos: bp})
	case '^':
		l.emit(Token{Kind: TokCaret, Pos: bp})
	case '$':
		l.emit(Token{Kind: TokDollar, Pos: bp})
	case '*':
		l.emit(Token{Kind: TokStar, Pos: bp})
	case '+':
		l.emit(Token{Kind: TokPlus, Pos: bp})
	case '?':
		l.emit(Token{Kind: TokQuestion, Pos: bp})
	case '|':
		l.emit(Token{Kind: TokPipe, Pos: bp})
	case '(':
		return l.scanGroupOpen(bp)
	case ')':
		l.emit(Token{Kind: TokRParen, Pos: bp})
	case '[':
		l.emit(Token{Kind: TokLBracket, Pos: bp})
		return l.scanCharClass(bp)
	case '{':
		return l.scanQuantifier(bp)
	default:
		l.emit(Token{Kind: TokLiteral, Rune: r, Pos: bp})
	}
	return nil
}

func (l *Lexer) scanEscape(bp int) error {
	r, ok := l.peek()
	if !ok {
		return syntaxErr(l.pattern, bp, "trailing backslash")
	}
	l.advance()

	switch r {
	case 'n':
		l.emit(Token{Kind: TokLiteral, Rune: '\n', Pos: bp})
	case 't':
		l.emit(Token{Kind: TokLiteral, Rune: '\t', Pos: bp})
	case 'r':
		l.emit(Token{Kind: TokLiteral, Rune: '\r', Pos: bp})
	case 'f':
		l.emit(Token{Kind: TokLiteral, Rune: '\f', Pos: bp})
	case 'a':
		l.emit(Token{Kind: TokLiteral, Rune: '\a', Pos: bp})
	case 'e':
		l.emit(Token{Kind: TokLiteral, Rune: 0x1B, Pos: bp})
	case 'd':
		l.emit(Token{Kind: TokCharClass, Value: `\d`, Pos: bp})
	case 'D':
		l.emit(Token{Kind: TokCharClass, Value: `\D`, Negated: true, Pos: bp})
	case 'w':
		l.emit(Token{Kind: TokCharClass, Value: `\w`, Pos: bp})
	case 'W':
		l.emit(Token{Kind: TokCharClass, Value: `\W`, Negated: true, Pos: bp})
	case 's':
		l.emit(Token{Kind: TokCharClass, Value: `\s`, Pos: bp})
	case 'S':
		l.emit(Token{Kind: TokCharClass, Value: `\S`, Negated: true, Pos: bp})
	case 'b':
		l.emit(Token{Kind: TokLiteral, Value: `\b`, Rune: -1, Pos: bp})
	case 'B':
		l.emit(Token{Kind: TokLiteral, Value: `\B`, Rune: -2, Pos: bp})
	case 'A':
		l.emit(Token{Kind: TokLiteral, Value: `\A`, Rune: -3, Pos: bp})
	case 'z':
		l.emit(Token{Kind: TokLiteral, Value: `\z`, Rune: -4, Pos: bp})
	case 'p', 'P':
		negated := r == 'P'
		return l.scanUnicodeClass(bp, negated)
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.emit(Token{Kind: TokBackref, Ref: int(r - '0'), Pos: bp})
	case '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '\\', '^', '$':
		l.emit(Token{Kind: TokLiteral, Rune: r, Pos: bp})
	default:
		l.emit(Token{Kind: TokLiteral, Rune: r, Pos: bp})
	}
	return nil
}

func (l *Lexer) scanUnicodeClass(bp int, negated bool) error {
	r, ok := l.peek()
	if !ok {
		return syntaxErr(l.pattern, bp, "incomplete unicode class")
	}
	if r == '{' {
		l.advance()
		var name []rune
		for {
			r, ok := l.peek()
			if !ok {
				return syntaxErr(l.pattern, bp, "unterminated unicode class")
			}
			l.advance()
			if r == '}' {
				break
			}
			name = append(name, r)
		}
		if len(name) == 0 {
			return syntaxErr(l.pattern, bp, "empty unicode class name")
		}
		l.emit(Token{Kind: TokUnicodeClass, Value: string(name), Negated: negated, Pos: bp})
		return nil
	}
	// Single letter shorthand: \pL
	l.advance()
	l.emit(Token{Kind: TokUnicodeClass, Value: string(r), Negated: negated, Pos: bp})
	return nil
}

func (l *Lexer) scanGroupOpen(bp int) error {
	r, ok := l.peek()
	if !ok || r != '?' {
		l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupCapture, Pos: bp})
		return nil
	}
	l.advance() // consume '?'

	r, ok = l.peek()
	if !ok {
		return syntaxErr(l.pattern, bp, "incomplete group")
	}

	switch r {
	case ':':
		l.advance()
		l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupNonCapture, Pos: bp})
	case '=':
		l.advance()
		l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupLookahead, Pos: bp})
	case '!':
		l.advance()
		l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupNegLookahead, Pos: bp})
	case '<':
		l.advance()
		r2, ok2 := l.peek()
		if !ok2 {
			return syntaxErr(l.pattern, bp, "incomplete group")
		}
		if r2 == '=' {
			l.advance()
			l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupLookbehind, Pos: bp})
		} else if r2 == '!' {
			l.advance()
			l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupNegLookbehind, Pos: bp})
		} else {
			name, err := l.scanGroupName('>')
			if err != nil {
				return err
			}
			l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupNamed, Name: name, Pos: bp})
		}
	case 'P':
		l.advance()
		r2, ok2 := l.peek()
		if !ok2 || r2 != '<' {
			return syntaxErr(l.pattern, bp, "expected '<' after (?P")
		}
		l.advance()
		name, err := l.scanGroupName('>')
		if err != nil {
			return err
		}
		l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupNamed, Name: name, Pos: bp})
	case '>':
		l.advance()
		l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupAtomic, Pos: bp})
	default:
		if isFlag(r) {
			flags := l.scanFlags()
			r2, ok2 := l.peek()
			if ok2 && r2 == ')' {
				l.advance()
				l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupFlags, Value: flags, Pos: bp})
			} else if ok2 && r2 == ':' {
				l.advance()
				l.emit(Token{Kind: TokGroupOpen, GroupKind: GroupNonCapture, Value: flags, Pos: bp})
			} else {
				return syntaxErr(l.pattern, bp, "invalid flag group")
			}
			return nil
		}
		return syntaxErr(l.pattern, bp, "unknown group type")
	}
	return nil
}

func isFlag(r rune) bool {
	return r == 'i' || r == 'm' || r == 's' || r == 'U' || r == 'u' || r == '-'
}

func (l *Lexer) scanFlags() string {
	var flags []rune
	for {
		r, ok := l.peek()
		if !ok || !isFlag(r) {
			break
		}
		flags = append(flags, r)
		l.advance()
	}
	return string(flags)
}

func (l *Lexer) scanGroupName(closer rune) (string, error) {
	var name []rune
	for {
		r, ok := l.peek()
		if !ok {
			return "", syntaxErr(l.pattern, l.bytePos(), "unterminated group name")
		}
		l.advance()
		if r == closer {
			break
		}
		name = append(name, r)
	}
	if len(name) == 0 {
		return "", syntaxErr(l.pattern, l.bytePos(), "empty group name")
	}
	return string(name), nil
}

func (l *Lexer) scanCharClass(startBp int) error {
	for {
		r, ok := l.peek()
		if !ok {
			return syntaxErr(l.pattern, startBp, "unterminated character class")
		}
		bp := l.bytePos()
		l.advance()
		if r == ']' {
			l.emit(Token{Kind: TokRBracket, Pos: bp})
			return nil
		}
		if r == '\\' {
			if err := l.scanEscape(bp); err != nil {
				return err
			}
		} else {
			l.emit(Token{Kind: TokLiteral, Rune: r, Pos: bp})
		}
	}
}

func (l *Lexer) scanQuantifier(bp int) error {
	start := l.pos
	min, max := 0, 0
	hasComma := false

	min, ok := l.scanInt()
	if !ok {
		l.pos = start
		l.emit(Token{Kind: TokLiteral, Rune: '{', Pos: bp})
		return nil
	}

	r, peek := l.peek()
	if peek && r == '}' {
		l.advance()
		l.emit(Token{Kind: TokQuantifier, Min: min, Max: min, Pos: bp})
		return nil
	}

	if peek && r == ',' {
		l.advance()
		hasComma = true
		r, peek = l.peek()
		if peek && r == '}' {
			l.advance()
			l.emit(Token{Kind: TokQuantifier, Min: min, Max: -1, Pos: bp})
			return nil
		}
		max, ok = l.scanInt()
		if !ok {
			l.pos = start
			l.emit(Token{Kind: TokLiteral, Rune: '{', Pos: bp})
			return nil
		}
		_ = hasComma
		r, peek = l.peek()
		if peek && r == '}' {
			l.advance()
			l.emit(Token{Kind: TokQuantifier, Min: min, Max: max, Pos: bp})
			return nil
		}
	}

	l.pos = start
	l.emit(Token{Kind: TokLiteral, Rune: '{', Pos: bp})
	return nil
}

func (l *Lexer) scanInt() (int, bool) {
	n := 0
	found := false
	for {
		r, ok := l.peek()
		if !ok || r < '0' || r > '9' {
			return n, found
		}
		l.advance()
		n = n*10 + int(r-'0')
		found = true
	}
}
