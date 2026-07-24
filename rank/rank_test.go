package rank

import (
	"math"
	"testing"
)

func mustRanker(t *testing.T, p Profile) *Ranker {
	t.Helper()
	r, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDefaultProfileOrdering(t *testing.T) {
	r := mustRanker(t, Default())

	remux := r.Rank("The.Matrix.1999.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.7.1.Atmos-FGT")
	webdl := r.Rank("The.Matrix.1999.1080p.WEB-DL.DDP5.1.H.264-GRP")
	cam := r.Rank("The.Matrix.1999.CAM.x264-TRASH")

	if !(remux.Rank > webdl.Rank && webdl.Rank > cam.Rank) {
		t.Fatalf("expected remux > webdl > cam, got %d / %d / %d", remux.Rank, webdl.Rank, cam.Rank)
	}
	if !remux.Fetch || !webdl.Fetch {
		t.Fatalf("remux/webdl should be fetchable: %+v %+v", remux.Rejections, webdl.Rejections)
	}
	if cam.Fetch {
		t.Fatal("CAM must be rejected by default")
	}
}

func TestTrashAndAdultVetoes(t *testing.T) {
	r := mustRanker(t, Default())

	if tr := r.Rank("Some.Movie.2020.720p.TS.x264"); tr.Fetch {
		t.Errorf("telesync should be trash-vetoed: %+v", tr)
	}
	if ad := r.Rank("Brazzers.Some.Adult.Title.XXX.1080p"); ad.Fetch {
		t.Errorf("adult content should be vetoed: %+v", ad)
	}

	lenient := Default()
	lenient.Options.RemoveTrash = false
	lenient.Options.RemoveAdult = false
	lenient.Options.MinRank = math.MinInt
	lr := mustRanker(t, lenient)
	// still blocked by the CAM attribute policy, but not by the trash veto
	tor := lr.Rank("Some.Movie.2020.720p.HDTV.x264")
	if !tor.Fetch {
		t.Errorf("hdtv should pass with lenient options: %+v", tor.Rejections)
	}
}

func TestResolutionGate(t *testing.T) {
	r := mustRanker(t, Default())
	if tor := r.Rank("Movie.2020.2160p.WEB-DL.HEVC"); !tor.Fetch {
		t.Fatalf("2160p should pass the default profile: %+v", tor.Rejections)
	}
	tor := r.Rank("Movie.2001.480p.WEB-DL.HEVC")
	if tor.Fetch {
		t.Fatalf("480p should be rejected by the default profile: %+v", tor)
	}
	found := false
	for _, rej := range tor.Rejections {
		if rej == "resolution:480p" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected resolution rejection, got %v", tor.Rejections)
	}

	only1080 := Default()
	only1080.Resolutions[Res2160p] = false
	only1080.Resolutions[Res1440p] = false
	r2 := mustRanker(t, only1080)
	if tor := r2.Rank("Movie.2020.2160p.WEB-DL.HEVC"); tor.Fetch {
		t.Fatalf("2160p should be rejected after disabling: %+v", tor)
	}
}

func TestRequireExcludePreferred(t *testing.T) {
	p := Default()
	p.Exclude = []string{`\bBLOCKED\b`}
	p.Preferred = []string{`\bSPARKS\b`}
	r := mustRanker(t, p)

	if tor := r.Rank("Movie.2020.1080p.WEB-DL.BLOCKED-GRP"); tor.Fetch {
		t.Errorf("exclude pattern should reject: %+v", tor)
	}
	plain := r.Rank("Movie.2020.1080p.WEB-DL-GRP")
	pref := r.Rank("Movie.2020.1080p.WEB-DL-SPARKS")
	if pref.Rank != plain.Rank+p.Options.PreferredBonus {
		t.Errorf("preferred bonus not applied: %d vs %d", pref.Rank, plain.Rank)
	}

	// require short-circuits attribute vetoes (xvid is unfetchable by
	// default) but not the trash veto
	p2 := Default()
	p2.Require = []string{`\bWANTED\b`}
	r2 := mustRanker(t, p2)
	tor := r2.Rank("Movie.2020.1080p.WEB-DL.XviD.WANTED-GRP")
	if !tor.Fetch {
		t.Errorf("require match should bypass attribute vetoes: %+v", tor.Rejections)
	}
	if cam := r2.Rank("Movie.2020.CAM.WANTED"); cam.Fetch {
		t.Error("require must not bypass the trash veto")
	}
}

func TestLanguageRules(t *testing.T) {
	p := Default()
	p.Languages.Exclude = []string{"ru"}
	r := mustRanker(t, p)

	if tor := r.Rank("Фильм Movie 2020 1080p WEB-DL RUS"); tor.Fetch {
		t.Errorf("excluded language should reject: %v %v", tor.Data.Languages, tor.Rejections)
	}
	// English bypasses exclusion by default
	if tor := r.Rank("Movie 2020 1080p WEB-DL RUS ENG"); !tor.Fetch {
		t.Errorf("AllowEnglish should bypass exclusion: %v %v", tor.Data.Languages, tor.Rejections)
	}

	p2 := Default()
	p2.Languages.Required = []string{"fr"}
	r2 := mustRanker(t, p2)
	if tor := r2.Rank("Movie.2020.1080p.WEB-DL"); tor.Fetch {
		t.Errorf("missing required language should reject: %+v", tor)
	}
	if tor := r2.Rank("Movie.2020.FRENCH.1080p.WEB-DL"); !tor.Fetch {
		t.Errorf("required language present should pass: %v %v", tor.Data.Languages, tor.Rejections)
	}
}

func TestTitleMatching(t *testing.T) {
	if !TitleMatch("The Matrix", "The Matrix", 0) {
		t.Fatal("identical titles must match")
	}
	if TitleMatch("The Matrix", "Completely Different Film", 0) {
		t.Fatal("different titles must not match")
	}
	if !TitleMatch("Amélie", "Amelie", 0) {
		t.Fatal("accent folding should make these match")
	}
	if !TitleMatch("Dungeons & Dragons", "Dungeons and Dragons", 0) {
		t.Fatal("& should fold to 'and'")
	}
	if !TitleMatch("The Office (US)", "The Office", 0.85, "The Office US") {
		t.Fatal("alias should match")
	}

	r := mustRanker(t, Default())
	tor := r.Rank("The.Matrix.1999.1080p.WEB-DL.H.264-GRP", RankOptions{TargetTitle: "The Matrix"})
	if !tor.Fetch || tor.TitleRatio < 0.99 {
		t.Fatalf("target title should match: ratio=%f rejections=%v", tor.TitleRatio, tor.Rejections)
	}
	miss := r.Rank("Totally.Other.Movie.2005.1080p.WEB-DL-GRP", RankOptions{TargetTitle: "The Matrix"})
	if miss.Fetch {
		t.Fatalf("title mismatch should reject: ratio=%f", miss.TitleRatio)
	}
}

func TestRankAllPreservesOrder(t *testing.T) {
	r := mustRanker(t, Default())
	titles := []string{
		"Movie.A.2020.1080p.WEB-DL-GRP",
		"Movie.B.2021.720p.HDTV.x264-GRP",
		"Movie.C.2019.CAM-TRASH",
		"Movie.D.2022.1080p.BluRay.REMUX-GRP",
	}
	out := r.RankAll(titles)
	if len(out) != len(titles) {
		t.Fatalf("length mismatch: %d", len(out))
	}
	for i, tor := range out {
		if tor.Raw != titles[i] {
			t.Fatalf("order not preserved at %d: %q", i, tor.Raw)
		}
	}
}

func TestSort(t *testing.T) {
	r := mustRanker(t, Default())
	titles := []string{
		"Movie.2020.720p.WEB-DL-A",
		"Movie.2020.2160p.WEB-DL.DV-B",
		"Movie.2020.1080p.BluRay.REMUX-C",
		"Movie.2020.1080p.WEB-DL-D",
		"Movie.2020.CAM-E",
	}
	out := Sort(r.RankAll(titles), SortOptions{FetchableOnly: true})
	if len(out) != 4 {
		t.Fatalf("CAM should be dropped, got %d", len(out))
	}
	if out[0].Raw != "Movie.2020.2160p.WEB-DL.DV-B" {
		t.Fatalf("2160p bucket should lead: %q", out[0].Raw)
	}
	if out[1].Raw != "Movie.2020.1080p.BluRay.REMUX-C" {
		t.Fatalf("remux should lead 1080p bucket: %q", out[1].Raw)
	}

	limited := Sort(r.RankAll(titles), SortOptions{FetchableOnly: true, BucketLimit: 1})
	if len(limited) != 3 { // one per bucket: 2160p, 1080p, 720p
		t.Fatalf("bucket limit: expected 3, got %d", len(limited))
	}

	only1080 := Sort(r.RankAll(titles), SortOptions{Resolutions: []Resolution{Res1080p}})
	for _, tor := range only1080 {
		if tor.Resolution() != Res1080p {
			t.Fatalf("resolution filter leaked: %q", tor.Raw)
		}
	}
}

func TestProfileRoundTrip(t *testing.T) {
	p := Default()
	p.Attributes = map[Attr]Policy{AttrRemux: {Fetch: false, Rank: -1}}
	path := t.TempDir() + "/profile.json"
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Attributes[AttrRemux].Rank != -1 || loaded.Attributes[AttrRemux].Fetch {
		t.Fatalf("round trip lost attribute policy: %+v", loaded.Attributes)
	}
	if loaded.Options.TitleThreshold != 0.85 {
		t.Fatalf("round trip lost options: %+v", loaded.Options)
	}
}
