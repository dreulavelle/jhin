package rules

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The builtin vocabulary. Deliberately small: anything an application needs
// beyond this is a registered function, which keeps the grammar closed and
// the checker's job finite.

// anyType marks a builtin parameter that takes whatever it is given, and a
// result that echoes the first argument's type.
var anyType = Type{K: KInvalid, Elem: 255}

// aggregateForms are the names that ask about the whole result set when they
// are called with a single yes/no argument.
var aggregateForms = map[string]bool{
	"count": true, "exists": true, "any": true, "none": true,
}

// matchedIdent refers to another rule by name; it is resolved by inlining
// before the checker ever sees it.
const matchedIdent = "matched"

// aggResolved is the internal call a lifted result-set question becomes. It
// is not a name a condition can write: the lexer would accept it, but the
// checker has no entry for it and reports it unknown.
const aggResolved = "\x00agg"

type builtin struct {
	params []Type
	result Type
	// variadic takes one or more of params[0].
	variadic bool
	// predicate is the two-argument collection form with a # placeholder.
	predicate bool
	fn        func(args []Value) (Value, error)
}

var builtins map[string]*builtin

func init() {
	builtins = map[string]*builtin{
		"len": {params: []Type{anyType}, result: Num, fn: func(a []Value) (Value, error) {
			switch a[0].Kind() {
			case KList:
				return NumOf(len(a[0].List())), nil
			case KStr:
				return NumOf(len([]rune(a[0].Str()))), nil
			}
			return invalidValue("len wants a list or text")
		}},
		"lower": {params: []Type{Str}, result: Str, fn: func(a []Value) (Value, error) {
			return StrOf(strings.ToLower(a[0].Str())), nil
		}},
		"upper": {params: []Type{Str}, result: Str, fn: func(a []Value) (Value, error) {
			return StrOf(strings.ToUpper(a[0].Str())), nil
		}},
		"trim": {params: []Type{Str}, result: Str, fn: func(a []Value) (Value, error) {
			return StrOf(strings.TrimSpace(a[0].Str())), nil
		}},
		"abs": {params: []Type{Num}, result: Num, fn: func(a []Value) (Value, error) {
			return NumOf(math.Abs(a[0].Num())), nil
		}},
		"floor": {params: []Type{Num}, result: Num, fn: func(a []Value) (Value, error) {
			return NumOf(math.Floor(a[0].Num())), nil
		}},
		"ceil": {params: []Type{Num}, result: Num, fn: func(a []Value) (Value, error) {
			return NumOf(math.Ceil(a[0].Num())), nil
		}},
		"round": {params: []Type{Num}, result: Num, fn: func(a []Value) (Value, error) {
			return NumOf(math.Round(a[0].Num())), nil
		}},
		"min": {params: []Type{Num}, result: Num, variadic: true, fn: func(a []Value) (Value, error) {
			m := a[0].Num()
			for _, v := range a[1:] {
				m = math.Min(m, v.Num())
			}
			return NumOf(m), nil
		}},
		"max": {params: []Type{Num}, result: Num, variadic: true, fn: func(a []Value) (Value, error) {
			m := a[0].Num()
			for _, v := range a[1:] {
				m = math.Max(m, v.Num())
			}
			return NumOf(m), nil
		}},
		"string": {params: []Type{anyType}, result: Str, fn: func(a []Value) (Value, error) {
			return StrOf(a[0].String()), nil
		}},
		"num": {params: []Type{Str}, result: Num, fn: func(a []Value) (Value, error) {
			// Unparseable text is zero rather than an error: a rule asking
			// num(bitrate) > 5 on a release with no bitrate should read false,
			// not remove the release.
			n, err := strconv.ParseFloat(strings.TrimSpace(a[0].Str()), 64)
			if err != nil {
				return NumOf(0), nil
			}
			return NumOf(n), nil
		}},
	}
	// collection predicates share one implementation shape; the evaluator
	// runs the body per element rather than calling fn.
	for _, name := range []string{"count", "any", "all", "none"} {
		result := Bool
		if name == "count" {
			result = Num
		}
		builtins[name] = &builtin{predicate: true, result: result}
	}
}

func invalidValue(msg string) (Value, error) { return Value{}, fmt.Errorf("%s", msg) }
