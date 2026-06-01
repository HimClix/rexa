package bitset

type BitSet struct {
	bits []uint64
}

func New(size int) *BitSet {
	return &BitSet{bits: make([]uint64, (size+63)/64)}
}

func (b *BitSet) Set(i int) {
	b.bits[i/64] |= 1 << uint(i%64)
}

func (b *BitSet) Has(i int) bool {
	return b.bits[i/64]&(1<<uint(i%64)) != 0
}

func (b *BitSet) Clear() {
	for i := range b.bits {
		b.bits[i] = 0
	}
}
