package parser

import (
	"strings"
	"testing"
)

// cleanTitleUnguarded is cleanTitle with every guard removed — the reference
// the guards must reproduce exactly. Keep it in sync when cleanTitle's regex
// sequence changes; the guards themselves belong only in cleanTitle.
func cleanTitleUnguarded(rawTitle string) string {
	title := strings.TrimSpace(rawTitle)

	title = strings.ReplaceAll(title, "_", " ")
	title = replaceAll(movieIndicatorRegex, title, "")
	title = replaceAll(notAllowedSymbolsAtStartAndEndRegex, title, "")
	for _, parts := range russianCastRegex.FindAllStringSubmatch(title, -1) {
		for i, mStr := range parts {
			if i != 0 {
				title = strings.Replace(title, mStr, "", 1)
			}
		}
	}
	title = replaceAll(releaseGroupMarkingAtStartRegex, title, "$1")
	title = replaceAll(releaseGroupMarkingAtEndRegex, title, "$1")
	title = replaceAll(altTitlesRegex, title, "")
	for i, mStr := range notOnlyNonEnglishRegex.FindStringSubmatch(title) {
		if i != 0 {
			title = strings.Replace(title, mStr, "", 1)
		}
	}
	title = replaceAll(remainingNotAllowedSymbolsAtStartAndEndRegex, title, "")
	title = replaceAll(emptyBracketsRegex, title, "")
	title = replaceAll(mp3AtEndRegex, title, "")
	title = replaceAll(parenthesesWithoutContentRegex, title, "")
	title = replaceAll(specialCharSpacingRegex, title, "")

	for _, b := range brackets {
		if strings.Count(title, b[0]) != strings.Count(title, b[1]) {
			title = strings.ReplaceAll(strings.ReplaceAll(title, b[0], ""), b[1], "")
		}
	}

	if !strings.Contains(title, " ") && strings.Contains(title, ".") {
		title = strings.ReplaceAll(title, ".", " ")
	}

	title = replaceAll(redundantSymbolsAtEnd, title, "")
	title = replaceAll(whitespacesRegex, title, " ")

	return strings.TrimSpace(title)
}

// guardProbes exercise each guard's boundary: a title that satisfies the
// necessary condition and one that just misses it.
var guardProbes = []string{
	"",
	" ",
	"_",
	"#",
	"]",
	")",
	"(",
	"[",
	"{",
	"★",
	"【",
	"】",
	"Movie Name (movie) 1080p",
	"[MOVIE] Title",
	"-Leading dash title",
	"Title trailing dash-",
	"Title ending colon:",
	"Title.ending.dot.",
	`Title ending backslash\`,
	"Мстители / Avengers (2018) BDRip",
	"Avengers (Вася Пупкин)",
	"Some Title (a/b) (cast)",
	"[SubsPlease] Frieren - 28 (1080p) [F1FF71EB].mkv",
	"【Group】Title★",
	"Title★",
	"Title】",
	"Title ()",
	"Title []",
	"Title {}",
	"Title (   )",
	"Track name mp3",
	"Track name MP3",
	"Title--  spaced",
	"Title++  spaced",
	"Title[]  spaced",
	"a\tb",
	"a\n\nb",
	"a  b",
	"a b",
	"no.spaces.at.all",
	"ſtitle 1080p",
	"K sign title",
	"日本語 / Japanese Title",
	"العربية / Arabic",
	"#hashtag start",
	"]bracket start",
}

func TestCleanTitleGuards(t *testing.T) {
	var titles []string
	titles = append(titles, guardProbes...)
	titles = append(titles, corpusTitles(t)...)
	// the real call site feeds cleanTitle a rune prefix of the working title,
	// so exercise prefixes too
	for _, ti := range corpusTitles(t) {
		for _, n := range []int{1, 2, 3, 7, 15, 31} {
			titles = append(titles, runePrefix(ti, n))
		}
	}

	for _, ti := range titles {
		if got, want := cleanTitle(ti), cleanTitleUnguarded(ti); got != want {
			t.Errorf("guarded cleanTitle diverged for %q:\n  guarded:   %q\n  unguarded: %q", ti, got, want)
		}
	}
}

func FuzzCleanTitleGuards(f *testing.F) {
	for _, s := range guardProbes {
		f.Add(s)
	}
	f.Add("Deadpool 2016 1080p BluRay x264 DTS-JYK")
	f.Add("Мстители: Война бесконечности / Avengers: Infinity War (2018) BDRip 1080p")
	f.Fuzz(func(t *testing.T, title string) {
		if got, want := cleanTitle(title), cleanTitleUnguarded(title); got != want {
			t.Errorf("guarded cleanTitle diverged for %q:\n  guarded:   %q\n  unguarded: %q", title, got, want)
		}
	})
}
