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

type evalState struct {
	facts Facts
	reg   *Registry
	kind  string
	aggs  *AggregateState
	hash  Value
	steps int
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
	switch t.op {
	case "and":
		l, err := eval(t.l, st)
		if err != nil || !l.Bool() {
			return BoolOf(false), err
		}
		r, err := eval(t.r, st)
		return BoolOf(r.Bool()), err
	case "or":
		l, err := eval(t.l, st)
		if err != nil {
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
