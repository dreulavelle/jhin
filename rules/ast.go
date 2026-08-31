package rules

// The AST. Nodes are immutable once compiled: Engine shares one tree across
// every goroutine evaluating a batch, so nothing here may carry per-release
// state.

type node interface{ pos() int }

type litNode struct {
	v Value
	p int
}

// fieldNode is a resolved attribute reference. Dotted paths are folded into
// one node at parse time — "probed.height" is a name, not a member access —
// because the language has no values with members.
type fieldNode struct {
	path string
	typ  Type
	tier string
	p    int
}

type listNode struct {
	items []node
	elem  Kind
	p     int
}

type unaryNode struct {
	op string
	x  node
	p  int
}

type binaryNode struct {
	op   string
	l, r node
	// re is the compiled pattern for `matches`, built once at compile time.
	re *regexpMatcher
	p  int
}

type ternaryNode struct {
	cond, then, els node
	p               int
}

type callNode struct {
	name string
	args []node
	p    int
}

// hashNode is the element placeholder inside a collection predicate. It is
// only legal in the second argument of count/any/all/none over a list.
type hashNode struct{ p int }

// aggNode is a result-set question after lifting: the inner condition has
// become its own program and this reads the precomputed answer by index.
type aggNode struct {
	idx  int
	form string // count | exists | none
	p    int
}

// scopedNode wraps an inlined matched() reference so the referenced rule's
// scope travels with its condition — a reference to a movie-only rule is
// false for a series whatever scope the referring rule has.
type scopedNode struct {
	scope map[string]bool
	x     node
	p     int
}

func (n *litNode) pos() int     { return n.p }
func (n *fieldNode) pos() int   { return n.p }
func (n *listNode) pos() int    { return n.p }
func (n *unaryNode) pos() int   { return n.p }
func (n *binaryNode) pos() int  { return n.p }
func (n *ternaryNode) pos() int { return n.p }
func (n *callNode) pos() int    { return n.p }
func (n *hashNode) pos() int    { return n.p }
func (n *aggNode) pos() int     { return n.p }
func (n *scopedNode) pos() int  { return n.p }

// size counts nodes, bounding matched() expansion: a reference is a copy, so
// a chain where each rule names the one below it twice doubles at every step.
func size(n node) int {
	if n == nil {
		return 0
	}
	switch t := n.(type) {
	case *litNode, *fieldNode, *hashNode, *aggNode:
		return 1
	case *listNode:
		total := 1
		for _, it := range t.items {
			total += size(it)
		}
		return total
	case *unaryNode:
		return 1 + size(t.x)
	case *binaryNode:
		return 1 + size(t.l) + size(t.r)
	case *ternaryNode:
		return 1 + size(t.cond) + size(t.then) + size(t.els)
	case *callNode:
		total := 1
		for _, a := range t.args {
			total += size(a)
		}
		return total
	case *scopedNode:
		return 1 + size(t.x)
	}
	return 1
}

// walk visits every node depth-first. Used to collect tiers and aggregate
// references after checking.
func walk(n node, fn func(node)) {
	if n == nil {
		return
	}
	fn(n)
	switch t := n.(type) {
	case *listNode:
		for _, it := range t.items {
			walk(it, fn)
		}
	case *unaryNode:
		walk(t.x, fn)
	case *binaryNode:
		walk(t.l, fn)
		walk(t.r, fn)
	case *ternaryNode:
		walk(t.cond, fn)
		walk(t.then, fn)
		walk(t.els, fn)
	case *callNode:
		for _, a := range t.args {
			walk(a, fn)
		}
	case *scopedNode:
		walk(t.x, fn)
	}
}
