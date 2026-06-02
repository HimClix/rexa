package rexa

import (
	"io"
	"strings"

	"github.com/himclix/rexa/compiler"
	"github.com/himclix/rexa/engine"
	"github.com/himclix/rexa/syntax"
)

type Regexp struct {
	pattern string
	prog    *compiler.Program
	tree    *syntax.Tree
	eng     engine.Engine
	meta    *engine.MetaEngine
	longest bool
}

func Compile(expr string) (*Regexp, error) {
	return CompileWithOptions(expr, CompileOptions{})
}

func CompileWithOptions(expr string, opts CompileOptions) (*Regexp, error) {
	tree, err := syntax.Parse(expr)
	if err != nil {
		return nil, err
	}
	prog, err := compiler.Compile(tree)
	if err != nil {
		return nil, err
	}
	compiler.Optimize(prog)

	btLimit := int64(DefaultBacktrackLimit)
	if opts.BacktrackLimit > 0 {
		btLimit = opts.BacktrackLimit
	} else if opts.BacktrackLimit < 0 {
		btLimit = -1
	}
	meta := engine.NewMetaEngineWithOptions(prog, btLimit, opts.CacheCapacity)

	return &Regexp{pattern: expr, prog: prog, tree: tree, eng: meta, meta: meta}, nil
}

func MustCompile(expr string) *Regexp {
	re, err := Compile(expr)
	if err != nil {
		panic("rexa: Compile(" + expr + "): " + err.Error())
	}
	return re
}

func MustCompileWithOptions(expr string, opts CompileOptions) *Regexp {
	re, err := CompileWithOptions(expr, opts)
	if err != nil {
		panic("rexa: Compile(" + expr + "): " + err.Error())
	}
	return re
}

func (re *Regexp) EngineUsed() string {
	if re.meta == nil {
		return "unknown"
	}
	return re.meta.Used.String()
}

func (re *Regexp) String() string { return re.pattern }
func (re *Regexp) NumSubexp() int { return re.prog.NumCap }
func (re *Regexp) Copy() *Regexp  { c := *re; return &c }
func (re *Regexp) Longest()       { re.longest = true }

func (re *Regexp) LiteralPrefix() (prefix string, complete bool) {
	return re.prog.LiteralPrefix, re.prog.PrefixComplete
}

func (re *Regexp) SubexpNames() []string {
	names := make([]string, re.prog.NumCap+1)
	for name, idx := range re.prog.CapNames {
		if idx < len(names) {
			names[idx] = name
		}
	}
	return names
}

func (re *Regexp) SubexpIndex(name string) int {
	idx, ok := re.prog.CapNames[name]
	if !ok {
		return -1
	}
	return idx
}

func (re *Regexp) search(input *engine.Input, pos int) *engine.MatchResult {
	return re.eng.Search(re.prog, input, pos)
}

// --- Match ---

func (re *Regexp) MatchString(s string) bool {
	if lit := re.meta.Literal(); lit != nil {
		_, _, ok := lit.SearchString(s, 0)
		return ok
	}
	return re.meta.SearchBool(re.prog, engine.MakeInput(s), 0)
}

func (re *Regexp) Match(b []byte) bool {
	return re.MatchString(string(b))
}

func (re *Regexp) MatchReader(r io.RuneReader) bool {
	var runes []rune
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		}
		runes = append(runes, ch)
	}
	return re.MatchString(string(runes))
}

// --- Find ---

func (re *Regexp) Find(b []byte) []byte {
	s := re.FindString(string(b))
	if s == "" {
		return nil
	}
	return []byte(s)
}

func (re *Regexp) FindString(s string) string {
	if lit := re.meta.Literal(); lit != nil {
		start, end, ok := lit.SearchString(s, 0)
		if !ok {
			return ""
		}
		return s[start:end]
	}
	input := engine.NewInputString(s)
	r := re.search(input, 0)
	if !r.Matched || len(r.Captures) == 0 {
		return ""
	}
	c := r.Captures[0]
	if c.Start < 0 || c.End < 0 {
		return ""
	}
	return input.SliceString(c.Start, c.End)
}

func (re *Regexp) FindIndex(b []byte) []int {
	return re.FindStringIndex(string(b))
}

func (re *Regexp) FindStringIndex(s string) []int {
	input := engine.NewInputString(s)
	r := re.search(input, 0)
	if !r.Matched || len(r.Captures) == 0 {
		return nil
	}
	c := r.Captures[0]
	if c.Start < 0 || c.End < 0 {
		return nil
	}
	return []int{c.Start, c.End}
}

func (re *Regexp) FindReaderIndex(r io.RuneReader) []int {
	var runes []rune
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		}
		runes = append(runes, ch)
	}
	return re.FindStringIndex(string(runes))
}

// --- FindSubmatch ---

func (re *Regexp) FindSubmatch(b []byte) [][]byte {
	ss := re.FindStringSubmatch(string(b))
	if ss == nil {
		return nil
	}
	out := make([][]byte, len(ss))
	for i, s := range ss {
		out[i] = []byte(s)
	}
	return out
}

func (re *Regexp) FindStringSubmatch(s string) []string {
	input := engine.NewInputString(s)
	r := re.search(input, 0)
	if !r.Matched {
		return nil
	}
	out := make([]string, len(r.Captures))
	for i, c := range r.Captures {
		if c.Start >= 0 && c.End >= 0 {
			out[i] = input.SliceString(c.Start, c.End)
		}
	}
	return out
}

func (re *Regexp) FindSubmatchIndex(b []byte) []int {
	return re.FindStringSubmatchIndex(string(b))
}

func (re *Regexp) FindStringSubmatchIndex(s string) []int {
	input := engine.NewInputString(s)
	r := re.search(input, 0)
	if !r.Matched {
		return nil
	}
	out := make([]int, len(r.Captures)*2)
	for i, c := range r.Captures {
		out[2*i] = c.Start
		out[2*i+1] = c.End
	}
	return out
}

func (re *Regexp) FindReaderSubmatchIndex(r io.RuneReader) []int {
	var runes []rune
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			break
		}
		runes = append(runes, ch)
	}
	return re.FindStringSubmatchIndex(string(runes))
}

// --- FindAll ---

func (re *Regexp) findAll(input *engine.Input, n int, fn func(*engine.MatchResult)) {
	pos := 0
	count := 0
	prevMatchEnd := -1
	for pos <= input.Length() && (n < 0 || count < n) {
		r := re.search(input, pos)
		if !r.Matched || len(r.Captures) == 0 {
			break
		}
		c := r.Captures[0]
		if c.Start < 0 || c.End < 0 {
			break
		}
		if c.Start == c.End && c.Start == prevMatchEnd {
			pos++
			continue
		}
		fn(r)
		count++
		prevMatchEnd = c.End
		if c.End > pos {
			pos = c.End
		} else {
			pos++
		}
	}
}

func (re *Regexp) FindAll(b []byte, n int) [][]byte {
	var out [][]byte
	input := engine.NewInputBytes(b)
	re.findAll(input, n, func(r *engine.MatchResult) {
		c := r.Captures[0]
		out = append(out, []byte(input.SliceString(c.Start, c.End)))
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (re *Regexp) FindAllString(s string, n int) []string {
	var out []string
	input := engine.NewInputString(s)
	re.findAll(input, n, func(r *engine.MatchResult) {
		c := r.Captures[0]
		out = append(out, input.SliceString(c.Start, c.End))
	})
	return out
}

func (re *Regexp) FindAllIndex(b []byte, n int) [][]int {
	var out [][]int
	input := engine.NewInputBytes(b)
	re.findAll(input, n, func(r *engine.MatchResult) {
		c := r.Captures[0]
		out = append(out, []int{c.Start, c.End})
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (re *Regexp) FindAllStringIndex(s string, n int) [][]int {
	var out [][]int
	input := engine.NewInputString(s)
	re.findAll(input, n, func(r *engine.MatchResult) {
		c := r.Captures[0]
		out = append(out, []int{c.Start, c.End})
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (re *Regexp) FindAllSubmatch(b []byte, n int) [][][]byte {
	var out [][][]byte
	input := engine.NewInputBytes(b)
	re.findAll(input, n, func(r *engine.MatchResult) {
		row := make([][]byte, len(r.Captures))
		for i, c := range r.Captures {
			if c.Start >= 0 && c.End >= 0 {
				row[i] = []byte(input.SliceString(c.Start, c.End))
			}
		}
		out = append(out, row)
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (re *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int {
	var out [][]int
	input := engine.NewInputBytes(b)
	re.findAll(input, n, func(r *engine.MatchResult) {
		row := make([]int, len(r.Captures)*2)
		for i, c := range r.Captures {
			row[2*i] = c.Start
			row[2*i+1] = c.End
		}
		out = append(out, row)
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (re *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	var out [][]string
	input := engine.NewInputString(s)
	re.findAll(input, n, func(r *engine.MatchResult) {
		row := make([]string, len(r.Captures))
		for i, c := range r.Captures {
			if c.Start >= 0 && c.End >= 0 {
				row[i] = input.SliceString(c.Start, c.End)
			}
		}
		out = append(out, row)
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func (re *Regexp) FindAllStringSubmatchIndex(s string, n int) [][]int {
	var out [][]int
	input := engine.NewInputString(s)
	re.findAll(input, n, func(r *engine.MatchResult) {
		row := make([]int, len(r.Captures)*2)
		for i, c := range r.Captures {
			row[2*i] = c.Start
			row[2*i+1] = c.End
		}
		out = append(out, row)
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

// --- Replace ---

func (re *Regexp) replaceAll(input *engine.Input, repl func(*engine.MatchResult) string) string {
	var buf strings.Builder
	pos := 0
	prevMatchEnd := -1
	for pos <= input.Length() {
		r := re.search(input, pos)
		if !r.Matched || len(r.Captures) == 0 {
			break
		}
		c := r.Captures[0]
		if c.Start < 0 || c.End < 0 {
			break
		}
		if c.Start == c.End && c.Start == prevMatchEnd {
			if pos < input.Length() {
				buf.WriteString(input.SliceString(pos, pos+1))
			}
			pos++
			continue
		}
		buf.WriteString(input.SliceString(pos, c.Start))
		buf.WriteString(repl(r))
		prevMatchEnd = c.End
		if c.End > pos {
			pos = c.End
		} else {
			if pos < input.Length() {
				buf.WriteString(input.SliceString(pos, pos+1))
			}
			pos++
		}
	}
	buf.WriteString(input.SliceString(pos, input.Length()))
	return buf.String()
}

func (re *Regexp) ReplaceAll(src, repl []byte) []byte {
	return []byte(re.ReplaceAllString(string(src), string(repl)))
}

func (re *Regexp) ReplaceAllString(src, repl string) string {
	input := engine.NewInputString(src)
	return re.replaceAll(input, func(_ *engine.MatchResult) string { return repl })
}

func (re *Regexp) ReplaceAllLiteral(src, repl []byte) []byte {
	return re.ReplaceAll(src, repl)
}

func (re *Regexp) ReplaceAllLiteralString(src, repl string) string {
	return re.ReplaceAllString(src, repl)
}

func (re *Regexp) ReplaceAllFunc(src []byte, repl func([]byte) []byte) []byte {
	input := engine.NewInputBytes(src)
	s := re.replaceAll(input, func(r *engine.MatchResult) string {
		c := r.Captures[0]
		matched := []byte(input.SliceString(c.Start, c.End))
		return string(repl(matched))
	})
	return []byte(s)
}

func (re *Regexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	input := engine.NewInputString(src)
	return re.replaceAll(input, func(r *engine.MatchResult) string {
		c := r.Captures[0]
		return repl(input.SliceString(c.Start, c.End))
	})
}

func (re *Regexp) Expand(dst []byte, template []byte, src []byte, match []int) []byte {
	return re.expand(dst, string(template), string(src), match)
}

func (re *Regexp) ExpandString(dst []byte, template string, src string, match []int) []byte {
	return re.expand(dst, template, src, match)
}

func (re *Regexp) expand(dst []byte, template string, src string, match []int) []byte {
	for i := 0; i < len(template); i++ {
		if template[i] == '$' && i+1 < len(template) {
			if template[i+1] >= '1' && template[i+1] <= '9' {
				idx := int(template[i+1] - '0')
				if 2*idx+1 < len(match) && match[2*idx] >= 0 {
					dst = append(dst, src[match[2*idx]:match[2*idx+1]]...)
				}
				i++
				continue
			}
			if template[i+1] == '{' {
				end := strings.IndexByte(template[i+2:], '}')
				if end >= 0 {
					name := template[i+2 : i+2+end]
					idx := re.SubexpIndex(name)
					if idx >= 0 && 2*idx+1 < len(match) && match[2*idx] >= 0 {
						dst = append(dst, src[match[2*idx]:match[2*idx+1]]...)
					}
					i = i + 2 + end
					continue
				}
			}
		}
		dst = append(dst, template[i])
	}
	return dst
}

// --- Split ---

func (re *Regexp) Split(s string, n int) []string {
	if n == 0 {
		return nil
	}

	if n < 0 {
		n = len(s) + 1
	}

	matches := re.FindAllStringIndex(s, n)
	strings := make([]string, 0, len(matches)+1)

	beg := 0
	for _, match := range matches {
		if len(strings) >= n-1 {
			break
		}
		end := match[0]
		if match[1] != 0 {
			strings = append(strings, s[beg:end])
		}
		beg = match[1]
	}

	strings = append(strings, s[beg:])
	return strings
}

// --- Marshaling ---

func (re *Regexp) MarshalText() ([]byte, error) {
	return []byte(re.pattern), nil
}

func (re *Regexp) UnmarshalText(text []byte) error {
	r, err := Compile(string(text))
	if err != nil {
		return err
	}
	*re = *r
	return nil
}

// --- Package-level functions ---

func Match(pattern string, b []byte) (bool, error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.Match(b), nil
}

func MatchString(pattern string, s string) (bool, error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(s), nil
}

func MatchReader(pattern string, r io.RuneReader) (bool, error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchReader(r), nil
}

func QuoteMeta(s string) string {
	special := `\.+*?()|[]{}^$`
	var buf strings.Builder
	for _, r := range s {
		if strings.ContainsRune(special, r) {
			buf.WriteByte('\\')
		}
		buf.WriteRune(r)
	}
	return buf.String()
}
