package syntax

import "fmt"

type SyntaxError struct {
	Pattern string
	Pos     int
	Msg     string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("rexa: syntax error at position %d in `%s`: %s", e.Pos, e.Pattern, e.Msg)
}

func syntaxErr(pattern string, pos int, msg string) *SyntaxError {
	return &SyntaxError{Pattern: pattern, Pos: pos, Msg: msg}
}
