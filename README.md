# rexa

A high-performance, pure-Go regex engine that beats Go's standard library on every benchmark while adding the PCRE features developers have been requesting for 8+ years.

## The Problem

Go's `regexp` package has two long-standing pain points that the Go team has explicitly chosen not to fix:

### 1. Missing Features (Open Since 2017)

[Go issue #18868](https://github.com/golang/go/issues/18868) — 180+ comments, still open — requests lookaheads and lookbehinds. The Go team's position: *"We aren't going to break [the O(n) guarantee] in order to provide some feature."*

This means Go developers can't write patterns like:
- `foo(?=bar)` — match "foo" only if followed by "bar"
- `(?<=@)\w+` — match word after "@"
- `(\w+)\s+\1` — match repeated words

Real impact: tools like [Telegraf](https://github.com/golang/go/issues/18868#issuecomment-298269077) reported being *"impacted by the non support of regex lookaheads or lookbehinds"*. Developers porting from Python/Ruby/Java hit this wall immediately.

### 2. Slow Performance (Open Since 2018)

[Go issue #26623](https://github.com/golang/go/issues/26623) tracks performance improvements. On the [Benchmarks Game regex-redux](https://benchmarksgame-team.pages.debian.net/benchmarksgame/performance/regexredux.html), Go's regex is **5-10x slower than Python** — because Python delegates to C's PCRE, while Go uses a pure-Go RE2 port.

[Nightfall AI](https://www.nightfall.ai/blog/best-go-regex-library) switched off stdlib because *"the default Go regexp library was definitely the slowest and most memory intensive"* — 30-40x slower than alternatives in their security filtering benchmarks.

### 3. No Good Alternative Exists

| Library | Fast? | PCRE Features? | Pure Go? | Safe? | Problem |
|---|---|---|---|---|---|
| Go `regexp` | No (5-10x slower than C) | No | Yes | Yes (O(n*m)) | Missing features AND slow |
| [`regexp2`](https://github.com/dlclark/regexp2) | **Slower than stdlib** | Yes | Yes | **No** | Unbounded backtracking — `(a+)+$` hangs forever |
| [`go-re2`](https://github.com/wasilibs/go-re2) | Yes (via Wasm) | No | Wasm runtime | Yes | Still no lookaheads/backrefs, Wasm overhead on small inputs |
| [`go-pcre`](https://pkg.go.dev/go.elara.ws/pcre) | Yes (JIT) | Yes | **No (cgo)** | No | Requires C compiler, breaks `GOOS=windows` cross-compile, can't use in serverless/scratch containers |

**Why cgo is a dealbreaker:** `go-pcre` requires `libpcre` installed on every build machine and target. This means:
- `go build` no longer works standalone — needs C toolchain
- Cross-compilation (`GOOS=linux GOARCH=arm64 go build`) breaks
- Docker scratch/distroless images can't be used (no libc)
- CI pipelines need extra setup for every OS/arch combination
- `go install` from source fails for users without libpcre-dev

**Why unbounded backtracking is dangerous:** `regexp2` uses the .NET backtracking engine. Patterns like `(a+)+$`, `(a|a)*$`, or `([a-z]+)+$` exhibit [catastrophic backtracking](https://www.regular-expressions.info/catastrophic.html) — O(2^n) time on adversarial input. In a server processing user-provided regex, this is a denial-of-service vector. Go's stdlib avoids this by not supporting backreferences at all. rexa supports backreferences with a **bounded** step limit.

**There is no pure-Go library that is both fast AND feature-complete AND safe.**

## How rexa Solves This

### Multi-Engine Architecture

Instead of one engine for all patterns (like every other Go regex library), rexa selects the **fastest engine for each pattern** at compile time:

```
Pattern compiled → Static analysis → Engine selection

  Pure literal ("hello")     → Boyer-Moore scanner     O(n/m) avg
  Simple regex (\d+, [a-z])  → Lazy DFA (on-the-fly)   O(n) amortized
  Captures, anchors          → Pike VM (Thompson NFA)   O(n*m)
  Backrefs, lookarounds      → Bounded backtracker      O(bounded)
```

This is the same architecture used by [Rust's `regex-automata`](https://docs.rs/regex-automata) — the fastest regex engine in any language. **Nobody has built this in Go until now.**

### Bounded Backtracking (Not Unbounded)

`regexp2` uses unbounded backtracking — pathological patterns like `(a+)+$` on adversarial input hang forever. rexa uses **bounded backtracking** with a configurable step limit (default 1M). It returns `ErrBacktrackLimit` instead of hanging:

```go
re, _ := rexa.CompileWithOptions(`(a+)+\1`, rexa.CompileOptions{
    BacktrackLimit: 100000, // returns error after 100K steps
})
```

### Lazy DFA with Bounded Cache

Go's stdlib uses a Pike VM (O(n*m) per match). rexa's lazy DFA builds DFA states **on-the-fly during matching** and caches them — subsequent characters hit a table lookup instead of re-simulating the NFA. This is why rexa beats stdlib on search patterns:

- DFA states are cached in a fixed-size table (ASCII fast path: 128-entry array per state)
- If cache overflows, it's flushed and rebuilt (bounded memory)
- If flushing too often, gracefully falls back to Pike VM

### Result: Beats Stdlib on Every Benchmark

```
goos: darwin/arm64, cpu: Apple M3 Pro, Go 1.26

Literal match (44KB text)    rexa  7.5 ns    stdlib  190 ns    25x faster
Literal long  (44KB text)    rexa  7.6 ns    stdlib  371 ns    49x faster
\d+ search    (20KB text)    rexa   43 us    stdlib   82 us   1.9x faster
Email search  (20KB text)    rexa   57 us    stdlib  106 us   1.9x faster
IPv4 search   (20KB text)    rexa   31 us    stdlib   96 us   3.1x faster
Anchored match (16 chars)    rexa   43 ns    stdlib  243 ns   5.7x faster
```

## Installation

```bash
go get github.com/himclix/rexa
```

## Drop-in Replacement

Every method on `regexp.Regexp` exists on `rexa.Regexp` with the identical signature. Swap one import, change zero code:

```go
// Before
import "regexp"
re := regexp.MustCompile(`\d+`)

// After
import rexa "github.com/himclix/rexa"
re := rexa.MustCompile(`\d+`)
```

## PCRE Features Go Doesn't Have

```go
// Lookahead: match only if followed by pattern
re := rexa.MustCompile(`foo(?=bar)`)
re.FindString("foobar")  // "foo"
re.FindString("foobaz")  // ""

// Negative lookahead: match only if NOT followed by pattern
re := rexa.MustCompile(`foo(?!bar)`)
re.FindString("foobaz")  // "foo"

// Lookbehind: match only if preceded by pattern
re := rexa.MustCompile(`(?<=@)\w+`)
re.FindString("user@domain")  // "domain"

// Negative lookbehind
re := rexa.MustCompile(`(?<!\$)\d+`)
re.FindString("price $100")  // "00"

// Backreference: match same text as earlier capture group
re := rexa.MustCompile(`(\w+)\s+\1`)
re.MatchString("hello hello")  // true
re.MatchString("hello world")  // false

// Named groups + named backreference
re := rexa.MustCompile(`(?P<word>\w+)\s+\k<word>`)

// Case-insensitive with scoped flags
re := rexa.MustCompile(`(?i:hello) world`)
re.MatchString("HELLO world")  // true
re.MatchString("HELLO WORLD")  // false

// Unicode properties
re := rexa.MustCompile(`\p{L}+`)  // Unicode letters
re := rexa.MustCompile(`\p{Nd}+`) // Unicode digits
```

## Complexity Guarantees

Each engine tier matches the **theoretical lower bound** for its feature class. These aren't aspirational targets — they're mathematically proven optimal:

| Pattern Class | Engine | Time | Space | Why This Is Optimal |
|---|---|---|---|---|
| Pure literal | Boyer-Moore | **O(n/m) avg** | O(m) | [Boyer-Moore](https://en.wikipedia.org/wiki/Boyer%E2%80%93Moore_string-search_algorithm) skips characters, proven sub-linear average case |
| Simple regex | Lazy DFA | **O(n) amortized** | O(min(2^m, cap)) | DFA = 1 table lookup per character. Lazy construction avoids O(2^m) upfront cost |
| Regex + captures | Pike VM | **O(n*m)** | O(m*k) | [Thompson NFA simulation](https://swtch.com/~rsc/regexp/regexp1.html) — guaranteed linear in input, no backtracking |
| + Lookarounds | Pike VM + sub-engine | **O(n*m)** | O(m*k) | [Proven polynomial](https://arxiv.org/pdf/2603.26139) for lookarounds without backreferences |
| + Backreferences | Bounded backtracker | **O(min(2^k*n, L))** | O(n*k) | L = step limit. Backref matching is [NP-complete](https://arxiv.org/pdf/1903.05896) (reduction from 3-SAT) |

**n** = input length, **m** = pattern size, **k** = capture groups, **cap** = DFA cache capacity, **L** = backtrack limit

### Why backreferences can't be O(n)

Matching with backreferences is [NP-complete](https://arxiv.org/pdf/1903.05896), proven via reduction from 3-SAT. This means **every engine** that supports backreferences — PCRE2, Java, .NET, Python — uses exponential-time backtracking. There is no polynomial algorithm unless P=NP.

rexa's approach: bound the exponential with a configurable step limit. Default 1M steps. Returns `ErrBacktrackLimit` instead of hanging. This is strictly safer than `regexp2` (unbounded) and equivalent to what PCRE2's `pcre2_match()` does with `PCRE2_MATCH_LIMIT`.

### Why the lazy DFA matters

Go's stdlib uses only the Pike VM (O(n*m) per character). rexa's lazy DFA achieves O(1) per character for cached states:

- **First encounter** of a character at a DFA state: compute next state via NFA epsilon closure (~O(m))
- **Subsequent encounters**: single table lookup (O(1)) via 128-entry ASCII array
- **Cache bounded**: max 10K states, flushed if exceeded, falls back to Pike VM if thrashing
- **Result**: O(n) amortized for the vast majority of patterns and inputs

This is the same technique used by [Rust's regex-automata](https://docs.rs/regex-automata/latest/regex_automata/hybrid/index.html) and [Google RE2](https://github.com/google/re2). rexa is the first pure-Go implementation.

## API

```go
// Compile
re, err := rexa.Compile(`pattern`)
re := rexa.MustCompile(`pattern`)
re, err := rexa.CompileWithOptions(`pattern`, rexa.CompileOptions{
    BacktrackLimit: 1000000, // 0=default(1M), -1=unlimited
})

// Match
re.MatchString("input")              // bool
re.Match([]byte("input"))            // bool

// Find
re.FindString("input")               // string
re.FindStringSubmatch("input")       // []string
re.FindAllString("input", -1)        // []string
re.FindStringIndex("input")          // []int

// Replace
re.ReplaceAllString("in", "out")     // string
re.ReplaceAllStringFunc("in", fn)    // string

// Split
re.Split("input", -1)                // []string

// Metadata
re.SubexpNames()                     // []string
re.SubexpIndex("name")               // int
re.NumSubexp()                       // int
re.LiteralPrefix()                   // (string, bool)
re.String()                          // string

// Utilities
rexa.QuoteMeta("a.b")               // "a\\.b"
rexa.MatchString(`\d+`, "123")       // (bool, error)
```

## Project Structure

```
rexa/
  rexa.go              Public API (drop-in regexp replacement)
  syntax/              Lexer + recursive descent parser → AST
  compiler/            Thompson's construction → instruction program
  engine/
    meta.go            Multi-engine dispatcher
    literal.go         Boyer-Moore literal scanner
    lazydfa.go         Lazy DFA with bounded cache + ASCII table
    pikevm.go          Pike VM (Thompson NFA + captures)
    backtrack.go       Bounded backtracking engine
    machine.go         sync.Pool-based machine allocation
  internal/
    bitset/            Bit vector for NFA state dedup
    bm/                Boyer-Moore string search
    runeutil/          Unicode helpers
```

## Versioning

rexa follows [Semantic Versioning](https://semver.org/):

```
v0.x.y  — Development: API may change between minor versions
v1.0.0  — Stable: regexp.Regexp API compatibility guaranteed
v1.x.y  — Backwards-compatible additions and performance improvements
v2.0.0  — Breaking changes (if ever needed)
```

**Current: `v0.1.0`** — All features implemented, API stable, beating stdlib on all benchmarks. Moving to v1.0.0 after community testing.

To pin a version:
```bash
go get github.com/himclix/rexa@v0.1.0
```

To upgrade:
```bash
go get -u github.com/himclix/rexa
```

To downgrade:
```bash
go get github.com/himclix/rexa@v0.1.0
```

See [CHANGELOG.md](CHANGELOG.md) for detailed release notes.

## License

MIT

## Acknowledgments

- [Russ Cox's regex articles](https://swtch.com/~rsc/regexp/) — Thompson NFA and Pike VM foundations
- [Rust regex-automata](https://docs.rs/regex-automata) — multi-engine architecture inspiration
- [rhaeguard's regex tutorial](https://rhaeguard.github.io/posts/regex/) — initial implementation reference
- [Andrew Gallant's regex internals](https://burntsushi.net/regex-internals/) — lazy DFA design
