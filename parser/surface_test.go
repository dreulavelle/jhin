package parser

import (
	"reflect"
	"testing"
)

func TestLanguageTranslation(t *testing.T) {
	got := TranslateLanguages([]string{"en", "ja", "nope", "pt"})
	want := []string{"English", "Japanese", "Portuguese"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	r := Parse("Movie.2020.FRENCH.1080p.WEB-DL")
	if names := r.LanguageNames(); len(names) == 0 || names[0] != "French" {
		t.Fatalf("LanguageNames: %v (langs %v)", names, r.Languages)
	}
}

func TestParseAllOrder(t *testing.T) {
	titles := []string{
		"Movie.A.2020.1080p.WEB-DL",
		"Show.B.S02E05.720p.HDTV.x264",
		"Movie.C.2019.2160p.BluRay.REMUX",
		"[SubsPlease] Anime D - 12 (1080p) [ABCD1234].mkv",
		"Movie.E.2021.CAM",
		"Show.F.S01.COMPLETE.1080p.WEB-DL",
		"Movie.G.1999.DVDRip.XviD",
		"Movie.H.2022.720p.WEBRip",
		"Movie.I.2023.1080p.WEB-DL.DDP5.1",
	}
	batch := ParseAll(titles)
	for i, title := range titles {
		single := Parse(title)
		if !reflect.DeepEqual(batch[i], single) {
			t.Fatalf("batch[%d] differs from single parse for %q", i, title)
		}
	}
}

func TestExtractHelpers(t *testing.T) {
	if s := ExtractSeasons("Show S02E05 720p"); !reflect.DeepEqual(s, []int{2}) {
		t.Fatalf("seasons: %v", s)
	}
	if e := ExtractEpisodes("Show S02E05 720p"); !reflect.DeepEqual(e, []int{5}) {
		t.Fatalf("episodes: %v", e)
	}
	if e := EpisodesFromSeason("Show S02E05 720p", 2); !reflect.DeepEqual(e, []int{5}) {
		t.Fatalf("eps from season 2: %v", e)
	}
	if e := EpisodesFromSeason("Show S02E05 720p", 3); len(e) != 0 {
		t.Fatalf("eps from wrong season should be empty: %v", e)
	}
}
