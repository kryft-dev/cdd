package match

import "slices"

// Scoring weights for one matched rune.
const (
	runeScore        = 16 // every matched rune
	wordStartBonus   = 8  // at the start of the text or of a word in it
	consecutiveBonus = 6  // right after the previous matched rune
	regionBonus      = 4  // in the name, or in the immediate parent
	typoPenalty      = 32 // per edit a typo match needed
)

// wordMatch is one word's match within a text.
type wordMatch struct {
	typo    bool
	score   int
	indexes []int
}

func (m *wordMatch) result() Result {
	return Result{Typo: m.typo, Score: m.score, Indexes: m.indexes}
}

// shift moves the indexes by off, for a word matched in a later segment.
func (m *wordMatch) shift(off int) {
	for i := range m.indexes {
		m.indexes[i] += off
	}
}

// bonusFrom adds bonus for every matched rune at or after from.
func (m *wordMatch) bonusFrom(from, bonus int) {
	for _, i := range m.indexes {
		if i >= from {
			m.score += bonus
		}
	}
}

// matchWord matches word within text[from:], comparing against fold (text
// with the query's case folding applied) and scoring against text. It
// tries a subsequence match first and a typo match after. early prefers
// the match that ends first over the one that scores best.
func matchWord(word, text, fold []rune, from int, early bool) (wordMatch, bool) {
	if idx, ok := subsequence(word, fold, from, text, early); ok {
		return wordMatch{score: scoreOf(idx, text), indexes: idx}, true
	}
	if idx, dist, ok := approximate(word, fold, from, early); ok {
		return wordMatch{typo: true, score: scoreOf(idx, text) - dist*typoPenalty, indexes: idx}, true
	}
	return wordMatch{}, false
}

// allowance is how many edits a word of n runes may need: none for short
// words, which would otherwise match nearly anything.
func allowance(n int) int {
	switch {
	case n <= 3:
		return 0
	case n <= 7:
		return 1
	default:
		return 2
	}
}

// scoreOf scores the matched runes idx of text.
func scoreOf(idx []int, text []rune) int {
	s := 0
	for k, j := range idx {
		s += runeScore + startBonus(text, j)
		if k > 0 {
			if gap := j - idx[k-1] - 1; gap == 0 {
				s += consecutiveBonus
			} else {
				s -= gap
			}
		}
	}
	return s
}

// startBonus rewards a rune that starts the text, a word after a
// separator, or a camelCase hump.
func startBonus(text []rune, j int) int {
	if j == 0 {
		return wordStartBonus
	}
	prev, cur := text[j-1], text[j]
	switch prev {
	case '/', '-', '_', '.', ' ':
		return wordStartBonus
	}
	if isLower(prev) && isUpper(cur) {
		return wordStartBonus
	}
	return 0
}

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }

// subsequence finds word as a subsequence of fold[from:], returning the
// matched offsets with the best score (or, when early, the best of those
// ending first).
func subsequence(word, fold []rune, from int, text []rune, early bool) ([]int, bool) {
	m, n := len(word), len(fold)
	if m == 0 || n-from < m {
		return nil, false
	}
	const none = -1 << 30
	// best[i][j]: best score with word[i] matched at fold[j];
	// prev[i][j]: where word[i-1] was matched on that path.
	best := make([][]int, m)
	prev := make([][]int, m)
	for i := range m {
		best[i] = make([]int, n)
		prev[i] = make([]int, n)
		run, runAt := none, -1 // max of best[i-1][j'] + j' over j' <= j-2
		for j := range n {
			best[i][j] = none
			if i > 0 && j >= 2 && best[i-1][j-2] != none && best[i-1][j-2]+j-2 > run {
				run, runAt = best[i-1][j-2]+j-2, j-2
			}
			if j < from || fold[j] != word[i] {
				continue
			}
			here := runeScore + startBonus(text, j)
			if i == 0 {
				best[i][j] = here
				continue
			}
			if j >= 1 && best[i-1][j-1] != none {
				best[i][j], prev[i][j] = here+best[i-1][j-1]+consecutiveBonus, j-1
			}
			if run != none && here+run-(j-1) > best[i][j] {
				best[i][j], prev[i][j] = here+run-(j-1), runAt
			}
		}
	}
	end := -1
	for j := from; j < n; j++ {
		if best[m-1][j] == none {
			continue
		}
		if end < 0 || (!early && best[m-1][j] > best[m-1][end]) {
			end = j
		}
		if early {
			break
		}
	}
	if end < 0 {
		return nil, false
	}
	idx := make([]int, m)
	for i, j := m-1, end; i >= 0; i-- {
		idx[i] = j
		j = prev[i][j]
	}
	return idx, true
}

// approximate finds a contiguous stretch of fold[from:] within word's typo
// allowance of word (substitutions, insertions, deletions and adjacent
// transpositions), returning the offsets of the runes that matched as-is
// and the edits needed. It prefers the fewest edits, then (unless early)
// the best score.
func approximate(word, fold []rune, from int, early bool) ([]int, int, bool) {
	k := allowance(len(word))
	if k == 0 {
		return nil, 0, false
	}
	text := fold[from:]
	m, n := len(word), len(text)
	// d[i][j]: fewest edits turning word[:i] into a stretch ending at text[j-1].
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
		d[i][0] = i
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 1
			if word[i-1] == text[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && word[i-1] == text[j-2] && word[i-2] == text[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	var bestIdx []int
	bestDist, bestScore := k+1, 0
	for j := 1; j <= n; j++ {
		dist := d[m][j]
		if dist > k || dist > bestDist {
			continue
		}
		idx := backtrace(d, word, text, j)
		if len(idx) == 0 {
			continue
		}
		for i := range idx {
			idx[i] += from
		}
		score := scoreOf(idx, fold)
		if dist < bestDist || (!early && score > bestScore) {
			bestIdx, bestDist, bestScore = idx, dist, score
		}
	}
	return bestIdx, bestDist, bestIdx != nil
}

// backtrace walks d back from (len(word), end) and returns the offsets of
// text runes that matched a word rune, transposed pairs included.
func backtrace(d [][]int, word, text []rune, end int) []int {
	var idx []int
	i, j := len(word), end
	for i > 0 && j > 0 {
		switch {
		case word[i-1] == text[j-1] && d[i][j] == d[i-1][j-1]:
			idx = append(idx, j-1)
			i, j = i-1, j-1
		case i > 1 && j > 1 && word[i-1] == text[j-2] && word[i-2] == text[j-1] && d[i][j] == d[i-2][j-2]+1:
			idx = append(idx, j-1, j-2)
			i, j = i-2, j-2
		case d[i][j] == d[i-1][j-1]+1:
			i, j = i-1, j-1
		case d[i][j] == d[i-1][j]+1:
			i--
		default:
			j--
		}
	}
	slices.Reverse(idx)
	return idx
}
