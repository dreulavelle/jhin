package rank

// Sorting: explicit, order-in/order-out. Evaluation never reorders — Sort
// returns a new slice ranked best-first.

import "sort"

// SortOptions tune Sort.
type SortOptions struct {
	// Resolutions, when non-empty, keeps only these buckets.
	Resolutions []Resolution
	// BucketLimit caps how many releases each resolution bucket contributes
	// (0 = unlimited).
	BucketLimit int
	// FetchableOnly drops releases that failed filtering.
	FetchableOnly bool
}

// Sort returns a new slice ordered by resolution bucket (highest first),
// then rank (highest first). The sort is stable: equal releases keep their
// input order.
func Sort(torrents []Torrent, opts ...SortOptions) []Torrent {
	var opt SortOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	keep := make([]Torrent, 0, len(torrents))
	var only map[Resolution]bool
	if len(opt.Resolutions) > 0 {
		only = make(map[Resolution]bool, len(opt.Resolutions))
		for _, r := range opt.Resolutions {
			only[r] = true
		}
	}
	for _, t := range torrents {
		if opt.FetchableOnly && !t.Fetch {
			continue
		}
		if only != nil && !only[t.Resolution()] {
			continue
		}
		keep = append(keep, t)
	}

	sort.SliceStable(keep, func(i, j int) bool {
		bi, bj := resolution_bucket[keep[i].Resolution()], resolution_bucket[keep[j].Resolution()]
		if bi != bj {
			return bi > bj
		}
		return keep[i].Rank > keep[j].Rank
	})

	if opt.BucketLimit > 0 {
		limited := keep[:0]
		counts := map[Resolution]int{}
		for _, t := range keep {
			res := t.Resolution()
			if counts[res] < opt.BucketLimit {
				limited = append(limited, t)
				counts[res]++
			}
		}
		keep = limited
	}
	return keep
}
