package rules

import (
	"strconv"
	"strings"
)

// render writes a tree back out as source. It is what deduplicates result-set
// questions — two rules asking the same thing in different spacing share one
// aggregate — and what lets a report name the condition a number belongs to.
//
// The output is canonical rather than faithful to what was typed: fully
// parenthesised where precedence would otherwise be load-bearing, so two trees
// render identically exactly when they mean the same thing.
func render(n node) string {
	var b strings.Builder
	writeNode(&b, n)
	return b.String()
}

func writeNode(b *strings.Builder, n node) {
	switch t := n.(type) {
	case nil:
		return

	case *litNode:
		writeValue(b, t.v)

	case *fieldNode:
		b.WriteString(t.path)

	case *hashNode:
		b.WriteByte('#')

	case *aggNode:
		b.WriteString(t.form)
		b.WriteString("#")
		b.WriteString(strconv.Itoa(t.idx))

	case *scopedNode:
		b.WriteString("scoped[")
		b.WriteString(strings.Join(sortedKeys(t.scope), ","))
		b.WriteString("](")
		writeNode(b, t.x)
		b.WriteByte(')')

	case *listNode:
		b.WriteByte('[')
		for i, it := range t.items {
			if i > 0 {
				b.WriteString(", ")
			}
			writeNode(b, it)
		}
		b.WriteByte(']')

	case *unaryNode:
		if t.op == "not" {
			b.WriteString("not (")
		} else {
			b.WriteString("-(")
		}
		writeNode(b, t.x)
		b.WriteByte(')')

	case *binaryNode:
		b.WriteByte('(')
		writeNode(b, t.l)
		b.WriteByte(' ')
		b.WriteString(t.op)
		b.WriteByte(' ')
		writeNode(b, t.r)
		b.WriteByte(')')

	case *ternaryNode:
		b.WriteByte('(')
		writeNode(b, t.cond)
		b.WriteString(" ? ")
		writeNode(b, t.then)
		b.WriteString(" : ")
		writeNode(b, t.els)
		b.WriteByte(')')

	case *callNode:
		b.WriteString(t.name)
		b.WriteByte('(')
		for i, a := range t.args {
			if i > 0 {
				b.WriteString(", ")
			}
			writeNode(b, a)
		}
		b.WriteByte(')')
	}
}

func writeValue(b *strings.Builder, v Value) {
	switch v.Kind() {
	case KStr:
		b.WriteString(quote(v.Str()))
	case KList:
		b.WriteByte('[')
		for i, e := range v.List() {
			if i > 0 {
				b.WriteString(", ")
			}
			writeValue(b, e)
		}
		b.WriteByte(']')
	default:
		b.WriteString(v.String())
	}
}

// quote writes a string literal the lexer reads back unchanged. Backslashes
// are literal inside a condition, so only the delimiter needs escaping — and
// a string already carrying a double quote takes single quotes instead, which
// keeps a pasted regex readable.
func quote(s string) string {
	if strings.Contains(s, `"`) && !strings.Contains(s, `'`) {
		return "'" + s + "'"
	}
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			b.WriteString(`\"`)
			continue
		}
		b.WriteByte(s[i])
	}
	b.WriteByte('"')
	return b.String()
}
