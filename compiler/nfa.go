package compiler

import "github.com/himclix/rexa/syntax"

const unpatched = -1

type fragment struct {
	start int
	out   patchList
}

type patchEntry struct {
	inst  int
	field int // 0 = Out, 1 = Out1
}

type patchList []patchEntry

func (c *Compiler) patchTo(pl patchList, target int) {
	for _, pe := range pl {
		if pe.field == 0 {
			c.prog.Insts[pe.inst].Out = target
		} else {
			c.prog.Insts[pe.inst].Out1 = target
		}
	}
}

type Compiler struct {
	prog *Program
}

func Compile(tree *syntax.Tree) (*Program, error) {
	c := &Compiler{
		prog: &Program{
			CapNames: tree.CapNames,
			NumCap:   tree.NumCap,
		},
	}

	frag, err := c.compile(tree.Root)
	if err != nil {
		return nil, err
	}

	matchIdx := c.prog.AddInst(Inst{Op: InstMatch})
	c.patchTo(frag.out, matchIdx)
	c.prog.Start = frag.start

	return c.prog, nil
}

func (c *Compiler) emit(inst Inst) int {
	return c.prog.AddInst(inst)
}

func (c *Compiler) compile(node *syntax.Node) (fragment, error) {
	switch node.Op {
	case syntax.OpEmpty:
		return c.compileEmpty()
	case syntax.OpLiteral:
		return c.compileLiteral(node)
	case syntax.OpDot:
		return c.compileDot(node)
	case syntax.OpCharClass:
		return c.compileCharClass(node)
	case syntax.OpConcat:
		return c.compileConcat(node)
	case syntax.OpAlternate:
		return c.compileAlternate(node)
	case syntax.OpStar:
		return c.compileRepeatOp(node, 0, -1, true)
	case syntax.OpPlus:
		return c.compileRepeatOp(node, 1, -1, true)
	case syntax.OpQuest:
		return c.compileRepeatOp(node, 0, 1, true)
	case syntax.OpRepeat:
		return c.compileRepeatOp(node, node.Min, node.Max, true)
	case syntax.OpLazy:
		return c.compileLazy(node)
	case syntax.OpCapture, syntax.OpNamedCapture:
		return c.compileCapture(node)
	case syntax.OpGroup:
		return c.compile(node.Children[0])
	case syntax.OpBeginLine:
		return c.compileZeroWidth(InstBeginLine)
	case syntax.OpEndLine:
		return c.compileZeroWidth(InstEndLine)
	case syntax.OpBeginText:
		return c.compileZeroWidth(InstBeginText)
	case syntax.OpEndText:
		return c.compileZeroWidth(InstEndText)
	case syntax.OpWordBoundary:
		return c.compileZeroWidth(InstWordBoundary)
	case syntax.OpNoWordBoundary:
		return c.compileZeroWidth(InstNoWordBoundary)
	case syntax.OpBackref:
		return c.compileBackref(node)
	case syntax.OpLookahead:
		return c.compileLookaround(node, InstLookaheadStart, InstLookaheadEnd, false)
	case syntax.OpNegLookahead:
		return c.compileLookaround(node, InstLookaheadStart, InstLookaheadEnd, true)
	case syntax.OpLookbehind:
		return c.compileLookaround(node, InstLookbehindStart, InstLookbehindEnd, false)
	case syntax.OpNegLookbehind:
		return c.compileLookaround(node, InstLookbehindStart, InstLookbehindEnd, true)
	case syntax.OpAtomic:
		return c.compileAtomic(node)
	case syntax.OpPossessive:
		return c.compilePossessive(node)
	default:
		return c.compileEmpty()
	}
}

func (c *Compiler) compileEmpty() (fragment, error) {
	idx := c.emit(Inst{Op: InstJump, Out: unpatched})
	return fragment{start: idx, out: patchList{{idx, 0}}}, nil
}

func (c *Compiler) compileLiteral(node *syntax.Node) (fragment, error) {
	ci := node.Flags&syntax.FlagCaseInsensitive != 0
	idx := c.emit(Inst{Op: InstRune, Rune: node.Rune, Out: unpatched, CaseInsensitive: ci})
	return fragment{start: idx, out: patchList{{idx, 0}}}, nil
}

func (c *Compiler) compileDot(node *syntax.Node) (fragment, error) {
	op := InstAnyNotNL
	if node.Flags&syntax.FlagDotAll != 0 {
		op = InstAnyChar
	}
	idx := c.emit(Inst{Op: op, Out: unpatched})
	return fragment{start: idx, out: patchList{{idx, 0}}}, nil
}

func (c *Compiler) compileCharClass(node *syntax.Node) (fragment, error) {
	var ranges []Range
	for _, cr := range node.Ranges {
		ranges = append(ranges, Range{Lo: cr.Lo, Hi: cr.Hi})
	}
	idx := c.emit(Inst{
		Op:           InstRuneClass,
		Ranges:       ranges,
		UnicodeTable: node.UnicodeTable,
		Negated:      node.Negated,
		Out:          unpatched,
	})
	return fragment{start: idx, out: patchList{{idx, 0}}}, nil
}

func (c *Compiler) compileConcat(node *syntax.Node) (fragment, error) {
	if len(node.Children) == 0 {
		return c.compileEmpty()
	}
	first, err := c.compile(node.Children[0])
	if err != nil {
		return fragment{}, err
	}
	for i := 1; i < len(node.Children); i++ {
		next, err := c.compile(node.Children[i])
		if err != nil {
			return fragment{}, err
		}
		c.patchTo(first.out, next.start)
		first = fragment{start: first.start, out: next.out}
	}
	return first, nil
}

func (c *Compiler) compileAlternate(node *syntax.Node) (fragment, error) {
	if len(node.Children) == 1 {
		return c.compile(node.Children[0])
	}
	frags := make([]fragment, len(node.Children))
	for i, child := range node.Children {
		f, err := c.compile(child)
		if err != nil {
			return fragment{}, err
		}
		frags[i] = f
	}
	result := frags[len(frags)-1]
	for i := len(frags) - 2; i >= 0; i-- {
		splitIdx := c.emit(Inst{Op: InstSplit, Out: frags[i].start, Out1: result.start, Greedy: true})
		var outs patchList
		outs = append(outs, frags[i].out...)
		outs = append(outs, result.out...)
		result = fragment{start: splitIdx, out: outs}
	}
	return result, nil
}

// compileRepeatOp handles *, +, ?, {n,m} with greedy/lazy control.
func (c *Compiler) compileRepeatOp(node *syntax.Node, min, max int, greedy bool) (fragment, error) {
	child := node.Children[0]

	// For OpRepeat, use node's own min/max
	if node.Op == syntax.OpRepeat {
		min = node.Min
		max = node.Max
	}

	if min == 0 && max == 0 {
		return c.compileEmpty()
	}

	var result *fragment

	// Emit min required copies
	for i := 0; i < min; i++ {
		body, err := c.compile(child)
		if err != nil {
			return fragment{}, err
		}
		if result == nil {
			result = &body
		} else {
			c.patchTo(result.out, body.start)
			result = &fragment{start: result.start, out: body.out}
		}
	}

	if max == -1 {
		// Unbounded: emit a loop (split + body looping back)
		body, err := c.compile(child)
		if err != nil {
			return fragment{}, err
		}
		splitIdx := c.emit(Inst{Op: InstSplit, Greedy: greedy})
		if greedy {
			c.prog.Insts[splitIdx].Out = body.start // prefer body
			// Out1 = skip (unpatched)
		} else {
			c.prog.Insts[splitIdx].Out1 = body.start // prefer skip
			// Out = skip (unpatched)
		}
		c.patchTo(body.out, splitIdx) // loop back

		var skipPatch patchList
		if greedy {
			skipPatch = patchList{{splitIdx, 1}}
		} else {
			skipPatch = patchList{{splitIdx, 0}}
		}

		if result == nil {
			return fragment{start: splitIdx, out: skipPatch}, nil
		}
		c.patchTo(result.out, splitIdx)
		return fragment{start: result.start, out: skipPatch}, nil
	}

	// Bounded optional copies: emit (max - min) optional (split + body) chains
	for i := min; i < max; i++ {
		body, err := c.compile(child)
		if err != nil {
			return fragment{}, err
		}
		splitIdx := c.emit(Inst{Op: InstSplit, Greedy: greedy})
		if greedy {
			c.prog.Insts[splitIdx].Out = body.start
			// Out1 = skip (collect as output)
		} else {
			c.prog.Insts[splitIdx].Out1 = body.start
			// Out = skip (collect as output)
		}

		var skipPatch patchList
		if greedy {
			skipPatch = patchList{{splitIdx, 1}}
		} else {
			skipPatch = patchList{{splitIdx, 0}}
		}

		// body's out feeds into the next optional or becomes final out
		var combinedOut patchList
		combinedOut = append(combinedOut, skipPatch...)
		combinedOut = append(combinedOut, body.out...)

		if result == nil {
			result = &fragment{start: splitIdx, out: combinedOut}
		} else {
			c.patchTo(result.out, splitIdx)
			result = &fragment{start: result.start, out: combinedOut}
		}
	}

	if result == nil {
		return c.compileEmpty()
	}
	return *result, nil
}

func (c *Compiler) compileLazy(node *syntax.Node) (fragment, error) {
	inner := node.Children[0]
	switch inner.Op {
	case syntax.OpStar:
		return c.compileRepeatOp(inner, 0, -1, false)
	case syntax.OpPlus:
		return c.compileRepeatOp(inner, 1, -1, false)
	case syntax.OpQuest:
		return c.compileRepeatOp(inner, 0, 1, false)
	case syntax.OpRepeat:
		return c.compileRepeatOp(inner, inner.Min, inner.Max, false)
	default:
		return c.compile(inner)
	}
}

func (c *Compiler) compileCapture(node *syntax.Node) (fragment, error) {
	startIdx := c.emit(Inst{Op: InstCapStart, Cap: node.Cap})
	body, err := c.compile(node.Children[0])
	if err != nil {
		return fragment{}, err
	}
	endIdx := c.emit(Inst{Op: InstCapEnd, Cap: node.Cap, Out: unpatched})

	c.prog.Insts[startIdx].Out = body.start
	c.patchTo(body.out, endIdx)

	return fragment{start: startIdx, out: patchList{{endIdx, 0}}}, nil
}

func (c *Compiler) compileZeroWidth(op OpCode) (fragment, error) {
	idx := c.emit(Inst{Op: op, Out: unpatched})
	return fragment{start: idx, out: patchList{{idx, 0}}}, nil
}

func (c *Compiler) compileBackref(node *syntax.Node) (fragment, error) {
	c.prog.NeedsBacktrack = true
	idx := c.emit(Inst{Op: InstBackref, Ref: node.Ref, Out: unpatched})
	return fragment{start: idx, out: patchList{{idx, 0}}}, nil
}

func (c *Compiler) compileLookaround(node *syntax.Node, startOp, endOp OpCode, negated bool) (fragment, error) {
	c.prog.NeedsBacktrack = true
	c.prog.NeedsLookaround = true

	body, err := c.compile(node.Children[0])
	if err != nil {
		return fragment{}, err
	}

	bodyMatch := c.emit(Inst{Op: InstMatch})
	c.patchTo(body.out, bodyMatch)

	startIdx := c.emit(Inst{Op: startOp, Out: body.start, Negated: negated, Out1: unpatched})
	return fragment{start: startIdx, out: patchList{{startIdx, 1}}}, nil
}

func (c *Compiler) compileAtomic(node *syntax.Node) (fragment, error) {
	c.prog.NeedsBacktrack = true

	startIdx := c.emit(Inst{Op: InstAtomicStart})
	body, err := c.compile(node.Children[0])
	if err != nil {
		return fragment{}, err
	}
	endIdx := c.emit(Inst{Op: InstAtomicEnd, Out: unpatched})

	c.prog.Insts[startIdx].Out = body.start
	c.patchTo(body.out, endIdx)

	return fragment{start: startIdx, out: patchList{{endIdx, 0}}}, nil
}

func (c *Compiler) compilePossessive(node *syntax.Node) (fragment, error) {
	inner := node.Children[0]
	atomicNode := &syntax.Node{Op: syntax.OpAtomic, Children: inner.Children}
	wrappedNode := &syntax.Node{Op: inner.Op, Children: []*syntax.Node{atomicNode}, Min: inner.Min, Max: inner.Max}

	switch inner.Op {
	case syntax.OpStar:
		return c.compileRepeatOp(wrappedNode, 0, -1, true)
	case syntax.OpPlus:
		return c.compileRepeatOp(wrappedNode, 1, -1, true)
	case syntax.OpQuest:
		return c.compileRepeatOp(wrappedNode, 0, 1, true)
	case syntax.OpRepeat:
		return c.compileRepeatOp(wrappedNode, inner.Min, inner.Max, true)
	default:
		return c.compileAtomic(&syntax.Node{Op: syntax.OpAtomic, Children: node.Children})
	}
}
