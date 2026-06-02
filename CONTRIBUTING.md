# Contributing to rexa

Thanks for your interest in contributing. This guide covers everything you need to get started.

## Development Setup

```bash
git clone https://github.com/HimClix/rexa.git
cd rexa
go test ./...          # run all tests
go test -race ./...    # run with race detector
go test -bench=. -benchmem  # run benchmarks
```

## Project Structure

```
syntax/       Lexer + parser → AST (start here to understand the pipeline)
compiler/     AST → instruction program (Thompson's construction)
engine/       Execution engines (where the performance work happens)
  meta.go     Engine dispatcher — reads this to understand how engines compose
  lazydfa.go  Lazy DFA — the primary performance engine
  pikevm.go   Pike VM — correctness reference engine
  backtrack.go Bounded backtracker — handles PCRE features
internal/     Shared utilities (bitset, Boyer-Moore, Unicode helpers)
```

## Performance Rules

Any PR that touches `engine/` must include benchmark results:

```bash
# Before your change
git stash
go test -bench=. -benchmem -count=5 > /tmp/before.txt

# After your change
git stash pop
go test -bench=. -benchmem -count=5 > /tmp/after.txt

# Compare
benchstat /tmp/before.txt /tmp/after.txt
```

**Hard rules:**
- No benchmark may regress by more than 5% without justification
- `MatchString` must remain 0 allocs/op
- Literal path must remain 0 allocs/op
- All engines must beat Go stdlib on their respective benchmarks

## Correctness Rules

- `go test ./... -race` must pass
- `TestStdlibParity` must pass — this verifies Match, Find, FindAll, Replace against `regexp`
- Any new feature needs tests for both positive and negative cases
- PCRE features need tests comparing expected output against Python/PCRE behavior

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Write tests first, then implementation
4. Run `go test ./... -race` and `go test -bench=. -benchmem`
5. Include benchmark results in the PR description
6. Open a PR against `main`

## Areas Where Help Is Welcome

- **Zero-alloc FindString path** — arena allocator for Pike VM capture state
- **One-pass DFA engine** — O(n) capture extraction for unambiguous patterns
- **`(?m)` multiline flag** — `^`/`$` matching at line boundaries
- **Fuzz testing** — `go test -fuzz` comparing against stdlib
- **More golden tests** — porting CPython `re_tests.py` and PCRE2 test vectors
- **Benchmarks against `regexp2` and `go-re2`** — expanding the comparison matrix
