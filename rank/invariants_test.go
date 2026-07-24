package rank

// Corpus-driven invariants: every guarantee the package documentation makes
// is enforced here against the full golden corpus.

import (
	"reflect"
	"testing"
)

func openCorpus(t *testing.T) ([]string, *Ranker) {
	t.Helper()
	titles := benchTitles(t)
	p := Default()
	r := mustRanker(t, p)
	return titles, r
}

func TestCorpusInvariants(t *testing.T) {
	titles, r := openCorpus(t)
	batch := r.RankAll(titles)

	for i, tor := range batch {
		// serial/parallel equivalence
		single := r.Rank(titles[i])
		if !reflect.DeepEqual(tor, single) {
			t.Fatalf("batch[%d] != serial rank for %q", i, titles[i])
		}
		// fetch verdict is exactly "no rejections"
		if tor.Fetch != (len(tor.Rejections) == 0) {
			t.Fatalf("fetch/rejections inconsistent for %q: %+v", titles[i], tor)
		}
		// the explanation always sums to the rank
		total := 0
		for _, c := range r.Explain(&tor) {
			total += c.Rank
		}
		if total != tor.Rank {
			t.Fatalf("explain sum %d != rank %d for %q", total, tor.Rank, titles[i])
		}
	}
}

func TestCorpusAttributeCoverage(t *testing.T) {
	titles, r := openCorpus(t)

	for _, title := range titles {
		tor := r.Rank(title)
		d := tor.Data
		if d.Quality != "" {
			if _, ok := qualityAttrs[d.Quality]; !ok {
				t.Errorf("unmapped quality %q from %q", d.Quality, title)
			}
		}
		if d.Codec != "" {
			if _, ok := codecAttrs[d.Codec]; !ok {
				t.Errorf("unmapped codec %q from %q", d.Codec, title)
			}
		}
		for _, h := range d.HDR {
			if _, ok := hdrAttrs[h]; !ok {
				t.Errorf("unmapped hdr %q from %q", h, title)
			}
		}
		for _, a := range d.Audio {
			if _, ok := audioAttrs[a]; !ok {
				t.Errorf("unmapped audio %q from %q", a, title)
			}
		}
		for _, c := range d.Channels {
			if _, ok := channelAttrs[c]; !ok {
				t.Errorf("unmapped channels %q from %q", c, title)
			}
		}
	}
}

func TestCorpusSortInvariants(t *testing.T) {
	titles, r := openCorpus(t)
	sorted := Sort(r.RankAll(titles))

	for i := 1; i < len(sorted); i++ {
		prevBucket := resolutionBucket[sorted[i-1].Resolution()]
		curBucket := resolutionBucket[sorted[i].Resolution()]
		if curBucket > prevBucket {
			t.Fatalf("bucket order violated at %d: %v after %v", i, sorted[i].Resolution(), sorted[i-1].Resolution())
		}
		if curBucket == prevBucket && sorted[i].Rank > sorted[i-1].Rank {
			t.Fatalf("rank order violated within bucket at %d", i)
		}
	}
}

func TestSimilarityProperties(t *testing.T) {
	pairs := [][2]string{
		{"The Matrix", "The Matrix"},
		{"The Matrix", "Matrix Reloaded"},
		{"Amélie", "Amelie"},
		{"", "anything"},
		{"a", "b"},
	}
	for _, p := range pairs {
		ab, ba := Similarity(p[0], p[1]), Similarity(p[1], p[0])
		if ab != ba {
			t.Errorf("similarity not symmetric for %q/%q: %f vs %f", p[0], p[1], ab, ba)
		}
		if ab < 0 || ab > 1 {
			t.Errorf("similarity out of range for %q/%q: %f", p[0], p[1], ab)
		}
	}
	if Similarity("Same Title", "Same Title") != 1 {
		t.Error("identity similarity must be 1")
	}
}

func TestResolutionOrderAndPatternRanks(t *testing.T) {
	// prefer 1080p over 2160p without disabling 4K
	p := Default()
	p.ResolutionOrder = []Resolution{Res1080p, Res720p, Res2160p}
	p.PatternRanks = []PatternRank{
		{Pattern: `\bIMAX\b`, Rank: 500},
		{Pattern: `\bHDCAM\b`, Rank: -2000},
	}
	r := mustRanker(t, p)

	torrents := r.RankAll([]string{
		"Movie.2020.2160p.WEB-DL.HEVC-A",
		"Movie.2020.1080p.WEB-DL.H264-B",
		"Movie.2020.720p.WEB-DL.H264-C",
	})
	sorted := r.Sort(torrents, SortOptions{FetchableOnly: true})
	if sorted[0].Resolution() != Res1080p || sorted[1].Resolution() != Res720p || sorted[2].Resolution() != Res2160p {
		t.Fatalf("custom resolution order not honored: %v %v %v",
			sorted[0].Resolution(), sorted[1].Resolution(), sorted[2].Resolution())
	}
	// free Sort keeps the default highest-first order
	def := Sort(torrents, SortOptions{FetchableOnly: true})
	if def[0].Resolution() != Res2160p {
		t.Fatalf("default order should lead with 2160p: %v", def[0].Resolution())
	}

	plain := r.Rank("Movie.2020.1080p.WEB-DL-GRP")
	imax := r.Rank("Movie.2020.IMAX.1080p.WEB-DL-GRP")
	if imax.Rank-plain.Rank < 500 {
		t.Fatalf("pattern rank not applied: %d vs %d", imax.Rank, plain.Rank)
	}
	found := false
	for _, c := range r.Explain(&imax) {
		if c.Source == `pattern:(?i)\bIMAX\b` && c.Rank == 500 {
			found = true
		}
	}
	if !found {
		t.Fatalf("pattern contribution missing from Explain: %+v", r.Explain(&imax))
	}
}

func TestSortCriteriaChain(t *testing.T) {
	r := mustRanker(t, Default())
	torrents := r.RankAll([]string{
		"Movie.2020.720p.BluRay.REMUX-A",
		"Movie.2020.2160p.WEBRip-B",
	})
	// rank-first chain ignores resolution buckets
	byRank := Sort(torrents, SortOptions{Criteria: []SortCriterion{{Key: SortByRank}}})
	if byRank[0].Raw != "Movie.2020.720p.BluRay.REMUX-A" {
		t.Fatalf("rank-first chain should lead with remux: %q", byRank[0].Raw)
	}
	// ascending rank reverses
	asc := Sort(torrents, SortOptions{Criteria: []SortCriterion{{Key: SortByRank, Ascending: true}}})
	if asc[0].Raw != "Movie.2020.2160p.WEBRip-B" {
		t.Fatalf("ascending rank chain wrong: %q", asc[0].Raw)
	}
}
