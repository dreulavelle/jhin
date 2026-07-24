// Package benchmarks is a standalone module comparing jhin against other
// torrent release-name parsers on speed and accuracy. It is not part of the
// jhin library — competitor dependencies live only here.
package benchmarks

import (
	"encoding/json"
	"fmt"
	"os"
)

// GoldEntry is one labeled title from parser/testdata/golden.json. Only the
// fields every benchmarked library claims to extract are decoded; the corpus
// carries many more (hdr, channels, editions, flags) that are compared in the
// feature matrix instead.
type GoldEntry struct {
	Title  string     `json:"title"`
	Result goldResult `json:"result"`
}

type goldResult struct {
	Title      string          `json:"title"`
	Year       json.RawMessage `json:"year"` // int, or a string range like "1980-1984"
	Seasons    []int           `json:"seasons"`
	Episodes   []int           `json:"episodes"`
	Resolution string          `json:"resolution"`
	Quality    string          `json:"quality"`
	Codec      string          `json:"codec"`
	Group      string          `json:"group"`
	Container  string          `json:"container"`
}

// yearString renders the gold year as a string ("2000", "1980-1984", or "").
func (g *goldResult) yearString() string {
	if len(g.Year) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(g.Year, &s); err == nil {
		return s
	}
	var n int
	if err := json.Unmarshal(g.Year, &n); err == nil {
		return fmt.Sprintf("%d", n)
	}
	return ""
}

// LoadCorpus reads the golden corpus relative to this module.
func LoadCorpus(path string) ([]GoldEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []GoldEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (g *GoldEntry) canonical() Canonical {
	return Canonical{
		Title:      g.Result.Title,
		Year:       g.Result.yearString(),
		Seasons:    g.Result.Seasons,
		Episodes:   g.Result.Episodes,
		Resolution: normResolution(g.Result.Resolution),
		Source:     normSource(g.Result.Quality),
		Codec:      normCodec(g.Result.Codec),
		Group:      g.Result.Group,
		Container:  normToken(g.Result.Container),
	}
}
