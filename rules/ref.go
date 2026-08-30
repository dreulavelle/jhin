package rules

import (
	"fmt"
	"strings"
)

// A rule reference — matched("Movies UHD BluRay T1") — reuses the
// classification another rule already expresses, so a tier list of release
// groups is written once and referred to from everywhere else that cares.
//
// References are resolved at compile time by inlining the referenced rule's
// condition, which is why one works wherever a condition does: on its own,
// inside a result-set question, inside a limit's grouping. Nothing is
// evaluated twice and rule order still does not matter.
//
// What a reference asks is whether the other rule's *condition* holds, not
// what its action did. For a score or reject rule those are the same thing;
// on a cap it means "counts against it", which is the only part of a cap
// knowable before the final ordering exists.

// maxExpansion caps how large a condition may grow once its references are
// inlined. Cycles are refused outright, but a reference is a copy rather than
// a call, so a chain where each rule names the one below it twice doubles at
// every step. The cap turns a set that would otherwise hang compilation into
// an error naming the rule.
const maxExpansion = 20_000

type refExpander struct {
	// byName maps a rule name to its definition. Library entries are added
	// first so a set's own rule under the same name shadows them.
	byName map[string]Rule
	// order records declaration order for a stable duplicate report.
	dupes map[string]bool
}

// newRefs builds an expander over a library and a rule set. References
// resolve against every rule, disabled ones included, so switching a rule off
// changes what a reference to it means rather than breaking the rules that
// name it.
func newRefs(library, ruleSet []Rule) *refExpander {
	r := &refExpander{byName: map[string]Rule{}, dupes: map[string]bool{}}
	for _, group := range [][]Rule{library, ruleSet} {
		seen := map[string]bool{}
		for _, rc := range group {
			name := strings.TrimSpace(rc.Name)
			if name == "" {
				continue
			}
			if seen[name] {
				r.dupes[name] = true
			}
			seen[name] = true
			r.byName[name] = rc
		}
	}
	return r
}

// expand replaces every matched() call in n with the referenced condition.
// stack carries the chain of names currently being inlined, for cycle
// detection and for an error that names the loop.
func (x *refExpander) expand(n node, self string, stack []string) (node, error) {
	var walkErr error
	out, err := x.rewrite(n, self, stack)
	if err != nil {
		return nil, err
	}
	if walkErr != nil {
		return nil, walkErr
	}
	if size(out) > maxExpansion {
		return nil, fmt.Errorf("condition grew past %d nodes once its references were included; a reference is a copy, so a chain that names the same rules repeatedly doubles at each step", maxExpansion)
	}
	return out, nil
}

func (x *refExpander) rewrite(n node, self string, stack []string) (node, error) {
	switch t := n.(type) {
	case nil:
		return nil, nil

	case *callNode:
		if t.name == matchedIdent {
			return x.inline(t, self, stack)
		}
		for i, a := range t.args {
			r, err := x.rewrite(a, self, stack)
			if err != nil {
				return nil, err
			}
			t.args[i] = r
		}
		return t, nil

	case *listNode:
		for i, it := range t.items {
			r, err := x.rewrite(it, self, stack)
			if err != nil {
				return nil, err
			}
			t.items[i] = r
		}
		return t, nil

	case *unaryNode:
		r, err := x.rewrite(t.x, self, stack)
		if err != nil {
			return nil, err
		}
		t.x = r
		return t, nil

	case *binaryNode:
		l, err := x.rewrite(t.l, self, stack)
		if err != nil {
			return nil, err
		}
		r, err := x.rewrite(t.r, self, stack)
		if err != nil {
			return nil, err
		}
		t.l, t.r = l, r
		return t, nil

	case *ternaryNode:
		c, err := x.rewrite(t.cond, self, stack)
		if err != nil {
			return nil, err
		}
		a, err := x.rewrite(t.then, self, stack)
		if err != nil {
			return nil, err
		}
		b, err := x.rewrite(t.els, self, stack)
		if err != nil {
			return nil, err
		}
		t.cond, t.then, t.els = c, a, b
		return t, nil

	case *scopedNode:
		r, err := x.rewrite(t.x, self, stack)
		if err != nil {
			return nil, err
		}
		t.x = r
		return t, nil
	}
	return n, nil
}

func (x *refExpander) inline(t *callNode, self string, stack []string) (node, error) {
	if len(t.args) != 1 {
		return nil, fmt.Errorf("matched() names one rule (at %d)", t.p)
	}
	lit, ok := t.args[0].(*litNode)
	if !ok || lit.v.Kind() != KStr {
		return nil, fmt.Errorf("matched() needs the rule's name written out (at %d)", t.p)
	}
	name := strings.TrimSpace(lit.v.Str())
	if x.dupes[name] {
		return nil, fmt.Errorf("matched(%q) is ambiguous: two rules share that name", name)
	}
	target, ok := x.byName[name]
	if !ok {
		return nil, fmt.Errorf("matched(%q) names no rule", name)
	}
	// A rule that is switched off classifies nothing, so a reference to it is
	// never true — and its condition is never looked at, which keeps a broken
	// rule you have turned off from blocking a save.
	if !target.IsEnabled() {
		return &litNode{v: BoolOf(false), p: t.p}, nil
	}
	for _, s := range append(stack, self) {
		if s == name {
			return nil, fmt.Errorf("matched(%q) closes a loop: %s", name, strings.Join(append(append([]string{}, stack...), self, name), " → "))
		}
	}
	inner, err := parse(target.When)
	if err != nil {
		return nil, fmt.Errorf("matched(%q): %w", name, err)
	}
	inner, err = x.expand(inner, name, append(append([]string{}, stack...), self))
	if err != nil {
		return nil, err
	}
	// The referenced rule's scope comes with its condition: a reference to a
	// movie-only rule is false for a series, whatever scope the referring
	// rule has.
	if sc := scopeSet(target.Scope); len(sc) > 0 {
		return &scopedNode{scope: sc, x: inner, p: t.p}, nil
	}
	return clone(inner), nil
}
