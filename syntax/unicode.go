package syntax

import "unicode"

var unicodeCategories = map[string]*unicode.RangeTable{
	"L":  unicode.L,
	"Lu": unicode.Lu,
	"Ll": unicode.Ll,
	"Lt": unicode.Lt,
	"Lm": unicode.Lm,
	"Lo": unicode.Lo,
	"M":  unicode.M,
	"Mn": unicode.Mn,
	"Mc": unicode.Mc,
	"Me": unicode.Me,
	"N":  unicode.N,
	"Nd": unicode.Nd,
	"Nl": unicode.Nl,
	"No": unicode.No,
	"P":  unicode.P,
	"Pc": unicode.Pc,
	"Pd": unicode.Pd,
	"Ps": unicode.Ps,
	"Pe": unicode.Pe,
	"Pi": unicode.Pi,
	"Pf": unicode.Pf,
	"Po": unicode.Po,
	"S":  unicode.S,
	"Sm": unicode.Sm,
	"Sc": unicode.Sc,
	"Sk": unicode.Sk,
	"So": unicode.So,
	"Z":  unicode.Z,
	"Zs": unicode.Zs,
	"Zl": unicode.Zl,
	"Zp": unicode.Zp,
	"C":  unicode.C,
	"Cc": unicode.Cc,
	"Cf": unicode.Cf,
	"Co": unicode.Co,
	"Cs": unicode.Cs,
}

var unicodeScripts = map[string]*unicode.RangeTable{
	"Arabic":     unicode.Arabic,
	"Armenian":   unicode.Armenian,
	"Bengali":    unicode.Bengali,
	"Cyrillic":   unicode.Cyrillic,
	"Devanagari": unicode.Devanagari,
	"Georgian":   unicode.Georgian,
	"Greek":      unicode.Greek,
	"Gujarati":   unicode.Gujarati,
	"Gurmukhi":   unicode.Gurmukhi,
	"Han":        unicode.Han,
	"Hangul":     unicode.Hangul,
	"Hebrew":     unicode.Hebrew,
	"Hiragana":   unicode.Hiragana,
	"Kannada":    unicode.Kannada,
	"Katakana":   unicode.Katakana,
	"Latin":      unicode.Latin,
	"Malayalam":  unicode.Malayalam,
	"Oriya":      unicode.Oriya,
	"Tamil":      unicode.Tamil,
	"Telugu":     unicode.Telugu,
	"Thai":       unicode.Thai,
	"Tibetan":    unicode.Tibetan,
}

func LookupUnicodeClass(name string) (*unicode.RangeTable, bool) {
	if t, ok := unicodeCategories[name]; ok {
		return t, true
	}
	if t, ok := unicodeScripts[name]; ok {
		return t, true
	}
	return nil, false
}
