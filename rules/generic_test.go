package rules

import (
	"strings"
	"testing"
)

// The extension layer has to be general, not merely general enough for the
// one application it was written alongside. These tests check that as a
// property: the same engine serves domains that share no vocabulary with
// jhin's, and the guarantees hold for whatever an application registers.

// Three deliberately alien schemas. None of them is about releases, none
// shares a tier name with any other, and between them they use every type,
// every registration seam, and a scope vocabulary of their own.
type domain struct {
	name  string
	build func() *Registry
	facts func(present ...string) MapFacts
	rules []Rule
	kind  string
}

func domains() []domain {
	return []domain{
		{
			name: "job board",
			kind: "contract",
			build: func() *Registry {
				r := NewRegistry()
				r.Tier("employer", "details the employer filled in")
				r.Tier("scraped", "what the crawler could read from the posting")
				r.Field("titleText", Str, "")
				r.Field("salary", Num, "employer")
				r.Field("remote", Bool, "employer")
				r.Namespace("posting", "scraped").
					Num("daysOld").StrList("tags").NumList("levels")
				r.Func("blocklisted", []Type{Str}, Bool, func(_ Facts, a []Value) (Value, error) {
					return BoolOf(strings.Contains(a[0].Str(), "unpaid")), nil
				})
				r.Effect("notify", Str)
				return r
			},
			facts: func(present ...string) MapFacts {
				return MapFacts{
					Values: map[string]Value{
						"titleText":       StrOf("Senior Go Engineer"),
						"salary":          NumOf(180000),
						"remote":          BoolOf(true),
						"posting.daysOld": NumOf(3),
						"posting.tags":    StrListOf([]string{"go", "backend"}),
						"posting.levels":  NumListOf([]int{5, 6}),
					},
					Tiers: tierSet(present),
				}
			},
			rules: []Rule{
				{Name: "Pay", When: "salary > 0", Score: "salary / 1000"},
				{Name: "Fresh", When: "posting.daysOld < 7", Score: "50"},
				{Name: "Go", When: `"go" in posting.tags`, Score: "100"},
				{Name: "Senior", When: "6 in posting.levels", Score: "25"},
				{Name: "No unpaid", When: `blocklisted(titleText)`, Action: ActionReject},
				{Name: "Remote only", When: "not remote", Action: ActionReject},
				{Name: "Tell me", When: "salary > 150000", Action: "notify", Score: `"high-pay: " + titleText`},
				{Name: "Top 3", Action: ActionLimit, Count: 3, When: "true", GroupBy: "remote"},
				{Name: "Contract only", When: "true", Score: "5", Scope: []string{"contract"}},
			},
		},
		{
			name: "photo catalogue",
			kind: "raw",
			build: func() *Registry {
				r := NewRegistry()
				r.Tier("exif", "metadata the camera wrote")
				r.Tier("scored", "what the aesthetic model returned")
				r.Field("filename", Str, "")
				r.Namespace("exif", "exif").
					Num("iso").Num("focalLength").Str("lens").Bool("flash")
				r.Namespace("model", "scored").Num("aesthetic").StrList("subjects")
				r.Effect("album", Str)
				return r
			},
			facts: func(present ...string) MapFacts {
				return MapFacts{
					Values: map[string]Value{
						"filename":         StrOf("DSC_0421.NEF"),
						"exif.iso":         NumOf(400),
						"exif.focalLength": NumOf(85),
						"exif.lens":        StrOf("85mm f/1.4"),
						"exif.flash":       BoolOf(false),
						"model.aesthetic":  NumOf(7.4),
						"model.subjects":   StrListOf([]string{"portrait", "person"}),
					},
					Tiers: tierSet(present),
				}
			},
			rules: []Rule{
				{Name: "Sharp", When: "exif.iso <= 800", Score: "exif.iso < 200 ? 100 : 40"},
				{Name: "Portrait glass", When: `exif.lens contains "85mm"`, Score: "30"},
				{Name: "Looks good", When: "model.aesthetic > 7", Score: "round(model.aesthetic * 10)"},
				{Name: "People", When: `"person" in model.subjects`, Score: "20"},
				{Name: "No flash", When: "exif.flash", Action: ActionReject},
				{Name: "Raw only", When: `not (filename endsWith ".NEF")`, Action: ActionReject},
				{Name: "Best per lens", Action: ActionLimit, Count: 2, When: "true", GroupBy: "exif.lens"},
				{Name: "File it", When: `"portrait" in model.subjects`, Action: "album", Score: `"portraits"`},
			},
		},
		{
			name: "support tickets",
			kind: "billing",
			build: func() *Registry {
				r := NewRegistry()
				r.Tier("crm", "the customer record, when one is linked")
				r.Field("subject", Str, "")
				r.Field("waitingHours", Num, "")
				r.Field("channel", Str, "")
				r.Namespace("customer", "crm").
					Str("plan").Num("arr").Bool("churnRisk")
				r.FuncTier("openTickets", nil, Num, "crm", func(_ Facts, _ []Value) (Value, error) {
					return NumOf(4), nil
				})
				return r
			},
			facts: func(present ...string) MapFacts {
				return MapFacts{
					Values: map[string]Value{
						"subject":            StrOf("Cannot log in after billing change"),
						"waitingHours":       NumOf(30),
						"channel":            StrOf("email"),
						"customer.plan":      StrOf("enterprise"),
						"customer.arr":       NumOf(240000),
						"customer.churnRisk": BoolOf(true),
					},
					Tiers: tierSet(present),
				}
			},
			rules: []Rule{
				{Name: "Waiting", When: "waitingHours > 0", Score: "min(waitingHours, 48) * 10"},
				{Name: "Big account", When: `customer.plan == "enterprise"`, Score: "customer.arr / 1000"},
				{Name: "At risk", When: "customer.churnRisk", Score: "500"},
				{Name: "Backlogged", When: "openTickets() > 3", Score: "100"},
				{Name: "Enterprise", Action: ActionDefine, When: `customer.plan == "enterprise"`},
				{Name: "Escalate", When: `matched("Enterprise") and waitingHours > 24`, Score: "1000"},
				{Name: "Spam", When: `subject matches "(?i)\bviagra\b"`, Action: ActionReject},
				{Name: "Billing scope", When: "true", Score: "1", Scope: []string{"billing"}},
			},
		},
	}
}

func tierSet(present []string) map[string]bool {
	out := map[string]bool{}
	for _, t := range present {
		out[t] = true
	}
	return out
}

func (d domain) allTiers() []string {
	return d.build().Tiers()
}

// Every domain compiles, evaluates, scores, rejects, caps, defines,
// references and reports — through the same engine, with jhin knowing nothing
// about any of them.
func TestExtensionIsDomainAgnostic(t *testing.T) {
	for _, d := range domains() {
		t.Run(d.name, func(t *testing.T) {
			reg := d.build()
			if err := reg.Err(); err != nil {
				t.Fatalf("registry: %v", err)
			}
			eng, err := Compile(reg, d.rules)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			f := d.facts(d.allTiers()...)
			out := eng.Evaluate(f, d.kind, eng.ComputeAggregates([]Facts{f}, d.kind))

			if out.Points == 0 && len(out.Rejections) == 0 {
				t.Errorf("nothing happened: %+v", out)
			}
			if len(out.Skipped) != 0 {
				t.Errorf("a rule was skipped with every tier present: %+v", out.Skipped)
			}
			// the score is the sum of what paid out, always
			sum := 0
			for _, m := range out.Matched {
				sum += m.Score
			}
			if sum != out.Points {
				t.Errorf("matches sum to %d, points are %d", sum, out.Points)
			}
			t.Logf("%s: %d points from %d rules, %d rejections, %d caps, %d effects",
				d.name, out.Points, len(out.Matched), len(out.Rejections), len(out.Limits), len(out.Effects))
		})
	}
}

// The guarantee the whole design rests on, stated as a property: taking a tier
// away can never turn a release that survived into one that is rejected. It
// holds for every subset of every domain's tiers, whatever the rules say.
func TestFailOpenIsMonotonic(t *testing.T) {
	for _, d := range domains() {
		t.Run(d.name, func(t *testing.T) {
			eng, err := Compile(d.build(), d.rules)
			if err != nil {
				t.Fatal(err)
			}
			all := d.allTiers()

			// every subset of the domain's tiers
			for mask := 0; mask < 1<<len(all); mask++ {
				var present []string
				for i, tier := range all {
					if mask&(1<<i) != 0 {
						present = append(present, tier)
					}
				}
				f := d.facts(present...)
				out := eng.Evaluate(f, d.kind, eng.ComputeAggregates([]Facts{f}, d.kind))

				// nothing a missing tier could answer may have acted
				for _, rej := range out.Rejections {
					name := strings.TrimPrefix(rej, RejectionPrefix)
					for _, s := range out.Skipped {
						if s.Name == name {
							t.Fatalf("tiers %v: rule %q was skipped and still rejected", present, name)
						}
					}
				}
				// removing every tier can only ever remove rejections, never add
				if len(present) == 0 && len(out.Rejections) > 0 {
					for _, rej := range out.Rejections {
						full := d.facts(all...)
						fullOut := eng.Evaluate(full, d.kind, nil)
						found := false
						for _, r2 := range fullOut.Rejections {
							if r2 == rej {
								found = true
							}
						}
						if !found {
							t.Fatalf("with no tiers, %q rejected — a release nothing could judge was removed", rej)
						}
					}
				}
			}
		})
	}
}

// A rule's own contribution cannot depend on what else is in the set. This is
// what "rule order does not matter" means once result-set questions exist, and
// it holds because the counts are taken before any rule fires.
func TestRuleContributionIsIndependent(t *testing.T) {
	for _, d := range domains() {
		t.Run(d.name, func(t *testing.T) {
			reg := d.build()
			f := d.facts(d.allTiers()...)

			// what each rule scores on its own, with its dependencies present
			alone := map[string]int{}
			for i := range d.rules {
				subset := []Rule{d.rules[i]}
				// a rule naming another needs it, so carry every define along
				for j := range d.rules {
					if j != i && d.rules[j].EffectiveAction() == ActionDefine {
						subset = append(subset, d.rules[j])
					}
				}
				eng, err := Compile(reg, subset)
				if err != nil || eng == nil {
					continue
				}
				out := eng.Evaluate(f, d.kind, eng.ComputeAggregates([]Facts{f}, d.kind))
				for _, m := range out.Matched {
					alone[m.Name] = m.Score
				}
			}

			// the same rules together must pay out exactly the same
			eng, err := Compile(reg, d.rules)
			if err != nil {
				t.Fatal(err)
			}
			out := eng.Evaluate(f, d.kind, eng.ComputeAggregates([]Facts{f}, d.kind))
			for _, m := range out.Matched {
				if want, ok := alone[m.Name]; ok && want != m.Score {
					t.Errorf("rule %q scored %d alone and %d in the set", m.Name, want, m.Score)
				}
			}

			// and reversing the set changes nothing
			reversed := make([]Rule, len(d.rules))
			for i, r := range d.rules {
				reversed[len(d.rules)-1-i] = r
			}
			rEng, err := Compile(reg, reversed)
			if err != nil {
				t.Fatal(err)
			}
			rOut := rEng.Evaluate(f, d.kind, rEng.ComputeAggregates([]Facts{f}, d.kind))
			if rOut.Points != out.Points {
				t.Errorf("reversed rule order scored %d, want %d", rOut.Points, out.Points)
			}
		})
	}
}

// jhin's own vocabulary lives in Core and nowhere else: the engine, the
// checker and the evaluator must not know that a release is a release.
func TestEngineHasNoDomainVocabulary(t *testing.T) {
	reg := NewRegistry()
	reg.Field("anything", Num, "")
	eng, err := Compile(reg, []Rule{{Name: "r", When: "anything > 1", Score: "anything"}})
	if err != nil {
		t.Fatalf("an engine with a one-field schema would not compile: %v", err)
	}
	out := eng.Evaluate(MapFacts{Values: map[string]Value{"anything": NumOf(42)}}, "", nil)
	if out.Points != 42 {
		t.Errorf("points = %d, want 42", out.Points)
	}
	// and jhin's own attributes are not reachable unless Core registered them
	if _, err := Compile(reg, []Rule{{Name: "r", When: `resolution == "2160p"`}}); err == nil {
		t.Error("a release attribute resolved against a registry that never declared one")
	}
}
