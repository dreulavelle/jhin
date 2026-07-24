package parser

// Literal prefiltering: most handlers' regexes require some literal substring
// (e.g. "webrip", "atmos", "telugu"). Running ~430 compiled regexes per title
// costs ~1µs each in fixed overhead; a strings.Contains gate skips the ones
// that provably cannot match.
//
// Correctness invariants:
//   - gate literals are derived from the handler's own pattern via
//     regexp/syntax as REQUIRED literals: if none occur (case-insensitively)
//     in the title, the regex cannot match. Literals are lowercased and
//     checked against a haystack lowered with fold_lower, which applies the
//     same simple case folding RE2 uses for (?i) — so the gate stays a
//     necessary condition for both case-sensitive and case-folded matches.
//   - regexes always run against the original (current) title; the lowercased
//     copy exists only for the Contains check.
//   - the haystack is recomputed whenever a handler mutates the title, since
//     a removal can splice two fragments into a new substring.
//
// TestPrefilterEquivalence and FuzzPrefilterEquivalence verify that gating
// never changes any parse result.

import (
	"regexp/syntax"
	"strings"
	"unicode"
)

// prefilter_enabled is a test hook: the equivalence test compares parses with
// gating on and off. Toggling it while parses run concurrently is a data
// race — tests must only flip it between serial parses.
var prefilter_enabled = true

// fold_lower lowercases for the gate haystack using the same simple case
// folding RE2 applies for (?i): runes whose fold orbit contains an ASCII
// letter (e.g. ſ, K) map to that letter, so a gate literal never misses a
// title its regex would match.
func fold_lower(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return strings.Map(fold_lower_rune, s)
		}
	}
	return strings.ToLower(s)
}

func fold_lower_rune(r rune) rune {
	if r < 0x80 {
		if 'A' <= r && r <= 'Z' {
			return r + 32
		}
		return r
	}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		if f < 0x80 {
			if 'A' <= f && f <= 'Z' {
				return f + 32
			}
			return f
		}
	}
	return unicode.ToLower(r)
}

const (
	gate_max_alternatives = 12
	gate_min_literal_len  = 2
	gate_merge_cap        = 16
)

type gateLits struct {
	lits []string // lowercase; matched against the lowercased title
}

func (g *gateLits) hit(lowerTitle string) bool {
	for _, l := range g.lits {
		if strings.Contains(lowerTitle, l) {
			return true
		}
	}
	return false
}

func gate(lits ...string) *gateLits {
	return &gateLits{lits: lits}
}

// derive_gate computes required literals for a pattern, or nil when the
// pattern cannot be gated safely.
func derive_gate(pattern string) *gateLits {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil
	}
	info := required_literals(re.Simplify())
	if info == nil || len(info.lits) == 0 || len(info.lits) > gate_max_alternatives {
		return nil
	}
	for _, l := range info.lits {
		if len(l) < gate_min_literal_len || !is_ascii(l) {
			return nil
		}
	}
	return &gateLits{lits: info.lits}
}

func is_ascii(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

type litInfo struct {
	lits []string
	// exact: lits enumerate the complete text this subexpression can match,
	// so adjacent exact parts may be concatenated via cross-product
	exact bool
}

func is_zero_width(op syntax.Op) bool {
	switch op {
	case syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary, syntax.OpEmptyMatch:
		return true
	}
	return false
}

// required_literals returns a set of literals of which AT LEAST ONE must be
// present for the regex to match, or nil if no such set can be derived.
func required_literals(re *syntax.Regexp) *litInfo {
	switch re.Op {
	case syntax.OpLiteral:
		return &litInfo{lits: []string{strings.ToLower(string(re.Rune))}, exact: true}
	case syntax.OpCapture:
		return required_literals(re.Sub[0])
	case syntax.OpPlus:
		if info := required_literals(re.Sub[0]); info != nil {
			return &litInfo{lits: info.lits}
		}
		return nil
	case syntax.OpRepeat:
		if re.Min >= 1 {
			if info := required_literals(re.Sub[0]); info != nil {
				return &litInfo{lits: info.lits, exact: info.exact && re.Min == 1 && re.Max == 1}
			}
		}
		return nil
	case syntax.OpConcat:
		return concat_literals(re.Sub)
	case syntax.OpAlternate:
		// every branch must contribute, else the gate is not a necessary
		// condition
		out := &litInfo{exact: true}
		for _, sub := range re.Sub {
			info := required_literals(sub)
			if info == nil {
				return nil
			}
			out.lits = append(out.lits, info.lits...)
			out.exact = out.exact && info.exact
			if len(out.lits) > gate_merge_cap {
				return nil
			}
		}
		return out
	default:
		return nil
	}
}

func concat_literals(subs []*syntax.Regexp) *litInfo {
	var best *litInfo
	consider := func(c *litInfo) {
		if c != nil && (best == nil || lit_stronger(c, best)) {
			best = c
		}
	}

	// merge maximal runs of adjacent exact parts via cross-product
	// (zero-width assertions are transparent: they consume no text)
	var run *litInfo
	runAll := true // run covers every non-zero-width child so far
	flush := func() {
		if run != nil {
			consider(run)
		}
		run = nil
	}
	for _, sub := range subs {
		if is_zero_width(sub.Op) {
			continue
		}
		info := required_literals(sub)
		if info == nil || !info.exact {
			flush()
			runAll = false
			consider(info)
			continue
		}
		if run == nil {
			run = &litInfo{lits: info.lits, exact: true}
			continue
		}
		if len(run.lits)*len(info.lits) > gate_merge_cap {
			// the run no longer covers the whole concatenation; its literals
			// stay valid as a gate but must not be treated as exact text
			flush()
			runAll = false
			run = &litInfo{lits: info.lits, exact: true}
			continue
		}
		merged := make([]string, 0, len(run.lits)*len(info.lits))
		for _, a := range run.lits {
			for _, b := range info.lits {
				merged = append(merged, a+b)
			}
		}
		run = &litInfo{lits: merged, exact: true}
	}
	flush()

	if best == nil {
		return nil
	}
	// exactness survives only when a single merged run covered everything
	return &litInfo{lits: best.lits, exact: best.exact && runAll}
}

func gate_min_len(lits []string) int {
	m := 1 << 30
	for _, s := range lits {
		m = min(m, len(s))
	}
	return m
}

func lit_stronger(a, b *litInfo) bool {
	la, lb := gate_min_len(a.lits), gate_min_len(b.lits)
	if la != lb {
		return la > lb
	}
	return len(a.lits) < len(b.lits)
}

func init() {
	for i := range handlers {
		h := &handlers[i]
		if h.Gate != nil {
			// explicit gate provided by the handler definition
			continue
		}
		if h.Pattern == nil {
			continue
		}
		if g := derive_gate(h.Pattern.String()); g != nil {
			h.Gate = g
		}
	}
}
