package rules

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// A result set is judged for one request, so the kind has to reach the
// aggregate: an inner condition may name a scoped rule, and counting it
// against no kind at all answers false for every scoped reference.
func TestAggregateCarriesKind(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "4K movie", Action: ActionDefine, When: `resolution == "2160p"`, Scope: []string{"movie"}},
		{Name: "Any 4K movie", When: `exists(matched("4K movie"))`, Score: "100"},
	})
	if err != nil {
		t.Fatal(err)
	}
	set := []Facts{release("2160p", "BluRay", nil)}

	if got := eng.Evaluate(set[0], "movie", eng.ComputeAggregates(set, "movie")).Points; got != 100 {
		t.Errorf("movie scored %d, want the scoped reference to count", got)
	}
	if got := eng.Evaluate(set[0], "series", eng.ComputeAggregates(set, "series")).Points; got != 0 {
		t.Errorf("series scored %d, want the movie-only reference not to count", got)
	}
}

// A compiled Engine is shared across a batch's workers, so nothing may mutate
// during evaluation.
func TestEngineIsConcurrencySafe(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "T1", Action: ActionDefine, When: `group in ["A", "B"]`},
		{Name: "Trusted", When: `matched("T1") and resolution == "2160p"`, Score: "min(grabs, 10) * 3"},
		{Name: "Regex", When: `releaseName matches "(?i)\bIMAX\b"`, Score: "5"},
		{Name: "Probe", When: `probed.height > 1000`, Score: "1"},
		{Name: "Cap", Action: ActionLimit, Count: 2, When: "true", GroupBy: "resolution"},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := release("2160p", "BluRay", nil, map[string]Value{
		"group":         StrOf("A"),
		"grabs":         NumOf(50),
		"releaseName":   StrOf("Movie IMAX 2160p"),
		"probed.height": NumOf(2160),
	})
	set := []Facts{f}
	aggs := eng.ComputeAggregates(set, "movie")

	want := eng.Evaluate(f, "movie", aggs)
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				got := eng.Evaluate(f, "movie", aggs)
				if got.Points != want.Points || len(got.Matched) != len(want.Matched) {
					t.Errorf("concurrent evaluation gave %+v, want %+v", got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// An effect carries whatever type it was registered with.
func TestEffectTypes(t *testing.T) {
	reg := testRegistry()
	reg.Effect("boost", Num)
	reg.Effect("flag", Bool)
	eng, err := Compile(reg, []Rule{
		{Name: "b", When: "true", Action: "boost", Score: "grabs * 2"},
		{Name: "f", When: "true", Action: "flag", Score: `resolution == "2160p"`},
		{Name: "t", When: "true", Action: "tag", Score: `"x" + quality`},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := eng.Evaluate(facts(map[string]Value{
		"grabs": NumOf(21), "resolution": StrOf("2160p"), "quality": StrOf("WEB"),
	}), "", nil)
	if len(out.Effects) != 3 {
		t.Fatalf("effects = %+v, want 3", out.Effects)
	}
	if out.Effects[0].Value.Num() != 42 {
		t.Errorf("boost = %v, want 42", out.Effects[0].Value)
	}
	if !out.Effects[1].Value.Bool() {
		t.Errorf("flag = %v, want true", out.Effects[1].Value)
	}
	if out.Effects[2].Value.Str() != "xWEB" {
		t.Errorf("tag = %q, want xWEB", out.Effects[2].Value.Str())
	}

	// and the value has to type-check against what was registered
	if _, err := Compile(reg, []Rule{
		{Name: "bad", When: "true", Action: "boost", Score: `"not a number"`},
	}); err == nil || !strings.Contains(err.Error(), "value has to be num") {
		t.Errorf("a mistyped effect value compiled: %v", err)
	}
	if _, err := Compile(reg, []Rule{
		{Name: "bare", When: "true", Action: "boost"},
	}); err == nil || !strings.Contains(err.Error(), "needs a value") {
		t.Errorf("an effect without a value compiled: %v", err)
	}
}

func TestRegistryErrors(t *testing.T) {
	for _, tc := range []struct {
		build    func(*Registry)
		contains string
	}{
		{func(r *Registry) { r.Field("sizeGB", Num, "nope") }, "not declared"},
		{func(r *Registry) { r.Field("resolution", Str, "") }, "declared twice"},
		{func(r *Registry) { r.Field("bad name", Num, "") }, "not a valid"},
		{func(r *Registry) {
			r.Func("len", nil, Num, func(Facts, []Value) (Value, error) { return Value{}, nil })
		}, "builtin"},
		{func(r *Registry) {
			r.Func("exists", nil, Num, func(Facts, []Value) (Value, error) { return Value{}, nil })
		}, "result set"},
		{func(r *Registry) {
			r.Func("matched", nil, Num, func(Facts, []Value) (Value, error) { return Value{}, nil })
		}, "another rule"},
		{func(r *Registry) { r.Func("x", nil, Num, nil) }, "no implementation"},
		{func(r *Registry) { r.Effect("score", Num) }, "built-in action"},
		{func(r *Registry) { r.Namespace("probed", "undeclared").Num("h") }, "not declared"},
	} {
		reg := Core()
		tc.build(reg)
		if err := reg.Err(); err == nil {
			t.Errorf("registry accepted it, want %q", tc.contains)
		} else if !strings.Contains(err.Error(), tc.contains) {
			t.Errorf("error %q, want it to contain %q", err, tc.contains)
		}
		// and Compile surfaces it rather than building against a broken schema
		if _, err := Compile(reg, []Rule{{Name: "r", When: "true"}}); err == nil {
			t.Error("Compile ignored a broken registry")
		}
	}
}

// An Engine holds its own copy of the schema, so registering more after
// compiling cannot change what already compiled.
func TestRegistrySnapshot(t *testing.T) {
	reg := Core()
	eng, err := Compile(reg, []Rule{{Name: "r", When: `resolution == "2160p"`, Score: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	reg.Field("addedLater", Num, "")
	if _, ok := eng.reg.Lookup("addedLater"); ok {
		t.Error("a compiled engine saw a field registered after it was built")
	}
}

func TestUnknownAction(t *testing.T) {
	_, err := Compile(testRegistry(), []Rule{{Name: "r", When: "true", Action: "explode"}})
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("error %v, want it to name the action", err)
	}
}

// An empty rule set compiles to nothing rather than to an engine that does
// nothing, so a caller can test for it.
func TestEmptySets(t *testing.T) {
	for _, rs := range [][]Rule{nil, {}, {{Name: "d", Action: ActionDefine, When: "true"}}} {
		eng, err := Compile(testRegistry(), rs)
		if err != nil {
			t.Fatal(err)
		}
		if eng.Len() != 0 {
			t.Errorf("%v compiled to %d rules", rs, eng.Len())
		}
		// every method has to survive the nil engine
		_ = eng.Evaluate(facts(nil), "", nil)
		_ = eng.ComputeAggregates(nil, "")
		_ = eng.HasAggregates()
		_ = eng.ReadsTier("measured")
		_ = eng.AggregateSources()
		_ = eng.Aggregates(nil)
	}
}

// ReadsTier lets a caller skip a lookup nothing will ask about.
func TestReadsTier(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{{Name: "r", When: "sizeGB > 1", Score: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if !eng.ReadsTier("reported") {
		t.Error("ReadsTier missed the tier the rule reads")
	}
	if eng.ReadsTier("measured") {
		t.Error("ReadsTier claimed a tier no rule reads")
	}
}

// Facts that answer nothing are the same as a release with no name at all:
// every attribute reads as its zero, and nothing panics.
func TestEmptyFacts(t *testing.T) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "r", When: `resolution == "" and year == 0 and len(traits) == 0 and title == ""`, Score: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := eng.Evaluate(MapFacts{}, "", nil).Points; got != 1 {
		t.Errorf("points = %d, want empty facts to read as zeros", got)
	}
	if got := eng.Evaluate(FromResult("", nil, nil), "", nil).Points; got != 1 {
		t.Errorf("points = %d, want a nil parse result to read as zeros", got)
	}
}

// Layer stacks an application's facts over jhin's, and the application wins.
func TestLayer(t *testing.T) {
	app := MapFacts{
		Values: map[string]Value{"resolution": StrOf("2160p")},
		Tiers:  map[string]bool{"measured": true},
	}
	base := MapFacts{Values: map[string]Value{"resolution": StrOf("1080p"), "quality": StrOf("WEB")}}
	l := Layer(app, base)

	if v, _ := l.Lookup("resolution"); v.Str() != "2160p" {
		t.Errorf("resolution = %q, want the application's value to win", v.Str())
	}
	if v, _ := l.Lookup("quality"); v.Str() != "WEB" {
		t.Errorf("quality = %q, want it to fall through", v.Str())
	}
	if !l.TierPresent("measured") || !l.TierPresent("") {
		t.Error("Layer lost a tier one of its sources carries")
	}
	if l.TierPresent("reported") {
		t.Error("Layer invented a tier no source carries")
	}
}

// A ternary reports the type of the branch that has one, which only differs
// from either branch when one side is an empty list.
func TestTernaryTypes(t *testing.T) {
	for _, tc := range []struct{ when string }{
		{`(true ? 1 : 2) == 1`},
		{`(true ? "a" : "b") == "a"`},
		{`(true ? true : false)`},
		{`"x" in (true ? ["x"] : [])`},
		{`"x" in (true ? [] : ["x"])`},
		{`(true ? traits : []) == traits`},
	} {
		if _, err := Compile(testRegistry(), []Rule{{Name: "r", When: tc.when}}); err != nil {
			t.Errorf("%s: %v", tc.when, err)
		}
	}
	// branches that genuinely disagree are still refused
	if _, err := Compile(testRegistry(), []Rule{{Name: "r", When: `(true ? 1 : "a") == 1`}}); err == nil {
		t.Error("mismatched ternary branches compiled")
	}
}

// An effect has to reach an application's own output with its value intact,
// not as an opaque name.
func TestValueJSON(t *testing.T) {
	reg := testRegistry()
	reg.Effect("boost", Num)
	reg.Effect("langs", StrList)
	eng, err := Compile(reg, []Rule{
		{Name: "t", When: "true", Action: "tag", Score: `"x"`},
		{Name: "b", When: "true", Action: "boost", Score: "1.5"},
		{Name: "l", When: "true", Action: "langs", Score: `languages`},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := eng.Evaluate(facts(map[string]Value{
		"languages": StrListOf([]string{"en", "de"}),
	}), "", nil)

	blob, err := json.Marshal(out.Effects)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"name":"tag","value":"x"},{"name":"boost","value":1.5},{"name":"langs","value":["en","de"]}]`
	if string(blob) != want {
		t.Errorf("effects marshalled to\n %s\nwant\n %s", blob, want)
	}

	// an empty list is [] rather than null, so a consumer can iterate it
	if b, _ := json.Marshal(ListOf(KStr)); string(b) != "[]" {
		t.Errorf("empty list marshalled to %s, want []", b)
	}
}
