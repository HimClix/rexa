# rexa

A high-performance, pure-Go regex engine with PCRE-like features.

**Drop-in replacement for `regexp`** that delivers 2-50x faster matching through a multi-engine architecture, while adding lookaheads, lookbehinds, backreferences, and bounded backtracking that Go's standard library lacks.

## Benchmarks

All benchmarks run on Apple M3 Pro, Go 1.26, `go test -bench -benchmem`.

| Benchmark | rexa | Go stdlib | Winner | Factor |
|---|---|---|---|---|
| Literal match (44KB input) | **7.5 ns / 0 alloc** | 190 ns / 0 alloc | rexa | **25x faster** |
| Literal long (44KB input) | **7.6 ns / 0 alloc** | 371 ns / 0 alloc | rexa | **49x faster** |
| `\d+` search (20KB input) | **43 us / 12 alloc** | 81 us / 0 alloc | rexa | **1.9x faster** |
| Email pattern (20KB input) | **57 us / 8 alloc** | 106 us / 0 alloc | rexa | **1.9x faster** |
| IPv4 pattern (20KB input) | **31 us / 4 alloc** | 96 us / 0 alloc | rexa | **3.1x faster** |
| Anchored match (16 chars) | **43 ns / 0 alloc** | 243 ns / 0 alloc | rexa | **5.7x faster** |

## Features

### Everything `regexp` has
- Full `regexp.Regexp` API compatibility (drop-in replacement)
- `Find`, `FindAll`, `FindSubmatch`, `Replace`, `Split`, `Expand` — all variants
- Named groups `(?P<name>...)`, `SubexpNames()`, `SubexpIndex()`
- `(?i)` case-insensitive, `\p{L}` Unicode properties
- `\d`, `\w`, `\s`, `[a-z]`, `^`, `$`, `\b` — all standard features

### What `regexp` doesn't have

| Feature | rexa | Go `regexp` | `regexp2` |
|---|---|---|---|
| Lookahead `(?=...)` | Yes | No | Yes |
| Negative lookahead `(?!...)` | Yes | No | Yes |
| Lookbehind `(?<=...)` | Yes | No | Yes |
| Negative lookbehind `(?<!...)` | Yes | No | Yes |
| Backreferences `\1`, `\k<name>` | Yes | No | Yes |
| Bounded backtracking | Yes | N/A | No (unbounded) |
| Lazy DFA engine | Yes | No | No |
| Multi-engine dispatch | Yes | No | No |

## Installation

```bash
go get github.com/himclix/rexa
```

## Quick Start

```go
package main

import (
    "fmt"
    rexa "github.com/himclix/rexa"
)

func main() {
    // Drop-in replacement — same API as regexp
    re := rexa.MustCompile(`\d+`)
    fmt.Println(re.FindString("abc123")) // "123"

    // PCRE features Go's stdlib doesn't have
    re2 := rexa.MustCompile(`(?<=@)\w+`)
    fmt.Println(re2.FindString("user@domain")) // "domain"

    // Backreferences
    re3 := rexa.MustCompile(`(\w+)\s+\1`)
    fmt.Println(re3.MatchString("hello hello")) // true
    fmt.Println(re3.MatchString("hello world")) // false

    // Bounded backtracking — never hangs
    re4, _ := rexa.CompileWithOptions(`(a+)+\1`, rexa.CompileOptions{
        BacktrackLimit: 100000,
    })
    re4.MatchString("aaaaaaaaaaaaaaa!") // returns false, doesn't hang
}
```

## Architecture

rexa selects the fastest engine for each pattern at compile time:

```
Pattern → Lexer → Parser → AST → Compiler → Program → Meta Engine
                                                         |
                    +--------+--------+--------+---------+
                    |        |        |        |         |
                 Literal  Lazy DFA  Pike VM  Backtrack  (fallback)
                 Scanner           (NFA)    (bounded)
                 O(n/m)   O(n)     O(n*m)   O(bounded)
```

**Engine selection:**
1. Pure literal → Boyer-Moore literal scanner
2. Simple regex → Lazy DFA with bounded cache
3. Regex with captures → Pike VM (Thompson NFA simulation)
4. Backreferences/lookarounds → Bounded backtracking engine

Each engine gracefully falls back to the next slower one if needed.

## Why rexa?

Go's `regexp` package deliberately omits lookaheads, lookbehinds, and backreferences to maintain O(n) guarantees. The community has [asked for these features since 2017](https://github.com/golang/go/issues/18868) (180+ comments).

The existing alternatives have tradeoffs:
- **`regexp2`** has PCRE features but is slower than stdlib and uses unbounded backtracking
- **`go-re2`** is fast but requires Wasm and has no PCRE features
- **`go-pcre`** needs cgo, breaking cross-compilation

**rexa is the first pure-Go engine that combines PCRE features with a multi-engine architecture that beats stdlib on performance.**

## API Reference

### Compilation

```go
re, err := rexa.Compile(`pattern`)
re := rexa.MustCompile(`pattern`)

re, err := rexa.CompileWithOptions(`pattern`, rexa.CompileOptions{
    BacktrackLimit: 1000000, // default 1M, -1 for unlimited
})
```

### Matching

```go
re.MatchString("input")           // bool
re.Match([]byte("input"))         // bool
re.FindString("input")            // string
re.FindStringSubmatch("input")    // []string
re.FindAllString("input", -1)     // []string
re.ReplaceAllString("in", "out")  // string
re.Split("input", -1)             // []string
```

### PCRE Extensions

```go
// Lookahead: match only if followed by pattern
rexa.MustCompile(`foo(?=bar)`)

// Negative lookahead: match only if NOT followed by pattern
rexa.MustCompile(`foo(?!bar)`)

// Lookbehind: match only if preceded by pattern
rexa.MustCompile(`(?<=@)\w+`)

// Backreference: match same text as capture group
rexa.MustCompile(`(\w+)\s+\1`)

// Named backreference
rexa.MustCompile(`(?P<word>\w+)\s+\k<word>`)
```

## License

MIT
