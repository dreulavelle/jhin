package parser

import (
	"slices"
	"testing"
)

// A language fused to "sub" with no separator (ENGSUB, ESub, VOSTFR) is
// subtitle evidence the generic sub handlers cannot see, and the language
// handlers that read it consume the text. These pin that both facts survive.
func TestCompoundSubtitleTokens(t *testing.T) {
	for _, tc := range []struct {
		title  string
		subbed bool
		lang   string
	}{
		{"Show.S01E01.1080p.WEB-DL.ENGSUB.x264-GRP", true, "en"},
		{"Movie.2018.1080p.WEB-DL.ESubs.x264-GRP", true, "en"},
		{"Movie (2018) 720p WEBRip x264 AAC HC-Esub - GRP.mkv", true, "en"},
		{"Movie.2017.KOREAN.ENSUBBED.1080p.WEBRip.x264-GRP", true, "en"},
		{"Film.2019.1080p.VOSTFR.x264-GRP", true, "fr"},
		{"Film.2019.1080p.VostFR.BRrip.x264", true, "fr"},
		{"Show.S06E07.SUBFRENCH.HDTV.x264-GRP.mkv", true, "fr"},
		{"Movie.2013.SWESUB.DANSUB.FiNSUB.720p.WEB-DL", true, "sv"},
		{"Show.S01.SweSub.720p.WEB-DL.H264", true, "sv"},
		{"Movie.2018.KORSUB.HDRip.XviD.MP3-GRP", true, "ko"},
		{"Movie.2025.PLSUB.1080p.WEB-DL.DDP5.1.H.264-GRP.mkv", true, "pl"},
		{"Show.COMPLETE.SLOSUBS.DVDRip.XviD", true, "sk"},
		{"Movie.Tetralogy.BRRip.XviD.AC3.RoSubbed-GRP", true, "ro"},
		{"Movie.2020.1080p.WEB-DL.DDP5.1.H.264.EN-ROSub-GRP", true, "ro"},
		{"Movie.2018.1080p.BluRay.ArabSub.x264-GRP", true, "ar"},
		// separated forms already reached the generic handlers
		{"Movie.2020.1080p.WEB-DL.KOR-SUB.x264-GRP", true, "ko"},
		{"Movie.2018.1080p.BluRay.Sub.Ita.x264-GRP", true, "it"},
		// fansub groups and title words are not subtitle tags
		{"[HorribleSubs] White Album 2 - 06 [1080p].mkv", false, ""},
		{"[SubsPlease] One Piece - 1111 (480p) [2E05E658].mkv", false, ""},
		{"[Kaerizaki-Fansub] One Piece 1098 FHD (1920x1080).mp4", false, ""},
		{"KNK E MMS Fansubs", false, ""},
		{"The.Substitute.1996.1080p.BluRay.x264-GRP", false, ""},
		{"Suburbicon.2017.1080p.BluRay.x264-GRP", false, ""},
		{"Vostok.Station.2019.1080p.WEB-DL.x264-GRP", false, ""},
		{"Subway.1985.1080p.BluRay.x264-GRP", false, ""},
	} {
		r := Parse(tc.title)
		if r.Subbed != tc.subbed {
			t.Errorf("%q: subbed=%v, want %v", tc.title, r.Subbed, tc.subbed)
		}
		if tc.lang != "" && !slices.Contains(r.Languages, tc.lang) {
			t.Errorf("%q: languages %v lack %q", tc.title, r.Languages, tc.lang)
		}
	}
}

// The fused-token handler claims subbed before the generic sub handlers run;
// those must still remove their text or a leftover MULTi reads as dubbed and
// a bare Sub leaks into the title or group.
func TestFusedSubtitleTokenWithSecondSubToken(t *testing.T) {
	for _, tc := range []struct {
		title      string
		wantTitle  string
		wantGroup  string
		wantSubbed bool
		wantDubbed bool
	}{
		{"Movie.2019.MULTI.SUBS.VOSTFR.1080p.BluRay.x264-GRP", "Movie", "GRP", true, false},
		{"Anime - 01 [1080p][ENGSUB][Multi-Subs]", "Anime", "", true, false},
		{"Movie.2019.1080p.WEB-DL.ESubs.MULTi.SUBS.x264", "Movie", "", true, false},
		{"Show.S01E01.VOSTFR.MULTi.SUBS.WEB-DL.1080p", "Show", "", true, false},
		{"[Eng Sub] Show S01E01 1080p ESub", "Show", "", true, false},
		{"Esub Subbed Movie 2019 1080p", "Movie", "", true, false},
	} {
		r := Parse(tc.title)
		if r.Title != tc.wantTitle || r.Group != tc.wantGroup || r.Subbed != tc.wantSubbed || r.Dubbed != tc.wantDubbed {
			t.Errorf("%q: title=%q group=%q subbed=%v dubbed=%v, want title=%q group=%q subbed=%v dubbed=%v",
				tc.title, r.Title, r.Group, r.Subbed, r.Dubbed, tc.wantTitle, tc.wantGroup, tc.wantSubbed, tc.wantDubbed)
		}
	}
}
