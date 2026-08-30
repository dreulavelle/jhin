package parser

import (
	"reflect"
	"testing"
)

func TestRangeCaps(t *testing.T) {
	if r := Parse("Show episodes 5-9999 x264"); len(r.Episodes) > maxRangeTill {
		t.Fatalf("range not capped: %d", len(r.Episodes))
	}
	if r := Parse("(500000000 of 1)"); len(r.Episodes) != 0 {
		t.Fatalf("range-till not capped: %d", len(r.Episodes))
	}
	if r := Parse("Show S01-S03 1080p"); len(r.Seasons) != 3 {
		t.Fatalf("legit season range broke: %v", r.Seasons)
	}
}

// A zero-padded upper bound wider than its lower bound is an anime absolute
// episode number trailing the season episode, not the end of a range.
func TestHybridAnimeEpisodeIsNotARange(t *testing.T) {
	for _, tc := range []struct {
		title string
		want  []int
	}{
		{"Dr..STONE.(2019)-S04E16-074-1080p-WEB-DL-GROUP", []int{16}},
		{"Dr.STONE.2019.S04E16.[074].1080p.WEB-DL-GROUP", []int{16}},
		{"Show.S04E16-074.1080p", []int{16}},
		{"Show.S01E01-E02.1080p", []int{1, 2}},
		{"Show.S01E01E02.1080p", []int{1, 2}},
		{"Show.S01E01-02.1080p", []int{1, 2}},
		{"Show.S01E01-E03.1080p", []int{1, 2, 3}},
		{"Show 001-012 1080p", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"Show 074-075 1080p", []int{74, 75}},
	} {
		if got := Parse(tc.title).Episodes; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: episodes = %v, want %v", tc.title, got, tc.want)
		}
	}
}
