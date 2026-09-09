package parser

import (
	"fmt"
	"strings"
	"testing"
)

// The subtitles field is a subset of languages, so a token may never resolve
// to one code here and a different one in the language handlers: that would
// make a single Result contradict itself. This walks every entry in the four
// token maps, puts the token in the shape that map is written for, and holds
// it against what the language handlers say about the same title.
func TestSubtitleMapsAgreeWithLanguageHandlers(t *testing.T) {
	const carrier = "Some Movie 2020 1080p BluRay %s x264-GRP"

	// The VOST family is deliberately absent: VOSTA is French audio with
	// English subtitles, so there the subtitle language is not the language
	// the handlers report at all.
	cases := []struct {
		name   string
		table  map[string]string
		shapes []string
	}{
		{"fused prefix", subtitleFusedPrefixLangs, []string{"%sSUB", "%sSUBS"}},
		{"fused suffix", subtitleFusedSuffixLangs, []string{"SUB%s"}},
		{"adjacent", subtitleAdjacentLangs, []string{"Subs %s", "Subs.%s"}},
	}

	for _, c := range cases {
		for tok, code := range c.table {
			for _, shape := range c.shapes {
				title := fmt.Sprintf(carrier, fmt.Sprintf(shape, strings.ToUpper(tok)))
				r := Parse(title)
				if len(r.Languages) == 0 {
					continue // the handlers have no opinion on this shape
				}
				found := false
				for _, l := range r.Languages {
					if l == code {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s map: %q -> %q, but the language handlers read %q as %v",
						c.name, tok, code, title, r.Languages)
				}
			}
		}
	}
}

// A sub marker introduces a run of languages, not just the first one.
func TestSubtitleRunCollectsEveryLanguage(t *testing.T) {
	for _, tc := range []struct {
		title string
		want  []string
	}{
		{"Divorzio allitaliana [XviD - Ita Mp3 - Sub Eng Esp Tur]", []string{"en", "es", "tr"}},
		{"Red Riding 1974 [2009 PAL DVD][En Subs[Sv.No.Fi]", []string{"en", "sv", "no", "fi"}},
		{"The Insider*1999*[DVD5][PAL][ENG, POL, sub. ROM, TUR]", []string{"ro", "tr"}},
		{"Show.S01.1080p.WEB-DL.Subs.EN.FR.DE.x264-GRP", []string{"en", "fr", "de"}},
		// guards: the run must not read title words, a negation, or a bare
		// multi marker as languages
		{"[Eng Subs] It Happened One Night 1934 1080p BluRay x264", []string{"en"}},
		{"Movie.2020.No.Subs.1080p.WEB-DL", nil},
		{"Movie.2020.MULTi-Subs.1080p.WEB-DL", nil},
	} {
		got := Parse(tc.title).Subtitles
		if len(got) != len(tc.want) {
			t.Errorf("%q: subtitles = %v, want %v", tc.title, got, tc.want)
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("%q: subtitles = %v, want %v", tc.title, got, tc.want)
				break
			}
		}
	}
}
