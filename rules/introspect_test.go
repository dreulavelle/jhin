package rules

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// A registry reports its whole vocabulary, so a rule editor or a capabilities
// endpoint reads what rules are checked against rather than a copy of it.

func TestTierDetails(t *testing.T) {
	reg := testRegistry()
	got := reg.TierDetails()
	want := []TierInfo{
		{Name: "measured", Description: "a probed file"},
		{Name: "reported", Description: "size, age or grabs, which a release name does not carry"},
	}
	if len(got) != len(want) {
		t.Fatalf("TierDetails = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("TierDetails[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	// the names agree with Tiers, which stays as it was
	names := reg.Tiers()
	for i := range got {
		if got[i].Name != names[i] {
			t.Errorf("TierDetails and Tiers disagree at %d: %q vs %q", i, got[i].Name, names[i])
		}
	}
	if n := len(Core().TierDetails()); n != 0 {
		t.Errorf("Core declares %d tiers, want none", n)
	}
}

// Every entry Funcs reports compiles when called the way it describes — so a
// client that validates against the list is validating against the checker.
func TestFuncsCompile(t *testing.T) {
	reg := testRegistry()
	reg.FuncTier("probedOK", nil, Bool, "measured", func(Facts, []Value) (Value, error) {
		return BoolOf(true), nil
	})
	reg.Func("between", []Type{Num, Num, Num}, Bool, func(Facts, []Value) (Value, error) {
		return BoolOf(true), nil
	})
	funcs := reg.Funcs()
	if len(funcs) == 0 {
		t.Fatal("no functions reported")
	}
	seen := map[string]bool{}
	for _, f := range funcs {
		call := sampleCall(f)
		// string() takes anything, so one shape covers every result type
		when := fmt.Sprintf(`string(%s) != "never"`, call)
		if _, err := Compile(reg, []Rule{{Name: "r", When: when, Score: "1"}}); err != nil {
			t.Errorf("%s (%s form): %q does not compile: %v", f.Name, f.Form, call, err)
		}
		if f.Variadic {
			// one more than the declared count has to be accepted too
			more := fmt.Sprintf(`string(%s(%s, %s)) != ""`, f.Name, sampleArg(f.Params[0]), sampleArg(f.Params[0]))
			if _, err := Compile(reg, []Rule{{Name: "r", When: more, Score: "1"}}); err != nil {
				t.Errorf("%s: variadic call %q does not compile: %v", f.Name, more, err)
			}
		}
		seen[f.Name+"/"+f.Form] = true
	}
	// what the language offers, and what was registered, are all there
	for _, want := range []string{
		"len/function", "lower/function", "upper/function", "trim/function",
		"abs/function", "floor/function", "ceil/function", "round/function",
		"min/function", "max/function", "string/function", "num/function",
		"count/collection", "any/collection", "all/collection", "none/collection",
		"count/aggregate", "exists/aggregate", "any/aggregate", "none/aggregate",
		"double/function", "probedOK/function", "between/function",
	} {
		if !seen[want] {
			t.Errorf("Funcs does not report %s", want)
		}
	}
	if seen["matched/function"] || seen["all/aggregate"] {
		t.Errorf("Funcs reports a name that is not callable: %v", seen)
	}
}

// sampleCall writes a call that matches the reported signature.
func sampleCall(f FuncInfo) string {
	if f.Form == FormCollection {
		return fmt.Sprintf(`%s(["x", "y"], # == "x")`, f.Name)
	}
	args := make([]string, len(f.Params))
	for i, p := range f.Params {
		args[i] = sampleArg(p)
	}
	return f.Name + "(" + strings.Join(args, ", ") + ")"
}

func sampleArg(p Type) string {
	switch {
	case p == Any, p == Num:
		return "1"
	case p == Str:
		return `"x"`
	case p == Bool:
		return "true"
	case p.K == KList && p.Elem == KNum:
		return "[1, 2]"
	case p.K == KList:
		return `["x"]`
	}
	return "?"
}

func TestFuncsDetails(t *testing.T) {
	reg := testRegistry()
	reg.FuncTier("probedOK", nil, Bool, "measured", func(Facts, []Value) (Value, error) {
		return BoolOf(true), nil
	})
	byKey := map[string]FuncInfo{}
	var order []string
	for _, f := range reg.Funcs() {
		byKey[f.Name+"/"+f.Form] = f
		order = append(order, f.Name+"/"+f.Form)
	}
	for i := 1; i < len(order); i++ {
		if order[i-1] >= order[i] {
			t.Errorf("Funcs is not sorted: %q before %q", order[i-1], order[i])
		}
	}
	checks := []struct {
		key  string
		want FuncInfo
	}{
		{"min/function", FuncInfo{Name: "min", Params: []Type{Num}, Result: Num, Form: FormFunction, Variadic: true}},
		{"len/function", FuncInfo{Name: "len", Params: []Type{Any}, Result: Num, Form: FormFunction}},
		{"string/function", FuncInfo{Name: "string", Params: []Type{Any}, Result: Str, Form: FormFunction}},
		{"any/collection", FuncInfo{Name: "any", Params: []Type{{K: KList}, Bool}, Result: Bool, Form: FormCollection}},
		{"count/collection", FuncInfo{Name: "count", Params: []Type{{K: KList}, Bool}, Result: Num, Form: FormCollection}},
		{"count/aggregate", FuncInfo{Name: "count", Params: []Type{Bool}, Result: Num, Form: FormAggregate}},
		{"exists/aggregate", FuncInfo{Name: "exists", Params: []Type{Bool}, Result: Bool, Form: FormAggregate}},
		{"double/function", FuncInfo{Name: "double", Params: []Type{Num}, Result: Num, Form: FormFunction}},
		{"probedOK/function", FuncInfo{Name: "probedOK", Params: []Type{}, Result: Bool, Form: FormFunction, Tier: "measured"}},
	}
	for _, c := range checks {
		got, ok := byKey[c.key]
		if !ok {
			t.Errorf("%s not reported", c.key)
			continue
		}
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%s = %+v, want %+v", c.key, got, c.want)
		}
	}
	// the report is a copy: editing it changes nothing a later call sees
	reg.Funcs()[0].Params = append(reg.Funcs()[0].Params, Str)
	if len(reg.Funcs()[0].Params) != len(byKey[order[0]].Params) {
		t.Error("Funcs handed out the registry's own parameter slice")
	}
}

// Types marshal as their names, so a signature is readable as JSON.
func TestTypeJSON(t *testing.T) {
	for typ, want := range map[Type]string{
		Num: "num", Str: "string", Bool: "bool", StrList: "list<string>",
		NumList: "list<num>", Any: "any", {K: KList}: "list",
	} {
		b, err := json.Marshal(typ)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		if err := json.Unmarshal(b, &got); err != nil || got != want {
			t.Errorf("%v marshals as %s, want %q", typ, b, want)
		}
	}
	b, err := json.Marshal(FuncInfo{Name: "min", Params: []Type{Num}, Result: Num, Form: FormFunction, Variadic: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"name":"min","params":["num"],"result":"num","form":"function","variadic":true}`; string(b) != want {
		t.Errorf("FuncInfo JSON = %s, want %s", b, want)
	}
}
