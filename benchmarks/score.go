package benchmarks

// Scoring: every title/field pair lands in exactly one outcome bucket.
// Accuracy is computed only over titles where the gold label has a value —
// a library is never rewarded for the corpus being sparse — and spurious
// extractions (gold empty, library extracted something) are reported
// separately because a wrong value corrupts downstream matching in a way an
// empty one does not.

import "sort"

// ScoredFields are the fields every benchmarked library claims to extract.
var ScoredFields = []string{
	"title", "year", "seasons", "episodes",
	"resolution", "source", "codec", "group", "container",
}

// FieldScore tallies one library's outcomes on one field.
type FieldScore struct {
	Correct  int // gold has a value, library matched it
	Wrong    int // gold has a value, library extracted something else
	Missed   int // gold has a value, library extracted nothing
	Spurious int // gold empty, library extracted something
	Absent   int // gold empty, library empty
}

// Present is how many corpus titles carry a gold label for this field.
func (f FieldScore) Present() int { return f.Correct + f.Wrong + f.Missed }

// Accuracy is Correct / Present.
func (f FieldScore) Accuracy() float64 {
	if f.Present() == 0 {
		return 0
	}
	return float64(f.Correct) / float64(f.Present())
}

// LibraryScore is one library's full scorecard.
type LibraryScore struct {
	Library *Library
	Fields  map[string]*FieldScore
	Errors  int // titles where the library returned an error or panicked
	Titles  int
}

// Overall is the mean of per-field accuracies (fields weight equally, so a
// rare field like container counts as much as title).
func (s *LibraryScore) Overall() float64 {
	sum := 0.0
	for _, f := range ScoredFields {
		sum += s.Fields[f].Accuracy()
	}
	return sum / float64(len(ScoredFields))
}

// fieldValues flattens a Canonical into comparable per-field strings;
// seasons/episodes are compared as int slices separately.
func fieldEqual(field string, gold, got Canonical) (goldHas, gotHas, equal bool) {
	switch field {
	case "seasons":
		return len(gold.Seasons) > 0, len(got.Seasons) > 0, sameInts(gold.Seasons, got.Seasons)
	case "episodes":
		return len(gold.Episodes) > 0, len(got.Episodes) > 0, sameInts(gold.Episodes, got.Episodes)
	}
	var g, o string
	switch field {
	case "title":
		g, o = normTitle(gold.Title), normTitle(got.Title)
	case "year":
		g, o = gold.Year, got.Year
	case "resolution":
		g, o = gold.Resolution, got.Resolution
	case "source":
		g, o = gold.Source, got.Source
	case "codec":
		g, o = gold.Codec, got.Codec
	case "group":
		g, o = normToken(gold.Group), normToken(got.Group)
	case "container":
		g, o = gold.Container, got.Container
	}
	return g != "", o != "", g == o
}

// Score runs one library over the corpus.
func Score(lib *Library, corpus []GoldEntry) *LibraryScore {
	s := &LibraryScore{Library: lib, Fields: map[string]*FieldScore{}, Titles: len(corpus)}
	for _, f := range ScoredFields {
		s.Fields[f] = &FieldScore{}
	}
	for i := range corpus {
		gold := corpus[i].canonical()
		got, err := safeParse(lib, corpus[i].Title)
		if err != nil {
			s.Errors++
			got = Canonical{} // an errored parse extracted nothing
		}
		for _, f := range ScoredFields {
			fs := s.Fields[f]
			goldHas, gotHas, equal := fieldEqual(f, gold, got)
			switch {
			case goldHas && equal:
				fs.Correct++
			case goldHas && gotHas:
				fs.Wrong++
			case goldHas:
				fs.Missed++
			case gotHas:
				fs.Spurious++
			default:
				fs.Absent++
			}
		}
	}
	return s
}

// Disagreement records one title where a library diverged from gold, for
// spot-check dumps.
type Disagreement struct {
	Title       string
	Field       string
	Gold, Got   string
	GoldS, GotS []int
}

// Disagreements lists every non-correct outcome for one library and field.
func Disagreements(lib *Library, corpus []GoldEntry, field string) []Disagreement {
	var out []Disagreement
	for i := range corpus {
		gold := corpus[i].canonical()
		got, err := safeParse(lib, corpus[i].Title)
		if err != nil {
			got = Canonical{}
		}
		goldHas, gotHas, equal := fieldEqual(field, gold, got)
		if equal || (!goldHas && !gotHas) {
			continue
		}
		d := Disagreement{Title: corpus[i].Title, Field: field}
		switch field {
		case "seasons":
			d.GoldS, d.GotS = gold.Seasons, got.Seasons
		case "episodes":
			d.GoldS, d.GotS = gold.Episodes, got.Episodes
		default:
			_, _, _ = goldHas, gotHas, equal
			gv, ov := flat(field, gold), flat(field, got)
			d.Gold, d.Got = gv, ov
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func flat(field string, c Canonical) string {
	switch field {
	case "title":
		return c.Title
	case "year":
		return c.Year
	case "resolution":
		return c.Resolution
	case "source":
		return c.Source
	case "codec":
		return c.Codec
	case "group":
		return c.Group
	case "container":
		return c.Container
	}
	return ""
}
