package rank

import "testing"

func TestEdgeCases(t *testing.T) {
	r := mustRanker(t, Default())

	if out := r.RankAll(nil); len(out) != 0 {
		t.Fatalf("empty batch: %v", out)
	}
	if out := r.RankAll([]string{}); len(out) != 0 {
		t.Fatalf("empty slice batch: %v", out)
	}

	empty := r.Rank("")
	if empty.Data == nil {
		t.Fatal("empty title must still produce a parse result")
	}

	junk := r.Rank("!!!###@@@")
	if junk.Data == nil {
		t.Fatal("junk title must still produce a parse result")
	}

	if got := Sort(nil); len(got) != 0 {
		t.Fatalf("Sort(nil): %v", got)
	}

	// profile with a bad pattern must fail loudly at construction
	bad := Default()
	bad.Exclude = []string{"("}
	if _, err := New(bad); err == nil {
		t.Fatal("invalid pattern should error at New")
	}

	// similarity edge cases
	if got := Similarity("", ""); got != 1 {
		t.Fatalf("empty-empty similarity: %f", got)
	}
	if got := Similarity("abc", ""); got != 0 {
		t.Fatalf("abc-empty similarity: %f", got)
	}
}
