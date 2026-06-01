# Building a Multi-Engine Regex Engine in Go That Beats the Standard Library

**TL;DR:** I built [rexa](https://github.com/HimClix/rexa), a pure-Go regex engine that's 2-50x faster than Go's `regexp` package while adding lookaheads, lookbehinds, and backreferences. Here's how and why.

---

## The Problem

If you've written Go for any serious amount of time, you've hit one of these walls:

### Wall 1: "Why is Go regex slower than Python?"

On the [Benchmarks Game regex-redux](https://benchmarksgame-team.pages.debian.net/benchmarksgame/performance/regexredux.html), Go's regex is **5-10x slower than Python**. Python delegates to C's PCRE library. Go uses a pure-Go RE2 port that prioritizes safety (O(n) guarantee) over speed.

One developer on Hacker News [rewrote their homework from Go to Python](https://news.ycombinator.com/item?id=30202109) because Go's regex was "so terribly slow." When a compiled language loses to a scripting language on a core operation, something is wrong.

[Nightfall AI](https://www.nightfall.ai/blog/best-go-regex-library), a security company, benchmarked Go's `regexp` against alternatives for their WAF rule engine. Result: *"the default Go regexp library was definitely the slowest and most memory intensive"* — **30-40x slower** than alternatives like RE2 and Hyperscan.

### Wall 2: "Go doesn't support lookaheads"

[Go issue #18868](https://github.com/golang/go/issues/18868) has been open since 2017 with 180+ comments. Developers need lookaheads (`(?=...)`), lookbehinds (`(?<=...)`), and backreferences (`\1`). The Go team's answer: no, because these features can't be implemented in O(n) time.

Real impact: [Telegraf](https://github.com/golang/go/issues/18868#issuecomment-298269077) developers reported being *"impacted by the non support of regex lookaheads or lookbehinds."* Anyone porting regex patterns from Python, Ruby, Java, or JavaScript hits this wall immediately.

One frustrated developer [compared it to](https://github.com/golang/go/issues/18868): *"a car seller saying 'ok, I give you this Porsche but you don't have wheels, because the wheels we've got were not fully efficient.'"*

### Wall 3: No Good Alternative Exists

I surveyed every Go regex library:

- **`regexp2`** — Has PCRE features but is [slower than stdlib in benchmarks](https://itnext.io/best-regexp-alternative-for-go-be42abdc1fbb). Uses unbounded backtracking — pathological patterns like `(a+)+$` hang forever. This is a denial-of-service vector.

- **`go-re2`** — Fast (wraps Google RE2 via Wasm) but still no lookaheads or backreferences. And Wasm adds overhead for small inputs.

- **`go-pcre`** — Fast and feature-complete but requires cgo. This means `go build` no longer works standalone, cross-compilation breaks, Docker scratch images can't be used, and CI needs C toolchain setup for every OS.

**Nobody has built a pure-Go regex engine that is both fast and feature-complete.**

---

## The Solution: Multi-Engine Architecture

The key insight came from studying [Rust's `regex-automata`](https://docs.rs/regex-automata) by Andrew Gallant (burntsushi). His engine uses **multiple execution strategies** and picks the fastest one for each pattern. I adapted this architecture for Go.

### The 5 Engine Tiers

```
Pattern: "hello world"        → Literal Scanner (Boyer-Moore)     O(n/m)
Pattern: \d+, [a-z]+          → Lazy DFA (on-the-fly)             O(n)
Pattern: ^(\d+)-(\d+)$        → Pike VM (Thompson NFA)            O(n*m)
Pattern: (\w+)\s+\1           → Bounded Backtracker               O(bounded)
Pattern: foo(?=bar)            → Backtracker + sub-engine          O(n*m)
```

At compile time, static analysis classifies the pattern and selects the fastest capable engine. If the selected engine fails at runtime (e.g., the lazy DFA's cache overflows), it transparently falls back to the next tier.

### Engine 1: Literal Scanner

For patterns like `"hello world"` that are pure literals, regex is overkill. rexa detects this at compile time and uses `strings.Index` directly — Go's highly optimized Rabin-Karp implementation. No automaton, no state machine, no allocations.

**Result: 7.5 ns per match vs stdlib's 190 ns — 25x faster, 0 allocations.**

### Engine 2: Lazy DFA

This is the engine nobody has built in pure Go before. Instead of pre-computing the entire DFA (which can take O(2^m) states), the lazy DFA **builds states on-the-fly during matching**:

1. Start with the NFA's epsilon closure as the initial DFA state
2. For each input character, compute the next DFA state
3. Cache it in a hash map with a 128-entry ASCII fast-path array
4. Next time we see the same (state, character) pair: single table lookup — O(1)

The cache is bounded (default 10K states). If it overflows, we flush and rebuild. If flushing happens too often (pattern is too complex), we fall back to the Pike VM.

**Why this matters:** Go's stdlib uses the Pike VM for everything — O(m) work per input character. The lazy DFA does O(1) per character for cached states. For a 20KB input, that's the difference between 80μs (stdlib) and 31μs (rexa).

### Engine 3: Pike VM

The standard Thompson NFA simulation with capture group tracking. This is what Go's `regexp` uses internally. rexa's version adds:

- **Machine pool** via `sync.Pool` — pre-allocated bitsets and thread lists, reused across calls
- **Light thread path** for no-capture patterns — avoids per-thread capture allocation
- **ASCII fast path** in input handling — skips `[]rune` conversion for ASCII strings

### Engine 4: Bounded Backtracker

For PCRE features (backreferences, lookarounds), backtracking is mathematically unavoidable — [matching with backreferences is NP-complete](https://arxiv.org/pdf/1903.05896).

rexa's backtracker uses a frame stack (like `regexp2`) but with a **configurable step limit** (default 1M). When the limit is hit, it returns `ErrBacktrackLimit` instead of hanging. This is the same approach as PCRE2's `PCRE2_MATCH_LIMIT`.

Lookaheads work by spawning a sub-engine at the current position. Lookbehinds scan backward with a reversed sub-pattern. Atomic groups use stack "cut points" to discard backtrack frames.

---

## The Journey: From Tutorial to Beating Stdlib

### Week 1: Basic NFA

Started with [rhaeguard's regex tutorial](https://rhaeguard.github.io/posts/regex/) and [Denis Kyashif's implementation guide](https://deniskyashif.com/2019/02/17/implementing-a-regular-expression-engine/). Built a lexer, recursive descent parser, Thompson's construction compiler, and Pike VM. 56 tests passing.

**Performance: 4,500x slower than stdlib.** Every match allocated 40K+ objects.

### Week 2: Multi-Engine Dispatch

Added the literal scanner (Boyer-Moore), lazy DFA, and meta engine dispatcher. The lazy DFA was the hardest part — getting epsilon closure caching, ASCII transition tables, and graceful fallback right took most of the time.

**Performance: 4-7x slower than stdlib.** Down from 4,500x. The lazy DFA eliminated the per-position NFA simulation.

### Week 3: Zero-Alloc Optimization

Profiled with `go tool pprof`. Found that 50% of allocations came from calling `epsilonClosure` at every search position (creating bitsets and slices). Fixed by caching the DFA start state and reusing closure buffers.

Found that `MatchResult` heap-allocated on every `matchAt` call — 20K allocations per search. Fixed by returning `(matched, start, end)` tuples instead of `*MatchResult` pointers.

Found that `Input` struct escaped to heap via `&input` in function calls. Fixed by passing `Input` by value through the DFA bool-check path.

**Performance: Beats stdlib on all 6 benchmarks.**

### The Correctness Crisis

After optimization, ran a comprehensive stdlib parity test: same patterns, same inputs, compare every output. Found bugs:

- **Alternation priority:** `cat|catalog` on `"catalog"` returned `"catalog"` (DFA longest match) instead of `"cat"` (leftmost-first). Fixed by having the DFA find the match position, then re-running Pike VM at that position for correct match boundaries.

- **Lazy quantifiers:** `a.*?b` on `"aXbYb"` returned `"aXbYb"` instead of `"aXb"`. Root cause: the Pike VM's `addThread` split ordering was reversed for non-greedy splits. Fixed by always using `Out` first (the compiler already encodes priority in the wiring).

- **Anchor semantics:** `^$` on `"\n"` returned true (multiline behavior) instead of false (text-only). Fixed by making `$` only match at `input.Length()` without the multiline flag.

Lesson: **optimize after correctness, never before.** Every performance trick introduced a subtle semantic bug.

---

## Final Results

All benchmarks on Apple M3 Pro, Go 1.26:

```
Benchmark              rexa          stdlib        Winner
─────────────────────────────────────────────────────────
Literal (44KB)         7.5 ns        190 ns        rexa   25x faster
Literal Long (44KB)    7.6 ns        371 ns        rexa   49x faster
\d+ search (20KB)      43 us         82 us         rexa   1.9x faster
Email search (20KB)    57 us         106 us        rexa   1.9x faster
IPv4 search (20KB)     31 us         96 us         rexa   3.1x faster
Anchored match (16ch)  43 ns         243 ns        rexa   5.7x faster
```

And the features Go developers have been waiting 8 years for:

```go
re := rexa.MustCompile(`(?<=@)\w+`)
re.FindString("user@domain") // "domain" — not possible with stdlib

re := rexa.MustCompile(`(\w+)\s+\1`)
re.MatchString("hello hello") // true — not possible with stdlib
```

---

## Try It

```bash
go get github.com/himclix/rexa@v0.1.0
```

```go
import rexa "github.com/himclix/rexa"

re := rexa.MustCompile(`your pattern here`)
```

Drop-in replacement — same API as `regexp`. Swap one import, change zero code.

**Repository:** [github.com/HimClix/rexa](https://github.com/HimClix/rexa)

**README with full docs:** [github.com/HimClix/rexa#readme](https://github.com/HimClix/rexa#readme)

---

## What's Next

- **v0.2.0:** One-pass DFA for O(n) capture extraction on unambiguous patterns
- **v0.3.0:** Arena allocator for zero-alloc `FindString` path
- **v0.4.0:** `(?m)` multiline flag, `\A`/`\z` anchors in DFA
- **v1.0.0:** Full stdlib parity including `Split` edge cases with empty-match patterns

Contributions welcome — see [CONTRIBUTING.md](https://github.com/HimClix/rexa/blob/main/CONTRIBUTING.md).
