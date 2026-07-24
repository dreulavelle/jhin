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

func TestCodexFindings(t *testing.T) {
	// threshold validation
	bad := Default()
	bad.Options.TitleThreshold = 1.5
	if _, err := New(bad); err == nil {
		t.Fatal("threshold > 1 should error")
	}

	// profile mutation after New must not affect the ranker
	p := Default()
	p.Attributes = map[Attr]Policy{AttrRemux: {Fetch: true, Rank: 123}}
	r := mustRanker(t, p)
	before := r.Rank("Movie.2020.1080p.BluRay.REMUX-GRP").Rank
	p.Attributes[AttrRemux] = Policy{Fetch: false, Rank: -999}
	p.Resolutions[Res1080p] = false
	after := r.Rank("Movie.2020.1080p.BluRay.REMUX-GRP")
	if after.Rank != before || !after.Fetch {
		t.Fatalf("ranker must snapshot the profile: before=%d after=%+v", before, after)
	}

	// per-entry infohash batch
	out := r.RankEntries([]Entry{
		{Title: "Movie.A.2020.1080p.WEB-DL", Infohash: "aaa"},
		{Title: "Movie.B.2021.720p.WEB-DL", Infohash: "bbb"},
	})
	if out[0].Infohash != "aaa" || out[1].Infohash != "bbb" {
		t.Fatalf("infohash not carried: %+v", out)
	}
}

func TestAttributesExported(t *testing.T) {
	r := mustRanker(t, Default())
	tor := r.Rank("Movie.2020.2160p.BluRay.REMUX.DV.TrueHD.7.1.Atmos-GRP")
	attrs := Attributes(tor.Data)
	want := map[Attr]bool{AttrRemux: true, AttrDolbyVision: true, AttrTrueHD: true, AttrAtmos: true, AttrSurround: true}
	found := map[Attr]bool{}
	for _, a := range attrs {
		found[a] = true
	}
	for a := range want {
		if !found[a] {
			t.Errorf("missing attribute %s in %v", a, attrs)
		}
	}
}
