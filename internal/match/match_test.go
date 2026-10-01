package match_test

import (
	"slices"
	"testing"

	"github.com/kryft-dev/cdd/internal/match"
)

// target is one Project as the Picker shows it: its parent directory,
// ending in a separator, and its name.
type target struct{ dir, name string }

var (
	fooBarbar = target{"~/Developer/domain/foo.com/", "barbar"}
	bazBarbar = target{"~/Developer/domain/baz.com/", "barbar"}
	toolsCdd  = target{"~/Developer/tools/", "cdd"}
	kryftCdd  = target{"~/kryft/tools/", "cdd"}
)

func TestParse_Empty(t *testing.T) {
	for _, q := range []string{"", " ", "   ", "\t"} {
		if !match.Parse(q).Empty() {
			t.Errorf("Parse(%q).Empty() = false, want true", q)
		}
	}
	for _, q := range []string{"a", "baz ", " br"} {
		if match.Parse(q).Empty() {
			t.Errorf("Parse(%q).Empty() = true, want false", q)
		}
	}
}

func TestMatch_Matches(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		target target
		want   bool
	}{
		{"whole path subsequence", "barbar", fooBarbar, true},
		{"whole path across separator", "ools/cdd", toolsCdd, true},
		{"whole path abbreviation", "br", fooBarbar, true},
		{"no match", "zzz", fooBarbar, false},
		{"parent then name", "baz br", bazBarbar, true},
		{"parent excludes other parent", "baz br", fooBarbar, false},
		{"parent must not match name", "barbar br", bazBarbar, false},
		{"name must not match parent", "baz baz", bazBarbar, false},
		{"any ancestor as parent", "dom br", fooBarbar, true},
		{"parent words in order", "kryft tools cdd", kryftCdd, true},
		{"parent words out of order", "tools kryft cdd", kryftCdd, false},
		{"earlier parent word yields to a later one", "to tools cdd", target{"~/to-x/tools/", "cdd"}, true},
		{"parent words missing one", "kryft tools cdd", toolsCdd, false},
		{"double space is one separator", "baz  br", bazBarbar, true},
		{"trailing space parent only", "baz ", bazBarbar, true},
		{"trailing space parent only excludes", "baz ", fooBarbar, false},
		{"leading space name only", " br", fooBarbar, true},
		{"leading space name only excludes parent hit", " dom", fooBarbar, false},
		{"smart case lower matches upper", "developer", toolsCdd, true},
		{"smart case upper is sensitive", "DEV", toolsCdd, false},
		{"smart case upper matches", "Dev", toolsCdd, true},
		{"typo substitution", "barbat", fooBarbar, true},
		{"typo transposition", "brabar", fooBarbar, true},
		{"typo missing rune", "barbr", fooBarbar, true},
		{"typo extra rune", "barbaar", fooBarbar, true},
		{"typo in parent word", "kryfy tools cdd", kryftCdd, true},
		{"short word gets no typo", "cdx", toolsCdd, false},
		{"one edit too many for medium word", "brbat", fooBarbar, false},
		{"two edits for long word", "developre", toolsCdd, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := match.Parse(tt.query).Match(tt.target.dir, tt.target.name)
			if ok != tt.want {
				t.Errorf("Parse(%q).Match(%q, %q) ok = %v, want %v", tt.query, tt.target.dir, tt.target.name, ok, tt.want)
			}
		})
	}
}

func TestMatch_Typo(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"barbar", false},
		{"br", false},
		{"baz br", false},
		{"barbat", true},
		{"brabar", true},
		{"bza barbat", false}, // bza is too short for a typo: no match at all
	}
	for _, tt := range tests {
		r, ok := match.Parse(tt.query).Match(bazBarbar.dir, bazBarbar.name)
		if tt.query == "bza barbat" {
			if ok {
				t.Errorf("Parse(%q) matched, want no match", tt.query)
			}
			continue
		}
		if !ok {
			t.Fatalf("Parse(%q) did not match", tt.query)
		}
		if r.Typo != tt.want {
			t.Errorf("Parse(%q).Typo = %v, want %v", tt.query, r.Typo, tt.want)
		}
	}
}

func TestMatch_Indexes(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		target target
		want   []int // rune indexes into dir+name
	}{
		// "~/Developer/tools/" is 18 runes, so the name starts at 18.
		{"name only", " cdd", toolsCdd, []int{18, 19, 20}},
		{"whole path prefers name", "cdd", toolsCdd, []int{18, 19, 20}},
		{"parent and name", "tools cdd", toolsCdd, []int{12, 13, 14, 15, 16, 18, 19, 20}},
		// barbat against barbar: the substituted last rune stays plain.
		{"typo highlights matched runes only", " barbat", target{"~/x/", "barbar"}, []int{4, 5, 6, 7, 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ok := match.Parse(tt.query).Match(tt.target.dir, tt.target.name)
			if !ok {
				t.Fatalf("Parse(%q) did not match", tt.query)
			}
			if !slices.Equal(r.Indexes, tt.want) {
				t.Errorf("Parse(%q).Indexes = %v, want %v", tt.query, r.Indexes, tt.want)
			}
		})
	}
}

// TestMatch_Ranking checks Less's order: exact before typo, then score.
func TestMatch_Ranking(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		better, worse target
	}{
		{"name hit beats parent hit", "tools", target{"~/x/", "tools"}, target{"~/tools/", "x"}},
		{"immediate parent beats ancestor", "dom br", target{"~/a/dom/", "barbar"}, target{"~/dom/a/", "barbar"}},
		{"consecutive beats scattered", "cdd", target{"~/", "cdd"}, target{"~/", "cxdxd"}},
		{"exact beats typo", "barbar", target{"~/", "barxbxaxr"}, target{"~/", "barbat"}},
		{"word start beats middle", "bar", target{"~/", "bar-x"}, target{"~/", "xxbar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := match.Parse(tt.query)
			b, ok := q.Match(tt.better.dir, tt.better.name)
			if !ok {
				t.Fatalf("better %v did not match %q", tt.better, tt.query)
			}
			w, ok := q.Match(tt.worse.dir, tt.worse.name)
			if !ok {
				t.Fatalf("worse %v did not match %q", tt.worse, tt.query)
			}
			if !match.Less(b, w) || match.Less(w, b) {
				t.Errorf("want %v (%+v) ranked before %v (%+v)", tt.better, b, tt.worse, w)
			}
		})
	}
}
