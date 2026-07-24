package rank

// Sorting: explicit, order-in/order-out. Evaluation never reorders — Sort
// returns a new slice ranked best-first. The default ordering is resolution
// bucket (highest first) then rank; SortOptions.Criteria composes custom
// chains, and a Profile.ResolutionOrder redefines what "highest" means when
// sorting through Ranker.Sort.

import "sort"

// SortKey selects a comparison dimension for one sort criterion.
type SortKey string

const (
	// SortByResolution orders by resolution precedence (see
	// Profile.ResolutionOrder; defaults to highest-first).
	SortByResolution SortKey = "resolution"
	// SortByRank orders by the computed score.
	SortByRank SortKey = "rank"
	// SortByTitleRatio orders by similarity to the target title.
	SortByTitleRatio SortKey = "title_ratio"
)

// SortCriterion is one link in a sort chain. Descending order (best first)
// is the default; set Ascending for the reverse.
type SortCriterion struct {
	Key       SortKey `json:"key"`
	Ascending bool    `json:"ascending,omitempty"`
}

// defaultCriteria is the classic ordering: resolution bucket, then rank.
var defaultCriteria = []SortCriterion{{Key: SortByResolution}, {Key: SortByRank}}

// SortOptions tune Sort.
type SortOptions struct {
	// Criteria is the sort chain; empty means resolution-then-rank.
	Criteria []SortCriterion
	// Resolutions, when non-empty, keeps only these buckets.
	Resolutions []Resolution
	// BucketLimit caps how many releases each resolution bucket contributes
	// (0 = unlimited).
	BucketLimit int
	// FetchableOnly drops releases that failed filtering.
	FetchableOnly bool
}

// resolutionPrecedence builds the precedence table for sorting: listed
// resolutions rank ahead of everything else in their given order, and
// unlisted ones keep their default relative order below them. A nil order
// yields the default highest-first table.
func resolutionPrecedence(order []Resolution) map[Resolution]int {
	out := make(map[Resolution]int, len(resolutionBucket))
	for res, bucket := range resolutionBucket {
		out[res] = bucket // defaults: 1..9
	}
	for i, res := range order {
		out[res] = 1000 - i // listed: always above unlisted
	}
	return out
}

var defaultPrecedence = resolutionPrecedence(nil)

// Sort returns a new slice ordered best-first using the default resolution
// precedence. The sort is stable: equal releases keep their input order.
func Sort(torrents []Torrent, opts ...SortOptions) []Torrent {
	var opt SortOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	return sortWith(torrents, opt, defaultPrecedence)
}

// Sort orders releases under this ranker's profile, honoring
// Profile.ResolutionOrder.
func (r *Ranker) Sort(torrents []Torrent, opts ...SortOptions) []Torrent {
	var opt SortOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	return sortWith(torrents, opt, r.resPrecedence)
}

func sortWith(torrents []Torrent, opt SortOptions, precedence map[Resolution]int) []Torrent {
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

	criteria := opt.Criteria
	if len(criteria) == 0 {
		criteria = defaultCriteria
	}

	sort.SliceStable(keep, func(i, j int) bool {
		for _, c := range criteria {
			var vi, vj float64
			switch c.Key {
			case SortByResolution:
				vi = float64(precedence[keep[i].Resolution()])
				vj = float64(precedence[keep[j].Resolution()])
			case SortByRank:
				vi, vj = float64(keep[i].Rank), float64(keep[j].Rank)
			case SortByTitleRatio:
				vi, vj = keep[i].TitleRatio, keep[j].TitleRatio
			default:
				continue
			}
			if vi == vj {
				continue
			}
			if c.Ascending {
				return vi < vj
			}
			return vi > vj
		}
		return false
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
