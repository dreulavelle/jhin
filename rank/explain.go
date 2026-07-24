package rank

// Explain: the per-clause breakdown of a score. Every point in a Torrent's
// Rank traces to a contribution here, so a surprising result is always
// answerable from the profile.

import "fmt"

// Contribution is one scored component of a release's rank.
type Contribution struct {
	// Source identifies the clause: "attribute:remux",
	// "preferred_pattern", or "preferred_language".
	Source string `json:"source"`
	Rank   int    `json:"rank"`
}

// Explain returns the score breakdown for a parsed release under this
// ranker's profile. The contributions sum to the release's Rank; zero-valued
// attribute contributions are included so present-but-neutral attributes are
// visible.
func (r *Ranker) Explain(t *Torrent) []Contribution {
	attrs := attributes(t.Data)
	out := make([]Contribution, 0, len(attrs)+2)
	for _, a := range attrs {
		out = append(out, Contribution{
			Source: fmt.Sprintf("attribute:%s", a),
			Rank:   r.policy(a).Rank,
		})
	}
	if len(r.preferred) > 0 && matchAny(r.preferred, t.Raw) {
		out = append(out, Contribution{Source: "preferred_pattern", Rank: r.profile.Options.PreferredBonus})
	}
	if len(r.preferredLangs) > 0 && langOverlap(t.Data.Languages, r.preferredLangs) {
		out = append(out, Contribution{Source: "preferred_language", Rank: r.profile.Options.PreferredBonus})
	}
	return out
}
