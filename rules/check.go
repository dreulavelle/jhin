package rules

import (
	"fmt"
	"regexp"
	"strings"
)

// The type checker. It resolves every attribute name against the registry,
// types every operator, compiles every literal regex, and — as a byproduct of
// resolving names — records which confidence tiers the expression reads.
//
// That last part is why jhin owns the checker rather than borrowing one: a
// separate pass over the tree to discover tiers is a second place for the two
// to disagree. Here a tier is learned at the moment a name is resolved.

type checker struct {
	reg *Registry
	// tiers collects what this expression depends on.
	tiers map[string]bool
	// hashType is the element type of the list a collection predicate is
	// running over, or invalid outside one. Predicates do not nest.
	hashType Type
	inHash   bool
	// aggs receives lifted result-set conditions.
	lift func(inner node, form string, tiers []string, p int) (int, error)
}

func newChecker(reg *Registry) *checker {
	return &checker{reg: reg, tiers: map[string]bool{}}
}

// check types an expression and returns its type.
func (c *checker) check(n node) (Type, error) {
	switch t := n.(type) {
	case *litNode:
		return t.v.Type(), nil

	case *hashNode:
		if !c.inHash {
			return invalid, fmt.Errorf("# is only meaningful inside count, any, all or none over a list (at %d)", t.p)
		}
		return c.hashType, nil

	case *fieldNode:
		f, ok := c.reg.Lookup(t.path)
		if !ok {
			return invalid, fmt.Errorf("unknown attribute %q%s", t.path, c.suggest(t.path))
		}
		t.typ, t.tier = f.Type, f.Tier
		if f.Tier != "" {
			c.tiers[f.Tier] = true
		}
		return f.Type, nil

	case *aggNode:
		if t.form == "count" {
			return Num, nil
		}
		return Bool, nil

	case *scopedNode:
		return c.check(t.x)

	case *listNode:
		return c.checkList(t)

	case *unaryNode:
		return c.checkUnary(t)

	case *binaryNode:
		return c.checkBinary(t)

	case *ternaryNode:
		return c.checkTernary(t)

	case *callNode:
		return c.checkCall(t)
	}
	return invalid, fmt.Errorf("cannot type this expression")
}

// suggest offers the closest declared name, which turns the commonest rule
// error — a typo — from a puzzle into a fix.
func (c *checker) suggest(path string) string {
	// Only worth doing for something that could plausibly be a typo; the
	// search is over every declared field, so a very long name would cost
	// more to be helpful about than it is worth.
	if len(path) > 64 {
		return ""
	}
	best, bestD := "", 1<<30
	for _, cand := range c.reg.Fields() {
		if d := editDistance(strings.ToLower(path), strings.ToLower(cand)); d < bestD {
			best, bestD = cand, d
		}
	}
	// only offer a suggestion close enough to be plausible
	if best != "" && bestD <= 1+len(path)/4 {
		return fmt.Sprintf(" (did you mean %q?)", best)
	}
	return ""
}

func (c *checker) checkList(t *listNode) (Type, error) {
	if len(t.items) == 0 {
		return Type{K: KList}, nil
	}
	var elem Type
	for i, it := range t.items {
		et, err := c.check(it)
		if err != nil {
			return invalid, err
		}
		if et.K == KList {
			return invalid, fmt.Errorf("a list cannot hold lists (at %d)", it.pos())
		}
		if i == 0 {
			elem = et
			continue
		}
		if et != elem {
			return invalid, fmt.Errorf("list mixes %s and %s (at %d)", elem, et, it.pos())
		}
	}
	t.elem = elem.K
	return Type{K: KList, Elem: elem.K}, nil
}

func (c *checker) checkUnary(t *unaryNode) (Type, error) {
	xt, err := c.check(t.x)
	if err != nil {
		return invalid, err
	}
	switch t.op {
	case "not":
		if xt != Bool {
			return invalid, fmt.Errorf("not needs a yes/no value, got %s (at %d)", xt, t.p)
		}
		return Bool, nil
	case "-":
		if xt != Num {
			return invalid, fmt.Errorf("- needs a number, got %s (at %d)", xt, t.p)
		}
		return Num, nil
	}
	return invalid, fmt.Errorf("unknown operator %q", t.op)
}

func (c *checker) checkBinary(t *binaryNode) (Type, error) {
	lt, err := c.check(t.l)
	if err != nil {
		return invalid, err
	}
	rt, err := c.check(t.r)
	if err != nil {
		return invalid, err
	}

	switch t.op {
	case "and", "or":
		if lt != Bool || rt != Bool {
			return invalid, fmt.Errorf("%s joins yes/no values, got %s and %s (at %d)", t.op, lt, rt, t.p)
		}
		return Bool, nil

	case "==", "!=":
		if !lt.assignable(rt) && !rt.assignable(lt) {
			return invalid, fmt.Errorf("cannot compare %s with %s (at %d)", lt, rt, t.p)
		}
		return Bool, nil

	case "<", "<=", ">", ">=":
		if lt != rt || (lt != Num && lt != Str) {
			return invalid, fmt.Errorf("%s compares two numbers or two strings, got %s and %s (at %d)", t.op, lt, rt, t.p)
		}
		return Bool, nil

	case "in", "not in":
		switch {
		case rt.K == KList:
			if rt.Elem != KInvalid && lt.K != rt.Elem {
				return invalid, fmt.Errorf("cannot look for %s in %s (at %d)", lt, rt, t.p)
			}
			if lt.K == KList {
				return invalid, fmt.Errorf("cannot look for a list inside a list (at %d)", t.p)
			}
		case rt == Str:
			if lt != Str {
				return invalid, fmt.Errorf("looking inside text needs text, got %s (at %d)", lt, t.p)
			}
		default:
			return invalid, fmt.Errorf("%s needs a list or text on the right, got %s (at %d)", t.op, rt, t.p)
		}
		return Bool, nil

	case "matches":
		if lt != Str || rt != Str {
			return invalid, fmt.Errorf("matches compares text against a pattern, got %s and %s (at %d)", lt, rt, t.p)
		}
		lit, ok := t.r.(*litNode)
		if !ok {
			return invalid, fmt.Errorf("the pattern for matches has to be written out, so it can be compiled once (at %d)", t.p)
		}
		re, err := regexp.Compile(lit.v.Str())
		if err != nil {
			return invalid, fmt.Errorf("bad pattern %q: %w", lit.v.Str(), err)
		}
		t.re = &regexpMatcher{re: re, src: lit.v.Str()}
		return Bool, nil

	case "contains", "startsWith", "endsWith":
		if lt != Str || rt != Str {
			return invalid, fmt.Errorf("%s compares text with text, got %s and %s (at %d)", t.op, lt, rt, t.p)
		}
		return Bool, nil

	case "+":
		if lt == Str && rt == Str {
			return Str, nil
		}
		if lt == Num && rt == Num {
			return Num, nil
		}
		return invalid, fmt.Errorf("+ adds two numbers or joins two strings, got %s and %s (at %d)", lt, rt, t.p)

	case "-", "*", "/", "%":
		if lt != Num || rt != Num {
			return invalid, fmt.Errorf("%s needs two numbers, got %s and %s (at %d)", t.op, lt, rt, t.p)
		}
		if t.op == "/" || t.op == "%" {
			if lit, ok := t.r.(*litNode); ok && lit.v.Num() == 0 {
				return invalid, fmt.Errorf("dividing by zero (at %d)", t.p)
			}
		}
		return Num, nil
	}
	return invalid, fmt.Errorf("unknown operator %q (at %d)", t.op, t.p)
}

func (c *checker) checkTernary(t *ternaryNode) (Type, error) {
	ct, err := c.check(t.cond)
	if err != nil {
		return invalid, err
	}
	if ct != Bool {
		return invalid, fmt.Errorf("the test before ? has to be yes/no, got %s (at %d)", ct, t.p)
	}
	at, err := c.check(t.then)
	if err != nil {
		return invalid, err
	}
	bt, err := c.check(t.els)
	if err != nil {
		return invalid, err
	}
	if !at.assignable(bt) && !bt.assignable(at) {
		return invalid, fmt.Errorf("the two branches of ? : give %s and %s; they have to agree (at %d)", at, bt, t.p)
	}
	// An empty list literal has no element type of its own, so the branch that
	// does carries the answer. Every other type is its own answer.
	if at.K == KList && at.Elem == KInvalid {
		return bt, nil
	}
	return at, nil
}

func (c *checker) checkCall(t *callNode) (Type, error) {
	// A result-set question is lifted out before it is typed: the inner
	// condition becomes its own program and this call becomes an index.
	//
	// Arity decides, so nothing has to be type-checked twice to find out what
	// it is: count/exists/any/none over one argument ask about the set, and
	// the two-argument form is the collection predicate.
	if aggregateForms[t.name] && len(t.args) == 1 {
		return c.checkAggregate(t)
	}
	if b, ok := builtins[t.name]; ok {
		return c.checkBuiltin(t, b)
	}
	if fn, ok := c.reg.funcs[t.name]; ok {
		if len(t.args) != len(fn.Params) {
			return invalid, fmt.Errorf("%s takes %d argument(s), got %d (at %d)", t.name, len(fn.Params), len(t.args), t.p)
		}
		for i, a := range t.args {
			at, err := c.check(a)
			if err != nil {
				return invalid, err
			}
			if !at.assignable(fn.Params[i]) {
				return invalid, fmt.Errorf("%s argument %d wants %s, got %s (at %d)", t.name, i+1, fn.Params[i], at, a.pos())
			}
		}
		if fn.Tier != "" {
			c.tiers[fn.Tier] = true
		}
		return fn.Result, nil
	}
	if t.name == matchedIdent {
		return invalid, fmt.Errorf("matched() names another rule and is resolved before this point (at %d)", t.p)
	}
	return invalid, fmt.Errorf("unknown function %q (at %d)", t.name, t.p)
}

// checkAggregate types a result-set question and lifts its condition out.
//
// The inner condition is checked by a checker of its own, with no lift of its
// own — which is both how result-set questions are stopped from nesting and
// why the tiers it reads stay its own. A rule asking whether the set holds a
// probed release does not itself need this release to be probed.
func (c *checker) checkAggregate(t *callNode) (Type, error) {
	if c.lift == nil {
		return invalid, fmt.Errorf("%s asks about the whole result set, which cannot be answered here — result-set questions do not nest (at %d)", t.name, t.p)
	}
	if c.inHash {
		return invalid, fmt.Errorf("%s cannot ask about the result set from inside a list predicate (at %d)", t.name, t.p)
	}
	sub := newChecker(c.reg)
	at, err := sub.check(t.args[0])
	if err != nil {
		return invalid, err
	}
	if at != Bool {
		return invalid, fmt.Errorf("%s over one argument asks a yes/no question about the result set, got %s — for a list, use the two-argument form like %s(hdr, # == \"DV\") (at %d)", t.name, at, t.name, t.p)
	}
	form := t.name
	if form == "any" {
		form = "exists"
	}
	idx, err := c.lift(t.args[0], form, sortedKeys(sub.tiers), t.p)
	if err != nil {
		return invalid, err
	}
	// rewrite in place: the call becomes a read of the precomputed answer
	*t = callNode{name: aggResolved, args: []node{&aggNode{idx: idx, form: form, p: t.p}}, p: t.p}
	if form == "count" {
		return Num, nil
	}
	return Bool, nil
}

func (c *checker) checkBuiltin(t *callNode, b *builtin) (Type, error) {
	if b.predicate {
		return c.checkPredicate(t, b)
	}
	if b.variadic {
		if len(t.args) == 0 {
			return invalid, fmt.Errorf("%s needs at least one argument (at %d)", t.name, t.p)
		}
		for _, a := range t.args {
			at, err := c.check(a)
			if err != nil {
				return invalid, err
			}
			if at != b.params[0] {
				return invalid, fmt.Errorf("%s takes %s, got %s (at %d)", t.name, b.params[0], at, a.pos())
			}
		}
		return b.result, nil
	}
	if len(t.args) != len(b.params) {
		return invalid, fmt.Errorf("%s takes %d argument(s), got %d (at %d)", t.name, len(b.params), len(t.args), t.p)
	}
	for i, a := range t.args {
		at, err := c.check(a)
		if err != nil {
			return invalid, err
		}
		want := b.params[i]
		if want.K == KList && want.Elem == KInvalid {
			if at.K != KList {
				return invalid, fmt.Errorf("%s argument %d wants a list, got %s (at %d)", t.name, i+1, at, a.pos())
			}
			continue
		}
		if want == anyType {
			continue
		}
		if !at.assignable(want) {
			return invalid, fmt.Errorf("%s argument %d wants %s, got %s (at %d)", t.name, i+1, want, at, a.pos())
		}
	}
	if b.result == anyType && len(t.args) > 0 {
		return c.check(t.args[0])
	}
	return b.result, nil
}

// checkPredicate types the two-argument collection forms — count(list, # > 3),
// any(hdr, # == "DV") — where # stands for the element under test.
func (c *checker) checkPredicate(t *callNode, b *builtin) (Type, error) {
	if len(t.args) != 2 {
		return invalid, fmt.Errorf("%s over a list takes the list and a test, e.g. %s(hdr, # == \"DV\") (at %d)", t.name, t.name, t.p)
	}
	lt, err := c.check(t.args[0])
	if err != nil {
		return invalid, err
	}
	if lt.K != KList {
		return invalid, fmt.Errorf("%s wants a list, got %s (at %d)", t.name, lt, t.args[0].pos())
	}
	if c.inHash {
		return invalid, fmt.Errorf("list predicates do not nest (at %d)", t.p)
	}
	c.inHash, c.hashType = true, Type{K: lt.Elem}
	pt, err := c.check(t.args[1])
	c.inHash, c.hashType = false, invalid
	if err != nil {
		return invalid, err
	}
	if pt != Bool {
		return invalid, fmt.Errorf("the test in %s has to be yes/no, got %s (at %d)", t.name, pt, t.args[1].pos())
	}
	return b.result, nil
}

// editDistance is Levenshtein, bounded by the shorter operand. Only ever run
// against the field list when a name failed to resolve.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(min(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
