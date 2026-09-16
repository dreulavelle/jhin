package rules

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// Facts supplies one release's attribute values. The schema was fixed at
// compile time, so an implementation only has to answer for names it declared
// — back it with a struct, a map, a database row, or a lazily-filled cache.
type Facts interface {
	// Lookup returns the value at a declared path. Reporting false is the
	// same as the attribute being absent, and yields the type's zero.
	Lookup(path string) (Value, bool)
	// TierPresent reports whether this release carries anything in a tier.
	// Returning false skips every rule that reads it — see Engine.Evaluate.
	TierPresent(tier string) bool
}

// maxSteps bounds one expression evaluation. Nothing in the grammar loops, so
// this only ever fires on pathological nesting that slipped past maxDepth;
// it exists so that untrusted input cannot turn into unbounded work.
const maxSteps = 100_000

// errBudget is returned when an evaluation exceeds maxSteps. Like any other
// runtime failure it skips the rule rather than rejecting the release.
var errBudget = errors.New("expression did more work than a rule is allowed")

// unanswerable reports that an expression's value turns on something this
// release does not carry: a field in a tier it has none of, a function that
// reads one, or a result-set question nothing in the set could answer.
//
// It travels as an error so that every operator propagates it without a case
// of its own — a comparison, a call or a `not` over an unknown is unknown.
// Only `and` and `or` look at it, because those are the operators whose
// result the other operand can settle: `false and unknown` is false and
// `true or unknown` is true whatever the unknown would have been. A rule is
// therefore skipped exactly when the missing fact could have changed its
// outcome, which is the guarantee the tier machinery exists for.
type unanswerable struct{ reason string }

func (u *unanswerable) Error() string { return u.reason }

// isUnanswerable is a plain assertion rather than errors.As: nothing wraps
// the value on its way up, and errors.As would cost an allocation per call on
// the hottest path there is.
func isUnanswerable(err error) bool {
	_, ok := err.(*unanswerable)
	return ok
}

type evalState struct {
	facts Facts
	reg   *Registry
	kind  string
	aggs  *AggregateState
	hash  Value
	steps int
	// partial is set while evaluating an expression that reads a tier this
	// release does not carry, so that field reads check for absence. An
	// expression whose tiers are all present — the common case — skips the
	// check at every read, and the compile-time tier list is what says so.
	partial bool
	// tiers memoises Facts.TierPresent for this release: every field of a
	// namespace asks about the same tier, and a rule set asks about the same
	// few tiers over and over. An absent tier's entry holds the skip reason,
	// built once rather than at every read. It is a fixed array rather than
	// a slice into one so that the state stays off the heap; a registry with
	// more tiers than fit is answered without memoising.
	tiers  [4]tierState
	ntiers int
}

type tierState struct {
	name   string
	absent error
}

// use points the state at another release. ComputeAggregates walks a whole
// set with one state, so what was learned about the last release must not
// carry over.
func (e *evalState) use(facts Facts) {
	e.facts = facts
	e.ntiers = 0
}

// missing reports why a read of tier cannot be answered, or nil when the
// release carries it. The empty tier is always present.
func (e *evalState) missing(tier string) error {
	if tier == "" {
		return nil
	}
	for i := 0; i < e.ntiers; i++ {
		if e.tiers[i].name == tier {
			return e.tiers[i].absent
		}
	}
	var absent error
	if !e.facts.TierPresent(tier) {
		absent = &unanswerable{reason: tierReason(e.reg, tier)}
	}
	if e.ntiers < len(e.tiers) {
		e.tiers[e.ntiers] = tierState{name: tier, absent: absent}
		e.ntiers++
	}
	return absent
}

// begin readies the state for an expression that reads tiers: field reads
// check for absence only when at least one of them is missing.
func (e *evalState) begin(tiers []string) {
	e.steps = 0
	e.partial = false
	for _, t := range tiers {
		if e.missing(t) != nil {
			e.partial = true
			return
		}
	}
}

// tierReason words an absent tier for the Skipped report, in the terms the
// application declared it with.
func tierReason(reg *Registry, tier string) string {
	if reg != nil {
		if d := reg.tiers[tier]; d != "" {
			return "needs " + d
		}
	}
	return "needs " + tier + " data, which this release has none of"
}

func (e *evalState) funcOf(name string) *Func {
	if e.reg == nil {
		return nil
	}
	return e.reg.funcs[name]
}

func (e *evalState) tick() error {
	e.steps++
	if e.steps > maxSteps {
		return errBudget
	}
	return nil
}

func eval(n node, st *evalState) (Value, error) {
	if err := st.tick(); err != nil {
		return Value{}, err
	}
	switch t := n.(type) {
	case *litNode:
		return t.v, nil

	case *hashNode:
		return st.hash, nil

	case *fieldNode:
		if st.partial {
			if err := st.missing(t.tier); err != nil {
				return Value{}, err
			}
		}
		if v, ok := st.facts.Lookup(t.path); ok && v.Kind() == t.typ.K {
			return v, nil
		}
		return zero(t.typ), nil

	case *aggNode:
		return st.aggs.value(t.idx, t.form)

	case *scopedNode:
		// the referenced rule's scope travels with its condition
		if !scopeAllows(t.scope, st.kind) {
			return BoolOf(false), nil
		}
		return eval(t.x, st)

	case *listNode:
		items := make([]Value, len(t.items))
		for i, it := range t.items {
			v, err := eval(it, st)
			if err != nil {
				return Value{}, err
			}
			items[i] = v
		}
		return ListOf(t.elem, items...), nil

	case *unaryNode:
		v, err := eval(t.x, st)
		if err != nil {
			return Value{}, err
		}
		if t.op == "not" {
			return BoolOf(!v.Bool()), nil
		}
		return NumOf(-v.Num()), nil

	case *ternaryNode:
		c, err := eval(t.cond, st)
		if err != nil {
			return Value{}, err
		}
		if c.Bool() {
			return eval(t.then, st)
		}
		return eval(t.els, st)

	case *binaryNode:
		return evalBinary(t, st)

	case *callNode:
		return evalCall(t, st)
	}
	return Value{}, fmt.Errorf("cannot evaluate this expression")
}

func evalBinary(t *binaryNode, st *evalState) (Value, error) {
	// and/or short-circuit, which is what lets a rule guard a division or a
	// lookup with the test that makes it safe.
	//
	// An operand that could not be answered does not decide on its own: the
	// other side is tried, and settles the result when it can — `false and
	// unknown` is false, `true or unknown` is true. When it cannot, the
	// operator is unknown and the report names the first thing that was
	// missing. A right side that fails outright while the left is unknown is
	// unknown too: had the guard been false the failure would never have
	// been reached, and had it been true the rule would be skipped anyway.
	switch t.op {
	case "and":
		l, err := eval(t.l, st)
		if err != nil {
			if !isUnanswerable(err) {
				return Value{}, err
			}
			if r, rerr := eval(t.r, st); rerr == nil && !r.Bool() {
				return BoolOf(false), nil
			}
			return Value{}, err
		}
		if !l.Bool() {
			return BoolOf(false), nil
		}
		r, err := eval(t.r, st)
		return BoolOf(r.Bool()), err
	case "or":
		l, err := eval(t.l, st)
		if err != nil {
			if !isUnanswerable(err) {
				return Value{}, err
			}
			if r, rerr := eval(t.r, st); rerr == nil && r.Bool() {
				return BoolOf(true), nil
			}
			return Value{}, err
		}
		if l.Bool() {
			return BoolOf(true), nil
		}
		r, err := eval(t.r, st)
		return BoolOf(r.Bool()), err
	}

	l, err := eval(t.l, st)
	if err != nil {
		return Value{}, err
	}
	r, err := eval(t.r, st)
	if err != nil {
		return Value{}, err
	}

	switch t.op {
	case "==":
		return BoolOf(l.equals(r)), nil
	case "!=":
		return BoolOf(!l.equals(r)), nil
	case "<", "<=", ">", ">=":
		return compare(t.op, l, r), nil
	case "in", "not in":
		found := contains(l, r)
		if t.op == "not in" {
			return BoolOf(!found), nil
		}
		return BoolOf(found), nil
	case "matches":
		return BoolOf(t.re.re.MatchString(l.Str())), nil
	case "contains":
		return BoolOf(strings.Contains(l.Str(), r.Str())), nil
	case "startsWith":
		return BoolOf(strings.HasPrefix(l.Str(), r.Str())), nil
	case "endsWith":
		return BoolOf(strings.HasSuffix(l.Str(), r.Str())), nil
	case "+":
		if l.Kind() == KStr {
			return StrOf(l.Str() + r.Str()), nil
		}
		return NumOf(l.Num() + r.Num()), nil
	case "-":
		return NumOf(l.Num() - r.Num()), nil
	case "*":
		return NumOf(l.Num() * r.Num()), nil
	case "/":
		if r.Num() == 0 {
			return Value{}, fmt.Errorf("dividing by zero")
		}
		return NumOf(l.Num() / r.Num()), nil
	case "%":
		if r.Num() == 0 {
			return Value{}, fmt.Errorf("taking a remainder by zero")
		}
		return NumOf(math.Mod(l.Num(), r.Num())), nil
	}
	return Value{}, fmt.Errorf("unknown operator %q", t.op)
}

func compare(op string, l, r Value) Value {
	var lt, eq bool
	if l.Kind() == KStr {
		lt, eq = l.Str() < r.Str(), l.Str() == r.Str()
	} else {
		lt, eq = l.Num() < r.Num(), l.Num() == r.Num()
	}
	switch op {
	case "<":
		return BoolOf(lt)
	case "<=":
		return BoolOf(lt || eq)
	case ">":
		return BoolOf(!lt && !eq)
	default:
		return BoolOf(!lt)
	}
}

func contains(needle, hay Value) bool {
	if hay.Kind() == KStr {
		return strings.Contains(hay.Str(), needle.Str())
	}
	for _, e := range hay.List() {
		if e.equals(needle) {
			return true
		}
	}
	return false
}

func evalCall(t *callNode, st *evalState) (Value, error) {
	if t.name == aggResolved {
		return eval(t.args[0], st)
	}
	if b, ok := builtins[t.name]; ok {
		if b.predicate {
			return evalPredicate(t, st)
		}
		args := make([]Value, len(t.args))
		for i, a := range t.args {
			v, err := eval(a, st)
			if err != nil {
				return Value{}, err
			}
			args[i] = v
		}
		return b.fn(args)
	}
	if fn := st.funcOf(t.name); fn != nil {
		if st.partial {
			if err := st.missing(fn.Tier); err != nil {
				return Value{}, err
			}
		}
		args := make([]Value, len(t.args))
		for i, a := range t.args {
			v, err := eval(a, st)
			if err != nil {
				return Value{}, err
			}
			args[i] = v
		}
		return fn.Fn(st.facts, args)
	}
	return Value{}, fmt.Errorf("unknown function %q", t.name)
}

// evalPredicate runs a collection form's body once per element, restoring the
// outer # afterwards so a predicate nested inside another expression is safe
// even though the checker forbids nesting predicates in each other.
func evalPredicate(t *callNode, st *evalState) (Value, error) {
	list, err := eval(t.args[0], st)
	if err != nil {
		return Value{}, err
	}
	saved := st.hash
	defer func() { st.hash = saved }()

	n := 0
	for _, e := range list.List() {
		if err := st.tick(); err != nil {
			return Value{}, err
		}
		st.hash = e
		v, err := eval(t.args[1], st)
		if err != nil {
			return Value{}, err
		}
		if v.Bool() {
			n++
			// any/none only need to know whether one matched
			if t.name == "any" {
				return BoolOf(true), nil
			}
			if t.name == "none" {
				return BoolOf(false), nil
			}
		} else if t.name == "all" {
			return BoolOf(false), nil
		}
	}
	switch t.name {
	case "count":
		return NumOf(n), nil
	case "any":
		return BoolOf(false), nil
	case "none":
		return BoolOf(true), nil
	default: // all
		return BoolOf(true), nil
	}
}
