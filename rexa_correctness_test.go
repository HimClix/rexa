package rexa

import (
	"regexp"
	"strings"
	"testing"
)

// TestStdlibParity runs the same patterns against both rexa and stdlib regexp.
// Any mismatch is a bug in rexa.
func TestStdlibParity(t *testing.T) {
	patterns := []string{
		`abc`, `a.c`, `a\.c`, `a\*c`,
		`\d+`, `\w+`, `\s+`, `\D+`, `\W+`, `\S+`,
		`[abc]`, `[a-z]`, `[^abc]`, `[a-zA-Z0-9]`, `[^\s]`,
		`a+`, `a{3}`, `a{2,4}`, `a{2,}`,
		`a+?`, `a{2,4}?`,
		`ab|cd`, `a(b|c)d`, `(a|b)*c`,
		`(abc)`, `(a)(b)(c)`, `(?:abc)`,
		`^abc`, `abc$`, `^abc$`,
		`a.*b`, `a.+b`, `a.?b`,
		`..`, `...`,
		`\bfoo\b`,
		`(?i)abc`, `(?i)ABC`, `(?i)[a-z]+`,
	}

	inputs := []string{
		"", "a", "ab", "abc", "abcd", "xabcx",
		"a.c", "a*c", "aXc",
		"123", "abc123", "123abc", "abc123def",
		"   ", " a b ", "hello world",
		"aaa", "aaaa", "aaaaa", "aa",
		"abcabc", "ababab", "acd", "abd", "aed",
		"ABC", "AbC", "HELLO", "Hello",
		"foobar", "foo bar", "bar foo baz", "xfoox",
		"user@example.com", "192.168.1.1",
		"the quick brown fox jumps over the lazy dog",
	}

	for _, pat := range patterns {
		stdRe, stdErr := regexp.Compile(pat)
		rexaRe, rexaErr := Compile(pat)

		if stdErr != nil && rexaErr != nil {
			continue // both reject = fine
		}
		if stdErr != nil || rexaErr != nil {
			t.Errorf("pattern %q: stdlib err=%v, rexa err=%v", pat, stdErr, rexaErr)
			continue
		}

		for _, input := range inputs {
			// MatchString
			stdMatch := stdRe.MatchString(input)
			rexaMatch := rexaRe.MatchString(input)
			if stdMatch != rexaMatch {
				t.Errorf("MatchString(%q, %q): stdlib=%v rexa=%v [engine=%s]",
					pat, input, stdMatch, rexaMatch, rexaRe.EngineUsed())
			}

			// FindString
			stdFind := stdRe.FindString(input)
			rexaFind := rexaRe.FindString(input)
			if stdFind != rexaFind {
				t.Errorf("FindString(%q, %q): stdlib=%q rexa=%q [engine=%s]",
					pat, input, stdFind, rexaFind, rexaRe.EngineUsed())
			}

			// FindAllString
			stdAll := stdRe.FindAllString(input, -1)
			rexaAll := rexaRe.FindAllString(input, -1)
			if len(stdAll) != len(rexaAll) {
				t.Errorf("FindAllString(%q, %q): stdlib has %d matches, rexa has %d [engine=%s]",
					pat, input, len(stdAll), len(rexaAll), rexaRe.EngineUsed())
			} else {
				for i := range stdAll {
					if stdAll[i] != rexaAll[i] {
						t.Errorf("FindAllString(%q, %q)[%d]: stdlib=%q rexa=%q",
							pat, input, i, stdAll[i], rexaAll[i])
					}
				}
			}

			// ReplaceAllString
			stdRepl := stdRe.ReplaceAllString(input, "X")
			rexaRepl := rexaRe.ReplaceAllString(input, "X")
			if stdRepl != rexaRepl {
				t.Errorf("ReplaceAllString(%q, %q, X): stdlib=%q rexa=%q",
					pat, input, stdRepl, rexaRepl)
			}

			// Split
			stdSplit := stdRe.Split(input, -1)
			rexaSplit := rexaRe.Split(input, -1)
			if len(stdSplit) != len(rexaSplit) {
				t.Errorf("Split(%q, %q): stdlib has %d parts, rexa has %d",
					pat, input, len(stdSplit), len(rexaSplit))
			} else {
				for i := range stdSplit {
					if stdSplit[i] != rexaSplit[i] {
						t.Errorf("Split(%q, %q)[%d]: stdlib=%q rexa=%q",
							pat, input, i, stdSplit[i], rexaSplit[i])
					}
				}
			}
		}
	}
}

// TestStdlibSubmatchParity checks capture groups match stdlib.
func TestStdlibSubmatchParity(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
	}{
		{`(a+)(b+)`, "aabb"},
		{`(a+)(b+)`, "xaabbx"},
		{`(\w+)@(\w+)`, "user@host"},
		{`(\d+)-(\d+)-(\d+)`, "2024-01-15"},
		{`(a)(b)(c)(d)`, "abcd"},
		{`(a+)`, "aaa"},
		{`(a*)`, ""},
		{`(a*)`, "bbb"},
		{`(a|b)+`, "abba"},
		{`(?:a)(b)`, "ab"},
		{`(a(?:bc))`, "abc"},
	}

	for _, tt := range tests {
		stdRe := regexp.MustCompile(tt.pattern)
		rexaRe := MustCompile(tt.pattern)

		stdSM := stdRe.FindStringSubmatch(tt.input)
		rexaSM := rexaRe.FindStringSubmatch(tt.input)

		if len(stdSM) != len(rexaSM) {
			t.Errorf("FindStringSubmatch(%q, %q): stdlib len=%d rexa len=%d\n  stdlib=%v\n  rexa=%v",
				tt.pattern, tt.input, len(stdSM), len(rexaSM), stdSM, rexaSM)
			continue
		}
		for i := range stdSM {
			if stdSM[i] != rexaSM[i] {
				t.Errorf("FindStringSubmatch(%q, %q)[%d]: stdlib=%q rexa=%q",
					tt.pattern, tt.input, i, stdSM[i], rexaSM[i])
			}
		}
	}
}

// TestDFAEndAnchorCorrectness specifically tests the DFA $ handling.
func TestDFAEndAnchorCorrectness(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// Basic $ anchor
		{`abc$`, "abc", true},
		{`abc$`, "abcx", false},
		{`abc$`, "xabc", true},
		{`abc$`, "xabcx", false},

		// ^ + $ combined
		{`^abc$`, "abc", true},
		{`^abc$`, "abcx", false},
		{`^abc$`, "xabc", false},
		{`^abc$`, " abc ", false},
		{`^$`, "", true},
		{`^$`, "a", false},
		{`^.+$`, "abc", true},
		{`^.+$`, "", false},

		// Quantifiers with $
		{`a+$`, "aaa", true},
		{`a+$`, "aaab", false},
		{`a+$`, "baaa", true},
		{`\d+$`, "abc123", true},
		{`\d+$`, "123abc", false},

		// Character classes with $
		{`[a-z]+$`, "hello", true},
		{`[a-z]+$`, "Hello", true}, // matches "ello" at end
		{`[a-z]+$`, "HELLO", false},
		{`[a-z]+$`, "HELLOworld", true},

		// Complex anchored patterns (the benchmark pattern)
		{`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "user@example.com", true},
		{`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "user@example", false},
		{`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "user@.com", false},
		{`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "@example.com", false},
		{`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "user@example.com extra", false},
		{`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "extra user@example.com", false},

		// Only $ (no ^)
		{`\d+$`, "abc", false},
		{`\d+$`, "abc123", true},
		{`\d+$`, "123abc", false},
		{`\d+$`, "123", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			got := re.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("MustCompile(%q).MatchString(%q) = %v, want %v [engine=%s]",
					tt.pattern, tt.input, got, tt.want, re.EngineUsed())
			}

			// Cross-check with stdlib for patterns it supports
			stdRe, err := regexp.Compile(tt.pattern)
			if err == nil {
				stdGot := stdRe.MatchString(tt.input)
				if stdGot != tt.want {
					t.Errorf("BUG IN TEST: stdlib says %v for pattern=%q input=%q", stdGot, tt.pattern, tt.input)
				}
			}
		})
	}
}

// TestEdgeCases covers tricky edge cases.
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		match   bool
		find    string
	}{
		{"empty pattern", ``, "abc", true, ""},
		{"empty input", `a*`, "", true, ""},
		{"empty both", ``, "", true, ""},
		{"dot newline", `.`, "\n", false, ""},
		{"dot non-newline", `.`, "a", true, "a"},
		{"star zero", `a*`, "bbb", true, ""},
		{"plus zero", `a+`, "bbb", false, ""},
		{"quest zero", `a?`, "bbb", true, ""},
		{"long repeat", `a{100}`, strings.Repeat("a", 100), true, strings.Repeat("a", 100)},
		{"long repeat fail", `a{100}`, strings.Repeat("a", 99), false, ""},
		{"alternation first", `cat|catalog`, "catalog", true, "cat"},
		{"greedy star", `a.*b`, "aXbYb", true, "aXbYb"},
		{"lazy star", `a.*?b`, "aXbYb", true, "aXb"},
		{"nested groups", `((a)(b))`, "ab", true, "ab"},
		{"unicode", `\p{L}+`, "hello", true, "hello"},
		{"unicode digits", `\p{Nd}+`, "123", true, "123"},
		{"case insensitive", `(?i)hello`, "HELLO", true, "HELLO"},
		{"escaped special", `\.\*\+\?`, ".*+?", true, ".*+?"},
		{"bracket dash", `[a-]`, "-", true, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := MustCompile(tt.pattern)
			if got := re.MatchString(tt.input); got != tt.match {
				t.Errorf("MatchString = %v, want %v", got, tt.match)
			}
			if got := re.FindString(tt.input); got != tt.find {
				t.Errorf("FindString = %q, want %q", got, tt.find)
			}
		})
	}
}

// TestLargeInput tests patterns on large inputs for correctness.
func TestLargeInput(t *testing.T) {
	large := strings.Repeat("abcdefghij", 10000) // 100K chars

	re := MustCompile(`\d+`)
	if re.MatchString(large) {
		t.Error("expected no digit match in alpha-only input")
	}

	// Pattern at the end
	largeWithMatch := large + "12345"
	re2 := MustCompile(`\d+$`)
	got := re2.FindString(largeWithMatch)
	if got != "12345" {
		t.Errorf("FindString on large input = %q, want %q", got, "12345")
	}

	// Pattern at the start
	largeWithStart := "12345" + large
	re3 := MustCompile(`^\d+`)
	got3 := re3.FindString(largeWithStart)
	if got3 != "12345" {
		t.Errorf("FindString on large input = %q, want %q", got3, "12345")
	}

	// Pattern in the middle
	largeMid := strings.Repeat("x", 50000) + "NEEDLE" + strings.Repeat("x", 50000)
	re4 := MustCompile(`NEEDLE`)
	if !re4.MatchString(largeMid) {
		t.Error("expected to find NEEDLE in large input")
	}
	got4 := re4.FindString(largeMid)
	if got4 != "NEEDLE" {
		t.Errorf("FindString = %q, want %q", got4, "NEEDLE")
	}

	// Anchored pattern on large input
	re5 := MustCompile(`^[a-z]+$`)
	allLower := strings.Repeat("abcdefghij", 10000)
	if !re5.MatchString(allLower) {
		t.Error("expected all-lowercase to match ^[a-z]+$")
	}
	allLowerPlusUpper := allLower + "X"
	if re5.MatchString(allLowerPlusUpper) {
		t.Error("expected mixed case NOT to match ^[a-z]+$")
	}
}

// TestFindAllCorrectness checks FindAllString against stdlib.
func TestFindAllCorrectness(t *testing.T) {
	patterns := []string{`\d+`, `[a-z]+`, `\w+`, `\s+`, `.`}
	inputs := []string{
		"hello world 123 foo 456",
		"  spaces  everywhere  ",
		"abc",
		"",
		"123",
		"a1b2c3d4",
	}

	for _, pat := range patterns {
		stdRe := regexp.MustCompile(pat)
		rexaRe := MustCompile(pat)

		for _, input := range inputs {
			stdAll := stdRe.FindAllString(input, -1)
			rexaAll := rexaRe.FindAllString(input, -1)

			if len(stdAll) != len(rexaAll) {
				t.Errorf("FindAllString(%q, %q): stdlib=%d matches, rexa=%d matches\n  stdlib=%v\n  rexa=%v",
					pat, input, len(stdAll), len(rexaAll), stdAll, rexaAll)
				continue
			}
			for i := range stdAll {
				if stdAll[i] != rexaAll[i] {
					t.Errorf("FindAllString(%q, %q)[%d]: stdlib=%q rexa=%q",
						pat, input, i, stdAll[i], rexaAll[i])
				}
			}
		}
	}
}

// TestPCREFeatures tests features that stdlib doesn't have.
func TestPCREFeatures(t *testing.T) {
	// Lookaheads
	t.Run("lookahead", func(t *testing.T) {
		re := MustCompile(`\w+(?=\d)`)
		if got := re.FindString("abc1"); got != "abc" {
			t.Errorf("got %q, want %q", got, "abc")
		}
		if got := re.FindString("abc"); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	// Negative lookahead
	t.Run("neg_lookahead", func(t *testing.T) {
		re := MustCompile(`foo(?!bar)`)
		if !re.MatchString("foobaz") {
			t.Error("expected match for foobaz")
		}
		if re.MatchString("foobar") {
			t.Error("expected no match for foobar")
		}
	})

	// Lookbehind
	t.Run("lookbehind", func(t *testing.T) {
		re := MustCompile(`(?<=@)\w+`)
		if got := re.FindString("user@domain"); got != "domain" {
			t.Errorf("got %q, want %q", got, "domain")
		}
	})

	// Negative lookbehind
	t.Run("neg_lookbehind", func(t *testing.T) {
		re := MustCompile(`(?<!@)\w+`)
		if got := re.FindString("user@domain"); got != "user" {
			t.Errorf("got %q, want %q", got, "user")
		}
	})

	// Backreference
	t.Run("backref_match", func(t *testing.T) {
		re := MustCompile(`(\w+)\s+\1`)
		if !re.MatchString("hello hello") {
			t.Error("expected match for repeated word")
		}
		if re.MatchString("hello world") {
			t.Error("expected no match for different words")
		}
	})

	// Backreference capture
	t.Run("backref_capture", func(t *testing.T) {
		re := MustCompile(`(\w+)\s+\1`)
		got := re.FindStringSubmatch("say hello hello world")
		if len(got) < 2 || got[0] != "hello hello" || got[1] != "hello" {
			t.Errorf("got %v", got)
		}
	})

	// Backtrack limit
	t.Run("backtrack_limit", func(t *testing.T) {
		re, _ := CompileWithOptions(`(a+)+\1`, CompileOptions{BacktrackLimit: 1000})
		got := re.MatchString(strings.Repeat("a", 30) + "!")
		// Should either match or hit limit — must NOT hang
		_ = got
	})
}
