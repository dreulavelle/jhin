package rank

import (
	"strings"
	"testing"

	"github.com/dreulavelle/jhin/rules"
)

// appRegistry is what a consuming application builds: jhin's own attributes
// plus its indexer and probe data.
func appRegistry() *rules.Registry {
	reg := rules.Core()
	reg.Tier("reported", "size, age or grabs, which a release name does not carry")
	reg.Tier("measured", "a probed file")
	reg.Field("sizeGB", rules.Num, "reported")
	reg.Field("ageDays", rules.Num, "reported")
	reg.Field("grabs", rules.Num, "reported")
	reg.Namespace("probed", "measured").Num("height").Num("bitDepth")
	return reg
}

// appFacts is the application's side: whatever it knows that a release name
// does not carry.
type appFacts struct {
	sizeGB, ageDays, grabs float64
	probedHeight           float64
	probed                 bool
	indexed                bool
}

func (a appFacts) Lookup(path string) (rules.Value, bool) {
	switch path {
	case "sizeGB":
		return rules.NumOf(a.sizeGB), true
	case "ageDays":
		return rules.NumOf(a.ageDays), true
	case "grabs":
		return rules.NumOf(a.grabs), true
	case "probed.height":
		return rules.NumOf(a.probedHeight), true
	}
	return rules.Value{}, false
}

func (a appFacts) TierPresent(tier string) bool {
	switch tier {
	case "reported":
		return a.indexed
	case "measured":
		return a.probed
	}
	return tier == ""
}

func rankerWith(t *testing.T, src string, tweak ...func(*Profile)) *Ranker {
	t.Helper()
	rs, err := rules.ParseText(src)
	if err != nil {
		t.Fatal(err)
	}
	p := Default()
	p.Rules = rs
	for _, fn := range tweak {
		fn(&p)
	}
	eng, err := p.CompileRules(appRegistry())
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(p, WithRules(eng))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Every worked example from the proposal, end to end through the ranker.
func TestRulesEndToEnd(t *testing.T) {
	r := rankerWith(t, `
Seeders: score min(grabs, 200) * 15 if grabs > 0
Freshness: score max(0, 500 - ageDays * 5) if true
Sweet spot: score 2000 - abs(sizeGB - 4) * 300 if sizeGB > 0
Oversized: reject if sizeGB > 12 and resolution != "2160p"
Real 10-bit: score 400 if probed.bitDepth >= 10
`)
	entries := []Entry{
		{
			Title: "Show.S01E01.1080p.WEB-DL.x264-GRP",
			Facts: appFacts{sizeGB: 4, ageDays: 20, grabs: 300, indexed: true},
		},
		{
			Title: "Show.S01E02.1080p.WEB-DL.x264-GRP",
			Facts: appFacts{sizeGB: 30, ageDays: 1, grabs: 5, indexed: true},
		},
	}
	got := r.RankEntries(entries)

	// the first is fresh, well-seeded and the right size
	if !got[0].Fetch {
		t.Errorf("first release was rejected: %v", got[0].Rejections)
	}
	// seeders 3000 + freshness 400 + sweet spot 2000
	var ruleTotal int
	for _, m := range got[0].RuleMatches {
		ruleTotal += m.Score
	}
	if ruleTotal != 3000+400+2000 {
		t.Errorf("rule points = %d, want 5400 (%+v)", ruleTotal, got[0].RuleMatches)
	}

	// the second is 30 GB of 1080p
	if got[1].Fetch {
		t.Error("the oversized release was not rejected")
	}
	if !strings.Contains(strings.Join(got[1].Rejections, " "), "Oversized") {
		t.Errorf("rejections = %v, want the rule named", got[1].Rejections)
	}

	// neither was probed, so the probe rule never ran
	for i := range got {
		if len(got[i].RuleSkipped) != 1 || got[i].RuleSkipped[0].Name != "Real 10-bit" {
			t.Errorf("release %d skipped = %+v, want the probe rule", i, got[i].RuleSkipped)
		}
	}
}

// The baseline still decides everything it decided before; rules add to it.
func TestRulesAddToBaseline(t *testing.T) {
	title := "Movie.2020.2160p.BluRay.REMUX.HEVC-GRP"
	plain, err := New(Default())
	if err != nil {
		t.Fatal(err)
	}
	base := plain.Rank(title)

	r := rankerWith(t, `Bonus: score 250 if "remux" in traits`)
	withRules := r.Rank(title)

	if withRules.Rank != base.Rank+250 {
		t.Errorf("rank = %d, want the baseline %d plus 250", withRules.Rank, base.Rank)
	}
	if withRules.Fetch != base.Fetch {
		t.Errorf("fetch changed from %v to %v", base.Fetch, withRules.Fetch)
	}
}

// A profile carrying rules that were never compiled scores exactly as it did
// before they were added.
func TestRulesInert(t *testing.T) {
	p := Default()
	p.Rules = []rules.Rule{{Name: "Bonus", When: "true", Score: "9999"}}
	r, err := New(p) // no WithRules
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := New(Default())
	title := "Movie.2020.1080p.WEB-DL-GRP"
	if r.Rank(title).Rank != plain.Rank(title).Rank {
		t.Error("uncompiled rules changed the score")
	}
}

// Reject X only when a better Y actually turned up.
func TestRulesResultSet(t *testing.T) {
	// the baseline vetoes upscales outright; this profile leaves the decision
	// to the rule, which is the whole point of asking about the set
	r := rankerWith(t,
		`Bad upscale: reject if upscaled and exists(resolution == "2160p" and "remux" in traits)`,
		func(p *Profile) {
			p.Attributes = map[Attr]Policy{AttrUpscaled: {Fetch: true, Rank: 0}}
		})

	upscale := "Movie.2020.2160p.AI.Upscaled.WEB-DL-GRP"
	remux := "Movie.2020.2160p.BluRay.REMUX-GRP"

	// alone, the upscale is all there is
	if got := r.RankAll([]string{upscale}); !got[0].Fetch {
		t.Errorf("upscale rejected with nothing better available: %v", got[0].Rejections)
	}
	// with a remux in the same batch, it goes
	got := r.RankAll([]string{upscale, remux})
	if got[0].Fetch {
		t.Error("upscale survived a batch containing a remux")
	}
	if !got[1].Fetch {
		t.Errorf("the remux was rejected: %v", got[1].Rejections)
	}
}

// Caps are decided after sorting, so they never cost you the release you
// wanted.
func TestRulesLimits(t *testing.T) {
	r := rankerWith(t, `Best 2 per resolution: keep 2 per resolution if true`)
	titles := []string{
		"A.2020.2160p.BluRay.REMUX-GRP",
		"B.2020.2160p.WEB-DL-GRP",
		"C.2020.2160p.WEBRip-GRP",
		"D.2020.1080p.BluRay-GRP",
	}
	sorted := r.Sort(r.RankAll(titles))
	ApplyLimits(sorted)

	kept := map[Resolution]int{}
	for i := range sorted {
		if sorted[i].Fetch {
			kept[sorted[i].Resolution()]++
		}
	}
	if kept[Res2160p] != 2 {
		t.Errorf("kept %d 2160p releases, want 2", kept[Res2160p])
	}
	if kept[Res1080p] != 1 {
		t.Errorf("kept %d 1080p releases, want 1", kept[Res1080p])
	}
	// the one dropped is the worst 2160p, not whichever arrived last
	for i := range sorted {
		if !sorted[i].Fetch && !strings.HasPrefix(sorted[i].Raw, "C.") {
			t.Errorf("dropped %q, want the lowest-ranked 2160p", sorted[i].Raw)
		}
	}
}

// Every point still traces to a clause, rules included.
func TestExplainIncludesRules(t *testing.T) {
	r := rankerWith(t, `Seeders: score min(grabs, 200) * 15 if grabs > 0`)
	got := r.RankEntries([]Entry{{
		Title: "Movie.2020.1080p.WEB-DL-GRP",
		Facts: appFacts{grabs: 10, indexed: true},
	}})

	total := 0
	found := false
	for _, c := range r.Explain(&got[0]) {
		total += c.Rank
		if c.Source == "rule:Seeders" {
			found = true
			if c.Rank != 150 {
				t.Errorf("rule contribution = %d, want 150", c.Rank)
			}
			if c.Detail != "min(grabs, 200) * 15" {
				t.Errorf("detail = %q, want the score expression", c.Detail)
			}
		}
	}
	if !found {
		t.Error("Explain did not mention the rule")
	}
	if total != got[0].Rank {
		t.Errorf("contributions sum to %d but rank is %d", total, got[0].Rank)
	}
}

// One rule reading a probe must not empty the list of everything unprobed.
func TestRulesFailOpenThroughRanker(t *testing.T) {
	r := rankerWith(t, `Small: reject if probed.height < 1080`)
	got := r.RankEntries([]Entry{
		{Title: "A.2020.1080p.WEB-DL-GRP", Facts: appFacts{indexed: true}},
		{Title: "B.2020.720p.WEB-DL-GRP", Facts: appFacts{probed: true, probedHeight: 720, indexed: true}},
	})
	if !got[0].Fetch {
		t.Errorf("unprobed release was rejected: %v", got[0].Rejections)
	}
	if got[1].Fetch {
		t.Error("a genuinely 720p probed release was not rejected")
	}
}

// Rules must be safe to evaluate from the batch workers.
func TestRulesConcurrent(t *testing.T) {
	r := rankerWith(t, `
Seeders: score min(grabs, 200) * 15 if grabs > 0
Tier: define if resolution == "2160p"
Uses: score 10 if matched("Tier")
`)
	entries := make([]Entry, 400)
	for i := range entries {
		entries[i] = Entry{
			Title: "Movie.2020.2160p.WEB-DL-GRP",
			Facts: appFacts{grabs: 10, indexed: true},
		}
	}
	got := r.RankEntries(entries)
	want := got[0].Rank
	for i := range got {
		if got[i].Rank != want {
			t.Fatalf("release %d scored %d, want %d — evaluation is not deterministic", i, got[i].Rank, want)
		}
	}
}

func TestProfileRulesRoundTripJSON(t *testing.T) {
	p := Default()
	p.Rules, _ = rules.ParseText("Seeders: score min(grabs, 200) * 15 if grabs > 0\n")
	dir := t.TempDir()
	path := dir + "/profile.json"
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rules) != 1 || back.Rules[0].Score != "min(grabs, 200) * 15" {
		t.Fatalf("rules did not survive the round trip: %+v", back.Rules)
	}
	if _, err := back.CompileRules(appRegistry()); err != nil {
		t.Fatalf("reloaded rules do not compile: %v", err)
	}
}

// A release another rule already rejected is gone, so it must not hold a slot
// a cap could have given to something the user can actually fetch.
func TestLimitsIgnoreAlreadyRejected(t *testing.T) {
	r := rankerWith(t, `
Drop webrips: reject if "webrip" in traits
Best 2 per resolution: keep 2 per resolution if true
`)
	titles := []string{
		"A.2020.2160p.BluRay.REMUX-GRP", // best
		"B.2020.2160p.WEBRip-GRP",       // rejected by the rule
		"C.2020.2160p.WEB-DL-GRP",       // should still make the cut
	}
	sorted := r.Sort(r.RankAll(titles), SortOptions{})
	ApplyLimits(sorted)

	byName := map[string]Torrent{}
	for _, t := range sorted {
		byName[strings.SplitN(t.Raw, ".", 2)[0]] = t
	}
	if !byName["A"].Fetch {
		t.Errorf("A was dropped: %v", byName["A"].Rejections)
	}
	if byName["B"].Fetch {
		t.Error("the webrip survived its own rejection")
	}
	if !byName["C"].Fetch {
		t.Errorf("C lost its slot to an already-rejected release: %v", byName["C"].Rejections)
	}
}

// A profile from a newer jhin is refused with an explanation rather than
// half-understood: a condition using syntax this build never heard of would
// otherwise fail as an unknown attribute.
func TestSyntaxVersion(t *testing.T) {
	p := Default()
	p.Rules = []rules.Rule{{Name: "r", When: "true", Score: "1"}}

	p.SyntaxVersion = rules.SyntaxVersion
	if _, err := p.CompileRules(nil); err != nil {
		t.Errorf("the current syntax version was refused: %v", err)
	}
	p.SyntaxVersion = 0 // written before the field existed
	if _, err := p.CompileRules(nil); err != nil {
		t.Errorf("a profile with no version was refused: %v", err)
	}
	p.SyntaxVersion = rules.SyntaxVersion + 1
	if _, err := p.CompileRules(nil); err == nil {
		t.Error("a profile from a newer build compiled")
	} else if !strings.Contains(err.Error(), "upgrade jhin") {
		t.Errorf("error %q does not say what to do about it", err)
	}
}

// The trait vocabulary is closed, so a rule naming a trait that does not
// exist fails when the profile is compiled rather than never firing.
func TestTraitVocabularyIsChecked(t *testing.T) {
	p := Default()
	p.Rules = []rules.Rule{{Name: "r", When: `"dual_audio" in traits`, Score: "1"}}
	if _, err := p.CompileRules(nil); err == nil {
		t.Fatal("a trait that does not exist compiled")
	} else if !strings.Contains(err.Error(), "never holds") {
		t.Errorf("error %q does not say the trait cannot occur", err)
	}

	// dual audio is carried as dubbed, which does exist
	p.Rules = []rules.Rule{{Name: "r", When: `"dubbed" in traits`, Score: "1"}}
	eng, err := p.CompileRules(nil)
	if err != nil {
		t.Fatalf("a real trait was refused: %v", err)
	}
	r, err := New(p, WithRules(eng))
	if err != nil {
		t.Fatal(err)
	}
	got := r.Rank("Anime.S01E01.1080p.BluRay.Dual.Audio.x265-GRP")
	if len(got.RuleMatches) != 1 {
		t.Errorf("dual audio did not read as dubbed: %+v", got.RuleMatches)
	}
}

// A release the baseline filters rejected is not "something better": a rule
// asking whether the set holds a remux must not count one the profile itself
// threw out.
func TestAggregatesIgnoreVetoedReleases(t *testing.T) {
	p := Default()
	p.Attributes = map[Attr]Policy{AttrUpscaled: {Fetch: true, Rank: 0}}
	// the only remux in the batch is excluded by the profile
	p.Exclude = []string{`\bREMUX\b`}
	p.Rules = []rules.Rule{{
		Name:   "Bad upscale",
		Action: rules.ActionReject,
		When:   `upscaled and exists(resolution == "2160p" and "remux" in traits)`,
	}}
	eng, err := p.CompileRules(nil)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(p, WithRules(eng))
	if err != nil {
		t.Fatal(err)
	}

	got := r.RankAll([]string{
		"Movie.2020.2160p.AI.Upscaled.WEB-DL-GRP",
		"Movie.2020.2160p.BluRay.REMUX-GRP",
	})
	if got[1].Fetch {
		t.Fatal("the remux should have been excluded by the baseline")
	}
	if !got[0].Fetch {
		t.Errorf("upscale rejected off a remux the profile itself excluded: %v", got[0].Rejections)
	}
}
