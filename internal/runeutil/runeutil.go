package runeutil

import "unicode"

func IsWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_' ||
		unicode.IsLetter(r) || unicode.IsDigit(r)
}

func EqualFold(a, b rune) bool {
	if a == b {
		return true
	}
	return unicode.ToLower(a) == unicode.ToLower(b)
}

func ToLower(r rune) rune {
	return unicode.ToLower(r)
}

func ToUpper(r rune) rune {
	return unicode.ToUpper(r)
}
