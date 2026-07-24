// Package jhin is an all-in-one library for working with torrent release
// names: parsing metadata out of raw titles, then ranking, filtering, and
// sorting releases against a user profile.
//
// The root package is a thin facade over the subpackages so that the common
// case stays a one-liner:
//
//	result := jhin.Parse("Deadpool 2016 1080p BluRay x264 DTS-JYK")
//
// For ranking, filtering, and sorting, use the rank subpackage:
//
//	ranker, _ := rank.New(rank.Default())
//	torrents := ranker.RankAll(titles)          // index-aligned with input
//	best := rank.Sort(torrents, rank.SortOptions{FetchableOnly: true})
//
// Subpackages:
//
//   - github.com/dreulavelle/jhin/parser — the title parsing engine
//   - github.com/dreulavelle/jhin/rank — ranking, filtering, and sorting
package jhin

import "github.com/dreulavelle/jhin/parser"

// Result holds all metadata extracted from a torrent title.
// It is an alias of [parser.Result].
type Result = parser.Result

// Parse extracts metadata from a torrent title.
func Parse(title string) *Result {
	return parser.Parse(title)
}

// GetPartialParser returns a parse function that only runs the handlers for
// the given field names. Useful when only a few fields are needed and
// throughput matters.
func GetPartialParser(fieldNames []string) func(title string) *Result {
	return parser.GetPartialParser(fieldNames)
}
