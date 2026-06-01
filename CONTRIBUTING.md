# Contributing to rexa

Thanks for your interest in contributing to rexa.

## Getting Started

```bash
git clone https://github.com/HimClix/rexa.git
cd rexa
go test ./...
```

## Development

### Running Tests

```bash
go test ./...                          # all tests
go test ./... -race                    # with race detector
go test -run TestStdlibParity          # stdlib compatibility
go test -run TestPCREFeatures          # PCRE feature tests
go test -bench=. -benchmem             # benchmarks
```

### Project Structure

```
syntax/     Lexer + parser → AST (start here to add new syntax)
compiler/   AST → instruction program (Thompson's construction)
engine/     Execution engines (where performance work happens)
  meta.go       Engine dispatcher
  lazydfa.go    Lazy DFA (O(n) matching)
  pikevm.go     Pike VM (captures/anchors)
  backtrack.go  Bounded backtracker (PCRE features)
  literal.go    Boyer-Moore literal scanner
internal/   Shared utilities (bitset, string search, Unicode)
```

### Adding a New Feature

1. Add token type in `syntax/token.go`
2. Handle it in the lexer (`syntax/lexer.go`)
3. Parse it into an AST node (`syntax/parser.go`, `syntax/tree.go`)
4. Compile it to instructions (`compiler/nfa.go`, `compiler/inst.go`)
5. Execute it in the appropriate engine (`engine/`)
6. Add tests that verify against Go's `regexp` where possible

### Performance Rules

- Never add allocations to the `MatchString` hot path (must stay 0 alloc)
- Benchmark before and after: `go test -bench=. -benchmem -count=5`
- Profile with: `go test -bench=BenchmarkX -cpuprofile=cpu.prof -memprofile=mem.prof`
- The lazy DFA path must stay O(n) amortized

## Reporting Issues

- Include the regex pattern, input string, expected vs actual output
- For performance issues, include benchmark numbers

## Pull Requests

- One feature per PR
- Add tests for new functionality
- Run `go test ./... -race` before submitting
- Don't break existing benchmarks (no performance regressions)
