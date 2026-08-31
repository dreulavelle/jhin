package rules

import (
	"strings"
	"testing"
	"time"
)

// Profiles are untrusted input: a rule set can arrive from a shared file, a
// remote library, or a text box. Compiling one must always terminate and must
// never panic — a malformed condition is an error naming the rule, never a
// crash that takes the search down with it.
func FuzzCompile(f *testing.F) {
	for _, seed := range []string{
		`resolution == "2160p"`,
		`sizeGB > 12 and resolution != "2160p"`,
		`min(grabs, 200) * 15`,
		`any(hdr, # == "DV")`,
		`count(resolution == "2160p") < 3`,
		`matched("x")`,
		`group in ["a", "b"]`,
		`releaseName matches "\bIMAX\b"`,
		`year > 2000 ? 1 : 2`,
		`not (a and b) or c`,
		`((((((true))))))`,
		`"\x41A\\"`,
		`1_000 + 1.5 % 2`,
		"`raw string`",
		`# == #`,
		`a.b.c.d`,
		``,
		`)`,
		`[[[[`,
	} {
		f.Add(seed)
	}

	reg := testRegistry()
	f.Fuzz(func(t *testing.T, src string) {
		// A condition, a score and a grouping reach the compiler by different
		// paths, so fuzz all three, plus both text-form entry points.
		_, _ = Compile(reg, []Rule{{Name: "fuzz", When: src}})
		_, _ = Compile(reg, []Rule{{Name: "fuzz", When: "true", Score: src}})
		_, _ = Compile(reg, []Rule{{Name: "fuzz", Action: ActionLimit, Count: 1, When: "true", GroupBy: src}})
		_, _ = ParseLine(src)
		_, _ = ParseText(src)
		// a rule may also name another, which is where inlining happens
		_, _ = Compile(reg, []Rule{
			{Name: "a", Action: ActionDefine, When: src},
			{Name: "b", When: `matched("a")`},
		})
	})
}

// Whatever compiles must also evaluate without panicking, however odd the
// facts it is handed.
func FuzzEvaluate(f *testing.F) {
	for _, seed := range []string{
		`resolution == "2160p"`,
		`len(traits) > 0 and traits[0] == "x"`,
		`sizeGB / (sizeGB - sizeGB) > 1`,
		`count(hdr, # == "DV") > 0`,
		`num(title) * 2 > 4`,
		`string(seasons) contains "1"`,
		`year % 0 == 0`,
		`min(year, sizeGB, grabs) > 0`,
	} {
		f.Add(seed)
	}

	reg := testRegistry()
	f.Fuzz(func(t *testing.T, src string) {
		eng, err := Compile(reg, []Rule{{Name: "fuzz", When: src}})
		if err != nil || eng == nil {
			return
		}
		for _, tiers := range [][]string{nil, {"measured"}, {"measured", "reported"}} {
			fx := facts(map[string]Value{
				"resolution": StrOf("2160p"),
				"title":      StrOf(""),
				"year":       NumOf(0),
				"sizeGB":     NumOf(0),
				"grabs":      NumOf(-1),
				"traits":     StrListOf(nil),
				"hdr":        StrListOf([]string{"DV"}),
				"seasons":    NumListOf([]int{1}),
			}, tiers...)
			out := eng.Evaluate(fx, "movie", nil)
			// an inconclusive rule never removes a release
			if len(out.Rejections) > 0 && len(out.Skipped) > 0 {
				t.Fatalf("%q both rejected and skipped: %+v", src, out)
			}
		}
	})
}

// A compiled expression must not be able to run for an unbounded time.
func TestEvaluationIsBounded(t *testing.T) {
	// deep-ish nesting that still parses, over a long list
	src := "count(traits, " + strings.Repeat("(", 40) + "# != \"\"" + strings.Repeat(")", 40) + ") >= 0"
	eng, err := Compile(testRegistry(), []Rule{{Name: "r", When: src}})
	if err != nil {
		t.Skipf("did not compile: %v", err)
	}
	big := make([]string, 20000)
	for i := range big {
		big[i] = "x"
	}
	done := make(chan Outcome, 1)
	go func() {
		done <- eng.Evaluate(facts(map[string]Value{"traits": StrListOf(big)}), "", nil)
	}()
	select {
	case out := <-done:
		// it either answered or gave up; either way it stopped
		_ = out
	case <-time.After(5 * time.Second):
		t.Fatal("evaluation did not finish")
	}
}

// The step budget stops a pathological expression rather than letting it run,
// and a rule that hit it is skipped rather than applied.
func TestStepBudget(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "heavy", When: `count(traits, # != "") >= 0`, Action: ActionReject},
	})
	if err != nil {
		t.Fatal(err)
	}
	big := make([]string, maxSteps+10)
	for i := range big {
		big[i] = "x"
	}
	out := eng.Evaluate(facts(map[string]Value{"traits": StrListOf(big)}), "", nil)
	if len(out.Rejections) != 0 {
		t.Errorf("a rule that ran out of budget still rejected: %+v", out)
	}
	if len(out.Skipped) != 1 {
		t.Errorf("skipped = %+v, want the over-budget rule reported", out.Skipped)
	}
}

// FuzzTextForm asserts the text form's contract: whatever parses, its
// canonical form parses back to the same rules, and formatting is stable —
// fmt applied twice is fmt applied once.
func FuzzTextForm(f *testing.F) {
	f.Add(`R: score 100 if title contains "two  spaces"`)
	f.Add("A [movie, series]: keep 3 per resolution if true")
	f.Add("B [off]: reject if proper\nC: define if repack")
	f.Add("Long: reject if\n    proper\n    and repack")
	f.Add("Split: score 10\n    + 5 if proper")
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 1<<14 {
			return
		}
		parsed, err := ParseText(src)
		if err != nil {
			return
		}
		text := FormatText(parsed)
		back, err := ParseText(text)
		if err != nil {
			t.Fatalf("canonical form does not parse back: %v\n%q", err, text)
		}
		if len(back) != len(parsed) {
			t.Fatalf("%d rules became %d:\n%q", len(parsed), len(back), text)
		}
		if again := FormatText(back); again != text {
			t.Fatalf("formatting is not stable:\n%q\n%q", text, again)
		}
	})
}
