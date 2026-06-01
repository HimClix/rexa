package rexa

import (
	"regexp"
	"testing"
)

func FuzzCompileAndMatch(f *testing.F) {
	f.Add("abc", "abc")
	f.Add(`\d+`, "123")
	f.Add(`(a|b)*`, "ababab")
	f.Add(`[a-z]+`, "hello")
	f.Add(`\w+@\w+`, "user@host")
	f.Add(`^abc$`, "abc")
	f.Add(`a{2,4}`, "aaa")
	f.Add(`(?i)hello`, "HELLO")
	f.Fuzz(func(t *testing.T, pattern, input string) {
		re, err := Compile(pattern)
		if err != nil {
			return
		}
		_ = re.MatchString(input)
		_ = re.FindString(input)
		_ = re.FindStringSubmatch(input)
		_ = re.FindAllString(input, 10)
	})
}

func FuzzAgainstStdlib(f *testing.F) {
	f.Add(`abc`, "xabcx")
	f.Add(`\d+`, "abc123def")
	f.Add(`[a-z]+`, "Hello World")
	f.Add(`\w+`, "test 123")
	f.Add(`a.c`, "axc abc adc")
	f.Fuzz(func(t *testing.T, pattern, input string) {
		stdRe, stdErr := regexp.Compile(pattern)
		rexaRe, rexaErr := Compile(pattern)
		if stdErr != nil || rexaErr != nil {
			return
		}
		stdMatch := stdRe.MatchString(input)
		rexaMatch := rexaRe.MatchString(input)
		if stdMatch != rexaMatch {
			t.Errorf("MatchString(%q, %q): stdlib=%v rexa=%v", pattern, input, stdMatch, rexaMatch)
		}
		stdFind := stdRe.FindString(input)
		rexaFind := rexaRe.FindString(input)
		if stdFind != rexaFind {
			t.Errorf("FindString(%q, %q): stdlib=%q rexa=%q", pattern, input, stdFind, rexaFind)
		}
	})
}
