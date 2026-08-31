package rank

import (
	"encoding/json"
	"hash/fnv"
	"os"
	"strings"
	"testing"

	"github.com/dreulavelle/jhin/rules"
)

// A realistic profile, run over every title in the parser's golden corpus.
// The point is not any one answer but the invariants: the same input always
// scores the same, a breakdown always adds up, and an unanswerable rule never
// removes a release.

const soakRules = `
UHD T1: define if group in ["FraMeSToR", "W4NK3R", "HiFi", "Positive", "TayTo"]
Trusted 4K: score 3000 if resolution == "2160p" and matched("UHD T1")
IMAX: score 2000 if releaseName matches "(?i)\bIMAX\b"
Seeders: score min(grabs, 200) * 15 if grabs > 0
Freshness: score max(0, 500 - ageDays * 5) if true
Sweet spot: score 2000 - abs(sizeGB - 4) * 300 if sizeGB > 0
Atmos: score 800 if "atmos" in traits
Ten-bit: score 1200 if "10bit" in traits
Anime bonus [anime_show]: score 400 if "hevc" in traits
Oversized: reject if sizeGB > 60 and resolution != "2160p"
DV without fallback: reject if dolbyVision and not hdrFallback
Real 10-bit: score 400 if probed.bitDepth >= 10
Bad upscale: reject if upscaled and exists(resolution == "2160p" and "remux" in traits)
Best 5 per flavour: keep 5 per resolution + " " + quality if true
Label: tag "t-" + resolution if true
`

// soakFacts derives stable pseudo-random application data from the title, so
// the corpus exercises present and absent tiers without the run depending on
// a clock or a seed.
type soakFacts struct {
	n uint32
}

func factsFor2(title string) soakFacts {
	h := fnv.New32a()
	_, _ = h.Write([]byte(title))
	return soakFacts{n: h.Sum32()}
}

func (s soakFacts) Lookup(path string) (rules.Value, bool) {
	switch path {
	case "sizeGB":
		return rules.NumOf(float64(s.n%80) + 0.5), true
	case "ageDays":
		return rules.NumOf(float64(s.n % 400)), true
	case "grabs":
		return rules.NumOf(float64(s.n % 500)), true
	case "probed.bitDepth":
		return rules.NumOf(float64(8 + s.n%3)), true
	case "probed.height":
		return rules.NumOf(float64(360 + s.n%2000)), true
	}
	return rules.Value{}, false
}

func (s soakFacts) TierPresent(tier string) bool {
	switch tier {
	case "reported":
		return s.n%5 != 0 // most releases come from an indexer
	case "measured":
		return s.n%7 == 0 // few have been probed
	}
	return tier == ""
}

func soakTitles(t *testing.T) []string {
	t.Helper()
	blob, err := os.ReadFile("../parser/testdata/golden.json")
	if err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}
	var cases []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(blob, &cases); err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(cases))
	for _, c := range cases {
		out = append(out, c.Title)
	}
	return out
}

func soakRanker(t *testing.T) *Ranker {
	t.Helper()
	reg := appRegistry()
	p := Default()
	p.Attributes = map[Attr]Policy{AttrUpscaled: {Fetch: true, Rank: -5000}}
	var err error
	if p.Rules, err = rules.ParseText(soakRules); err != nil {
		t.Fatal(err)
	}
	reg.Effect("tag", rules.Str)
	eng, err := rules.Compile(reg, p.Rules)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(p, WithRules(eng))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func soakEntries(titles []string) []Entry {
	entries := make([]Entry, len(titles))
	for i, title := range titles {
		entries[i] = Entry{Title: title, Facts: factsFor2(title)}
	}
	return entries
}

func TestSoakInvariants(t *testing.T) {
	titles := soakTitles(t)
	r := soakRanker(t)
	got := r.RankEntries(soakEntries(titles), RankOptions{Kind: "movie"})

	if len(got) != len(titles) {
		t.Fatalf("%d results for %d titles", len(got), len(titles))
	}

	skippedSeen, matchedSeen, rejectedSeen, effectsSeen := 0, 0, 0, 0
	for i := range got {
		tr := &got[i]

		// index alignment is the contract RankEntries has always kept
		if tr.Raw != titles[i] {
			t.Fatalf("result %d is %q, want %q", i, tr.Raw, titles[i])
		}

		// every point traces to a clause
		total := 0
		for _, c := range r.Explain(tr) {
			total += c.Rank
		}
		if total != tr.Rank {
			t.Fatalf("%q: contributions sum to %d but rank is %d", tr.Raw, total, tr.Rank)
		}

		// fetch and rejections agree
		if tr.Fetch != (len(tr.Rejections) == 0) {
			t.Fatalf("%q: fetch=%v with rejections %v", tr.Raw, tr.Fetch, tr.Rejections)
		}

		// a rule cannot both be skipped and have paid out
		skipped := map[string]bool{}
		for _, s := range tr.RuleSkipped {
			skipped[s.Name] = true
			if s.Reason == "" {
				t.Fatalf("%q: rule %q was skipped with no reason", tr.Raw, s.Name)
			}
		}
		for _, m := range tr.RuleMatches {
			if skipped[m.Name] {
				t.Fatalf("%q: rule %q both skipped and matched", tr.Raw, m.Name)
			}
		}
		// nor can a skipped rule have rejected
		for _, rej := range tr.Rejections {
			name := strings.TrimPrefix(rej, rules.RejectionPrefix)
			if skipped[name] {
				t.Fatalf("%q: rule %q was skipped but rejected anyway", tr.Raw, name)
			}
		}

		if len(tr.RuleSkipped) > 0 {
			skippedSeen++
		}
		if len(tr.RuleMatches) > 0 {
			matchedSeen++
		}
		if !tr.Fetch {
			rejectedSeen++
		}
		if len(tr.Effects) > 0 {
			effectsSeen++
		}
	}

	// the corpus has to actually exercise each path, or the invariants above
	// are being checked against nothing
	for name, n := range map[string]int{
		"skipped": skippedSeen, "matched": matchedSeen,
		"rejected": rejectedSeen, "effects": effectsSeen,
	} {
		if n == 0 {
			t.Errorf("no release exercised the %s path", name)
		}
	}
	t.Logf("%d titles: %d matched, %d skipped a rule, %d rejected, %d tagged",
		len(got), matchedSeen, skippedSeen, rejectedSeen, effectsSeen)
}

// Same input, same score — every time, whatever order the workers happen to
// run in.
func TestSoakDeterminism(t *testing.T) {
	titles := soakTitles(t)
	r := soakRanker(t)
	entries := soakEntries(titles)

	first := r.RankEntries(entries, RankOptions{Kind: "movie"})
	for run := range 4 {
		again := r.RankEntries(entries, RankOptions{Kind: "movie"})
		for i := range first {
			if first[i].Rank != again[i].Rank || first[i].Fetch != again[i].Fetch {
				t.Fatalf("run %d, %q: %d/%v then %d/%v",
					run, titles[i], first[i].Rank, first[i].Fetch, again[i].Rank, again[i].Fetch)
			}
			if len(first[i].RuleMatches) != len(again[i].RuleMatches) {
				t.Fatalf("run %d, %q: rule matches differ", run, titles[i])
			}
		}
	}
}

// Caps hold over the whole corpus: no bucket keeps more than it was allowed,
// and nothing that survives was rejected for a cap.
func TestSoakLimits(t *testing.T) {
	titles := soakTitles(t)
	r := soakRanker(t)
	sorted := r.Sort(r.RankEntries(soakEntries(titles), RankOptions{Kind: "movie"}), SortOptions{})
	ApplyLimits(sorted)

	kept := map[string]int{}
	for i := range sorted {
		if !sorted[i].Fetch {
			continue
		}
		for _, lm := range sorted[i].Limits {
			kept[lm.Name+"/"+lm.Group]++
			if kept[lm.Name+"/"+lm.Group] > lm.Count {
				t.Fatalf("bucket %q kept %d, cap is %d", lm.Name+"/"+lm.Group, kept[lm.Name+"/"+lm.Group], lm.Count)
			}
		}
	}
	if len(kept) == 0 {
		t.Error("no cap was exercised")
	}
}

// The whole corpus, with rules, still parses and ranks without the baseline
// changing for a profile whose rules do not fire.
func TestSoakBaselineUnchanged(t *testing.T) {
	titles := soakTitles(t)
	plain, err := New(Default())
	if err != nil {
		t.Fatal(err)
	}
	p := Default()
	p.Rules = []rules.Rule{{Name: "never", When: `year == 99999`, Score: "1000"}}
	eng, err := p.CompileRules(appRegistry())
	if err != nil {
		t.Fatal(err)
	}
	withRules, err := New(p, WithRules(eng))
	if err != nil {
		t.Fatal(err)
	}

	base := plain.RankAll(titles)
	ruled := withRules.RankAll(titles)
	for i := range base {
		if base[i].Rank != ruled[i].Rank || base[i].Fetch != ruled[i].Fetch {
			t.Fatalf("%q: rules that never fire changed %d/%v to %d/%v",
				titles[i], base[i].Rank, base[i].Fetch, ruled[i].Rank, ruled[i].Fetch)
		}
	}
}

func BenchmarkSoakRankEntries(b *testing.B) {
	blob, err := os.ReadFile("../parser/testdata/golden.json")
	if err != nil {
		b.Skip(err)
	}
	var cases []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(blob, &cases); err != nil {
		b.Fatal(err)
	}
	titles := make([]string, len(cases))
	for i, c := range cases {
		titles[i] = c.Title
	}

	reg := appRegistry()
	reg.Effect("tag", rules.Str)
	p := Default()
	p.Rules, _ = rules.ParseText(soakRules)
	eng, err := rules.Compile(reg, p.Rules)
	if err != nil {
		b.Fatal(err)
	}
	r, _ := New(p, WithRules(eng))
	entries := soakEntries(titles)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = r.RankEntries(entries, RankOptions{Kind: "movie"})
	}
}

// The same corpus without rules, so the cost of the rule layer is the
// difference rather than a number on its own.
func BenchmarkSoakBaseline(b *testing.B) {
	blob, err := os.ReadFile("../parser/testdata/golden.json")
	if err != nil {
		b.Skip(err)
	}
	var cases []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(blob, &cases); err != nil {
		b.Fatal(err)
	}
	entries := make([]Entry, len(cases))
	for i, c := range cases {
		entries[i] = Entry{Title: c.Title}
	}
	r, _ := New(Default())
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = r.RankEntries(entries)
	}
}
