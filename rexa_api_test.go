package rexa

import (
	"strings"
	"testing"
)

func TestFind(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.Find([]byte("abc123def"))
	if string(got) != "123" {
		t.Errorf("got %q, want %q", got, "123")
	}
	if re.Find([]byte("abc")) != nil {
		t.Error("expected nil for no match")
	}
}

func TestFindIndex(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.FindIndex([]byte("abc123"))
	if got == nil || got[0] != 3 || got[1] != 6 {
		t.Errorf("got %v, want [3 6]", got)
	}
}

func TestFindSubmatch(t *testing.T) {
	re := MustCompile(`(a+)(b+)`)
	got := re.FindSubmatch([]byte("xaabbx"))
	if len(got) != 3 || string(got[0]) != "aabb" || string(got[1]) != "aa" || string(got[2]) != "bb" {
		t.Errorf("got %v", got)
	}
}

func TestFindSubmatchIndex(t *testing.T) {
	re := MustCompile(`(a+)(b+)`)
	got := re.FindSubmatchIndex([]byte("xaabbx"))
	if got == nil || got[0] != 1 || got[1] != 5 {
		t.Errorf("got %v", got)
	}
}

func TestFindAll(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.FindAll([]byte("a1b22c333"), -1)
	if len(got) != 3 || string(got[0]) != "1" || string(got[1]) != "22" || string(got[2]) != "333" {
		t.Errorf("got %v", got)
	}
}

func TestFindAllIndex(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.FindAllIndex([]byte("a1b22"), -1)
	if len(got) != 2 || got[0][0] != 1 || got[1][0] != 3 {
		t.Errorf("got %v", got)
	}
}

func TestFindAllStringSubmatch(t *testing.T) {
	re := MustCompile(`(\w+)@(\w+)`)
	got := re.FindAllStringSubmatch("a@b c@d", -1)
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2", len(got))
	}
	if got[0][0] != "a@b" || got[0][1] != "a" || got[0][2] != "b" {
		t.Errorf("match 0: %v", got[0])
	}
	if got[1][0] != "c@d" || got[1][1] != "c" || got[1][2] != "d" {
		t.Errorf("match 1: %v", got[1])
	}
}

func TestFindAllSubmatchIndex(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.FindAllSubmatchIndex([]byte("a1b22"), -1)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
}

func TestFindAllStringSubmatchIndex(t *testing.T) {
	re := MustCompile(`(\d+)`)
	got := re.FindAllStringSubmatchIndex("a1b22", -1)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
}

func TestReplaceAll(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.ReplaceAll([]byte("a1b22"), []byte("N"))
	if string(got) != "aNbN" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceAllLiteralString(t *testing.T) {
	re := MustCompile(`\d+`)
	got := re.ReplaceAllLiteralString("a1b22", "$1")
	if got != "a$1b$1" {
		t.Errorf("got %q, want %q", got, "a$1b$1")
	}
}

func TestReplaceAllFunc(t *testing.T) {
	re := MustCompile(`\w+`)
	got := re.ReplaceAllFunc([]byte("hello world"), func(m []byte) []byte {
		return []byte(strings.ToUpper(string(m)))
	})
	if string(got) != "HELLO WORLD" {
		t.Errorf("got %q", got)
	}
}

func TestReplaceAllStringFunc(t *testing.T) {
	re := MustCompile(`\w+`)
	got := re.ReplaceAllStringFunc("hello world", strings.ToUpper)
	if got != "HELLO WORLD" {
		t.Errorf("got %q", got)
	}
}

func TestExpandString(t *testing.T) {
	re := MustCompile(`(\w+)@(\w+)`)
	match := re.FindStringSubmatchIndex("user@host")
	dst := re.ExpandString(nil, "name=$1 domain=$2", "user@host", match)
	if string(dst) != "name=user domain=host" {
		t.Errorf("got %q", dst)
	}
}

func TestExpandNamedGroup(t *testing.T) {
	re := MustCompile(`(?P<user>\w+)@(?P<host>\w+)`)
	match := re.FindStringSubmatchIndex("admin@server")
	dst := re.ExpandString(nil, "u=${user} h=${host}", "admin@server", match)
	if string(dst) != "u=admin h=server" {
		t.Errorf("got %q", dst)
	}
}

func TestLiteralPrefix(t *testing.T) {
	re := MustCompile(`http://\w+`)
	prefix, complete := re.LiteralPrefix()
	if prefix != "http://" {
		t.Errorf("prefix = %q, want %q", prefix, "http://")
	}
	if complete {
		t.Error("expected complete=false")
	}

	re2 := MustCompile(`hello`)
	prefix2, complete2 := re2.LiteralPrefix()
	if prefix2 != "hello" || !complete2 {
		t.Errorf("got prefix=%q complete=%v", prefix2, complete2)
	}
}

func TestCopy(t *testing.T) {
	re := MustCompile(`\d+`)
	re2 := re.Copy()
	if re2.String() != re.String() {
		t.Error("copy has different pattern")
	}
	if !re2.MatchString("123") {
		t.Error("copy doesn't match")
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	re := MustCompile(`\d+`)
	data, err := re.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `\d+` {
		t.Errorf("marshal: got %q", data)
	}

	var re2 Regexp
	if err := re2.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	if !re2.MatchString("123") {
		t.Error("unmarshaled regex doesn't match")
	}
}

func TestQuoteMeta(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"a.b", `a\.b`},
		{"a*b+c?", `a\*b\+c\?`},
		{"(a|b)", `\(a\|b\)`},
		{"[a]", `\[a\]`},
		{"a{3}", `a\{3\}`},
	}
	for _, tt := range tests {
		got := QuoteMeta(tt.in)
		if got != tt.want {
			t.Errorf("QuoteMeta(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMatchPackageLevel(t *testing.T) {
	matched, err := Match(`\d+`, []byte("abc123"))
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Error("expected match")
	}
}

func TestMatchStringPackageLevel(t *testing.T) {
	matched, err := MatchString(`\d+`, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Error("expected match")
	}
}
