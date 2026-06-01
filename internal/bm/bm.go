package bm

// Searcher implements Boyer-Moore-Horspool string search.
// For short patterns it falls back to a simple scan, but for
// patterns >= 4 runes the bad-character shift table gives
// O(n/m) average-case performance.
type Searcher struct {
	pattern []rune
	shift   [256]int // bad-character shift table (ASCII portion)
	fullMap map[rune]int // shift for non-ASCII runes in the pattern
}

func New(pattern []rune) *Searcher {
	s := &Searcher{
		pattern: pattern,
		fullMap: make(map[rune]int),
	}
	m := len(pattern)
	if m == 0 {
		return s
	}

	// Default shift = pattern length
	for i := range s.shift {
		s.shift[i] = m
	}

	// For each character in the pattern (except last), shift = distance from end
	for i := 0; i < m-1; i++ {
		r := pattern[i]
		dist := m - 1 - i
		if r < 256 {
			s.shift[r] = dist
		}
		s.fullMap[r] = dist
	}

	return s
}

// Search returns the index of the first occurrence of the pattern
// in text starting at or after position start. Returns -1 if not found.
func (s *Searcher) Search(text []rune, start int) int {
	m := len(s.pattern)
	n := len(text)

	if m == 0 {
		return start
	}
	if m > n-start {
		return -1
	}

	// For very short patterns, use naive scan
	if m <= 3 {
		return s.naiveSearch(text, start)
	}

	i := start + m - 1
	for i < n {
		j := m - 1
		for j >= 0 && text[i-(m-1-j)] == s.pattern[j] {
			j--
		}
		if j < 0 {
			return i - (m - 1)
		}

		// Shift by bad character rule
		badChar := text[i]
		shift := s.getShift(badChar)
		if shift < 1 {
			shift = 1
		}
		i += shift
	}
	return -1
}

// SearchAll returns all non-overlapping occurrences starting at or after start.
func (s *Searcher) SearchAll(text []rune, start int) []int {
	var results []int
	pos := start
	for {
		idx := s.Search(text, pos)
		if idx < 0 {
			break
		}
		results = append(results, idx)
		pos = idx + len(s.pattern)
		if len(s.pattern) == 0 {
			pos++
		}
	}
	return results
}

func (s *Searcher) getShift(r rune) int {
	if r < 256 {
		return s.shift[r]
	}
	if d, ok := s.fullMap[r]; ok {
		return d
	}
	return len(s.pattern)
}

func (s *Searcher) naiveSearch(text []rune, start int) int {
	m := len(s.pattern)
	n := len(text)
	for i := start; i <= n-m; i++ {
		match := true
		for j := 0; j < m; j++ {
			if text[i+j] != s.pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
