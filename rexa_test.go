package rexa

import "testing"

func TestMatchString(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// Literals
		{"abc", "abc", true},
		{"abc", "xabc", true},
		{"abc", "ab", false},
		{"abc", "xabcx", true},

		// Dot
		{"a.c", "abc", true},
		{"a.c", "aXc", true},
		{"a.c", "ac", false},
		{"a.c", "a\nc", false},

		// Alternation
		{"cat|dog", "I have a cat", true},
		{"cat|dog", "I have a dog", true},
		{"cat|dog", "I have a bird", false},

		// Star
		{"ab*c", "ac", true},
		{"ab*c", "abc", true},
		{"ab*c", "abbc", true},
		{"ab*c", "abbbc", true},
		{"ab*c", "adc", false},

		// Plus
		{"ab+c", "ac", false},
		{"ab+c", "abc", true},
		{"ab+c", "abbc", true},

		// Question
		{"ab?c", "ac", true},
		{"ab?c", "abc", true},
		{"ab?c", "abbc", false},

		// Groups
		{"a(b|c)*d", "ad", true},
		{"a(b|c)*d", "abd", true},
		{"a(b|c)*d", "abcbcd", true},
		{"a(b|c)*d", "aed", false},

		// Non-capturing groups
		{"a(?:b|c)*d", "abcbcd", true},

		// Anchors
		{"^abc", "abc", true},
		{"^abc", "xabc", false},
		{"abc$", "abc", true},
		{"abc$", "abcx", false},
		{"^abc$", "abc", true},
		{"^abc$", "abcx", false},

		// Escaped chars
		{`a\.b`, "a.b", true},
		{`a\.b`, "axb", false},
		{`a\*b`, "a*b", true},

		// Character classes (shorthand)
		{`\d+`, "abc123def", true},
		{`\d+`, "abcdef", false},
		{`\w+`, "hello", true},
		{`\s`, "hello world", true},
		{`\s`, "helloworld", false},

		// Quantifier bounds
		{`a{3}`, "aaa", true},
		{`a{3}`, "aa", false},
		{`a{2,4}`, "aa", true},
		{`a{2,4}`, "aaaa", true},
		{`a{2,4}`, "a", false},
		{`a{2,}`, "aa", true},
		{`a{2,}`, "aaaaaaa", true},
		{`a{2,}`, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).MatchString(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestFindString(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{"abc", "xabcx", "abc"},
		{"a.c", "xaXcx", "aXc"},
		{`\d+`, "abc123def", "123"},
		{"cat|dog", "I have a cat and a dog", "cat"},
		{"ab*c", "xabbcx", "abbc"},
		{"xyz", "abc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).FindString(%q) = %q, want %q", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestFindStringSubmatch(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    []string
	}{
		{"(a+)(b+)", "aaabb", []string{"aaabb", "aaa", "bb"}},
		{"(\\w+)@(\\w+)", "user@host", []string{"user@host", "user", "host"}},
		{"(a)(b)(c)", "abc", []string{"abc", "a", "b", "c"}},
		{"(a+)", "bbb", nil},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindStringSubmatch(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("got %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("submatch[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindAllString(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		n       int
		want    []string
	}{
		{`\d+`, "a1b22c333", -1, []string{"1", "22", "333"}},
		{`\d+`, "a1b22c333", 2, []string{"1", "22"}},
		{"cat", "cat cat cat", -1, []string{"cat", "cat", "cat"}},
		{"xyz", "abc", -1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindAllString(tt.input, tt.n)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("match[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReplaceAllString(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		repl    string
		want    string
	}{
		{`\d+`, "a1b22c333", "N", "aNbNcN"},
		{"cat", "cat and cat", "dog", "dog and dog"},
		{"xyz", "abc", "Z", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.ReplaceAllString(tt.input, tt.repl)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		n       int
		want    []string
	}{
		{`,`, "a,b,c", -1, []string{"a", "b", "c"}},
		{`\s+`, "hello  world foo", -1, []string{"hello", "world", "foo"}},
		{`,`, "a,b,c", 2, []string{"a", "b,c"}},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.Split(tt.input, tt.n)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("part[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSyntaxError(t *testing.T) {
	patterns := []string{
		`(abc`,
		`\`,
		`[abc`,
	}

	for _, pat := range patterns {
		t.Run(pat, func(t *testing.T) {
			_, err := Compile(pat)
			if err == nil {
				t.Errorf("expected error for pattern %q, got nil", pat)
			}
		})
	}
}

func TestMustCompilePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid pattern")
		}
	}()
	MustCompile(`(abc`)
}

func TestSubexpNames(t *testing.T) {
	re := MustCompile(`(?P<year>\d+)-(?P<month>\d+)`)
	names := re.SubexpNames()
	if names[1] != "year" {
		t.Errorf("names[1] = %q, want \"year\"", names[1])
	}
	if names[2] != "month" {
		t.Errorf("names[2] = %q, want \"month\"", names[2])
	}
	if re.SubexpIndex("year") != 1 {
		t.Errorf("SubexpIndex(\"year\") = %d, want 1", re.SubexpIndex("year"))
	}
	if re.SubexpIndex("month") != 2 {
		t.Errorf("SubexpIndex(\"month\") = %d, want 2", re.SubexpIndex("month"))
	}
	if re.SubexpIndex("nope") != -1 {
		t.Errorf("SubexpIndex(\"nope\") = %d, want -1", re.SubexpIndex("nope"))
	}
}

func TestWordBoundary(t *testing.T) {
	re := MustCompile(`\bfoo\b`)
	if !re.MatchString("foo bar") {
		t.Error("expected match for 'foo bar'")
	}
	if !re.MatchString("bar foo") {
		t.Error("expected match for 'bar foo'")
	}
	if re.MatchString("foobar") {
		t.Error("expected no match for 'foobar'")
	}
}

func TestCharClassBracket(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"[abc]", "a", true},
		{"[abc]", "d", false},
		{"[a-z]", "m", true},
		{"[a-z]", "M", false},
		{"[^abc]", "d", true},
		{"[^abc]", "a", false},
		{"[a-zA-Z0-9]", "Z", true},
		{"[a-zA-Z0-9]", "5", true},
		{"[a-zA-Z0-9]", "!", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).MatchString(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestCaseInsensitive(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{`(?i)hello`, "HELLO", true},
		{`(?i)hello`, "Hello", true},
		{`(?i)hello`, "hello", true},
		{`(?i)hello`, "world", false},
		{`(?i)abc`, "AbC", true},
		{`(?i:hello) world`, "HELLO world", true},
		{`(?i:hello) world`, "HELLO WORLD", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).MatchString(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestUnicodeClass(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{`\p{L}+`, "hello", true},
		{`\p{L}+`, "12345", false},
		{`\p{L}+`, "héllo", true},
		{`\p{Nd}+`, "12345", true},
		{`\p{Nd}+`, "hello", false},
		{`\P{L}+`, "12345", true},
		{`\P{L}+`, "hello", false},
		{`\pL+`, "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).MatchString(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestFindStringCaseInsensitive(t *testing.T) {
	re := MustCompile(`(?i)hello`)
	got := re.FindString("say HELLO world")
	if got != "HELLO" {
		t.Errorf("got %q, want %q", got, "HELLO")
	}
}

func TestEngineSelection(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"hello world", "literal"},
		{"abc", "literal"},
		{`\d+`, "lazydfa"},
		{`a(b|c)*d`, "lazydfa"},
		{`[a-z]+`, "lazydfa"},
		{`^abc$`, "lazydfa"},
		{`\bfoo\b`, "pikevm"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			re.FindString("test input hello world abc 123")
			got := re.EngineUsed()
			if got != tt.want {
				t.Errorf("MustCompile(%q).EngineUsed() = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestLiteralPrefilter(t *testing.T) {
	input := "aaa bbb http://example.com ccc"
	re := MustCompile(`http://\w+\.\w+`)
	got := re.FindString(input)
	if got != "http://example.com" {
		t.Errorf("got %q, want %q", got, "http://example.com")
	}
}

func TestBackreference(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{`(\w+)\s+\1`, "hello hello", true},
		{`(\w+)\s+\1`, "hello world", false},
		{`(\w+)\s+\1`, "abc abc", true},
		{`(\w+)\s+\1`, "abc abd", false},
		{`(a+)(b+)\s+\1\2`, "aabb aabb", true},
		{`(a+)(b+)\s+\1\2`, "aabb abbb", true}, // matches "abb abb" at offset 1
		{`(a+)(b+)\s+\1\2`, "aabb xyzq", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).MatchString(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestBackreferenceSubmatch(t *testing.T) {
	re := MustCompile(`(\w+)\s+\1`)
	got := re.FindStringSubmatch("say hello hello world")
	if len(got) < 2 {
		t.Fatalf("expected submatch, got %v", got)
	}
	if got[0] != "hello hello" {
		t.Errorf("got[0] = %q, want %q", got[0], "hello hello")
	}
	if got[1] != "hello" {
		t.Errorf("got[1] = %q, want %q", got[1], "hello")
	}
}

func TestBackreferenceEngineSelection(t *testing.T) {
	re := MustCompile(`(\w+)\s+\1`)
	re.FindString("hello hello")
	if re.EngineUsed() != "backtrack" {
		t.Errorf("expected backtrack engine, got %q", re.EngineUsed())
	}
}

func TestBacktrackLimit(t *testing.T) {
	// (a+)+$ on "aaa...!" is pathological — exponential backtracking
	input := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa!"
	re, err := CompileWithOptions(`(a+)+$`, CompileOptions{BacktrackLimit: 10000})
	if err != nil {
		t.Fatal(err)
	}
	// Should return false (no match) because limit is hit, not hang
	got := re.MatchString(input)
	if got {
		t.Error("expected no match (backtrack limit should trigger)")
	}
}

func TestBacktrackUnlimited(t *testing.T) {
	re, err := CompileWithOptions(`(\w+)\s+\1`, CompileOptions{BacktrackLimit: -1})
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("hello hello") {
		t.Error("expected match")
	}
}

func TestLookahead(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{`foo(?=bar)`, "foobar", "foo"},
		{`foo(?=bar)`, "foobaz", ""},
		{`\w+(?=\.)`, "hello.world", "hello"},
		{`\w+(?=\s)`, "hello world", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).FindString(%q) = %q, want %q", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestNegativeLookahead(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{`foo(?!bar)`, "foobaz", "foo"},
		{`foo(?!bar)`, "foobar", ""},
		{`\d+(?!\.)`, "123.456", "12"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).FindString(%q) = %q, want %q", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestLookbehind(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{`(?<=@)\w+`, "user@domain", "domain"},
		{`(?<=\$)\d+`, "price $100", "100"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).FindString(%q) = %q, want %q", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestNegativeLookbehind(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    string
	}{
		{`(?<!@)\w+`, "user@domain", "user"},
		{`(?<!\$)\d+`, "price $100", "00"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.FindString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).FindString(%q) = %q, want %q", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestLookaroundEngineSelection(t *testing.T) {
	re := MustCompile(`foo(?=bar)`)
	re.FindString("foobar")
	if re.EngineUsed() != "backtrack" {
		t.Errorf("expected backtrack engine for lookahead, got %q", re.EngineUsed())
	}
}
