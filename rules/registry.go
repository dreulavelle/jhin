package rules

import (
	"fmt"
	"sort"
	"strings"
)

// A Registry declares what a rule may name: which attributes exist, what type
// each one is, and how much it can be trusted. It is the whole extension
// surface — an application adds a fact source, a function or an action here,
// and never by changing the language.
//
// Build one, register against it, then hand it to Compile. A Registry is not
// safe for concurrent modification, but a compiled Engine no longer refers to
// it, so registration order and timing stop mattering once compilation is done.
type Registry struct {
	fields  map[string]Field
	tiers   map[string]string
	funcs   map[string]*Func
	effects map[string]Type
	err     error
}

// Field is one declared attribute.
type Field struct {
	Type Type
	// Tier names the confidence group this attribute belongs to. The empty
	// tier is always present: it is for facts every release carries.
	Tier string
}

// Func is an application-supplied function. Params are exact — the checker
// rejects a call that does not match, so a misspelt rule fails when the
// profile is saved rather than when a search runs.
type Func struct {
	Name   string
	Params []Type
	Result Type
	// Tier, when set, makes a call to this function depend on that tier the
	// same way naming one of its fields would.
	Tier string
	Fn   func(f Facts, args []Value) (Value, error)
}

// NewRegistry returns an empty registry. Most callers want Core, which
// returns one already carrying everything jhin can read off a release name.
func NewRegistry() *Registry {
	return &Registry{
		fields:  map[string]Field{},
		tiers:   map[string]string{},
		funcs:   map[string]*Func{},
		effects: map[string]Type{},
	}
}

func (r *Registry) fail(format string, args ...any) *Registry {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
	return r
}

// Tier declares a confidence group and what it means. Rules reading a tier
// fail open: a release carrying nothing in it skips them rather than being
// judged against zero values. See Engine.Evaluate.
func (r *Registry) Tier(name, description string) *Registry {
	if name == "" {
		return r.fail("tier: name is empty")
	}
	r.tiers[name] = description
	return r
}

// Field declares a single attribute at the top level.
func (r *Registry) Field(path string, t Type, tier string) *Registry {
	if !validPath(path) {
		return r.fail("field %q: not a valid attribute name", path)
	}
	if !t.valid() {
		return r.fail("field %q: no type", path)
	}
	if tier != "" {
		if _, ok := r.tiers[tier]; !ok {
			return r.fail("field %q: tier %q is not declared", path, tier)
		}
	}
	if _, dup := r.fields[path]; dup {
		return r.fail("field %q: declared twice", path)
	}
	r.fields[path] = Field{Type: t, Tier: tier}
	return r
}

// Namespace groups fields under a prefix, all sharing one tier:
//
//	reg.Namespace("probed", "measured").Num("height").Bool("dolbyVision")
//
// which declares probed.height and probed.dolbyVision.
func (r *Registry) Namespace(prefix, tier string) *Namespace {
	if !validName(prefix) {
		r.fail("namespace %q: not a valid name", prefix)
	}
	if tier != "" {
		if _, ok := r.tiers[tier]; !ok {
			r.fail("namespace %q: tier %q is not declared", prefix, tier)
		}
	}
	return &Namespace{reg: r, prefix: prefix, tier: tier}
}

// Namespace is a fluent builder for a group of fields under one prefix.
type Namespace struct {
	reg    *Registry
	prefix string
	tier   string
}

func (n *Namespace) add(name string, t Type) *Namespace {
	n.reg.Field(n.prefix+"."+name, t, n.tier)
	return n
}

func (n *Namespace) Num(name string) *Namespace     { return n.add(name, Num) }
func (n *Namespace) Str(name string) *Namespace     { return n.add(name, Str) }
func (n *Namespace) Bool(name string) *Namespace    { return n.add(name, Bool) }
func (n *Namespace) StrList(name string) *Namespace { return n.add(name, StrList) }
func (n *Namespace) NumList(name string) *Namespace { return n.add(name, NumList) }

// Done returns to the registry, for callers who prefer one chain.
func (n *Namespace) Done() *Registry { return n.reg }

// Func registers an application function. Anything the grammar deliberately
// cannot say, Go can:
//
//	reg.Func("imdbRating", nil, Num, fetchRating)
//	reg.Func("closerThan", []Type{Num, Num}, Bool, closerThan)
func (r *Registry) Func(name string, params []Type, result Type, fn func(Facts, []Value) (Value, error)) *Registry {
	if !validName(name) {
		return r.fail("function %q: not a valid name", name)
	}
	if _, reserved := builtins[name]; reserved {
		return r.fail("function %q: that name is a builtin", name)
	}
	if aggregateForms[name] {
		return r.fail("function %q: that name asks about the result set", name)
	}
	if name == matchedIdent {
		return r.fail("function %q: that name refers to another rule", name)
	}
	if fn == nil {
		return r.fail("function %q: no implementation", name)
	}
	if _, dup := r.funcs[name]; dup {
		return r.fail("function %q: registered twice", name)
	}
	r.funcs[name] = &Func{Name: name, Params: params, Result: result, Fn: fn}
	return r
}

// FuncTier is Func for a function that reads a tier, so a rule calling it
// fails open exactly as one naming a field of that tier would.
func (r *Registry) FuncTier(name string, params []Type, result Type, tier string, fn func(Facts, []Value) (Value, error)) *Registry {
	r.Func(name, params, result, fn)
	if f, ok := r.funcs[name]; ok {
		if _, known := r.tiers[tier]; !known && tier != "" {
			return r.fail("function %q: tier %q is not declared", name, tier)
		}
		f.Tier = tier
	}
	return r
}

// Effect declares an action jhin evaluates but does not interpret. A rule
// whose action is the effect's name reports Effect{Name, Value} in the
// outcome and nothing else happens — routing, tagging and anything else an
// application invents needs no change here.
//
// The value type is what the rule's "score" expression must produce; pass
// Bool for an effect that is only ever a flag.
func (r *Registry) Effect(name string, value Type) *Registry {
	if !validName(name) {
		return r.fail("effect %q: not a valid name", name)
	}
	if reservedActions[name] {
		return r.fail("effect %q: that is a built-in action", name)
	}
	if !value.valid() {
		return r.fail("effect %q: no value type", name)
	}
	r.effects[name] = value
	return r
}

// Err reports the first registration that failed. Compile returns it too, so
// checking here is optional.
func (r *Registry) Err() error { return r.err }

// Fields lists every declared attribute, sorted. Useful for a rule editor
// that wants to offer completions.
func (r *Registry) Fields() []string {
	out := make([]string, 0, len(r.fields))
	for p := range r.fields {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Lookup returns a declared field.
func (r *Registry) Lookup(path string) (Field, bool) {
	f, ok := r.fields[path]
	return f, ok
}

// Tiers lists every declared tier, sorted.
func (r *Registry) Tiers() []string {
	out := make([]string, 0, len(r.tiers))
	for t := range r.tiers {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// clone snapshots the registry so a compiled Engine is unaffected by later
// registration on the caller's copy.
func (r *Registry) clone() *Registry {
	c := NewRegistry()
	for k, v := range r.fields {
		c.fields[k] = v
	}
	for k, v := range r.tiers {
		c.tiers[k] = v
	}
	for k, v := range r.funcs {
		c.funcs[k] = v
	}
	for k, v := range r.effects {
		c.effects[k] = v
	}
	c.err = r.err
	return c
}

func validName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !isIdentStart(r) {
			return false
		}
		if i > 0 && !isIdentPart(r) {
			return false
		}
	}
	return true
}

func validPath(s string) bool {
	if s == "" {
		return false
	}
	for _, part := range strings.Split(s, ".") {
		if !validName(part) {
			return false
		}
	}
	return true
}
