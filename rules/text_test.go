package rules

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseText(t *testing.T) {
	off := false
	for _, tc := range []struct {
		line string
		want Rule
	}{
		{
			`Atmos: score -800 if "atmos" in traits`,
			Rule{Name: "Atmos", Action: ActionScore, Score: "-800", When: `"atmos" in traits`},
		},
		{
			`DV without HDR fallback: reject if dolbyVision and not hdrFallback`,
			Rule{Name: "DV without HDR fallback", Action: ActionReject, When: "dolbyVision and not hdrFallback"},
		},
		{
			`At most 3 in 4K [movie]: keep 3 if resolution == "2160p"`,
			Rule{Name: "At most 3 in 4K", Action: ActionLimit, Count: 3, Scope: []string{"movie"}, When: `resolution == "2160p"`},
		},
		{
			`Best 3 of each flavour: keep 3 per resolution + " " + quality if true`,
			Rule{Name: "Best 3 of each flavour", Action: ActionLimit, Count: 3, GroupBy: `resolution + " " + quality`, When: "true"},
		},
		{
			`UHD T1: define if group in ["FraMeSToR", "W4NK3R"]`,
			Rule{Name: "UHD T1", Action: ActionDefine, When: `group in ["FraMeSToR", "W4NK3R"]`},
		},
		{
			`Old experiment [off]: score 100 if "remux" in traits`,
			Rule{Name: "Old experiment", Action: ActionScore, Score: "100", When: `"remux" in traits`, Enabled: &off},
		},
		{
			`Both [anime_show] [off]: score 1 if true`,
			Rule{Name: "Both", Action: ActionScore, Score: "1", When: "true", Scope: []string{"anime_show"}, Enabled: &off},
		},
		{
			`Seeders: score min(grabs, 200) * 15 if grabs > 0`,
			Rule{Name: "Seeders", Action: ActionScore, Score: "min(grabs, 200) * 15", When: "grabs > 0"},
		},
		{
			`Label: tag "uhd" if resolution == "2160p"`,
			Rule{Name: "Label", Action: "tag", Score: `"uhd"`, When: `resolution == "2160p"`},
		},
		// a ternary in the score, and a colon in the condition
		{
			`Tier: score resolution == "2160p" ? 100 : 10 if year > 2000`,
			Rule{Name: "Tier", Action: ActionScore, Score: `resolution == "2160p" ? 100 : 10`, When: "year > 2000"},
		},
		// the word "if" inside a string is not the separator
		{
			`Odd: score 1 if title contains " if "`,
			Rule{Name: "Odd", Action: ActionScore, Score: "1", When: `title contains " if "`},
		},
	} {
		got, err := ParseLine(tc.line)
		if err != nil {
			t.Errorf("%s: %v", tc.line, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s\n got %+v\nwant %+v", tc.line, got, tc.want)
		}
	}
}

func TestTextRoundTrip(t *testing.T) {
	src := `Atmos: score -800 if "atmos" in traits
DV without HDR fallback: reject if dolbyVision and not hdrFallback
At most 3 in 4K [movie]: keep 3 if resolution == "2160p"
Best 3 of each flavour: keep 3 per resolution + " " + quality if true
UHD T1: define if group in ["FraMeSToR", "W4NK3R"]
Old experiment [off]: score 100 if "remux" in traits
Seeders: score min(grabs, 200) * 15 if grabs > 0
`
	rules, err := ParseText(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatText(rules); got != src {
		t.Errorf("round trip changed the text:\n got %q\nwant %q", got, src)
	}
	// and the result still compiles
	if _, err := Compile(testRegistry(), rules); err != nil {
		t.Fatalf("round-tripped rules do not compile: %v", err)
	}
}

func TestParseTextErrors(t *testing.T) {
	for _, tc := range []struct{ line, contains string }{
		{`no colon here`, "Name: action if condition"},
		{`Name: score 1`, "if <condition>"},
		{`Name: if true`, "an action"},
		{`Name: reject 5 if true`, "takes nothing"},
		{`Name: keep if true`, "keep needs a number"},
		{`Name: keep x if true`, "keep needs a number"},
		{`Name: keep 3 by resolution if true`, "per <grouping>"},
		{`: score 1 if true`, "no name"},
		{`Name []: score 1 if true`, "empty []"},
	} {
		if _, err := ParseLine(tc.line); err == nil {
			t.Errorf("%s: parsed, want %q", tc.line, tc.contains)
		} else if !strings.Contains(err.Error(), tc.contains) {
			t.Errorf("%s: error %q, want it to contain %q", tc.line, err, tc.contains)
		}
	}
}

func TestParseTextSkipsBlanksAndComments(t *testing.T) {
	rules, err := ParseText("\n# a note\n// another\nA: score 1 if true\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("%d rules, want 1", len(rules))
	}
}

func TestParseTextReportsLine(t *testing.T) {
	_, err := ParseText("A: score 1 if true\nB: keep x if true\n")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error %v, want it to name line 2", err)
	}
}

// A condition worth writing is often too long to read on one line.
func TestParseTextContinuation(t *testing.T) {
	src := `Untrusted UHD encode: reject if
    resolution == "2160p" and "bluray" in traits
    and not (matched("UHD T1") or matched("UHD T2"))
    and exists(resolution == "2160p" and "remux" in traits)
UHD T1: define if group in ["FraMeSToR"]
UHD T2: define if group in ["HiFi"]
`
	rules, err := ParseText(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("%d rules, want 3: %+v", len(rules), rules)
	}
	want := "resolution == \"2160p\" and \"bluray\" in traits\n" +
		"and not (matched(\"UHD T1\") or matched(\"UHD T2\"))\n" +
		"and exists(resolution == \"2160p\" and \"remux\" in traits)"
	if rules[0].When != want {
		t.Errorf("condition folded to\n %q\nwant\n %q", rules[0].When, want)
	}
	// the canonical form is still one line
	if strings.Contains(FormatLine(rules[0]), "\n") {
		t.Error("FormatLine kept the author's line breaks")
	}
	if _, err := Compile(testRegistry(), rules); err != nil {
		t.Fatalf("the folded rule does not compile: %v", err)
	}

	// a blank line ends a rule, so an indented line after one starts fresh
	if _, err := ParseText("A: score 1 if true\n\n    B: score 2 if true\n"); err != nil {
		t.Errorf("an indented line after a blank should stand alone: %v", err)
	}
	// and the error still names the line the rule started on
	_, err = ParseText("A: score 1 if true\nB: keep x if\n    true\n")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error %v, want it to name line 2", err)
	}
}
