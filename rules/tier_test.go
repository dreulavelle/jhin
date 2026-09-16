package rules

import (
	"strings"
	"testing"
)

// Absence is judged on a rule's outcome, not on the names its condition
// mentions: a read of an absent tier is unknown, and/or settle it where the
// other side can, and only a rule the missing fact could have swung is skipped.
func TestAbsentTierSettlesWhereItCan(t *testing.T) {
	unprobed := facts(map[string]Value{
		"resolution": StrOf("1080p"), "sizeGB": NumOf(0), "traits": StrListOf([]string{"remux"}),
	}, "measured")

	tests := []struct {
		when string
		fire bool
		skip bool
	}{
		// the rows from the truth table
		{`resolution == "1080p" or probed.bitDepth == 10`, true, false},
		{`probed.bitDepth == 10 or resolution == "1080p"`, true, false},
		{`resolution == "720p" or probed.bitDepth == 10`, false, true},
		{`resolution == "1080p" and probed.bitDepth == 10`, false, true},
		{`resolution == "720p" and probed.bitDepth == 10`, false, false},
		{`probed.bitDepth == 10 and resolution == "720p"`, false, false},
		{`not probed.dolbyVision`, false, true},
		{`resolution == "1080p" and not probed.dolbyVision`, false, true},
		// unknown travels through everything else
		{`probed.height < 1080`, false, true},
		{`probed.height + 1 > 1`, false, true},
		{`probed.dolbyVision ? true : true`, false, true},
		{`probed.height in [1080, 2160]`, false, true},
		{`string(probed.height) == "0"`, false, true},
		{`any(traits, # == probed.dynamicRange)`, false, true},
		// and settles once the answerable half of a nested condition has
		{`not (resolution == "720p" and probed.dolbyVision)`, true, false},
		{`(resolution == "720p" and probed.dolbyVision) or resolution == "1080p"`, true, false},
		{`resolution == "1080p" or (probed.height > 2000 and probed.bitDepth == 10)`, true, false},
		// two optional halves: neither can answer, so it skips
		{`probed.height > 2000 or probed.bitDepth == 10`, false, true},
		// a guard on the absent tier is unknown before what it guards runs
		{`probed.height > 0 and 1 / probed.height > 0`, false, true},
		// a guard the release answers still short-circuits
		{`sizeGB > 0 and 1 / sizeGB > 0`, false, false},
		// ...and an unknown guard over a failing right side is unknown, not
		// a runtime failure
		{`probed.height > 0 and 1 / sizeGB > 0`, false, true},
		{`probed.height > 0 or 1 / sizeGB > 0`, false, true},
	}
	for _, tc := range tests {
		eng := compile1(t, tc.when)
		out := eng.Evaluate(unprobed, "", nil)
		if fired := len(out.Matched) == 1; fired != tc.fire {
			t.Errorf("%s: fired = %v, want %v (%+v)", tc.when, fired, tc.fire, out)
		}
		if skipped := len(out.Skipped) == 1; skipped != tc.skip {
			t.Errorf("%s: skipped = %v, want %v (%+v)", tc.when, skipped, tc.skip, out)
		}
		for _, s := range out.Skipped {
			if s.Reason != "needs a probed file" {
				t.Errorf("%s: skip reason %q, want the tier's description", tc.when, s.Reason)
			}
		}
	}
}

// Once the tier is there the same conditions are judged on their values.
func TestPresentTierIsJudged(t *testing.T) {
	probed := facts(map[string]Value{
		"resolution": StrOf("1080p"), "probed.bitDepth": NumOf(8), "probed.height": NumOf(1080),
	})
	for when, want := range map[string]bool{
		`resolution == "1080p" or probed.bitDepth == 10`:  true,
		`resolution == "1080p" and probed.bitDepth == 10`: false,
		`resolution == "720p" and probed.bitDepth == 10`:  false,
		`not probed.dolbyVision`:                          true,
		`probed.height < 1080`:                            false,
	} {
		if got := evalBool(t, when, probed); got != want {
			t.Errorf("%s = %v, want %v", when, got, want)
		}
	}
}

// A rule's payout and grouping read tiers too. A condition the release answers
// does not make a score it cannot answer act.
func TestAbsentTierInScoreOrGroupingSkips(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "pay by depth", When: `resolution == "1080p"`, Score: "probed.bitDepth * 10"},
		{Name: "cap by height", When: `resolution == "1080p"`, Action: ActionLimit, Count: 1, GroupBy: "probed.height"},
		{Name: "tag by range", When: `resolution == "1080p"`, Action: "tag", Score: "probed.dynamicRange"},
	})
	if err != nil {
		t.Fatal(err)
	}
	unprobed := facts(map[string]Value{"resolution": StrOf("1080p")}, "measured")
	out := eng.Evaluate(unprobed, "", nil)
	if out.Points != 0 || len(out.Matched) != 0 || len(out.Limits) != 0 || len(out.Effects) != 0 {
		t.Errorf("an unprobed release was acted on: %+v", out)
	}
	if len(out.Skipped) != 3 {
		t.Fatalf("skipped = %+v, want all three", out.Skipped)
	}
	for _, s := range out.Skipped {
		if s.Reason != "needs a probed file" {
			t.Errorf("%s: reason %q, want the tier's description", s.Name, s.Reason)
		}
	}
}

// A function registered against a tier reads like a field of that tier.
func TestAbsentTierThroughFunction(t *testing.T) {
	reg := testRegistry()
	reg.FuncTier("probedOK", nil, Bool, "measured", func(Facts, []Value) (Value, error) {
		return BoolOf(true), nil
	})
	eng, err := Compile(reg, []Rule{
		{Name: "either", When: `resolution == "1080p" or probedOK()`, Score: "1"},
		{Name: "both", When: `resolution == "1080p" and probedOK()`, Score: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := eng.Evaluate(facts(map[string]Value{"resolution": StrOf("1080p")}, "measured"), "", nil)
	if len(out.Matched) != 1 || out.Matched[0].Name != "either" {
		t.Errorf("matched = %+v, want only the rule the name settles", out.Matched)
	}
	if len(out.Skipped) != 1 || out.Skipped[0].Name != "both" || !strings.Contains(out.Skipped[0].Reason, "probed file") {
		t.Errorf("skipped = %+v, want the rule the call could have swung", out.Skipped)
	}
}

// The reason names the first missing thing the condition turned on, and a
// tier with no description is still named.
func TestAbsentTierReasonNamesTheTier(t *testing.T) {
	reg := Core()
	reg.Tier("seadex", "")
	reg.Namespace("seadex", "seadex").Bool("best")
	eng, err := Compile(reg, []Rule{{Name: "r", When: `seadex.best and resolution == "1080p"`, Score: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	out := eng.Evaluate(MapFacts{Values: map[string]Value{"resolution": StrOf("1080p")}}, "", nil)
	if len(out.Skipped) != 1 || out.Skipped[0].Reason != "needs seadex data, which this release has none of" {
		t.Errorf("skipped = %+v", out.Skipped)
	}
}

// The set is counted the same way: a release whose answer turns on a tier it
// lacks neither counts nor answers, but one the answerable half settles does.
func TestAggregateSettlesWhereItCan(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "either", When: `exists(resolution == "2160p" or probed.height >= 2000)`, Score: "1"},
		{Name: "both", When: `exists(resolution == "2160p" and probed.height >= 2000)`, Score: "1"},
		{Name: "count", When: `count(resolution == "2160p" and probed.height >= 2000) == 0`, Score: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	uhd := facts(map[string]Value{"resolution": StrOf("2160p")}, "measured")
	hd := facts(map[string]Value{"resolution": StrOf("1080p")}, "measured")

	st := eng.ComputeAggregates([]Facts{uhd, hd}, "")
	reports := eng.Aggregates(st)
	if len(reports) != 2 { // "both" and "count" ask the same question
		t.Fatalf("reports = %+v", reports)
	}
	// the `or` question: the 2160p release answers yes on its name alone
	if !reports[0].Known || reports[0].Count != 1 {
		t.Errorf("or: %+v, want known with one match", reports[0])
	}
	// the `and` question: the 1080p release answers no, the 2160p one cannot
	if !reports[1].Known || reports[1].Count != 0 {
		t.Errorf("and: %+v, want known with no match", reports[1])
	}

	// every question has an answer: "both" is answered no, so it does not fire
	out := eng.Evaluate(hd, "", st)
	if len(out.Matched) != 2 || out.Matched[0].Name != "either" || out.Matched[1].Name != "count" {
		t.Errorf("matched = %+v, want the or-question and the zero count", out.Matched)
	}
	if len(out.Skipped) != 0 {
		t.Errorf("skipped = %+v", out.Skipped)
	}

	// with only the 1080p release in the set, the or-question is unknown —
	// the probe could have made it true — while the and-question is still no
	st = eng.ComputeAggregates([]Facts{hd}, "")
	out = eng.Evaluate(hd, "", st)
	if len(out.Matched) != 1 || out.Matched[0].Name != "count" {
		t.Errorf("matched = %+v, want only the zero count", out.Matched)
	}
	if len(out.Skipped) != 1 || out.Skipped[0].Name != "either" || !strings.Contains(out.Skipped[0].Reason, "result set") {
		t.Errorf("skipped = %+v, want the or-question the set cannot answer", out.Skipped)
	}
}
