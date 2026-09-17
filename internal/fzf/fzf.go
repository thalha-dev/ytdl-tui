// Package fzf wraps junegunn/fzf's fuzzy matching engine so every picker in
// the TUI filters and ranks results exactly like the fzf CLI does.
package fzf

import (
	"sort"
	"strings"
	"sync"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func init() {
	// Build fzf's char-class / bonus tables for the default scheme.
	algo.Init("default")
}

var (
	slabOnce sync.Once
	slab     *util.Slab
)

func getSlab() *util.Slab {
	slabOnce.Do(func() {
		slab = util.MakeSlab(64*1024, 4*1024*1024)
	})
	return slab
}

// Match is a single fuzzy-match result.
type Match struct {
	Index int    // index of the item in the original slice
	Text  string // the full item text that matched
	Score int    // fzf relevance score; higher is better
}

// Filter ranks items against query using fzf's FuzzyMatchV2 algorithm.
// Items that do not match are dropped; the rest are returned sorted by
// score, best first. An empty (or whitespace-only) query keeps every item
// in its original order.
//
// Matching is smart-case (a query containing uppercase letters is matched
// case-sensitively) and normalizes accented runes, mirroring fzf defaults.
func Filter(query string, items []string) []Match {
	q := strings.TrimSpace(query)
	if q == "" {
		out := make([]Match, len(items))
		for i, t := range items {
			out[i] = Match{Index: i, Text: t}
		}
		return out
	}

	caseSensitive := strings.ToLower(q) != q
	pattern := []rune(strings.ToLower(q))
	if caseSensitive {
		pattern = []rune(q)
	}

	matches := make([]Match, 0, len(items))
	for i, text := range items {
		chars := util.ToChars([]byte(text))
		res, _ := algo.FuzzyMatchV2(caseSensitive, true, true, &chars, pattern, false, getSlab())
		if res.Start < 0 {
			continue // no match
		}
		matches = append(matches, Match{Index: i, Text: text, Score: res.Score})
	}

	sort.SliceStable(matches, func(a, b int) bool {
		return matches[a].Score > matches[b].Score
	})
	return matches
}
