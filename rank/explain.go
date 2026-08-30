package rank

// Explain: the per-clause breakdown of a score. Every point in a Torrent's
// Rank traces to a contribution here, so a surprising result is always
// answerable from the profile.

import (
	"fmt"

	"github.com/dreulavelle/jhin/rules"
)

// Contribution is one scored component of a release's rank.
type Contribution struct {
	// Source identifies the clause: "attribute:remux",
	// "preferred_pattern", "preferred_language", or "rule:<name>".
	Source string `json:"source"`
	Rank   int    `json:"rank"`
	// Detail carries a rule's score expression, so a breakdown can show the
	// working behind a number that was computed rather than looked up.
	Detail string `json:"detail,omitempty"`
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
	for _, pr := range r.patternRanks {
		if pr.re.MatchString(t.Raw) {
			out = append(out, Contribution{Source: "pattern:" + pr.re.String(), Rank: pr.rank})
		}
	}
	// Rules are reported from the evaluated release rather than re-run: a
	// rule may read facts only the caller has, so re-deriving them here would
	// give a different answer than the score actually used.
	for _, m := range t.RuleMatches {
		out = append(out, Contribution{Source: "rule:" + m.Name, Rank: m.Score, Detail: m.Source})
	}
	return out
}

// Skipped lists the rules that did not run for a release, and why. It is the
// half of a breakdown a static weight table never needed: a rule reading a
// fact the release does not carry is skipped rather than judged against zero,
// and a surprising result should say so rather than leave it to be inferred.
func (r *Ranker) Skipped(t *Torrent) []rules.Skip { return t.RuleSkipped }
