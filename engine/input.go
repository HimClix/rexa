package engine

import "unicode/utf8"

type Input struct {
	runes []rune
	str   string
	len   int
	init  bool
	ascii bool // true if string is pure ASCII
}

func NewInputString(s string) *Input {
	inp := MakeInput(s)
	return &inp
}

func MakeInput(s string) Input {
	a := isASCII(s)
	l := len(s)
	if !a {
		l = utf8.RuneCountInString(s)
	}
	return Input{str: s, ascii: a, len: l}
}

func NewInputBytes(b []byte) *Input {
	return NewInputString(string(b))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func (inp *Input) ensureRunes() {
	if !inp.init {
		inp.runes = []rune(inp.str)
		inp.init = true
	}
}

func (inp *Input) RuneAt(pos int) (rune, bool) {
	if pos < 0 || pos >= inp.len {
		return 0, false
	}
	if inp.ascii {
		return rune(inp.str[pos]), true
	}
	inp.ensureRunes()
	return inp.runes[pos], true
}

func (inp *Input) Length() int {
	return inp.len
}

func (inp *Input) Slice(start, end int) []rune {
	if start < 0 {
		start = 0
	}
	if end > inp.len {
		end = inp.len
	}
	if start >= end {
		return nil
	}
	inp.ensureRunes()
	return inp.runes[start:end]
}

func (inp *Input) String() string {
	return inp.str
}

func (inp *Input) Runes() []rune {
	inp.ensureRunes()
	return inp.runes
}

func (inp *Input) SliceString(start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > inp.len {
		end = inp.len
	}
	if start >= end {
		return ""
	}
	if inp.ascii {
		return inp.str[start:end]
	}
	return string(inp.Slice(start, end))
}
