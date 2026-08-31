package rules

import (
	"strings"
	"testing"
)

func release(res, quality string, traits []string, extra ...map[string]Value) MapFacts {
	v := map[string]Value{
		"resolution": StrOf(res),
		"quality":    StrOf(quality),
		"traits":     StrListOf(traits),
		"upscaled":   BoolOf(false),
	}
	for _, m := range extra {
		for k, val := range m {
			v[k] = val
		}
	}
	return MapFacts{Values: v, Tiers: map[string]bool{"measured": true, "reported": true}}
}

// The reason to want set-wide questions at all: reject X only when a better Y
// actually turned up, rather than rejecting X and hoping.
func TestAggregates(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "Bad upscale", Action: ActionReject,
			When: `upscaled and exists(resolution == "2160p" and "remux" in traits)`},
		{Name: "Scarce 4K", When: `count(resolution == "2160p") < 3`, Score: "500"},
		{Name: "No web", When: `none(quality == "WEB-DL")`, Score: "200"},
	})
	if err != nil {
		t.Fatal(err)
	}

	upscale := release("2160p", "WEB-DL", nil, map[string]Value{"upscaled": BoolOf(true)})
	remux := release("2160p", "BluRay", []string{"remux"})

	// with a remux on offer, the upscale goes
	set := []Facts{upscale, remux}
	st := eng.ComputeAggregates(set, "")
	if out := eng.Evaluate(upscale, "", st); len(out.Rejections) != 1 {
		t.Errorf("upscale survived a set containing a remux: %+v", out)
	}
	// the remux itself is not an upscale, so it stays
	if out := eng.Evaluate(remux, "", st); len(out.Rejections) != 0 {
		t.Errorf("remux was rejected: %+v", out)
	}

	// alone, the upscale is all there is — nothing better exists, so it stays
	st = eng.ComputeAggregates([]Facts{upscale}, "")
	if out := eng.Evaluate(upscale, "", st); len(out.Rejections) != 0 {
		t.Errorf("upscale rejected with no replacement available: %+v", out)
	}

	// the set includes the release being judged
	st = eng.ComputeAggregates([]Facts{remux}, "")
	if out := eng.Evaluate(remux, "", st); out.Points != 500+200 {
		t.Errorf("points = %d, want scarce-4K and no-web to both fire", out.Points)
	}
}

// Counts are taken before any rule fires, so a rule that rejects cannot
// change what another rule counted.
func TestAggregateOrderIndependence(t *testing.T) {
	forward := []Rule{
		{Name: "drop web", When: `quality == "WEB-DL"`, Action: ActionReject},
		{Name: "count web", When: `count(quality == "WEB-DL") > 0`, Score: "7"},
	}
	backward := []Rule{forward[1], forward[0]}

	set := []Facts{release("1080p", "WEB-DL", nil), release("1080p", "BluRay", nil)}
	for _, rs := range [][]Rule{forward, backward} {
		eng, err := Compile(testRegistry(), rs)
		if err != nil {
			t.Fatal(err)
		}
		st := eng.ComputeAggregates(set, "")
		if got := eng.Evaluate(set[1], "", st).Points; got != 7 {
			t.Errorf("points = %d, want 7 regardless of rule order", got)
		}
	}
}

// Two rules asking the same thing share one aggregate, so the set is walked
// once for both.
func TestAggregateDedup(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "a", When: `count(resolution == "2160p") > 0`, Score: "1"},
		{Name: "b", When: `count( resolution=="2160p" ) > 1`, Score: "1"},
		{Name: "c", When: `exists(resolution == "2160p")`, Score: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(eng.aggs); n != 1 {
		t.Errorf("%d aggregates, want 1: %v", n, eng.AggregateSources())
	}
}

// On a fresh search where nothing has been probed, none(probed...) must not
// read as "there is no good 4K" and reject everything.
func TestAggregateFailOpen(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "no good 4K", When: `none(probed.height >= 2000)`, Action: ActionReject},
	})
	if err != nil {
		t.Fatal(err)
	}
	unprobed := release("1080p", "WEB-DL", nil)
	unprobed.Tiers = map[string]bool{"reported": true}

	st := eng.ComputeAggregates([]Facts{unprobed}, "")
	out := eng.Evaluate(unprobed, "", st)
	if len(out.Rejections) != 0 {
		t.Errorf("rejected on an unanswerable question: %v", out.Rejections)
	}
	if len(out.Skipped) != 1 || !strings.Contains(out.Skipped[0].Reason, "result set") {
		t.Errorf("skipped = %+v, want it to say the set could not answer", out.Skipped)
	}

	// once something in the set has been probed, the question has an answer
	probed := release("1080p", "WEB-DL", nil, map[string]Value{"probed.height": NumOf(1080)})
	st = eng.ComputeAggregates([]Facts{unprobed, probed}, "")
	if out := eng.Evaluate(probed, "", st); len(out.Rejections) != 1 {
		t.Errorf("want a rejection once the set can answer, got %+v", out)
	}
}

// count(list) still judges one release; count(condition) judges the set. The
// two are told apart by the argument's type.
func TestCountAmbiguity(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "over a list", When: `count(traits, # == "remux") == 1`, Score: "1"},
		{Name: "over the set", When: `count(resolution == "2160p") == 1`, Score: "10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(eng.aggs) != 1 {
		t.Fatalf("%d aggregates, want only the set question lifted", len(eng.aggs))
	}
	f := release("2160p", "BluRay", []string{"remux"})
	st := eng.ComputeAggregates([]Facts{f}, "")
	if got := eng.Evaluate(f, "", st).Points; got != 11 {
		t.Errorf("points = %d, want both forms to work", got)
	}
}

func TestAggregatesCannotNest(t *testing.T) {
	_, err := Compile(testRegistry(), []Rule{
		{Name: "r", When: `exists(count(resolution == "2160p") > 1)`},
	})
	if err == nil {
		t.Fatal("nested result-set questions compiled")
	}
}

// A tier list is written once and referred to from everywhere else that cares.
func TestMatched(t *testing.T) {
	set := []Rule{
		{Name: "UHD T1", Action: ActionDefine, When: `group in ["FraMeSToR", "W4NK3R"]`},
		{Name: "UHD T2", Action: ActionDefine, When: `group in ["HiFi"]`},
		{Name: "Trusted 4K", When: `resolution == "2160p" and matched("UHD T1")`, Score: "3000"},
		{Name: "Untrusted 4K", Action: ActionReject,
			When: `resolution == "2160p" and not (matched("UHD T1") or matched("UHD T2"))`},
	}
	eng, err := Compile(testRegistry(), set)
	if err != nil {
		t.Fatal(err)
	}
	// a define rule never joins the set: it pays nothing and appears nowhere
	if eng.Len() != 2 {
		t.Errorf("%d acting rules, want 2 — defines should not act", eng.Len())
	}

	trusted := release("2160p", "BluRay", nil, map[string]Value{"group": StrOf("FraMeSToR")})
	if out := eng.Evaluate(trusted, "", nil); out.Points != 3000 || len(out.Rejections) != 0 {
		t.Errorf("trusted group: %+v", out)
	}
	unknown := release("2160p", "BluRay", nil, map[string]Value{"group": StrOf("NOBODY")})
	if out := eng.Evaluate(unknown, "", nil); len(out.Rejections) != 1 {
		t.Errorf("unknown group was not rejected: %+v", out)
	}
	if out := eng.Evaluate(unknown, "", nil); len(out.Matched) != 0 {
		t.Errorf("define leaked into the breakdown: %+v", out.Matched)
	}
}

func TestMatchedCarriesScopeAndTier(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "movie tier", Action: ActionDefine, When: `resolution == "2160p"`, Scope: []string{"movie"}},
		{Name: "uses it", When: `matched("movie tier")`, Score: "5"},
		{Name: "probe def", Action: ActionDefine, When: `probed.height > 100`},
		{Name: "uses probe", When: `matched("probe def")`, Score: "9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := release("2160p", "BluRay", nil, map[string]Value{"probed.height": NumOf(2160)})

	// the referenced rule's scope travels with its condition
	if got := eng.Evaluate(f, "movie", nil).Points; got != 5+9 {
		t.Errorf("movie scored %d, want both rules", got)
	}
	if got := eng.Evaluate(f, "series", nil).Points; got != 9 {
		t.Errorf("series scored %d, want only the unscoped rule", got)
	}

	// and so does its tier: referring to a probe rule makes you probe-dependent
	unprobed := release("2160p", "BluRay", nil)
	unprobed.Tiers = map[string]bool{"reported": true}
	out := eng.Evaluate(unprobed, "movie", nil)
	if out.Points != 5 {
		t.Errorf("points = %d, want the probe-dependent rule skipped", out.Points)
	}
	if len(out.Skipped) != 1 || out.Skipped[0].Name != "uses probe" {
		t.Errorf("skipped = %+v", out.Skipped)
	}
}

func TestMatchedErrors(t *testing.T) {
	for _, tc := range []struct {
		rules    []Rule
		contains string
	}{
		{[]Rule{{Name: "a", When: `matched("nope")`}}, "names no rule"},
		{[]Rule{{Name: "a", When: `matched("a")`}}, "closes a loop"},
		{
			[]Rule{
				{Name: "a", When: `matched("b")`},
				{Name: "b", Action: ActionDefine, When: `matched("a")`},
			},
			"closes a loop",
		},
		{[]Rule{{Name: "a", When: `matched(resolution)`}}, "written out"},
		{[]Rule{{Name: "a", When: `matched("x", "y")`}}, "names one rule"},
		{
			[]Rule{
				{Name: "dup", Action: ActionDefine, When: "true"},
				{Name: "dup", Action: ActionDefine, When: "false"},
				{Name: "a", When: `matched("dup")`},
			},
			"already has this name",
		},
	} {
		if _, err := Compile(testRegistry(), tc.rules); err == nil {
			t.Errorf("%v: compiled, want %q", tc.rules[0].When, tc.contains)
		} else if !strings.Contains(err.Error(), tc.contains) {
			t.Errorf("error %q, want it to contain %q", err, tc.contains)
		}
	}

	// duplicates the set itself cannot refuse — two library rules sharing a
	// name — still make a reference to them ambiguous
	lib := []Rule{
		{Name: "dup", Action: ActionDefine, When: "true"},
		{Name: "dup", Action: ActionDefine, When: "false"},
	}
	_, err := Compile(testRegistry(), []Rule{{Name: "a", When: `matched("dup")`}}, lib...)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error %v, want the library duplicate to be ambiguous", err)
	}
}

// A rule that is switched off classifies nothing, so a reference to it is
// never true — and its condition is never looked at.
func TestMatchedDisabled(t *testing.T) {
	off := false
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "tier", Action: ActionDefine, When: "not valid at all", Enabled: &off},
		{Name: "uses it", When: `matched("tier")`, Score: "5"},
		{Name: "or not", When: `not matched("tier")`, Score: "3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.Evaluate(release("1080p", "WEB-DL", nil), "", nil).Points; got != 3 {
		t.Errorf("points = %d, want a reference to a disabled rule to be false", got)
	}
}

// A library holds shared definitions; a profile rule under the same name
// shadows the library's version.
func TestLibraryShadowing(t *testing.T) {
	lib := []Rule{{Name: "T1", Action: ActionDefine, When: `group == "LIBRARY"`}}
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "uses", When: `matched("T1")`, Score: "1"},
	}, lib...)
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.Evaluate(release("x", "y", nil, map[string]Value{"group": StrOf("LIBRARY")}), "", nil).Points; got != 1 {
		t.Errorf("library define not used, points = %d", got)
	}

	eng, err = Compile(testRegistry(), []Rule{
		{Name: "T1", Action: ActionDefine, When: `group == "MINE"`},
		{Name: "uses", When: `matched("T1")`, Score: "1"},
	}, lib...)
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.Evaluate(release("x", "y", nil, map[string]Value{"group": StrOf("MINE")}), "", nil).Points; got != 1 {
		t.Errorf("profile rule did not shadow the library, points = %d", got)
	}
}

// A reference is a copy, so a chain naming the rules below it twice doubles at
// every step. The cap turns a set that would hang compilation into an error.
func TestExpansionCap(t *testing.T) {
	rules := []Rule{{Name: "r0", Action: ActionDefine, When: "true"}}
	for i := 1; i <= 40; i++ {
		prev := rules[i-1].Name
		rules = append(rules, Rule{
			Name:   "r" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Action: ActionDefine,
			When:   `matched("` + prev + `") and matched("` + prev + `")`,
		})
	}
	rules = append(rules, Rule{Name: "top", When: `matched("` + rules[len(rules)-1].Name + `")`})
	if _, err := Compile(testRegistry(), rules); err == nil {
		t.Fatal("a doubling reference chain compiled")
	} else if !strings.Contains(err.Error(), "past 20000 nodes") {
		t.Errorf("error %q, want it to name the expansion cap", err)
	}
}

func TestLimits(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "Best 3 per flavour", Action: ActionLimit, Count: 3,
			When: "true", GroupBy: `resolution + " " + quality`},
	})
	if err != nil {
		t.Fatal(err)
	}
	kinds := []struct{ res, qual string }{
		{"2160p", "REMUX"}, {"2160p", "REMUX"}, {"2160p", "REMUX"}, {"2160p", "REMUX"},
		{"2160p", "WEB-DL"}, {"1080p", "BluRay"},
	}
	perItem := make([][]LimitMatch, len(kinds))
	order := make([]int, len(kinds))
	for i, k := range kinds {
		perItem[i] = eng.Evaluate(release(k.res, k.qual, nil), "", nil).Limits
		order[i] = i
	}
	got := ApplyLimits(perItem, order)
	// the fourth 2160p REMUX is the only one over its bucket's cap
	for i, r := range got {
		want := i == 3
		if (r != "") != want {
			t.Errorf("item %d (%s %s) rejected=%v, want %v (%q)", i, kinds[i].res, kinds[i].qual, r != "", want, r)
		}
	}
	if !strings.Contains(got[3], "2160p REMUX") {
		t.Errorf("rejection %q does not name the bucket", got[3])
	}
}

// Survivors are the best by final score, so a cap never costs the release you
// wanted — order, not arrival, decides.
func TestLimitsFollowOrder(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "cap", Action: ActionLimit, Count: 1, When: "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	perItem := make([][]LimitMatch, 3)
	for i := range perItem {
		perItem[i] = eng.Evaluate(release("1080p", "WEB-DL", nil), "", nil).Limits
	}
	got := ApplyLimits(perItem, []int{2, 0, 1}) // item 2 is best
	if got[2] != "" {
		t.Errorf("the best release was capped: %q", got[2])
	}
	if got[0] == "" || got[1] == "" {
		t.Errorf("the other two survived a cap of 1: %q %q", got[0], got[1])
	}
}

// Grouping by an attribute of a tier the release lacks makes the whole rule
// tier-dependent, rather than bucketing every unprobed release together.
func TestLimitGroupingCarriesTier(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "per height", Action: ActionLimit, Count: 1, When: "true", GroupBy: "probed.height"},
	})
	if err != nil {
		t.Fatal(err)
	}
	unprobed := release("1080p", "WEB-DL", nil)
	unprobed.Tiers = map[string]bool{"reported": true}
	out := eng.Evaluate(unprobed, "", nil)
	if len(out.Limits) != 0 {
		t.Errorf("an unprobed release counted against a probe-grouped cap: %+v", out.Limits)
	}
	if len(out.Skipped) != 1 {
		t.Errorf("skipped = %+v, want the cap skipped", out.Skipped)
	}
}

func TestLimitValidation(t *testing.T) {
	if _, err := Compile(testRegistry(), []Rule{
		{Name: "r", Action: ActionLimit, When: "true"},
	}); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Errorf("a cap of zero compiled: %v", err)
	}
	if _, err := Compile(testRegistry(), []Rule{
		{Name: "r", When: "true", Score: "1", GroupBy: "resolution"},
	}); err == nil || !strings.Contains(err.Error(), "only a limit rule") {
		t.Errorf("a score rule accepted a grouping: %v", err)
	}
}

// count over a list of yes/no values counts the true ones and judges this
// release; count over a condition judges the whole set. The argument's type
// is what tells them apart.
func TestCountOverBoolList(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "flags", When: `count([proper, repack, remastered]) >= 2`, Score: "1"},
		{Name: "set", When: `count(resolution == "2160p") >= 1`, Score: "10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(eng.aggs) != 1 {
		t.Fatalf("%d aggregates, want only the set question lifted", len(eng.aggs))
	}
	f := release("2160p", "BluRay", nil, map[string]Value{
		"proper": BoolOf(true), "repack": BoolOf(true), "remastered": BoolOf(false),
	})
	if got := eng.Evaluate(f, "", eng.ComputeAggregates([]Facts{f}, "")).Points; got != 11 {
		t.Errorf("points = %d, want both forms to work", got)
	}
	// one flag is not two
	f = release("2160p", "BluRay", nil, map[string]Value{
		"proper": BoolOf(true), "repack": BoolOf(false), "remastered": BoolOf(false),
	})
	if got := eng.Evaluate(f, "", eng.ComputeAggregates([]Facts{f}, "")).Points; got != 10 {
		t.Errorf("points = %d, want only the set question", got)
	}
	// a list of anything else still names the two-argument form
	_, err = Compile(testRegistry(), []Rule{{Name: "r", When: `count(traits) > 0`}})
	if err == nil || !strings.Contains(err.Error(), "two-argument form") {
		t.Errorf("error %v, want it to name the two-argument form", err)
	}
}

// A release one cap turns away consumes no slot in another, so the caps'
// declaration order cannot decide who survives.
func TestLimitsAreIndependent(t *testing.T) {
	fourK := Rule{Name: "at most 2 in 4K", Action: ActionLimit, Count: 2, When: `resolution == "2160p"`}
	remux := Rule{Name: "at most 1 remux", Action: ActionLimit, Count: 1, When: `"remux" in traits`}

	rel := func(res string, traits ...string) Facts {
		return facts(map[string]Value{"resolution": StrOf(res), "traits": StrListOf(traits)})
	}
	set := []Facts{
		rel("2160p", "remux"), // best
		rel("2160p", "remux"), // over the remux cap — must not hold a 4K slot
		rel("2160p"),          // second 4K survivor
	}

	for _, ruleSet := range [][]Rule{{fourK, remux}, {remux, fourK}} {
		eng, err := Compile(testRegistry(), ruleSet)
		if err != nil {
			t.Fatal(err)
		}
		perItem := make([][]LimitMatch, len(set))
		for i, f := range set {
			perItem[i] = eng.Evaluate(f, "", nil).Limits
		}
		got := ApplyLimits(perItem, []int{0, 1, 2})
		if got[0] != "" || got[2] != "" {
			t.Errorf("%s first: survivors wrong: %q", ruleSet[0].Name, got)
		}
		if got[1] == "" {
			t.Errorf("%s first: the second remux should be over its cap", ruleSet[0].Name)
		}
	}
}

// Two live rules under one name would merge everything they report.
func TestDuplicateNamesRefused(t *testing.T) {
	_, err := Compile(testRegistry(), []Rule{
		{Name: "dupe", When: "proper", Score: "1"},
		{Name: "dupe", When: "repack", Score: "2"},
	})
	if err == nil || !strings.Contains(err.Error(), "already has this name") {
		t.Errorf("duplicate names compiled: %v", err)
	}

	// a disabled duplicate blocks nothing — it classifies nothing
	off := false
	if _, err := Compile(testRegistry(), []Rule{
		{Name: "dupe", When: "proper", Score: "1"},
		{Name: "dupe", When: "broken ((", Enabled: &off},
	}); err != nil {
		t.Errorf("a disabled duplicate blocked the save: %v", err)
	}
}

// Group keys must tell two releases apart: ["a b"] and ["a", "b"] are
// different buckets.
func TestGroupKeysDoNotCollide(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "cap", Action: ActionLimit, Count: 1, GroupBy: "hdr", When: "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := eng.Evaluate(facts(map[string]Value{"hdr": StrListOf([]string{"a b"})}), "", nil)
	b := eng.Evaluate(facts(map[string]Value{"hdr": StrListOf([]string{"a", "b"})}), "", nil)
	if len(a.Limits) != 1 || len(b.Limits) != 1 {
		t.Fatalf("caps did not match: %+v %+v", a, b)
	}
	if a.Limits[0].Group == b.Limits[0].Group {
		t.Errorf("two different lists share the bucket %q", a.Limits[0].Group)
	}
}
