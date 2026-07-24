package parser

import "testing"

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
