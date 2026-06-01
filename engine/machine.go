package engine

import (
	"sync"

	"github.com/himclix/rexa/internal/bitset"
)

// machine holds pre-allocated per-match state, pooled via sync.Pool.
// This eliminates the 9 allocs/op from bitsets, thread lists, and Input.
type machine struct {
	nInst   int
	seenA   *bitset.BitSet
	seenB   *bitset.BitSet
	lightA  []lightThread
	lightB  []lightThread
	threadA []thread
	threadB []thread
	input   Input // reusable input slot, avoids heap escape
}

type lightThread struct {
	pc    int
	start int
}

type machinePool struct {
	pool sync.Pool
}

func newMachinePool(nInst int) *machinePool {
	return &machinePool{
		pool: sync.Pool{
			New: func() any {
				return newMachine(nInst)
			},
		},
	}
}

func newMachine(nInst int) *machine {
	return &machine{
		nInst:   nInst,
		seenA:   bitset.New(nInst),
		seenB:   bitset.New(nInst),
		lightA:  make([]lightThread, 0, nInst),
		lightB:  make([]lightThread, 0, nInst),
		threadA: make([]thread, 0, nInst),
		threadB: make([]thread, 0, nInst),
	}
}

func (mp *machinePool) get() *machine {
	return mp.pool.Get().(*machine)
}

func (mp *machinePool) put(m *machine) {
	m.seenA.Clear()
	m.seenB.Clear()
	m.lightA = m.lightA[:0]
	m.lightB = m.lightB[:0]
	m.threadA = m.threadA[:0]
	m.threadB = m.threadB[:0]
	mp.pool.Put(m)
}
