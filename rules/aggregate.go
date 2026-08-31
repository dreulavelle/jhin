package rules

import (
	"fmt"
	"sort"
)

// A result-set question — count(...), exists(...), any(...) or none(...) —
// judges the whole set where the rest of a rule judges one release. The inner
// condition is lifted out at compile time and evaluated once per request
// against every release; the rule itself is rewritten to read the precomputed
// value, so evaluation stays a single pass whatever the rule says about the
// set.
//
// Because the counts are taken before any rule fires, a rule that rejects can
// never change what another rule counted — so rule order does not matter.

type aggregate struct {
	program node
	// tiers are the tiers the inner condition reads. A release missing one
	// cannot be judged and does not count; a set where no release carries
	// them leaves the aggregate unknown.
	tiers []string
	// scope, when set, comes from an inlined reference inside the condition.
	source string
}

// AggregateState is one result set's computed values, shared by every release
// in it.
type AggregateState struct {
	values []int
	// answered mirrors values: whether anyone in the set carried the tiers
	// the condition reads, so a fresh search skips rather than counting zero.
	answered []bool
}

// value reads a computed answer. A caller that skipped the known check gets
// an error rather than a silent zero.
func (st *AggregateState) value(idx int, form string) (Value, error) {
	if st == nil || idx >= len(st.values) {
		return Value{}, fmt.Errorf("a result-set question was not computed for this set")
	}
	if !st.answered[idx] {
		return Value{}, fmt.Errorf("nothing in the result set could answer this question")
	}
	n := st.values[idx]
	switch form {
	case "count":
		return NumOf(n), nil
	case "none":
		return BoolOf(n == 0), nil
	default: // exists
		return BoolOf(n > 0), nil
	}
}

// known reports whether an aggregate could be judged at all.
func (st *AggregateState) known(idx int) bool {
	if st == nil || idx >= len(st.answered) {
		return false
	}
	return st.answered[idx]
}

// liftAggregate registers one inner condition, deduplicating by source text
// so two rules asking the same thing count it once.
//
// The condition arrives already checked — the checker types it once, in a
// context of its own, and hands the annotated tree straight here. Re-checking
// a copy would be wasted work, and storing an unchecked copy would leave its
// field references untyped, which reads as absent however present the
// attribute is.
func (e *Engine) liftAggregate(inner node, form string, tiers []string, aggIdx map[string]int) (int, error) {
	src := render(inner)
	if idx, ok := aggIdx[src]; ok {
		return idx, nil
	}
	idx := len(e.aggs)
	e.aggs = append(e.aggs, aggregate{program: inner, tiers: tiers, source: src})
	aggIdx[src] = idx
	return idx, nil
}

// HasAggregates reports whether any rule reads the result set, so a caller
// can skip the extra pass entirely.
func (e *Engine) HasAggregates() bool {
	return e != nil && len(e.aggs) > 0
}

// ComputeAggregates evaluates every result-set question over the whole set,
// once, before any rule runs.
//
// The set is whatever the caller passes — jhin's ranker passes the releases
// its baseline filters let through, so a rejected release is not "on offer"
// to a question like exists(...). A nil entry is skipped, which is how a
// caller keeps a slice index-aligned while leaving some releases out. The set
// is fixed before anything fires, so a viable 4K remux always has
// count(resolution == "2160p") >= 1.
//
// Fail-open extends here: a release missing a tier the inner condition reads
// is not counted, and when no release in the set carries that tier the
// question is unanswerable, so rules reading it are skipped rather than fed a
// zero. On a fresh search where nothing has been probed,
// none(probed.height >= 2000) must not read as "there is no good 4K".
//
// kind is the content kind the request is for, matched against the scope of
// any rule an inner condition references.
func (e *Engine) ComputeAggregates(set []Facts, kind string) *AggregateState {
	if e == nil || len(e.aggs) == 0 {
		return nil
	}
	st := &AggregateState{
		values:   make([]int, len(e.aggs)),
		answered: make([]bool, len(e.aggs)),
	}
	// The kind travels with the question: an inner condition may name a
	// scoped rule through matched(), and a whole result set is judged for one
	// request, so counting it against no kind at all would silently answer
	// false for every scoped reference.
	ev := &evalState{reg: e.reg, kind: kind}
	for i := range e.aggs {
		agg := &e.aggs[i]
		for _, facts := range set {
			if facts == nil || !tiersPresent(agg.tiers, facts) {
				continue
			}
			st.answered[i] = true
			ev.facts = facts
			ev.steps = 0
			v, err := eval(agg.program, ev)
			if err != nil {
				continue
			}
			if v.Bool() {
				st.values[i]++
			}
		}
	}
	return st
}

func tiersPresent(tiers []string, facts Facts) bool {
	for _, t := range tiers {
		if !facts.TierPresent(t) {
			return false
		}
	}
	return true
}

// AggregateReport is one computed result-set question, for a preview that
// wants to show its work rather than leave a rule's behaviour to be inferred.
type AggregateReport struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
	Known  bool   `json:"known"`
}

// Aggregates describes what ComputeAggregates worked out.
func (e *Engine) Aggregates(st *AggregateState) []AggregateReport {
	if e == nil || st == nil {
		return nil
	}
	out := make([]AggregateReport, 0, len(e.aggs))
	for i := range e.aggs {
		out = append(out, AggregateReport{Source: e.aggs[i].source, Count: st.values[i], Known: st.answered[i]})
	}
	return out
}

// ApplyLimits decides which releases survive the caps they matched.
//
// "At most three of these" is about the final score order — which three are
// best is only known after every point is in and the list is sorted, later
// than any condition runs. So a limit is an action, and this is where it is
// counted.
//
// perItem[i] is the caps release i matched; order lists indices best-first.
// The result is index-aligned with perItem: an empty string means the release
// survived, otherwise it is the rejection to record.
//
// Rules count independently. A release dropped by one cap does not take a slot
// in another: it is gone, so it is not competing.
func ApplyLimits(perItem [][]LimitMatch, order []int) []string {
	out := make([]string, len(perItem))
	if len(perItem) == 0 {
		return out
	}
	type bucket struct{ name, group string }
	seen := map[bucket]int{}

	for _, i := range order {
		if i < 0 || i >= len(perItem) {
			continue
		}
		// Every cap is tested before any is counted: a release one cap turns
		// away is gone, so it must not have consumed a slot in another — the
		// caps' declaration order must not decide who survives.
		rejected := false
		for _, lm := range perItem[i] {
			if seen[bucket{lm.Name, lm.Group}]+1 > lm.Count {
				out[i] = limitRejection(lm)
				rejected = true
				break
			}
		}
		if rejected {
			continue
		}
		for _, lm := range perItem[i] {
			seen[bucket{lm.Name, lm.Group}]++
		}
	}
	return out
}

func limitRejection(lm LimitMatch) string {
	// A cap that groups names the group when it turns a release away:
	// "over the limit of 3" on a rule that offered nine reads as a
	// contradiction on its own.
	if lm.Group != "" {
		return fmt.Sprintf("%s%s (over the limit of %d for %s)", RejectionPrefix, lm.Name, lm.Count, lm.Group)
	}
	return fmt.Sprintf("%s%s (over the limit of %d)", RejectionPrefix, lm.Name, lm.Count)
}

// AggregateSources lists the result-set questions this set asks, sorted. Two
// rules asking the same thing share one entry — which is the point.
func (e *Engine) AggregateSources() []string {
	if e == nil {
		return nil
	}
	out := make([]string, 0, len(e.aggs))
	for i := range e.aggs {
		out = append(out, e.aggs[i].source)
	}
	sort.Strings(out)
	return out
}
