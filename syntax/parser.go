package syntax

import "unicode"

type Parser struct {
	tokens   []Token
	pos      int
	capIndex int
	capNames map[string]int
	pattern  string
	flags    Flags
}

func Parse(pattern string) (*Tree, error) {
	lex := NewLexer(pattern)
	tokens, err := lex.Tokenize()
	if err != nil {
		return nil, err
	}

	p := &Parser{
		tokens:   tokens,
		pattern:  pattern,
		capNames: make(map[string]int),
	}

	root, err := p.parseAlternate()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != TokEOF {
		return nil, syntaxErr(pattern, p.current().Pos, "unexpected token")
	}

	return &Tree{
		Root:     root,
		NumCap:   p.capIndex,
		CapNames: p.capNames,
	}, nil
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: TokEOF, Pos: len(p.pattern)}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	t := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}

func (p *Parser) expect(kind TokenKind) (Token, error) {
	t := p.current()
	if t.Kind != kind {
		return t, syntaxErr(p.pattern, t.Pos, "unexpected token")
	}
	p.advance()
	return t, nil
}

// Alternate -> Concat ('|' Concat)*
func (p *Parser) parseAlternate() (*Node, error) {
	left, err := p.parseConcat()
	if err != nil {
		return nil, err
	}

	if p.current().Kind != TokPipe {
		return left, nil
	}

	children := []*Node{left}
	for p.current().Kind == TokPipe {
		p.advance()
		child, err := p.parseConcat()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	return &Node{Op: OpAlternate, Children: children}, nil
}

// Concat -> Repeat*
func (p *Parser) parseConcat() (*Node, error) {
	var children []*Node

	for {
		t := p.current()
		if t.Kind == TokEOF || t.Kind == TokPipe || t.Kind == TokRParen {
			break
		}
		child, err := p.parseRepeat()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	if len(children) == 0 {
		return &Node{Op: OpEmpty}, nil
	}
	if len(children) == 1 {
		return children[0], nil
	}
	return &Node{Op: OpConcat, Children: children}, nil
}

// Repeat -> Atom Quantifier?
func (p *Parser) parseRepeat() (*Node, error) {
	atom, err := p.parseAtom()
	if err != nil {
		return nil, err
	}

	t := p.current()
	switch t.Kind {
	case TokStar:
		p.advance()
		node := &Node{Op: OpStar, Children: []*Node{atom}}
		return p.parseLazyOrPossessive(node), nil
	case TokPlus:
		p.advance()
		node := &Node{Op: OpPlus, Children: []*Node{atom}}
		return p.parseLazyOrPossessive(node), nil
	case TokQuestion:
		p.advance()
		node := &Node{Op: OpQuest, Children: []*Node{atom}}
		return p.parseLazyOrPossessive(node), nil
	case TokQuantifier:
		p.advance()
		node := &Node{Op: OpRepeat, Children: []*Node{atom}, Min: t.Min, Max: t.Max}
		return p.parseLazyOrPossessive(node), nil
	}

	return atom, nil
}

func (p *Parser) parseLazyOrPossessive(node *Node) *Node {
	if p.current().Kind == TokQuestion {
		p.advance()
		return &Node{Op: OpLazy, Children: []*Node{node}}
	}
	if p.current().Kind == TokPlus {
		p.advance()
		return &Node{Op: OpPossessive, Children: []*Node{node}}
	}
	return node
}

// Atom -> Literal | '.' | Group | CharClass | Anchor | Backref | CharClassShorthand
func (p *Parser) parseAtom() (*Node, error) {
	t := p.current()

	switch t.Kind {
	case TokLiteral:
		p.advance()
		if t.Value == `\b` {
			return &Node{Op: OpWordBoundary}, nil
		}
		if t.Value == `\B` {
			return &Node{Op: OpNoWordBoundary}, nil
		}
		if t.Value == `\A` {
			return &Node{Op: OpBeginText}, nil
		}
		if t.Value == `\z` {
			return &Node{Op: OpEndText}, nil
		}
		return &Node{Op: OpLiteral, Rune: t.Rune, Flags: p.flags}, nil

	case TokDot:
		p.advance()
		return &Node{Op: OpDot, Flags: p.flags}, nil

	case TokCaret:
		p.advance()
		return &Node{Op: OpBeginLine}, nil

	case TokDollar:
		p.advance()
		return &Node{Op: OpEndLine}, nil

	case TokCharClass:
		p.advance()
		return p.buildShorthandClass(t), nil

	case TokUnicodeClass:
		p.advance()
		return p.buildUnicodeClass(t)

	case TokBackref:
		p.advance()
		return &Node{Op: OpBackref, Ref: t.Ref, Name: t.Name}, nil

	case TokGroupOpen:
		return p.parseGroup()

	case TokLBracket:
		return p.parseCharClass()

	default:
		return nil, syntaxErr(p.pattern, t.Pos, "unexpected token in atom")
	}
}

func (p *Parser) buildShorthandClass(t Token) *Node {
	var ranges []CharRange
	switch t.Value {
	case `\d`, `\D`:
		ranges = []CharRange{{Lo: '0', Hi: '9'}}
	case `\w`, `\W`:
		ranges = []CharRange{
			{Lo: '0', Hi: '9'},
			{Lo: 'A', Hi: 'Z'},
			{Lo: '_', Hi: '_'},
			{Lo: 'a', Hi: 'z'},
		}
	case `\s`, `\S`:
		ranges = []CharRange{
			{Lo: '\t', Hi: '\t'},
			{Lo: '\n', Hi: '\n'},
			{Lo: '\f', Hi: '\f'},
			{Lo: '\r', Hi: '\r'},
			{Lo: ' ', Hi: ' '},
		}
	}
	return &Node{Op: OpCharClass, Ranges: ranges, Negated: t.Negated}
}

func (p *Parser) parseGroup() (*Node, error) {
	t := p.advance() // consume TokGroupOpen

	switch t.GroupKind {
	case GroupCapture:
		p.capIndex++
		cap := p.capIndex
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpCapture, Children: []*Node{inner}, Cap: cap}, nil

	case GroupNamed:
		p.capIndex++
		cap := p.capIndex
		p.capNames[t.Name] = cap
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpNamedCapture, Children: []*Node{inner}, Cap: cap, Name: t.Name}, nil

	case GroupNonCapture:
		oldFlags := p.flags
		if t.Value != "" {
			p.applyFlags(t.Value)
		}
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		p.flags = oldFlags
		return &Node{Op: OpGroup, Children: []*Node{inner}}, nil

	case GroupLookahead:
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpLookahead, Children: []*Node{inner}}, nil

	case GroupNegLookahead:
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpNegLookahead, Children: []*Node{inner}}, nil

	case GroupLookbehind:
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpLookbehind, Children: []*Node{inner}}, nil

	case GroupNegLookbehind:
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpNegLookbehind, Children: []*Node{inner}}, nil

	case GroupAtomic:
		inner, err := p.parseAlternate()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, syntaxErr(p.pattern, t.Pos, "unclosed group")
		}
		return &Node{Op: OpAtomic, Children: []*Node{inner}}, nil

	case GroupFlags:
		oldFlags := p.flags
		p.applyFlags(t.Value)
		_ = oldFlags
		return &Node{Op: OpEmpty}, nil

	default:
		return nil, syntaxErr(p.pattern, t.Pos, "unsupported group type")
	}
}

func (p *Parser) applyFlags(flags string) {
	negate := false
	for _, c := range flags {
		if c == '-' {
			negate = true
			continue
		}
		var f Flags
		switch c {
		case 'i':
			f = FlagCaseInsensitive
		case 'm':
			f = FlagMultiline
		case 's':
			f = FlagDotAll
		case 'U':
			f = FlagUngreedy
		case 'u':
			f = FlagUnicode
		}
		if negate {
			p.flags &^= f
		} else {
			p.flags |= f
		}
	}
}

func (p *Parser) buildUnicodeClass(t Token) (*Node, error) {
	rt, ok := LookupUnicodeClass(t.Value)
	if !ok {
		return nil, syntaxErr(p.pattern, t.Pos, "unknown unicode class: "+t.Value)
	}
	return &Node{Op: OpCharClass, UnicodeTable: rt, Negated: t.Negated}, nil
}

func (p *Parser) parseCharClass() (*Node, error) {
	// TokLBracket already consumed by scanToken which also scanned contents
	// The lexer emits: TokLBracket, then literals/charclasses inside, then TokRBracket
	// We already consumed TokLBracket in scanToken before calling scanCharClass,
	// but parseAtom sees TokLBracket. We need to skip it here.
	startTok := p.advance() // consume TokLBracket

	negated := false
	if p.current().Kind == TokLiteral && p.current().Rune == '^' {
		negated = true
		p.advance()
	}

	var ranges []CharRange
	for p.current().Kind != TokRBracket && p.current().Kind != TokEOF {
		t := p.advance()
		if t.Kind == TokCharClass {
			node := p.buildShorthandClass(t)
			ranges = append(ranges, node.Ranges...)
			continue
		}
		if t.Kind != TokLiteral {
			continue
		}
		lo := t.Rune
		if p.current().Kind == TokLiteral && p.current().Rune == '-' {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind != TokRBracket {
				p.advance() // consume '-'
				hiTok := p.advance()
				if hiTok.Kind == TokLiteral {
					ranges = append(ranges, CharRange{Lo: lo, Hi: hiTok.Rune})
					continue
				}
			}
		}
		ranges = append(ranges, CharRange{Lo: lo, Hi: lo})
	}

	if p.current().Kind != TokRBracket {
		return nil, syntaxErr(p.pattern, startTok.Pos, "unterminated character class")
	}

	// If case-insensitive, expand ranges to include both cases
	if p.flags&FlagCaseInsensitive != 0 {
		var expanded []CharRange
		for _, cr := range ranges {
			expanded = append(expanded, cr)
			// Add case-folded variants
			for r := cr.Lo; r <= cr.Hi; r++ {
				lo := unicode.ToLower(r)
				hi := unicode.ToUpper(r)
				if lo != r {
					expanded = append(expanded, CharRange{Lo: lo, Hi: lo})
				}
				if hi != r {
					expanded = append(expanded, CharRange{Lo: hi, Hi: hi})
				}
			}
		}
		ranges = expanded
	}
	p.advance() // consume TokRBracket

	return &Node{Op: OpCharClass, Ranges: ranges, Negated: negated}, nil
}
