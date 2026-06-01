# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-01

### Added
- Multi-engine regex architecture with 5 execution tiers
  - Literal Scanner (Boyer-Moore): O(n/m) for pure literal patterns
  - Lazy DFA with bounded cache and ASCII transition table: O(n) amortized
  - Pike VM (Thompson NFA simulation): O(n*m) with capture tracking
  - Bounded Backtracking engine: configurable step limit for PCRE features
  - Meta engine dispatcher: selects fastest engine per pattern at compile time
- Full `regexp.Regexp` API compatibility (drop-in replacement)
  - All Find, FindAll, FindSubmatch variants (string, []byte, index)
  - ReplaceAll, ReplaceAllFunc, ReplaceAllString, ReplaceAllStringFunc
  - Split, Expand, ExpandString
  - MatchString, Match, MatchReader
  - SubexpNames, SubexpIndex, NumSubexp
  - LiteralPrefix, Longest, Copy
  - MarshalText, UnmarshalText
  - QuoteMeta (package-level)
- PCRE-like features not available in Go's standard library
  - Positive lookahead `(?=...)`
  - Negative lookahead `(?!...)`
  - Positive lookbehind `(?<=...)`
  - Negative lookbehind `(?<!...)`
  - Backreferences `\1` through `\9`
  - Named backreferences `\k<name>`
  - Named capture groups `(?P<name>...)` and `(?<name>...)`
  - Atomic groups `(?>...)`
  - Possessive quantifiers `*+`, `++`, `?+`
- Bounded backtracking with configurable step limit
  - Default 1M steps, configurable via `CompileOptions{BacktrackLimit: N}`
  - Returns `ErrBacktrackLimit` on pathological input instead of hanging
  - Unlimited mode available with `BacktrackLimit: -1`
- Unicode support
  - Unicode property classes `\p{L}`, `\p{Nd}`, `\P{L}`, `\pL`
  - Unicode-aware `\w`, `\b` (via `unicode.IsLetter`/`unicode.IsDigit`)
  - Case-insensitive matching `(?i)` with Unicode case folding
  - Scoped flags `(?i:...)`
- Performance optimizations
  - Zero-alloc `MatchString` path via sync.Pool machine reuse
  - Zero-alloc literal path via `strings.Index` bypass
  - ASCII fast path in Input (avoids `[]rune` conversion)
  - DFA start state caching
  - Arena allocation for DFA epsilon closure results
  - Pre-allocated capture slots on MetaEngine
- Correctness verification
  - 150+ tests covering all features
  - Stdlib parity tests (Match, Find, FindAll, Replace verified against `regexp`)
  - Edge case tests (empty matches, Unicode, anchors, large input)
  - Race detector clean (`go test -race`)

### Performance vs Go stdlib `regexp`
- Literal: 25x faster (7.5 ns vs 190 ns)
- Literal Long: 49x faster (7.6 ns vs 371 ns)
- `\d+` search: 1.9x faster (43 us vs 82 us)
- Email search: 1.9x faster (57 us vs 106 us)
- IPv4 search: 3.1x faster (31 us vs 96 us)
- Anchored match: 5.7x faster (43 ns vs 243 ns)
