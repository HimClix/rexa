// Package rexa is a high-performance, pure-Go regex engine with PCRE-like features.
//
// rexa is a drop-in replacement for Go's standard regexp package that delivers
// 2-50x faster matching through a multi-engine architecture while adding
// lookaheads, lookbehinds, backreferences, and bounded backtracking that
// the standard library lacks.
//
// # Quick Start
//
//	re := rexa.MustCompile(`\d+`)
//	fmt.Println(re.FindString("abc123")) // "123"
//
// # Drop-in Replacement
//
// Every method on regexp.Regexp exists on rexa.Regexp with the same signature:
//
//	// Before
//	import "regexp"
//	re := regexp.MustCompile(`pattern`)
//
//	// After — zero code changes needed
//	import rexa "github.com/himclix/rexa"
//	re := rexa.MustCompile(`pattern`)
//
// # PCRE Features
//
// Features not available in Go's standard regexp package:
//
//	// Lookahead
//	rexa.MustCompile(`foo(?=bar)`).FindString("foobar") // "foo"
//
//	// Negative lookahead
//	rexa.MustCompile(`foo(?!bar)`).FindString("foobaz") // "foo"
//
//	// Lookbehind
//	rexa.MustCompile(`(?<=@)\w+`).FindString("user@domain") // "domain"
//
//	// Backreference
//	rexa.MustCompile(`(\w+)\s+\1`).MatchString("hello hello") // true
//
// # Bounded Backtracking
//
// Patterns with backreferences use a bounded backtracking engine that
// returns ErrBacktrackLimit instead of hanging on pathological input:
//
//	re, _ := rexa.CompileWithOptions(`(a+)+\1`, rexa.CompileOptions{
//	    BacktrackLimit: 100000,
//	})
//
// # Architecture
//
// rexa selects the fastest engine for each pattern at compile time:
//
//   - Literal Scanner (Boyer-Moore): O(n/m) for pure literal patterns
//   - Lazy DFA: O(n) amortized for simple regex patterns
//   - Pike VM: O(n*m) for patterns with captures or anchors
//   - Bounded Backtracker: for backreferences and lookarounds
package rexa
