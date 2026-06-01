package pool

import "sync"

type SlicePool[T any] struct {
	pool sync.Pool
	size int
}

func NewSlicePool[T any](size int) *SlicePool[T] {
	return &SlicePool[T]{
		size: size,
		pool: sync.Pool{
			New: func() any {
				s := make([]T, size)
				return &s
			},
		},
	}
}

func (p *SlicePool[T]) Get() []T {
	sp := p.pool.Get().(*[]T)
	s := *sp
	if len(s) != p.size {
		s = make([]T, p.size)
	}
	return s
}

func (p *SlicePool[T]) Put(s []T) {
	if cap(s) >= p.size {
		s = s[:p.size]
		p.pool.Put(&s)
	}
}
