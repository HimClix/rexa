package compiler

type Program struct {
	Insts           []Inst
	Start           int
	NumCap          int
	CapNames        map[string]int
	NeedsBacktrack  bool
	NeedsLookaround bool
	HasAnchors      bool
	AnchoredStart   bool // pattern starts with ^ or \A
	OnlyStartAnchor bool // only anchor is ^ at start (safe for DFA)
	IsLiteral       bool
	LiteralPrefix   string
	PrefixComplete  bool
}

func (p *Program) HasWordBoundary() bool {
	for _, inst := range p.Insts {
		if inst.Op == InstWordBoundary || inst.Op == InstNoWordBoundary {
			return true
		}
	}
	return false
}

func (p *Program) AddInst(inst Inst) int {
	idx := len(p.Insts)
	p.Insts = append(p.Insts, inst)
	return idx
}
