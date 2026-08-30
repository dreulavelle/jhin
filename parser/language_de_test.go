package parser

import (
	"slices"
	"testing"
)

// A bare uppercase DE is a German scene tag in metadata and the Romance
// preposition "de" in a title; these pin which contexts jhin trusts.
func TestBareDEAsGerman(t *testing.T) {
	for _, tc := range []struct {
		title  string
		german bool
	}{
		// past the title region
		{"Movie.Name.2020.DE.1080p.BluRay.x264-GROUP", true},
		{"Movie.Name.2020.1080p.BluRay.x264.DE-GROUP", true},
		{"Movie Name 2020 DE DL 1080p BluRay x264", true},
		// abutting a metadata token that survives to the language handlers
		{"Movie.Name.2020.DE.DL.1080p.BluRay.x264-GROUP", true},
		{"Show.S01.2020.DE.DUBBED.1080p.WEB-DL", true},
		{"Movie.Name.2020.MULTi.DE.EN.1080p.BluRay", true},
		// a run of two language codes, which the code-run gate alone missed
		{"Movie.Name.2020.EN.DE.1080p.WEB-DL", true},
		{"Movie.Name.2020.1080p.WEB-DL.DE.EN", true},
		// Romance prepositions, upper and title case
		{"LOS SIMPSONS TEMP 7 DVDRIP ESPANOL DE ESPANA", false},
		{"EL CLUB DE LA LUCHA 1080p BluRay x264", false},
		{"LA CASA DE PAPEL S01E01 1080p WEB-DL", false},
		{"La.Casa.De.Papel.S01E01.1080p.WEB-DL", false},
		{"PELICULA.DE.TERROR.2020.1080p.WEB-DL", false},
		{"CIUDAD DE DIOS 2002 1080p BluRay x264", false},
		{"Anatomia De Grey - Temporada 19 [HDTV][Cap.1905][Castellano]", false},
		{"Avatar La Voie de l'eau.FRENCH.CAMHD.H264.AAC", false},
		// a German tracker domain is not a language tag
		{"www.SerienJunkies.DE - Movie.Name.2020.1080p.WEB-DL", false},
	} {
		got := slices.Contains(Parse(tc.title).Languages, "de")
		if got != tc.german {
			t.Errorf("%q: de=%v, want %v (languages %v)",
				tc.title, got, tc.german, Parse(tc.title).Languages)
		}
	}
}
