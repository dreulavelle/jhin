package parser

import (
	"slices"
	"testing"
)

// A bare uppercase JA is a Japanese tag in metadata (jhin #39) but "yes" in
// German and "I" in Polish titles; these pin which contexts jhin trusts,
// mirroring the bare-DE cases in language_de_test.go.
func TestBareJAAsJapanese(t *testing.T) {
	for _, tc := range []struct {
		title    string
		japanese bool
	}{
		// the report: compact codes must behave like the full words
		{"Anime.Show.S01E01.1080p.WEB-DL.JA.EN.DDP5.1.H.264-GRP", true},
		{"Anime.Show.S01E01.1080p.WEB-DL.Japanese.English.DDP5.1.H.264-GRP", true},
		// past the title region
		{"Movie.Name.2020.JA.1080p.BluRay.x264-GROUP", true},
		{"Movie.Name.2020.1080p.BluRay.x264.JA-GROUP", true},
		// abutting a metadata token that survives to the language handlers
		{"Show.S01.2020.JA.DUBBED.1080p.WEB-DL", true},
		{"Movie.Name.2020.MULTi.JA.EN.1080p.BluRay", true},
		// a run of two language codes
		{"Movie.Name.2020.EN.JA.1080p.WEB-DL", true},
		{"Movie.Name.2020.1080p.WEB-DL.JA.EN", true},
		// prose: German "ja" and Polish "ja" inside the title, any case
		{"Ja.Ja.Ding.Dong.2020.1080p.WEB-DL", false},
		{"JA JA DING DONG 2020 1080p WEB-DL", false},
		{"Sag.Ja.Zum.Leben.2019.1080p.WEB-DL", false},
		{"Ja.Ich.Will.2021.German.1080p.BluRay", false},
		// a tracker domain is not a language tag
		{"www.Anime.JA - Show.S01E01.1080p.WEB-DL", false},
	} {
		got := slices.Contains(Parse(tc.title).Languages, "ja")
		if got != tc.japanese {
			t.Errorf("%q: ja=%v, want %v (languages %v)",
				tc.title, got, tc.japanese, Parse(tc.title).Languages)
		}
	}
}

// Bare uppercase AR fails the same way JA did: ARA, ARABIC and ArabSub parse,
// the compact code is dropped in every position. AR is trusted only next to
// a metadata token or inside a run of codes — "ar" is a Portuguese and
// Catalan word, and the PTT corpus has "XviD.AR [PT ENG ESP]" where AR is
// not a language, so a lone AR past the title stays unmatched.
func TestBareARAsArabic(t *testing.T) {
	for _, tc := range []struct {
		title  string
		arabic bool
	}{
		// long forms already work; the compact code must match them
		{"Movie.Name.2020.1080p.WEB-DL.Arabic.DDP5.1.H.264-GRP", true},
		{"Movie.Name.2020.1080p.WEB-DL.AR.EN.DDP5.1.H.264-GRP", true},
		{"Movie.Name.2020.1080p.WEB-DL.EN.AR.DDP5.1.H.264-GRP", true},
		// abutting a surviving metadata token
		{"Movie.Name.2020.MULTi.AR.EN.1080p.BluRay", true},
		{"Show.S01E01.1080p.NF.WEB-DL.DDP5.1.H.264.AR.SUBS-GRP", true},
		{"Show.S01.2020.AR.DUBBED.1080p.WEB-DL", true},
		// three-code run with JA
		{"Anime.Show.S01E01.1080p.WEB-DL.JA.AR.DDP5.1-GRP", true},
		// a lone AR past the title is not trusted (see the corpus title above)
		{"Movie.Name.2020.AR.1080p.WEB-DL.H.264-GRP", false},
		{"Movie.Name.2020.1080p.WEB-DL.AR-GRP", false},
		{"Un.Homme.Et.Une.Femme.1966.DVDRip.XviD.AR [PT ENG ESP]", false},
		// prose: Portuguese and Catalan "ar" inside the title, any case
		{"Castelos.De.Ar.2020.1080p.WEB-DL", false},
		{"CASTELOS DE AR 2020 1080p WEB-DL", false},
		{"Homes.A.L.Ar.2019.1080p.WEB-DL", false},
		// a tracker domain is not a language tag
		{"www.Movies.AR - Movie.Name.2020.1080p.WEB-DL", false},
	} {
		got := slices.Contains(Parse(tc.title).Languages, "ar")
		if got != tc.arabic {
			t.Errorf("%q: ar=%v, want %v (languages %v)",
				tc.title, got, tc.arabic, Parse(tc.title).Languages)
		}
	}
}
