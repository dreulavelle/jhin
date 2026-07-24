// Package jhin is an all-in-one library for working with torrent release
// names: parsing metadata out of raw titles, and (coming soon) ranking,
// filtering, and sorting torrents.
//
// The root package is a thin facade over the subpackages so that the common
// case stays a one-liner:
//
//	result := jhin.Parse("Deadpool 2016 1080p BluRay x264 DTS-JYK")
//
// For advanced use, import the subpackages directly:
//
//   - github.com/dreulavelle/jhin/parser — the title parsing engine
//   - github.com/dreulavelle/jhin/rank — torrent ranking (planned)
//   - github.com/dreulavelle/jhin/filter — torrent filtering (planned)
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
