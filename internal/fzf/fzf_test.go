package fzf

import "testing"

func TestFilterEmptyQueryKeepsAll(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	got := Filter("", items)
	if len(got) != len(items) {
		t.Fatalf("got %d matches, want %d", len(got), len(items))
	}
	for i, m := range got {
		if m.Index != i {
			t.Errorf("match[%d].Index = %d, want %d", i, m.Index, i)
		}
	}
}

func TestFilterDropsNonMatches(t *testing.T) {
	items := []string{"1080p", "720p", "1080p60 H.264"}
	got := Filter("1080", items)
	if len(got) != 2 {
		t.Fatalf("got %d matches %v, want 2", len(got), got)
	}
	for _, m := range got {
		if m.Index == 1 {
			t.Errorf("720p should not match query 1080")
		}
	}
}

func TestFilterMatchesAcrossDesc(t *testing.T) {
	// picker feeds "Title Desc" as one string — verify words match loosely
	got := Filter("opus", []string{"Opus · 130 kbps", "AAC · 128 kbps"})
	if len(got) != 1 || got[0].Index != 0 {
		t.Fatalf("want only the opus row, got %v", got)
	}
}

func TestFilterSmartCase(t *testing.T) {
	items := []string{"h264", "H264"}
	got := Filter("H264", items) // uppercase query => case-sensitive
	if len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("case-sensitive query should match only H264, got %v", got)
	}
	got = Filter("h264", items) // lowercase query matches both
	if len(got) != 2 {
		t.Fatalf("lowercase query should match both, got %v", got)
	}
}

func TestFilterRankingPrefersBoundaries(t *testing.T) {
	items := []string{"some-random-word-tag", "tag"}
	got := Filter("tag", items)
	if len(got) != 2 {
		t.Fatalf("want both matches, got %v", got)
	}
	if got[0].Index != 1 {
		t.Errorf("exact item should rank first, got %v", got)
	}
}
