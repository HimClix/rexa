<p align="center">
  <h1 align="center">rexa</h1>
  <p align="center">
    A high-performance, pure-Go regex engine with PCRE-like features.<br/>
    Drop-in <code>regexp</code> replacement. 2-50x faster. Zero cgo.
  </p>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/himclix/rexa"><img src="https://pkg.go.dev/badge/github.com/himclix/rexa.svg" alt="Go Reference"></a>
  <a href="https://github.com/HimClix/rexa/actions"><img src="https://github.com/HimClix/rexa/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/himclix/rexa"><img src="https://goreportcard.com/badge/github.com/himclix/rexa" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="CHANGELOG.md"><img src="https://img.shields.io/badge/version-v0.1.0-green.svg" alt="Version"></a>
</p>

---

## Why rexa Exists

Go's `regexp` package has two long-standing problems that the Go team has chosen not to fix:

**Missing features.** [Go issue #18868](https://github.com/golang/go/issues/18868) (open since 2017, 180+ comments) requests lookaheads and lookbehinds. The Go team's response: *"We aren't going to break [the O(n) guarantee] in order to provide some feature."* This blocks patterns that every other language supports — `(?<=@)\w+`, `foo(?=bar)`, `(\w+)\s+\1`.

**Slow performance.** [Go issue #26623](https://github.com/golang/go/issues/26623) tracks regex performance. On the [Benchmarks Game](https://benchmarksgame-team.pages.debian.net/benchmarksgame/performance/regexredux.html), Go's regex is 5-10x slower than Python. [Nightfall AI](https://www.nightfall.ai/blog/best-go-regex-library) measured it at 30-40x slower than RE2/Hyperscan for security filtering.

**No good alternative.** `regexp2` has PCRE features but is [slower than stdlib](https://itnext.io/best-regexp-alternative-for-go-be42abdc1fbb) with unbounded backtracking (DoS risk). `go-pcre` is fast but requires cgo (breaks cross-compilation, Docker scratch, CI). `go-re2` is fast via Wasm but still lacks PCRE features.

rexa fills the gap: **fast + feature-complete + pure Go + safe.**

---

## Benchmarks

Apple M3 Pro, Go 1.26. Run yourself: `go test -bench=. -benchmem`

| Pattern | Input | rexa | stdlib | Winner |
|---|---|---|---|---|
| Literal `"lazy dog"` | 44 KB text | **7.5 ns** / 0 alloc | 190 ns / 0 alloc | **rexa 25x** |
| Literal `"jumps over the lazy"` | 44 KB text | **7.6 ns** / 0 alloc | 371 ns / 0 alloc | **rexa 49x** |
| `\d+` | 20 KB text | **43 μs** / 12 alloc | 82 μs / 0 alloc | **rexa 1.9x** |
| `[\w.+-]+@[\w.-]+\.[\w.-]+` | 20 KB text | **57 μs** / 8 alloc | 106 μs / 0 alloc | **rexa 1.9x** |
| `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}` | 20 KB text | **31 μs** / 4 alloc | 96 μs / 0 alloc | **rexa 3.1x** |
| `^[a-zA-Z0-9._%+-]+@...+\.[a-zA-Z]{2,}$` | 16 chars | **43 ns** / 0 alloc | 243 ns / 0 alloc | **rexa 5.7x** |

---

## Quick Start

```bash
go get github.com/himclix/rexa@v0.1.0
```

```go
package main

import (
    "fmt"
    rexa "github.com/himclix/rexa"
)

func main() {
    // Drop-in replacement — same API as regexp
    re := rexa.MustCompile(`\d+`)
    fmt.Println(re.FindString("abc123"))        // "123"
    fmt.Println(re.FindAllString("a1b2c3", -1)) // [1 2 3]
    fmt.Println(re.ReplaceAllString("a1b2", "X")) // "aXbX"

    // PCRE features Go's stdlib doesn't have
    re2 := rexa.MustCompile(`(?<=@)\w+`)
    fmt.Println(re2.FindString("user@domain"))  // "domain"

    re3 := rexa.MustCompile(`(\w+)\s+\1`)
    fmt.Println(re3.MatchString("hello hello")) // true
    fmt.Println(re3.MatchString("hello world")) // false
}
```

### Drop-in Replacement

Swap one import. Change zero code.

```go
// Before                              // After
import "regexp"                        import rexa "github.com/himclix/rexa"
re := regexp.MustCompile(`\d+`)       re := rexa.MustCompile(`\d+`)
```

Every method on `regexp.Regexp` exists on `rexa.Regexp` with the identical signature:
`Find`, `FindAll`, `FindSubmatch`, `FindAllStringSubmatch`, `ReplaceAll`, `ReplaceAllFunc`,
`Split`, `Expand`, `MatchString`, `Match`, `MatchReader`, `SubexpNames`, `SubexpIndex`,
`NumSubexp`, `LiteralPrefix`, `Longest`, `Copy`, `MarshalText`, `UnmarshalText`, `String`.

---

## Feature Comparison

| Feature | rexa | Go `regexp` | `regexp2` | `go-pcre` |
|---|:---:|:---:|:---:|:---:|
| Lookahead `(?=...)` | Yes | -- | Yes | Yes |
| Negative lookahead `(?!...)` | Yes | -- | Yes | Yes |
| Lookbehind `(?<=...)` | Yes | -- | Yes | Yes |
| Negative lookbehind `(?<!...)` | Yes | -- | Yes | Yes |
| Backreferences `\1` | Yes | -- | Yes | Yes |
| Named groups `(?P<n>...)` | Yes | Yes | Yes | Yes |
| Unicode `\p{L}` | Yes | Yes | Yes | Yes |
| Case-insensitive `(?i)` | Yes | Yes | Yes | Yes |
| Bounded backtracking | Yes | N/A | -- | -- |
| Multi-engine dispatch | Yes | -- | -- | -- |
| Lazy DFA | Yes | -- | -- | -- |
| Pure Go (no cgo) | Yes | Yes | Yes | -- |
| Zero-alloc MatchString | Yes | Yes | -- | -- |

---

## PCRE Features

```go
// ── Lookahead ──────────────────────────────────────────
// Match "foo" only if followed by "bar"
rexa.MustCompile(`foo(?=bar)`).FindString("foobar")   // "foo"
rexa.MustCompile(`foo(?=bar)`).FindString("foobaz")   // ""

// Match "foo" only if NOT followed by "bar"
rexa.MustCompile(`foo(?!bar)`).FindString("foobaz")   // "foo"

// ── Lookbehind ─────────────────────────────────────────
// Match word after "@"
rexa.MustCompile(`(?<=@)\w+`).FindString("user@domain")  // "domain"

// Match digits NOT preceded by "$"
rexa.MustCompile(`(?<!\$)\d+`).FindString("price $100")  // "00"

// ── Backreferences ─────────────────────────────────────
// Match repeated words
rexa.MustCompile(`(\w+)\s+\1`).MatchString("hello hello") // true
rexa.MustCompile(`(\w+)\s+\1`).MatchString("hello world") // false

// Named backreference
rexa.MustCompile(`(?P<w>\w+)\s+\k<w>`)

// ── Flags ──────────────────────────────────────────────
// Case-insensitive (scoped)
rexa.MustCompile(`(?i:hello) world`).MatchString("HELLO world") // true

// Unicode properties
rexa.MustCompile(`\p{L}+`).FindString("hello")   // Unicode letters
rexa.MustCompile(`\p{Nd}+`).FindString("123")     // Unicode digits

// ── Bounded Backtracking ───────────────────────────────
// Pathological patterns return error instead of hanging
re, _ := rexa.CompileWithOptions(`(a+)+\1`, rexa.CompileOptions{
    BacktrackLimit: 100000, // default: 1M steps
})
re.MatchString("aaaaaaaaaa!")  // false (bounded, doesn't hang)
```

---

## Architecture

rexa selects the **fastest engine** for each pattern at compile time:

```
┌──────────────────────────────────────────────────────────────┐
│                     Compile(pattern)                          │
│                           │                                   │
│              Lexer → Parser → AST → Compiler                 │
│                           │                                   │
│                      ┌────▼─────┐                            │
│                      │ Optimizer │  extract literal prefix,   │
│                      │          │  detect anchors, classify   │
│                      └────┬─────┘                            │
│                           │                                   │
│                    ┌──────▼───────┐                           │
│                    │ Meta Engine  │  picks fastest engine      │
│                    └──────┬───────┘                           │
│                           │                                   │
│         ┌─────────┬───────┼────────┬──────────┐              │
│         ▼         ▼       ▼        ▼          ▼              │
│    ┌─────────┐┌───────┐┌──────┐┌───────┐┌─────────┐         │
│    │ Literal ││ Lazy  ││ Pike ││Bounded││  (fall  │         │
│    │ Scanner ││ DFA   ││ VM   ││ Back- ││  back)  │         │
│    │         ││       ││      ││ track ││         │         │
│    │ O(n/m)  ││ O(n)  ││O(n·m)││O(lim) ││         │         │
│    │ 0 alloc ││ amort ││      ││       ││         │         │
│    └─────────┘└───┬───┘└──────┘└───────┘└─────────┘         │
│                   │ cache                                     │
│                   │ overflow → falls back to Pike VM          │
└──────────────────────────────────────────────────────────────┘
```

**Engine selection rules:**

| Pattern | Engine | Why |
|---|---|---|
| `"hello"` (pure literal) | Literal Scanner | `strings.Index` — no automaton overhead |
| `\d+`, `[a-z]+` | Lazy DFA | O(1) per cached character via ASCII table |
| `^(\d+)-(\d+)$` | Lazy DFA → Pike VM | DFA finds position, Pike VM extracts captures |
| `\bfoo\b` | Pike VM | Word boundaries need position context |
| `(\w+)\s+\1` | Bounded Backtracker | Backreferences require backtracking |
| `foo(?=bar)` | Bounded Backtracker | Lookaround spawns sub-engine |

---

## Complexity Guarantees

Each tier matches the **theoretical lower bound** for its feature class:

| Pattern Class | Engine | Time | Space | Proof |
|---|---|---|---|---|
| Pure literal | Boyer-Moore | **O(n/m)** avg | O(m) | [Boyer-Moore (1977)](https://en.wikipedia.org/wiki/Boyer%E2%80%93Moore_string-search_algorithm) |
| Simple regex | Lazy DFA | **O(n)** amortized | O(min(2^m, cap)) | [Thompson (1968)](https://swtch.com/~rsc/regexp/regexp1.html) |
| + Captures | Pike VM | **O(n·m)** | O(m·k) | [Pike NFA simulation](https://swtch.com/~rsc/regexp/regexp2.html) |
| + Lookarounds | Pike VM + sub | **O(n·m)** | O(m·k) | [Barriere et al. (2024)](https://arxiv.org/pdf/2603.26139) |
| + Backreferences | Bounded BT | **O(min(2^k·n, L))** | O(n·k) | [NP-complete (Aho, 1990)](https://arxiv.org/pdf/1903.05896) |

> **n** = input length, **m** = pattern size, **k** = capture groups, **L** = backtrack limit

**Why backreferences can't be O(n):** Matching with backreferences is [NP-complete](https://arxiv.org/pdf/1903.05896) — proven by reduction from 3-SAT. Every engine supporting them (PCRE2, Java, .NET) uses exponential backtracking. rexa bounds this with a configurable step limit (default 1M). Returns `ErrBacktrackLimit` instead of hanging — same approach as PCRE2's `PCRE2_MATCH_LIMIT`.

**Why the lazy DFA matters:** Go's stdlib does O(m) work per character (NFA simulation). rexa's lazy DFA caches states — after first computation, each character is a single array lookup (O(1)). ASCII inputs use a 128-entry transition table per state; non-ASCII falls back to a hash map. Cache is bounded at 10K states; overflow flushes and rebuilds; excessive flushing falls back to Pike VM.

---

## API Reference

<details>
<summary><strong>Compilation</strong></summary>

```go
re, err := rexa.Compile(`pattern`)
re := rexa.MustCompile(`pattern`)
re, err := rexa.CompileWithOptions(`pattern`, rexa.CompileOptions{
    BacktrackLimit: 1000000, // 0 = default (1M), -1 = unlimited
})
```
</details>

<details>
<summary><strong>Matching</strong></summary>

```go
re.MatchString("input")           // bool
re.Match([]byte("input"))         // bool
re.MatchReader(r)                 // bool
```
</details>

<details>
<summary><strong>Finding</strong></summary>

```go
re.FindString("input")                   // string
re.FindStringSubmatch("input")           // []string
re.FindStringIndex("input")              // []int
re.FindStringSubmatchIndex("input")      // []int
re.FindAllString("input", -1)            // []string
re.FindAllStringSubmatch("input", -1)    // [][]string
// ... all byte/reader/index variants available
```
</details>

<details>
<summary><strong>Replacing</strong></summary>

```go
re.ReplaceAllString("in", "out")                // string
re.ReplaceAllStringFunc("in", strings.ToUpper)  // string
re.ReplaceAllLiteralString("in", "$1")          // string (no expansion)
re.Expand(dst, template, src, match)            // []byte
// ... all byte variants available
```
</details>

<details>
<summary><strong>Splitting & Metadata</strong></summary>

```go
re.Split("input", -1)          // []string
re.SubexpNames()               // []string
re.SubexpIndex("name")         // int
re.NumSubexp()                 // int
re.LiteralPrefix()             // (string, bool)
re.String()                    // string
rexa.QuoteMeta("a.b")         // "a\\.b"
```
</details>

---

## Project Structure

```
rexa/
├── rexa.go                  Public API (drop-in regexp replacement)
├── rexa_options.go          CompileOptions, ErrBacktrackLimit
├── doc.go                   Package documentation
│
├── syntax/                  Compilation frontend
│   ├── lexer.go             Pattern string → token stream
│   ├── parser.go            Recursive descent → AST
│   ├── tree.go              AST node types (Op, Node, Tree)
│   ├── token.go             Token/GroupKind definitions
│   ├── unicode.go           Unicode category/script tables
│   └── error.go             SyntaxError with position
│
├── compiler/                Compilation backend
│   ├── nfa.go               Thompson's construction (AST → instructions)
│   ├── inst.go              Instruction opcodes
│   ├── program.go           Compiled program + metadata
│   └── optimize.go          Literal prefix, anchor detection
│
├── engine/                  Execution engines
│   ├── meta.go              Multi-engine dispatcher
│   ├── literal.go           Boyer-Moore / strings.Index
│   ├── lazydfa.go           Lazy DFA + ASCII transition table
│   ├── pikevm.go            Pike VM (Thompson NFA + captures)
│   ├── backtrack.go         Bounded backtracking + lookarounds
│   ├── machine.go           sync.Pool machine allocation
│   ├── engine.go            Engine interface, MatchResult
│   ├── input.go             Input abstraction (ASCII fast path)
│   └── state.go             Thread type
│
└── internal/
    ├── bitset/              Bit vector for NFA state dedup
    ├── bm/                  Boyer-Moore string search
    ├── pool/                Generic sync.Pool wrapper
    └── runeutil/            Unicode helpers (IsWordChar, EqualFold)
```

---

## Versioning

rexa follows [Semantic Versioning](https://semver.org/). See [CHANGELOG.md](CHANGELOG.md) for release notes.

```
v0.x.y  Development (current) — API may change between minor versions
v1.0.0  Stable — regexp.Regexp API compatibility guaranteed
v1.x.y  Backwards-compatible additions and optimizations
```

```bash
go get github.com/himclix/rexa@v0.1.0   # pin version
go get -u github.com/himclix/rexa        # upgrade to latest
go get github.com/himclix/rexa@v0.1.0    # downgrade
```

---

## Contributing

Bug reports, performance improvements, new features, and docs fixes are all welcome.

See **[CONTRIBUTING.md](CONTRIBUTING.md)** for development setup, project structure walkthrough, performance rules, and PR guidelines.

---

## Acknowledgments

- [Russ Cox's regex articles](https://swtch.com/~rsc/regexp/) — Thompson NFA and Pike VM foundations
- [Rust regex-automata](https://docs.rs/regex-automata) — multi-engine architecture inspiration
- [Andrew Gallant's regex internals](https://burntsushi.net/regex-internals/) — lazy DFA design
- [rhaeguard's regex tutorial](https://rhaeguard.github.io/posts/regex/) — initial implementation reference

## License

[MIT](LICENSE)
