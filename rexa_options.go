package rexa

import (
	"errors"

	"github.com/himclix/rexa/syntax"
)

var ErrBacktrackLimit = errors.New("rexa: backtrack limit exceeded")

const DefaultBacktrackLimit = 1_000_000

type CompileOptions struct {
	BacktrackLimit int64        // 0 = default (1M), -1 = unlimited
	CacheCapacity  int          // lazy DFA cache size, 0 = default
	Flags          syntax.Flags // initial flags
}
