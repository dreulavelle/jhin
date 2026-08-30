package rules

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Built-in actions. An application adds its own with Registry.Effect.
const (
	// ActionScore adds Score's value to the release's points.
	ActionScore = "score"
	// ActionReject removes the release, naming this rule.
	ActionReject = "reject"
	// ActionLimit caps how many matching releases survive. Which ones is only
	// knowable after every point is in and the list is ordered, so the match
	// is recorded here and counted by ApplyLimits.
	ActionLimit = "limit"
	// ActionDefine does nothing. The rule exists to be named by matched().
	ActionDefine = "define"
)

var reservedActions = map[string]bool{
	ActionScore: true, ActionReject: true, ActionLimit: true, ActionDefine: true,
}

// ScopeAll is the scope of a rule that applies to every content kind.
const ScopeAll = "all"

// SyntaxVersion is the rule language this build understands.
//
// A profile records the version it was written against so that a file from a
// newer jhin is refused with an explanation rather than half-understood: a
// condition using syntax this build has never heard of would otherwise fail
// as an unknown attribute, which sends the reader looking in the wrong place.
// Zero means a profile written before the field existed, which is version 1.
const SyntaxVersion = 1

// Rule is one named condition and what happens when it holds. It is plain
// data: JSON in a profile, a row in an editor, a line in the text form.
type Rule struct {
	// Name identifies the rule in the outcome and to matched().
	Name string `json:"name"`
	// When is the condition, which must evaluate to yes or no.
	When string `json:"when"`
	// Action is what happens when When holds. Empty means score.
	Action string `json:"action,omitempty"`
	// Score is what a matching release earns, as an expression — so points
	// can be computed from the release rather than fixed. Empty means zero.
	// For an application-registered effect this carries the effect's value.
	Score string `json:"score,omitempty"`
	// Count is how many matching releases survive, for a limit rule.
	Count int `json:"count,omitempty"`
	// GroupBy splits a limit rule's cap into one bucket per value, so Count
	// is kept per bucket rather than across the whole set.
	GroupBy string `json:"group_by,omitempty"`
	// Scope limits the rule to certain content kinds. Empty applies to all.
	Scope []string `json:"scope,omitempty"`
	// Enabled turns a rule off without deleting it. Nil means enabled, so a
	// rule written before the flag existed keeps working.
	Enabled *bool `json:"enabled,omitempty"`
}

// IsEnabled reports whether the rule takes part.
func (r Rule) IsEnabled() bool { return r.Enabled == nil || *r.Enabled }

// EffectiveAction is the rule's action with the default applied.
func (r Rule) EffectiveAction() string {
	a := strings.ToLower(strings.TrimSpace(r.Action))
	if a == "" {
		return ActionScore
	}
	return a
}

// Error is a rule that would not compile, named so an editor can point at the
// row rather than at the profile.
type Error struct {
	Rule string
	Err  error
}

func (e *Error) Error() string { return "rule " + e.Rule + ": " + e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Match is a rule that paid out.
type Match struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	// Source is the score expression, so a breakdown can show the working.
	Source string `json:"source,omitempty"`
}

// Skip is a rule that did not run, and why. It exists so a surprising result
// says "this rule needs a probed file" instead of leaving it to be inferred.
type Skip struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// LimitMatch is one cap a release counts against.
type LimitMatch struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	// Group is the bucket the release falls in, empty for an ungrouped cap.
	Group string `json:"group,omitempty"`
}

// Effect is an application-defined action that fired.
type Effect struct {
	Name  string `json:"name"`
	Value Value  `json:"value"`
}

// Outcome is what a rule set did to one release.
type Outcome struct {
	Points     int          `json:"points"`
	Matched    []Match      `json:"matched,omitempty"`
	Rejections []string     `json:"rejections,omitempty"`
	Skipped    []Skip       `json:"skipped,omitempty"`
	Limits     []LimitMatch `json:"limits,omitempty"`
	Effects    []Effect     `json:"effects,omitempty"`
}

type compiled struct {
	name    string
	action  string
	scope   map[string]bool
	when    node
	score   node
	scoreIx string
	count   int
	groupBy node
	tiers   []string
	aggs    []int
}

// Engine is a compiled rule set. It is immutable, so one Engine serves every
// goroutine evaluating a batch.
type Engine struct {
	reg   *Registry
	rules []compiled
	aggs  []aggregate
}

// Compile turns rules into an evaluable set, checked against the registry.
//
// Compilation is strict: every condition must type-check and must yield a
// yes/no. A rule that cannot compile fails the whole call, because a set that
// silently dropped one would filter differently than configured and give no
// sign of it. Disabled rules are dropped rather than checked, so a half-written
// rule that is switched off never blocks a save.
//
// library carries shared definitions that exist only to be referenced.
// matched() resolves against the set's own rules first — a rule under the same
// name shadows a library's — and library rules never join the set themselves.
func Compile(reg *Registry, ruleSet []Rule, library ...Rule) (*Engine, error) {
	if reg == nil {
		return nil, fmt.Errorf("no registry")
	}
	if err := reg.Err(); err != nil {
		return nil, err
	}
	if len(ruleSet) == 0 {
		return nil, nil
	}
	eng := &Engine{reg: reg.clone()}
	refs := newRefs(library, ruleSet)
	aggIdx := map[string]int{}

	for i := range ruleSet {
		rc := ruleSet[i]
		name := ruleLabel(rc, i)
		if !rc.IsEnabled() {
			continue
		}
		c, err := eng.compileRule(rc, name, refs, aggIdx)
		if err != nil {
			return nil, &Error{Rule: name, Err: err}
		}
		if c != nil {
			eng.rules = append(eng.rules, *c)
		}
	}
	if len(eng.rules) == 0 {
		return nil, nil
	}
	return eng, nil
}

func ruleLabel(r Rule, i int) string {
	if n := strings.TrimSpace(r.Name); n != "" {
		return n
	}
	return fmt.Sprintf("#%d", i+1)
}

func (e *Engine) compileRule(rc Rule, name string, refs *refExpander, aggIdx map[string]int) (*compiled, error) {
	action := rc.EffectiveAction()
	if !reservedActions[action] {
		if _, ok := e.reg.effects[action]; !ok {
			return nil, fmt.Errorf("unknown action %q", rc.Action)
		}
	}
	if strings.TrimSpace(rc.When) == "" {
		return nil, fmt.Errorf("condition is empty")
	}
	if action == ActionLimit && rc.Count < 1 {
		return nil, fmt.Errorf("a limit rule has to keep at least one release")
	}
	if action != ActionLimit && strings.TrimSpace(rc.GroupBy) != "" {
		return nil, fmt.Errorf("only a limit rule can group by %s", strings.TrimSpace(rc.GroupBy))
	}

	c := &compiled{name: name, action: action, count: rc.Count, scope: scopeSet(rc.Scope)}
	tiers := map[string]bool{}

	when, err := e.compileExpr(rc.When, refs, name, aggIdx, tiers, Bool, "condition")
	if err != nil {
		return nil, err
	}
	c.when = when

	// A define rule is validated as strictly as any other — a broken
	// definition must fail the save whether or not anything references it yet
	// — but it never joins the set: nothing is judged by it, so it pays
	// nothing out and appears nowhere.
	if action == ActionDefine {
		return nil, nil
	}

	switch {
	case action == ActionScore:
		if strings.TrimSpace(rc.Score) != "" {
			n, err := e.compileExpr(rc.Score, refs, name, aggIdx, tiers, Num, "score")
			if err != nil {
				return nil, err
			}
			c.score, c.scoreIx = n, strings.TrimSpace(rc.Score)
		}
	case action == ActionLimit:
		if gb := strings.TrimSpace(rc.GroupBy); gb != "" {
			// A grouping is not required to yield any particular type: what a
			// bucket needs is an identity, not a value, and every type has one
			// once it is written out.
			n, err := e.compileExpr(rc.GroupBy, refs, name, aggIdx, tiers, anyType, "group by")
			if err != nil {
				return nil, err
			}
			c.groupBy = n
		}
	case action != ActionReject:
		want := e.reg.effects[action]
		if strings.TrimSpace(rc.Score) == "" {
			return nil, fmt.Errorf("the %s action needs a value", action)
		}
		n, err := e.compileExpr(rc.Score, refs, name, aggIdx, tiers, want, "value")
		if err != nil {
			return nil, err
		}
		c.score, c.scoreIx = n, strings.TrimSpace(rc.Score)
	}

	c.tiers = sortedKeys(tiers)
	c.aggs = e.aggsFor(c.when, c.score, c.groupBy)
	return c, nil
}

// compileExpr parses, inlines references, lifts result-set questions and type
// checks one expression, accumulating the tiers it reads.
//
// Tiers are judged on the rewritten condition, so a rule asking whether the
// set holds a probed release does not itself need this release to be probed.
func (e *Engine) compileExpr(src string, refs *refExpander, self string, aggIdx map[string]int, tiers map[string]bool, want Type, what string) (node, error) {
	n, err := parse(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, locate(src, err))
	}
	if n, err = refs.expand(n, self, nil); err != nil {
		return nil, fmt.Errorf("%s: %w", what, locate(src, err))
	}

	ck := newChecker(e.reg)
	ck.lift = func(inner node, form string, tiers []string, _ int) (int, error) {
		return e.liftAggregate(inner, form, tiers, aggIdx)
	}
	got, err := ck.check(n)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, locate(src, err))
	}
	if want != anyType && !got.assignable(want) {
		return nil, fmt.Errorf("%s has to be %s, but it gives %s", what, want, got)
	}
	for t := range ck.tiers {
		tiers[t] = true
	}
	return n, nil
}

func (e *Engine) aggsFor(nodes ...node) []int {
	seen := map[int]bool{}
	var out []int
	for _, n := range nodes {
		walk(n, func(x node) {
			if a, ok := x.(*aggNode); ok && !seen[a.idx] {
				seen[a.idx] = true
				out = append(out, a.idx)
			}
		})
	}
	sort.Ints(out)
	return out
}

func scopeSet(scopes []string) map[string]bool {
	out := map[string]bool{}
	for _, s := range scopes {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || s == ScopeAll {
			return nil // applies everywhere
		}
		out[s] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func scopeAllows(scope map[string]bool, kind string) bool {
	if len(scope) == 0 {
		return true
	}
	return scope[strings.ToLower(strings.TrimSpace(kind))]
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Len reports how many acting rules the set holds.
func (e *Engine) Len() int {
	if e == nil {
		return 0
	}
	return len(e.rules)
}

// ReadsTier reports whether any rule reads a tier, so a caller can skip an
// expensive lookup — a probe, a network call — that nothing will ask about.
func (e *Engine) ReadsTier(tier string) bool {
	if e == nil {
		return false
	}
	for i := range e.rules {
		for _, t := range e.rules[i].tiers {
			if t == tier {
				return true
			}
		}
	}
	return false
}

// Evaluate runs the rules that apply to this content kind.
//
// A rule is skipped, not failed, when it reads a tier the release carries
// nothing in. Judging those against zero values would let a single rule like
// `probed.height < 1080` reject every release that was never probed, which is
// the opposite of what it asks. The same contract covers a runtime failure: a
// compiled, type-checked rule that fails on data no test covered skips, so an
// inconclusive check never removes a release.
//
// A set with result-set questions expects the caller to have computed them —
// ComputeAggregates over the whole set, then pass the state here.
func (e *Engine) Evaluate(facts Facts, kind string, aggs *AggregateState) Outcome {
	var out Outcome
	if e == nil {
		return out
	}
	st := &evalState{facts: facts, reg: e.reg, kind: kind, aggs: aggs}

	for i := range e.rules {
		r := &e.rules[i]
		if !scopeAllows(r.scope, kind) {
			continue
		}
		if reason, skip := e.skipReason(r, facts, aggs); skip {
			out.Skipped = append(out.Skipped, Skip{Name: r.name, Reason: reason})
			continue
		}
		st.steps = 0
		v, err := eval(r.when, st)
		if err != nil || !v.Bool() {
			if err != nil {
				out.Skipped = append(out.Skipped, Skip{Name: r.name, Reason: err.Error()})
			}
			continue
		}
		e.apply(r, st, &out)
	}
	return out
}

func (e *Engine) skipReason(r *compiled, facts Facts, aggs *AggregateState) (string, bool) {
	for _, tier := range r.tiers {
		if !facts.TierPresent(tier) {
			if d := e.reg.tiers[tier]; d != "" {
				return "needs " + d, true
			}
			return "needs " + tier + " data, which this release has none of", true
		}
	}
	for _, idx := range r.aggs {
		if !aggs.known(idx) {
			return "asks about the result set, and nothing in it could answer", true
		}
	}
	return "", false
}

func (e *Engine) apply(r *compiled, st *evalState, out *Outcome) {
	switch r.action {
	case ActionReject:
		out.Rejections = append(out.Rejections, RejectionPrefix+r.name)

	case ActionLimit:
		group, ok := groupOf(r, st)
		if !ok {
			// A grouping that fails at runtime cannot say which bucket this
			// release belongs in, and a cap that does not know that cannot
			// count it. Dropping the match rather than guessing keeps the
			// promise the tier checks make: a rule that cannot be judged
			// never removes a release.
			out.Skipped = append(out.Skipped, Skip{Name: r.name, Reason: "the grouping could not be worked out"})
			return
		}
		out.Limits = append(out.Limits, LimitMatch{Name: r.name, Count: r.count, Group: group})

	case ActionScore:
		points := 0
		if r.score != nil {
			st.steps = 0
			v, err := eval(r.score, st)
			if err != nil {
				out.Skipped = append(out.Skipped, Skip{Name: r.name, Reason: "the score could not be worked out: " + err.Error()})
				return
			}
			p, ok := clampPoints(v.Num())
			if !ok {
				out.Skipped = append(out.Skipped, Skip{Name: r.name, Reason: "the score did not come out as a number"})
				return
			}
			points = p
		}
		out.Points += points
		out.Matched = append(out.Matched, Match{Name: r.name, Score: points, Source: r.scoreIx})

	default: // an application-registered effect
		st.steps = 0
		v, err := eval(r.score, st)
		if err != nil {
			out.Skipped = append(out.Skipped, Skip{Name: r.name, Reason: "the value could not be worked out: " + err.Error()})
			return
		}
		out.Effects = append(out.Effects, Effect{Name: r.action, Value: v})
	}
}

// RejectionPrefix marks a rejection as coming from a rule, so it reads apart
// from the built-in ones in the same list.
const RejectionPrefix = "rule:"

// maxPoints bounds one rule's payout. Converting a float64 outside the
// integer's range is platform-defined in Go, and a payout near int's edge
// would let two rules overflow the total — a bound none of that can reach
// keeps score arithmetic exact however wild the expression.
const maxPoints = 1 << 40

// clampPoints turns a computed score into points. Infinities clamp — the
// author's direction survives — but NaN has no direction to keep, so it
// reports false and the rule is skipped.
func clampPoints(n float64) (int, bool) {
	if math.IsNaN(n) {
		return 0, false
	}
	if n > maxPoints {
		return maxPoints, true
	}
	if n < -maxPoints {
		return -maxPoints, true
	}
	return int(n), true
}

func groupOf(r *compiled, st *evalState) (string, bool) {
	if r.groupBy == nil {
		return "", true
	}
	st.steps = 0
	v, err := eval(r.groupBy, st)
	if err != nil {
		return "", false
	}
	return v.String(), true
}
