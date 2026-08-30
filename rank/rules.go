package rank

// Rules: the profile's expression layer. Everything derivable from a release
// name is already profile-configurable through Policy and the pattern lists;
// rules are how an application folds in what only it knows — size, age, grab
// count, what a probe measured — without any of it becoming Go.

import (
	"fmt"

	"github.com/dreulavelle/jhin/rules"
)

// Facts is the application's per-release data, as the rule engine sees it.
type Facts = rules.Facts

// WithRules attaches a compiled rule set to a Ranker.
//
// The registry the engine was compiled against must include rules.Core(),
// since the ranker always supplies those facts itself.
func WithRules(eng *rules.Engine) Option {
	return func(r *Ranker) { r.rules = eng }
}

// Option configures a Ranker at construction.
type Option func(*Ranker)

// CompileRules builds an engine for a profile's rules against a registry.
// Applications that register their own attributes pass their own registry;
// passing nil uses rules.Core() alone.
func (p Profile) CompileRules(reg *rules.Registry) (*rules.Engine, error) {
	if len(p.Rules) == 0 {
		return nil, nil
	}
	if reg == nil {
		reg = rules.Core()
	}
	eng, err := rules.Compile(reg, p.Rules, p.RuleLibrary...)
	if err != nil {
		return nil, fmt.Errorf("rules: %w", err)
	}
	return eng, nil
}

// coreFacts answers the built-in schema for one evaluated release. An
// application's own facts are layered over it — see Entry.Facts.
func coreFacts(t *Torrent) rules.Facts {
	attrs := attributes(t.Data)
	traits := make([]string, len(attrs))
	for i, a := range attrs {
		traits[i] = string(a)
	}
	return rules.FromResult(t.Raw, t.Data, traits)
}

// factsFor layers the application's facts over the ones jhin derives itself.
// The application's come first so it can override a parsed value with a
// measured one.
func factsFor(t *Torrent, app rules.Facts) rules.Facts {
	if app == nil {
		return coreFacts(t)
	}
	return rules.Layer(app, coreFacts(t))
}

// applyRules folds a rule outcome into an evaluated release.
//
// Points join the score, rejections join the rejection list, and caps are
// carried on the Torrent until Sort has put the set in final order — which is
// the only point at which "the best three of these" means anything.
func (r *Ranker) applyRules(t *Torrent, opt *RankOptions, aggs *rules.AggregateState) {
	if r.rules == nil {
		return
	}
	out := r.rules.Evaluate(factsFor(t, opt.factsFor(t)), opt.Kind, aggs)
	t.Rank += out.Points
	t.RuleMatches = out.Matched
	t.RuleSkipped = out.Skipped
	t.Limits = out.Limits
	t.Effects = out.Effects
	t.Rejections = append(t.Rejections, out.Rejections...)
	t.Fetch = len(t.Rejections) == 0
}

// ApplyLimits resolves the caps a batch matched, after sorting has decided
// which releases are best.
//
// Pass the sorted slice: a cap keeps the best Count by final score, so it is
// the order that decides, not the order the indexer returned. Releases dropped
// here have the cap recorded in Rejections and Fetch cleared.
func ApplyLimits(sorted []Torrent) {
	perItem := make([][]rules.LimitMatch, len(sorted))
	order := make([]int, 0, len(sorted))
	for i := range sorted {
		// A release something already rejected is gone, so it is not
		// competing: letting it hold a slot would cost the cap a release the
		// user could actually have had.
		if !sorted[i].Fetch {
			continue
		}
		perItem[i] = sorted[i].Limits
		order = append(order, i)
	}
	for i, reason := range rules.ApplyLimits(perItem, order) {
		if reason == "" {
			continue
		}
		sorted[i].Rejections = append(sorted[i].Rejections, reason)
		sorted[i].Fetch = false
	}
}

// runRules applies the rule set to a whole batch.
//
// It runs after every release is parsed and evaluated, because result-set
// questions are computed over the finished set — which is also what makes
// rule order irrelevant: nothing a rule does can change what another counted.
func (r *Ranker) runRules(batch []Torrent, opt *RankOptions) {
	if r.rules == nil || len(batch) == 0 {
		return
	}
	var aggs *rules.AggregateState
	if r.rules.HasAggregates() {
		set := make([]rules.Facts, len(batch))
		for i := range batch {
			set[i] = factsFor(&batch[i], opt.factsFor(&batch[i]))
		}
		aggs = r.rules.ComputeAggregates(set, opt.Kind)
	}
	for i := range batch {
		r.applyRules(&batch[i], opt, aggs)
	}
}
