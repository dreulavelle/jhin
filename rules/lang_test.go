package rules

import (
	"strings"
	"testing"
)

// testRegistry is Core plus a probe tier, an indexer tier, a function and an
// effect — the shape an application actually builds.
func testRegistry() *Registry {
	r := Core()
	r.Tier("measured", "a probed file")
	r.Tier("reported", "size, age or grabs, which a release name does not carry")
	r.Namespace("probed", "measured").
		Num("height").Num("bitDepth").Bool("dolbyVision").Str("dynamicRange")
	r.Field("sizeGB", Num, "reported")
	r.Field("ageDays", Num, "reported")
	r.Field("grabs", Num, "reported")
	r.Field("kind", Str, "")
	r.Func("double", []Type{Num}, Num, func(_ Facts, a []Value) (Value, error) {
		return NumOf(a[0].Num() * 2), nil
	})
	r.Effect("tag", Str)
	return r
}

// facts builds a release with everything present unless a tier is named as
// absent.
func facts(values map[string]Value, absentTiers ...string) MapFacts {
	tiers := map[string]bool{"measured": true, "reported": true}
	for _, t := range absentTiers {
		delete(tiers, t)
	}
	return MapFacts{Values: values, Tiers: tiers}
}

func compile1(t *testing.T, when string) *Engine {
	t.Helper()
	eng, err := Compile(testRegistry(), []Rule{{Name: "r", When: when}})
	if err != nil {
		t.Fatalf("compile %q: %v", when, err)
	}
	return eng
}

// evalBool compiles a condition as a rule and reports whether it fired.
func evalBool(t *testing.T, when string, f Facts) bool {
	t.Helper()
	eng := compile1(t, when)
	out := eng.Evaluate(f, "", nil)
	return len(out.Matched) == 1
}

func TestExpressions(t *testing.T) {
	f := facts(map[string]Value{
		"resolution":    StrOf("2160p"),
		"quality":       StrOf("WEB-DL"),
		"year":          NumOf(2020),
		"group":         StrOf("FraMeSToR"),
		"traits":        StrListOf([]string{"remux", "hevc", "10bit"}),
		"hdr":           StrListOf([]string{"DV", "HDR10"}),
		"languages":     StrListOf([]string{"en", "de"}),
		"seasons":       NumListOf([]int{1}),
		"episodes":      NumListOf([]int{}),
		"title":         StrOf("The Matrix"),
		"releaseName":   StrOf("The.Matrix.1999.2160p.IMAX.WEB-DL"),
		"dolbyVision":   BoolOf(true),
		"hdrFallback":   BoolOf(true),
		"upscaled":      BoolOf(false),
		"sizeGB":        NumOf(28.5),
		"ageDays":       NumOf(3),
		"grabs":         NumOf(42),
		"probed.height": NumOf(2160),
	})

	for _, tc := range []struct {
		when string
		want bool
	}{
		// comparison and logic
		{`resolution == "2160p"`, true},
		{`resolution != "1080p"`, true},
		{`year > 2000 and year < 2030`, true},
		{`year >= 2020 and year <= 2020`, true},
		{`not upscaled`, true},
		{`upscaled or year == 2020`, true},
		{`false or false`, false},

		// membership
		{`"remux" in traits`, true},
		{`"cam" in traits`, false},
		{`"cam" not in traits`, true},
		{`"DV" in hdr`, true},
		{`1 in seasons`, true},
		{`group in ["FraMeSToR", "W4NK3R"]`, true},
		{`group in []`, false},

		// text
		{`releaseName matches "(?i)\bIMAX\b"`, true},
		{`releaseName matches "(?i)\bDIRECTORS\b"`, false},
		{`releaseName contains "Matrix"`, true},
		{`title startsWith "The"`, true},
		{`title endsWith "Matrix"`, true},
		{`lower(group) == "framestor"`, true},
		{`"Matrix" in releaseName`, true},

		// arithmetic and ternary
		{`sizeGB * 2 > 50`, true},
		{`sizeGB - 28.5 == 0`, true},
		{`(year - 2000) / 2 == 10`, true},
		{`year % 2 == 0`, true},
		{`(resolution == "2160p" ? 10 : 1) == 10`, true},
		{`-year < 0`, true},

		// builtins
		{`len(traits) == 3`, true},
		{`len(title) == 10`, true},
		{`min(grabs, 10) == 10`, true},
		{`max(grabs, 10) == 42`, true},
		{`abs(0 - grabs) == 42`, true},
		{`round(sizeGB) == 29`, true},
		{`floor(sizeGB) == 28`, true},
		{`ceil(sizeGB) == 29`, true},
		{`double(grabs) == 84`, true},
		{`num("12") == 12`, true},
		{`num("nope") == 0`, true},
		{`string(year) == "2020"`, true},
		{`upper(quality) == "WEB-DL"`, true},
		{`trim("  x ") == "x"`, true},

		// collection predicates with #
		{`any(hdr, # == "DV")`, true},
		{`all(hdr, # != "")`, true},
		{`none(hdr, # == "HLG")`, true},
		{`count(hdr, # startsWith "H") == 1`, true},

		// the composite the pitch leads on
		{`dolbyVision and not hdrFallback`, false},
		{`sizeGB > 12 and resolution != "2160p"`, false},
		{`probed.height >= 2000`, true},
	} {
		if got := evalBool(t, tc.when, f); got != tc.want {
			t.Errorf("%s = %v, want %v", tc.when, got, tc.want)
		}
	}
}

func TestCompileErrors(t *testing.T) {
	for _, tc := range []struct{ when, contains string }{
		{`resolutoin == "x"`, "did you mean"},
		{`nosuchfield`, "unknown attribute"},
		{`year`, "has to be bool"},
		{`year == "2020"`, "cannot compare"},
		{`year and true`, "joins yes/no"},
		{`not year`, "needs a yes/no"},
		{`1 < 2 < 3`, "do not chain"},
		{`year > "x"`, "compares two numbers or two strings"},
		{`"a" in year`, "needs a list or text"},
		{`year matches "x"`, "compares text against a pattern"},
		{`title matches title`, "written out"},
		{`title matches "("`, "bad pattern"},
		{`year / 0 == 1`, "dividing by zero"},
		{`nope(1)`, "unknown function"},
		{`double(1, 2)`, "takes 1 argument"},
		{`double("x")`, "wants num"},
		{`len()`, "takes 1 argument"},
		{`# == 1`, "only meaningful inside"},
		{`any(traits, # == 1)`, "cannot compare"},
		{`any(year, # == 1)`, "wants a list"},
		{`[1, "a"]`, "mixes"},
		{`(year > 1 ? 1 : "x") == 1`, "have to agree"},
		{`year >`, "expected a value"},
		{`(year`, "expected \")\""},
		{`"unterminated`, "unterminated string"},
		{`year @ 2`, "unexpected character"},
	} {
		_, err := Compile(testRegistry(), []Rule{{Name: "r", When: tc.when}})
		if err == nil {
			t.Errorf("%s: compiled, want error containing %q", tc.when, tc.contains)
			continue
		}
		if !strings.Contains(err.Error(), tc.contains) {
			t.Errorf("%s: error %q, want it to contain %q", tc.when, err, tc.contains)
		}
	}
}

// A pasted regex has to mean what it says: \b is a word boundary, not a
// backspace, and \+ is a literal plus rather than a refused save.
func TestRawRegexEscapes(t *testing.T) {
	f := facts(map[string]Value{"releaseName": StrOf("Movie C++ 2020 IMAX")})
	for _, when := range []string{
		`releaseName matches "\bIMAX\b"`,
		`releaseName matches "C\+\+"`,
		`releaseName matches "\d{4}"`,
		`releaseName matches "\\bIMAX\\b"`, // defensively doubled means the same
	} {
		if !evalBool(t, when, f) {
			t.Errorf("%s did not match", when)
		}
	}
	// a real string escape still works
	if !evalBool(t, `"a\tb" contains "\t"`, f) {
		t.Error(`\t lost its string meaning`)
	}
}

func TestScoreIsAnExpression(t *testing.T) {
	f := facts(map[string]Value{"grabs": NumOf(300), "ageDays": NumOf(20), "sizeGB": NumOf(6)})
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "Seeders", When: "grabs > 0", Score: "min(grabs, 200) * 15"},
		{Name: "Freshness", When: "true", Score: "max(0, 500 - ageDays * 5)"},
		{Name: "Sweet spot", When: "sizeGB > 0", Score: "2000 - abs(sizeGB - 4) * 300"},
		{Name: "Flat", When: "true", Score: "-800"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := eng.Evaluate(f, "", nil)
	want := map[string]int{"Seeders": 3000, "Freshness": 400, "Sweet spot": 1400, "Flat": -800}
	for _, m := range out.Matched {
		if want[m.Name] != m.Score {
			t.Errorf("%s scored %d, want %d", m.Name, m.Score, want[m.Name])
		}
	}
	if out.Points != 3000+400+1400-800 {
		t.Errorf("points = %d, want %d", out.Points, 3000+400+1400-800)
	}
}

// Without this, one probe rule empties every result list of everything except
// the releases that happen to have been opened.
func TestFailOpen(t *testing.T) {
	rules := []Rule{
		{Name: "Small probe", When: "probed.height < 1080", Action: ActionReject},
		{Name: "Old", When: "ageDays > 1000", Action: ActionReject},
		{Name: "Always", When: "true", Score: "10"},
	}
	eng, err := Compile(testRegistry(), rules)
	if err != nil {
		t.Fatal(err)
	}

	// a release nobody probed: the probe rule is skipped, not applied
	unprobed := facts(map[string]Value{"ageDays": NumOf(5)}, "measured")
	out := eng.Evaluate(unprobed, "", nil)
	if len(out.Rejections) != 0 {
		t.Errorf("unprobed release was rejected: %v", out.Rejections)
	}
	if len(out.Skipped) != 1 || out.Skipped[0].Name != "Small probe" {
		t.Errorf("skipped = %v, want the probe rule", out.Skipped)
	}
	if out.Points != 10 {
		t.Errorf("points = %d, want the always-rule to have run", out.Points)
	}

	// a release that was probed and really is small: now it rejects
	small := facts(map[string]Value{"probed.height": NumOf(720), "ageDays": NumOf(5)})
	out = eng.Evaluate(small, "", nil)
	if len(out.Rejections) != 1 {
		t.Errorf("probed 720p release was not rejected: %v", out.Rejections)
	}

	// the reason names what is missing rather than leaving it to be inferred
	out = eng.Evaluate(facts(nil, "measured", "reported"), "", nil)
	if len(out.Skipped) != 2 {
		t.Fatalf("skipped = %v, want both tier rules", out.Skipped)
	}
	if !strings.Contains(out.Skipped[0].Reason, "probed file") {
		t.Errorf("skip reason %q does not say what is missing", out.Skipped[0].Reason)
	}
}

func TestScope(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "movies only", When: "true", Score: "100", Scope: []string{"movie"}},
		{Name: "anime only", When: "true", Score: "10", Scope: []string{"anime_show"}},
		{Name: "everything", When: "true", Score: "1", Scope: []string{"all"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for kind, want := range map[string]int{"movie": 101, "anime_show": 11, "series": 1, "": 1} {
		if got := eng.Evaluate(facts(nil), kind, nil).Points; got != want {
			t.Errorf("kind %q scored %d, want %d", kind, got, want)
		}
	}
}

func TestEffects(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "label", When: `resolution == "2160p"`, Action: "tag", Score: `"uhd-" + quality`},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := eng.Evaluate(facts(map[string]Value{
		"resolution": StrOf("2160p"), "quality": StrOf("WEB-DL"),
	}), "", nil)
	if len(out.Effects) != 1 || out.Effects[0].Name != "tag" || out.Effects[0].Value.Str() != "uhd-WEB-DL" {
		t.Fatalf("effects = %+v", out.Effects)
	}
}

func TestDisabledRule(t *testing.T) {
	off := false
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "on", When: "true", Score: "1"},
		{Name: "off", When: "true", Score: "100", Enabled: &off},
		// a disabled rule is never compiled, so a broken one cannot block a save
		{Name: "broken but off", When: "this is not valid", Score: "1", Enabled: &off},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.Evaluate(facts(nil), "", nil).Points; got != 1 {
		t.Errorf("points = %d, want only the enabled rule", got)
	}
}

// Untrusted profiles must fail rather than crash or hang.
func TestHostileInput(t *testing.T) {
	deep := strings.Repeat("(", 500) + "true" + strings.Repeat(")", 500)
	if _, err := Compile(testRegistry(), []Rule{{Name: "r", When: deep}}); err == nil {
		t.Error("deeply nested expression compiled")
	}
	// a long flat chain is legal up to the token cap, and refused past it
	// rather than recursed over
	long := "true" + strings.Repeat(" and true", 2000)
	if _, err := Compile(testRegistry(), []Rule{{Name: "r", When: long}}); err != nil {
		t.Errorf("a long flat expression should compile: %v", err)
	}
	huge := "true" + strings.Repeat(" and true", 40000)
	if _, err := Compile(testRegistry(), []Rule{{Name: "r", When: huge}}); err == nil {
		t.Error("an expression past the token cap compiled")
	}
	if _, err := Compile(testRegistry(), []Rule{{Name: "r", When: strings.Repeat("a", maxSource+1)}}); err == nil {
		t.Error("an oversized expression compiled")
	}
}
