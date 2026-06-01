package compiler

func Optimize(p *Program) {
	extractLiteralPrefix(p)
	checkIsLiteral(p)
	checkHasAnchors(p)
}

func checkHasAnchors(p *Program) {
	for _, inst := range p.Insts {
		switch inst.Op {
		case InstBeginLine, InstEndLine, InstBeginText, InstEndText,
			InstWordBoundary, InstNoWordBoundary:
			p.HasAnchors = true
		}
	}
	hasEndAnchor := false
	hasWordBoundary := false
	hasStartAnchor := false
	for _, inst := range p.Insts {
		switch inst.Op {
		case InstBeginLine, InstBeginText:
			hasStartAnchor = true
		case InstEndLine, InstEndText:
			hasEndAnchor = true
		case InstWordBoundary, InstNoWordBoundary:
			hasWordBoundary = true
		}
	}

	// Check if pattern starts with ^/\A
	pc := p.Start
	seen := make(map[int]bool)
	for pc >= 0 && pc < len(p.Insts) && !seen[pc] {
		seen[pc] = true
		inst := &p.Insts[pc]
		switch inst.Op {
		case InstBeginLine, InstBeginText:
			p.AnchoredStart = true
			pc = inst.Out
		case InstCapStart, InstCapEnd, InstJump:
			pc = inst.Out
		default:
			goto done
		}
	}
done:
	p.OnlyStartAnchor = hasStartAnchor && !hasEndAnchor && !hasWordBoundary
}

// extractLiteralPrefix walks the instruction list from Start,
// following only InstRune instructions (no splits, no classes).
// If a contiguous sequence of literal runes is found at the start,
// it's stored in Program.LiteralPrefix for use as a prefilter.
func extractLiteralPrefix(p *Program) {
	var prefix []rune
	pc := p.Start
	seen := make(map[int]bool)

	for pc >= 0 && pc < len(p.Insts) && !seen[pc] {
		seen[pc] = true
		inst := &p.Insts[pc]

		switch inst.Op {
		case InstRune:
			if inst.CaseInsensitive {
				goto done
			}
			prefix = append(prefix, inst.Rune)
			pc = inst.Out
		case InstCapStart, InstCapEnd:
			pc = inst.Out
		case InstJump:
			pc = inst.Out
		default:
			goto done
		}
	}

done:
	if len(prefix) > 0 {
		p.LiteralPrefix = string(prefix)
	}
}

// checkIsLiteral checks if the entire program is a simple literal match
// (no quantifiers, classes, or capture groups — just a sequence of InstRune → InstMatch).
func checkIsLiteral(p *Program) {
	if p.NumCap > 0 {
		return
	}
	pc := p.Start
	seen := make(map[int]bool)

	for pc >= 0 && pc < len(p.Insts) && !seen[pc] {
		seen[pc] = true
		inst := &p.Insts[pc]

		switch inst.Op {
		case InstRune:
			if inst.CaseInsensitive {
				return
			}
			pc = inst.Out
		case InstCapStart, InstCapEnd:
			pc = inst.Out
		case InstJump:
			pc = inst.Out
		case InstMatch:
			p.IsLiteral = true
			p.PrefixComplete = true
			return
		default:
			return
		}
	}
}
