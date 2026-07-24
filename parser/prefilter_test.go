package parser

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func corpusTitles(t testing.TB) []string {
	blob, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []goldenCase
	if err := json.Unmarshal(blob, &cases); err != nil {
		t.Fatal(err)
	}
	titles := make([]string, len(cases))
	for i, c := range cases {
		titles[i] = c.Title
	}
	return titles
}

// TestPrefilterEquivalence guarantees the literal gate never changes a parse:
// every corpus title must produce byte-identical results with gating on/off.
func TestPrefilterEquivalence(t *testing.T) {
	defer func() { prefilter_enabled = true }()
	for _, title := range corpusTitles(t) {
		prefilter_enabled = true
		gated := Parse(title)
		prefilter_enabled = false
		ungated := Parse(title)
		if !reflect.DeepEqual(gated, ungated) {
			g, _ := json.Marshal(gated)
			u, _ := json.Marshal(ungated)
			t.Errorf("prefilter changed result for %q:\n  gated:   %s\n  ungated: %s", title, g, u)
		}
	}
}

func FuzzPrefilterEquivalence(f *testing.F) {
	f.Add("Deadpool 2016 1080p BluRay x264 DTS-JYK")
	f.Add("www.Torrenting.com   -    14.Peaks.Nothing.Is.Impossible.2021.1080p.WEB.h264-RUMOUR")
	f.Add("[SubsPlease] Sousou no Frieren - 28 (1080p) [F1FF71EB].mkv")
	f.Add("Мстители: Война бесконечности / Avengers: Infinity War (2018) BDRip 1080p")
	f.Add("AEW DARK 4th December 2020 WEBRip h264-TJ")
	f.Add("Show hardſub 1080p WEBRip")
	f.Add("Movie remaſtered uncenſored 720p")
	f.Add("Kelvin K sign 1080p WEB-DL")
	f.Fuzz(func(t *testing.T, title string) {
		defer func() { prefilter_enabled = true }()
		prefilter_enabled = true
		gated := Parse(title)
		prefilter_enabled = false
		ungated := Parse(title)
		if !reflect.DeepEqual(gated, ungated) {
			t.Errorf("prefilter changed result for %q", title)
		}
	})
}
