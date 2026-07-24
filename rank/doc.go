// Package rank scores, filters, and sorts torrent releases.
//
// A Profile declares everything tunable: per-attribute policies (fetch
// yes/no + score), require/exclude/preferred regex patterns, resolution
// gates, language rules, and global options. Compile it once into a Ranker
// and evaluate any number of releases concurrently:
//
//	ranker, err := rank.New(rank.Default())
//	torrents := ranker.RankAll(titles) // index-aligned with the input
//
//	best := rank.Sort(torrents, rank.SortOptions{
//		FetchableOnly: true,
//		BucketLimit:   5,
//	})
//
// Evaluation never reorders: RankAll's result is index-aligned with its
// input, each entry annotated with the parse result, score, fetch verdict,
// and rejection reasons. Sort is a separate, explicit step.
//
// To pin releases to a specific piece of media, pass a target title:
//
//	t := ranker.Rank(raw, rank.RankOptions{
//		TargetTitle: "The Matrix",
//		Aliases:     []string{"Matrix"},
//	})
//	// t.TitleRatio holds the similarity; below Options.TitleThreshold the
//	// release is rejected with "title_mismatch".
//
// Helpers rank.TitleMatch, rank.Similarity, and rank.Normalize are exported
// for standalone use.
package rank
