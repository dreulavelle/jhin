package rules

import (
	"math"
	"strconv"
	"strings"
)

// Kind is a value's runtime type. The set is closed: a rule can only ever
// hold what a release attribute can be, which keeps the checker's promotion
// rules to a page and the evaluator free of reflection.
type Kind uint8

const (
	KInvalid Kind = iota
	KBool
	KNum
	KStr
	KList
)

func (k Kind) String() string {
	switch k {
	case KBool:
		return "bool"
	case KNum:
		return "num"
	case KStr:
		return "string"
	case KList:
		return "list"
	}
	return "invalid"
}

// Type is a Kind plus, for lists, the kind of their elements. Nesting stops
// there: a list of lists has no meaning for any release attribute.
type Type struct {
	K    Kind
	Elem Kind
}

var (
	Bool    = Type{K: KBool}
	Num     = Type{K: KNum}
	Str     = Type{K: KStr}
	StrList = Type{K: KList, Elem: KStr}
	NumList = Type{K: KList, Elem: KNum}
	invalid = Type{}
)

// List is the list type over elem.
func List(elem Type) Type { return Type{K: KList, Elem: elem.K} }

func (t Type) String() string {
	if t.K == KList {
		return "list<" + t.Elem.String() + ">"
	}
	return t.K.String()
}

func (t Type) valid() bool { return t.K != KInvalid }

// assignable reports whether a value of type t satisfies want. An empty list
// literal has no element type of its own and fits any list.
func (t Type) assignable(want Type) bool {
	if t == want {
		return true
	}
	if t.K == KList && want.K == KList && (t.Elem == KInvalid || want.Elem == KInvalid) {
		return true
	}
	return false
}

// Value is a rule's runtime datum. It is passed by value and carries no
// pointers beyond a list's backing array, which keeps a condition's cost to
// the walk itself: a rule that does not fire allocates only its evaluation
// state, and one that does allocates what it reports.
type Value struct {
	k    Kind
	num  float64
	str  string
	list []Value
}

func BoolOf(b bool) Value {
	return Value{k: KBool, num: map[bool]float64{true: 1, false: 0}[b]}
}

// NumOf builds a numeric value. Every integer a release carries — year,
// season, bit depth, grab count — is exact in float64's integer range, so
// one numeric type serves both.
func NumOf[T int | int64 | float64](n T) Value { return Value{k: KNum, num: float64(n)} }

func StrOf(s string) Value { return Value{k: KStr, str: s} }

// ListOf builds a list value. elem records the element kind so an empty list
// still type-checks against a typed list field.
func ListOf(elem Kind, items ...Value) Value {
	return Value{k: KList, num: float64(elem), list: items}
}

// StrListOf is the common case: a list of strings.
func StrListOf(items []string) Value {
	vs := make([]Value, len(items))
	for i, s := range items {
		vs[i] = StrOf(s)
	}
	return ListOf(KStr, vs...)
}

// NumListOf is the other common case.
func NumListOf(items []int) Value {
	vs := make([]Value, len(items))
	for i, n := range items {
		vs[i] = NumOf(n)
	}
	return ListOf(KNum, vs...)
}

func (v Value) Kind() Kind     { return v.k }
func (v Value) Bool() bool     { return v.num != 0 }
func (v Value) Num() float64   { return v.num }
func (v Value) Str() string    { return v.str }
func (v Value) List() []Value  { return v.list }
func (v Value) elemKind() Kind { return Kind(v.num) }
func (v Value) Type() Type {
	if v.k == KList {
		return Type{K: KList, Elem: v.elemKind()}
	}
	return Type{K: v.k}
}

// equals is value identity, used by == and by `in`. Lists compare element by
// element; two lists of different length are never equal.
func (v Value) equals(o Value) bool {
	if v.k != o.k {
		return false
	}
	switch v.k {
	case KBool:
		return v.Bool() == o.Bool()
	case KNum:
		return v.num == o.num
	case KStr:
		return v.str == o.str
	case KList:
		if len(v.list) != len(o.list) {
			return false
		}
		for i := range v.list {
			if !v.list[i].equals(o.list[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// String renders a value for a limit rule's bucket key and for error text.
// A bucket only needs to tell two releases apart, so every type needs some
// written form — including a list, which buckets as a whole.
func (v Value) String() string {
	switch v.k {
	case KBool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case KNum:
		if v.num == math.Trunc(v.num) && !math.IsInf(v.num, 0) {
			return strconv.FormatInt(int64(v.num), 10)
		}
		return strconv.FormatFloat(v.num, 'g', -1, 64)
	case KStr:
		return v.str
	case KList:
		var b strings.Builder
		b.WriteByte('[')
		for i, e := range v.list {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(e.String())
		}
		b.WriteByte(']')
		return b.String()
	}
	return ""
}

// zero is the value a field of type t holds when nothing supplied it. It is
// only ever reached for a field in a tier the release does carry — a missing
// tier skips the rule outright rather than reading a zero (see Engine.Evaluate).
func zero(t Type) Value {
	switch t.K {
	case KBool:
		return BoolOf(false)
	case KNum:
		return NumOf(0)
	case KStr:
		return StrOf("")
	case KList:
		return ListOf(t.Elem)
	}
	return Value{}
}
