// Package match decides whether a Picker query matches a Project's shown
// path, how well, and which runes to highlight.
//
// A query without a space matches the whole shown path (parent directory
// then name). With a space, the last word matches the name and every
// earlier word must match, in order, within the parent directory: "baz br"
// picks baz.com/barbar over foo.com/barbar. A trailing space leaves the
// name unconstrained and a leading one leaves the parent unconstrained.
//
// Each word matches as a subsequence ("br" matches "barbar") or, failing
// that, as a contiguous stretch within a few edits of it ("brabar" matches
// "barbar"). Every subsequence match ranks before every typo match.
package match

import (
	"strings"
	"unicode"
)

// Query is a parsed Picker query.
type Query struct {
	empty     bool
	whole     bool     // no space: name is matched against the whole path
	parents   [][]rune // words matched in order within the parent directory
	name      []rune   // word matched against the name; may be empty
	sensitive bool     // smart case: the query holds an uppercase letter
}

// Result is how a Query matched one Project.
type Result struct {
	// Typo reports whether any word matched only within its typo allowance.
	Typo bool
	// Score orders Results within the same Typo tier; higher is better.
	Score int
	// Indexes are the matched runes' offsets into dir+name, ascending.
	Indexes []int
}

// Less reports whether a ranks before b: subsequence matches before typo
// matches, then by Score.
func Less(a, b Result) bool {
	if a.Typo != b.Typo {
		return !a.Typo
	}
	return a.Score > b.Score
}

// Parse splits q into its words. A query of only whitespace is Empty.
func Parse(q string) Query {
	if strings.TrimSpace(q) == "" {
		return Query{empty: true}
	}
	query := Query{sensitive: strings.IndexFunc(q, unicode.IsUpper) >= 0}
	i := strings.LastIndexByte(q, ' ')
	if i < 0 {
		query.whole = true
		query.name = query.fold([]rune(q))
		return query
	}
	query.name = query.fold([]rune(q[i+1:]))
	for _, w := range strings.Fields(q[:i]) {
		query.parents = append(query.parents, query.fold([]rune(w)))
	}
	return query
}

// Empty reports whether the query filters nothing.
func (q Query) Empty() bool { return q.empty }

// fold lowercases rs unless the query is case sensitive.
func (q Query) fold(rs []rune) []rune {
	if q.sensitive {
		return rs
	}
	out := make([]rune, len(rs))
	for i, r := range rs {
		out[i] = unicode.ToLower(r)
	}
	return out
}

// Match matches q against a Project shown as dir (ending in a separator)
// followed by name.
func (q Query) Match(dir, name string) (Result, bool) {
	if q.empty {
		return Result{}, true
	}
	d, n := []rune(dir), []rune(name)
	if q.whole {
		text := append(append([]rune{}, d...), n...)
		m, ok := matchWord(q.name, text, q.fold(text), 0, false)
		if !ok {
			return Result{}, false
		}
		m.bonusFrom(len(d), regionBonus)
		return m.result(), true
	}

	var res Result
	if len(q.parents) > 0 {
		pm, ok := q.matchParents(d, false)
		if !ok {
			pm, ok = q.matchParents(d, true)
		}
		if !ok {
			return Result{}, false
		}
		res = pm
	}
	if len(q.name) > 0 {
		m, ok := matchWord(q.name, n, q.fold(n), 0, false)
		if !ok {
			return Result{}, false
		}
		m.shift(len(d))
		r := m.result()
		res.Typo = res.Typo || r.Typo
		res.Score += r.Score
		res.Indexes = append(res.Indexes, r.Indexes...)
	}
	return res, true
}

// matchParents matches the parent words in order within dir, each starting
// after the previous one ends. Greedy on score first; early takes each
// word's earliest-ending match instead, so a later word is never crowded
// out by a better-scoring earlier one.
func (q Query) matchParents(dir []rune, early bool) (Result, bool) {
	fold := q.fold(dir)
	immediate := immediateParent(dir)
	var res Result
	from := 0
	for _, w := range q.parents {
		m, ok := matchWord(w, dir, fold, from, early)
		if !ok {
			return Result{}, false
		}
		m.bonusFrom(immediate, regionBonus)
		r := m.result()
		res.Typo = res.Typo || r.Typo
		res.Score += r.Score
		res.Indexes = append(res.Indexes, r.Indexes...)
		from = r.Indexes[len(r.Indexes)-1] + 1
	}
	return res, true
}

// immediateParent returns the rune offset where dir's last segment starts.
func immediateParent(dir []rune) int {
	for i := len(dir) - 2; i >= 0; i-- {
		if dir[i] == '/' {
			return i + 1
		}
	}
	return 0
}
