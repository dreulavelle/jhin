package rules

import (
	"fmt"
	"regexp"
	"strings"
)

// regexpMatcher holds a pattern compiled once, at rule-compile time. A
// profile cannot build a regex per release: `matches` requires a literal
// right-hand side, which is what makes that guarantee hold.
type regexpMatcher struct {
	re  *regexp.Regexp
	src string
}

// maxDepth bounds recursive descent. Profiles are untrusted input, and a few
// thousand nested parentheses would otherwise reach the goroutine stack limit
// as a crash rather than as an error.
const maxDepth = 96

// maxSource and maxTokens bound an expression's size.
//
// maxDepth alone is not enough: `and` and `or` chains are built iteratively,
// so a flat chain of a hundred thousand terms makes a tree that deep down its
// left spine, and everything that walks a tree afterwards — checking,
// inlining, rendering, evaluating — recurses over it. A node can never
// outnumber the tokens it was built from, so capping tokens caps the tree.
const (
	maxSource = 128 << 10
	maxTokens = 20_000
)

// maxPathParts bounds a dotted attribute name. Two is the shape every real
// one has — a namespace and a field — and the cap is what stops an untrusted
// profile from naming one long enough to be expensive to even reject.
const maxPathParts = 8

type exprParser struct {
	toks  []token
	i     int
	depth int
}

// parse turns source into an unchecked tree. Types, field names and function
// arity are the checker's business (see check.go); this stage only decides
// shape.
func parse(src string) (node, error) {
	if len(src) > maxSource {
		return nil, fmt.Errorf("expression is %d bytes; the limit is %d", len(src), maxSource)
	}
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &exprParser{toks: toks}
	n, err := p.expr()
	if err != nil {
		return nil, err
	}
	if p.cur().kind != tEOF {
		return nil, fmt.Errorf("unexpected %s at %d", p.describe(p.cur()), p.cur().pos)
	}
	return n, nil
}

func (p *exprParser) cur() token  { return p.toks[p.i] }
func (p *exprParser) next() token { t := p.toks[p.i]; p.i++; return t }

func (p *exprParser) describe(t token) string {
	switch t.kind {
	case tEOF:
		return "end of expression"
	case tStr:
		return fmt.Sprintf("string %q", t.text)
	default:
		return fmt.Sprintf("%q", t.text)
	}
}

func (p *exprParser) enter() error {
	p.depth++
	if p.depth > maxDepth {
		return fmt.Errorf("expression nested too deeply (limit %d)", maxDepth)
	}
	return nil
}

func (p *exprParser) leave() { p.depth-- }

// isWordOp reports whether the current token is an identifier standing in
// operator position.
func (p *exprParser) isWordOp(name string) bool {
	t := p.cur()
	return t.kind == tIdent && t.text == name
}

func (p *exprParser) expr() (node, error) { return p.ternary() }

func (p *exprParser) ternary() (node, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	cond, err := p.or()
	if err != nil {
		return nil, err
	}
	if !p.cur().is("?") {
		return cond, nil
	}
	q := p.next()
	then, err := p.ternary()
	if err != nil {
		return nil, err
	}
	if !p.cur().is(":") {
		return nil, fmt.Errorf("expected \":\" for the ? : at %d", q.pos)
	}
	p.next()
	els, err := p.ternary()
	if err != nil {
		return nil, err
	}
	return &ternaryNode{cond: cond, then: then, els: els, p: q.pos}, nil
}

func (p *exprParser) or() (node, error) {
	l, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.isWordOp("or") {
		op := p.next()
		r, err := p.and()
		if err != nil {
			return nil, err
		}
		l = &binaryNode{op: "or", l: l, r: r, p: op.pos}
	}
	return l, nil
}

func (p *exprParser) and() (node, error) {
	l, err := p.notExpr()
	if err != nil {
		return nil, err
	}
	for p.isWordOp("and") {
		op := p.next()
		r, err := p.notExpr()
		if err != nil {
			return nil, err
		}
		l = &binaryNode{op: "and", l: l, r: r, p: op.pos}
	}
	return l, nil
}

func (p *exprParser) notExpr() (node, error) {
	if p.isWordOp("not") {
		// `not in` is a comparison operator, not a prefix not — only treat a
		// leading `not` as negation when what follows could start an operand.
		if p.toks[p.i+1].kind != tIdent || p.toks[p.i+1].text != "in" {
			op := p.next()
			if err := p.enter(); err != nil {
				return nil, err
			}
			defer p.leave()
			x, err := p.notExpr()
			if err != nil {
				return nil, err
			}
			return &unaryNode{op: "not", x: x, p: op.pos}, nil
		}
	}
	return p.comparison()
}

// comparison is non-associative on purpose: `a < b < c` is a compile error
// rather than the silent nonsense it is in most C-family languages.
func (p *exprParser) comparison() (node, error) {
	l, err := p.additive()
	if err != nil {
		return nil, err
	}
	op, pos, ok := p.compOp()
	if !ok {
		return l, nil
	}
	r, err := p.additive()
	if err != nil {
		return nil, err
	}
	if _, _, again := p.compOp(); again {
		return nil, fmt.Errorf("comparisons do not chain; parenthesise the one you mean (at %d)", p.cur().pos)
	}
	return &binaryNode{op: op, l: l, r: r, p: pos}, nil
}

func (p *exprParser) compOp() (string, int, bool) {
	t := p.cur()
	if t.kind == tOp {
		switch t.text {
		case "==", "!=", "<", "<=", ">", ">=":
			p.next()
			return t.text, t.pos, true
		}
		return "", 0, false
	}
	if t.kind != tIdent {
		return "", 0, false
	}
	switch t.text {
	case "in", "matches", "contains", "startsWith", "endsWith":
		p.next()
		return t.text, t.pos, true
	case "not":
		if p.toks[p.i+1].kind == tIdent && p.toks[p.i+1].text == "in" {
			p.next()
			p.next()
			return "not in", t.pos, true
		}
	}
	return "", 0, false
}

func (p *exprParser) additive() (node, error) {
	l, err := p.multiplicative()
	if err != nil {
		return nil, err
	}
	for p.cur().is("+") || p.cur().is("-") {
		op := p.next()
		r, err := p.multiplicative()
		if err != nil {
			return nil, err
		}
		l = &binaryNode{op: op.text, l: l, r: r, p: op.pos}
	}
	return l, nil
}

func (p *exprParser) multiplicative() (node, error) {
	l, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.cur().is("*") || p.cur().is("/") || p.cur().is("%") {
		op := p.next()
		r, err := p.unary()
		if err != nil {
			return nil, err
		}
		l = &binaryNode{op: op.text, l: l, r: r, p: op.pos}
	}
	return l, nil
}

func (p *exprParser) unary() (node, error) {
	if p.cur().is("-") {
		op := p.next()
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer p.leave()
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: "-", x: x, p: op.pos}, nil
	}
	return p.primary()
}

func (p *exprParser) primary() (node, error) {
	t := p.cur()
	switch {
	case t.kind == tNum || t.kind == tStr:
		p.next()
		return &litNode{v: t.val, p: t.pos}, nil

	case t.kind == tHash:
		p.next()
		return &hashNode{p: t.pos}, nil

	case t.is("("):
		p.next()
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer p.leave()
		n, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.cur().is(")") {
			return nil, fmt.Errorf("expected \")\" to close the group opened at %d", t.pos)
		}
		p.next()
		return n, nil

	case t.is("["):
		return p.listLit()

	case t.kind == tIdent:
		return p.identOrCall()
	}
	return nil, fmt.Errorf("expected a value but found %s at %d", p.describe(t), t.pos)
}

func (p *exprParser) listLit() (node, error) {
	open := p.next()
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()
	n := &listNode{p: open.pos}
	if p.cur().is("]") {
		p.next()
		return n, nil
	}
	for {
		it, err := p.expr()
		if err != nil {
			return nil, err
		}
		n.items = append(n.items, it)
		if p.cur().is(",") {
			p.next()
			// tolerate a trailing comma before the bracket
			if p.cur().is("]") {
				break
			}
			continue
		}
		break
	}
	if !p.cur().is("]") {
		return nil, fmt.Errorf("expected \"]\" to close the list opened at %d", open.pos)
	}
	p.next()
	return n, nil
}

func (p *exprParser) identOrCall() (node, error) {
	t := p.next()
	if t.text == "true" || t.text == "false" {
		return &litNode{v: BoolOf(t.text == "true"), p: t.pos}, nil
	}
	if p.cur().is("(") {
		return p.call(t)
	}
	// A dotted path is one name. The language has no values with members, so
	// "probed.height" is folded here rather than left as an access chain.
	//
	// The depth cap matters as much as the builder does: an attribute name is
	// a namespace and a field, and appending to a string in a loop is
	// quadratic, so untrusted input could otherwise spend gigabytes writing a
	// name nothing will ever resolve.
	if !p.cur().is(".") {
		return &fieldNode{path: t.text, p: t.pos}, nil
	}
	var b strings.Builder
	b.WriteString(t.text)
	for parts := 0; p.cur().is("."); parts++ {
		if parts >= maxPathParts {
			return nil, fmt.Errorf("attribute name has too many parts (limit %d) at %d", maxPathParts, t.pos)
		}
		p.next()
		part := p.cur()
		if part.kind != tIdent {
			return nil, fmt.Errorf("expected an attribute name after \".\" at %d", p.cur().pos)
		}
		p.next()
		b.WriteByte('.')
		b.WriteString(part.text)
	}
	return &fieldNode{path: b.String(), p: t.pos}, nil
}

func (p *exprParser) call(name token) (node, error) {
	p.next() // consume "("
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()
	n := &callNode{name: name.text, p: name.pos}
	if p.cur().is(")") {
		p.next()
		return n, nil
	}
	for {
		a, err := p.expr()
		if err != nil {
			return nil, err
		}
		n.args = append(n.args, a)
		if p.cur().is(",") {
			p.next()
			continue
		}
		break
	}
	if !p.cur().is(")") {
		return nil, fmt.Errorf("expected \")\" to close the call to %s at %d", name.text, name.pos)
	}
	p.next()
	return n, nil
}
