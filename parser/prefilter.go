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
//     checked against a haystack lowered with foldLower, which applies the
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

// prefilterEnabled is a test hook: the equivalence test compares parses with
// gating on and off. Toggling it while parses run concurrently is a data
// race — tests must only flip it between serial parses.
var prefilterEnabled = true

// foldLower lowercases for the gate haystack using the same simple case
// folding RE2 applies for (?i): runes whose fold orbit contains an ASCII
// letter (e.g. ſ, K) map to that letter, so a gate literal never misses a
// title its regex would match.
func foldLower(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return strings.Map(foldLowerRune, s)
		}
	}
	return strings.ToLower(s)
}

func foldLowerRune(r rune) rune {
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
	gateMaxAlternatives = 12
	gateMinLiteralLen   = 2
	gateMergeCap        = 16
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

// deriveGate computes required literals for a pattern, or nil when the
// pattern cannot be gated safely.
func deriveGate(pattern string) *gateLits {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil
	}
	info := requiredLiterals(re.Simplify())
	if info == nil || len(info.lits) == 0 || len(info.lits) > gateMaxAlternatives {
		return nil
	}
	for _, l := range info.lits {
		if len(l) < gateMinLiteralLen || !isASCII(l) {
			return nil
		}
	}
	return &gateLits{lits: info.lits}
}

func isASCII(s string) bool {
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

func isZeroWidth(op syntax.Op) bool {
	switch op {
	case syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary, syntax.OpEmptyMatch:
		return true
	}
	return false
}

// requiredLiterals returns a set of literals of which AT LEAST ONE must be
// present for the regex to match, or nil if no such set can be derived.
func requiredLiterals(re *syntax.Regexp) *litInfo {
	switch re.Op {
	case syntax.OpLiteral:
		return &litInfo{lits: []string{strings.ToLower(string(re.Rune))}, exact: true}
	case syntax.OpCapture:
		return requiredLiterals(re.Sub[0])
	case syntax.OpPlus:
		if info := requiredLiterals(re.Sub[0]); info != nil {
			return &litInfo{lits: info.lits}
		}
		return nil
	case syntax.OpRepeat:
		if re.Min >= 1 {
			if info := requiredLiterals(re.Sub[0]); info != nil {
				return &litInfo{lits: info.lits, exact: info.exact && re.Min == 1 && re.Max == 1}
			}
		}
		return nil
	case syntax.OpConcat:
		return concatLiterals(re.Sub)
	case syntax.OpAlternate:
		// every branch must contribute, else the gate is not a necessary
		// condition
		out := &litInfo{exact: true}
		for _, sub := range re.Sub {
			info := requiredLiterals(sub)
			if info == nil {
				return nil
			}
			out.lits = append(out.lits, info.lits...)
			out.exact = out.exact && info.exact
			if len(out.lits) > gateMergeCap {
				return nil
			}
		}
		return out
	default:
		return nil
	}
}

func concatLiterals(subs []*syntax.Regexp) *litInfo {
	var best *litInfo
	consider := func(c *litInfo) {
		if c != nil && (best == nil || litStronger(c, best)) {
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
		if isZeroWidth(sub.Op) {
			continue
		}
		info := requiredLiterals(sub)
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
		if len(run.lits)*len(info.lits) > gateMergeCap {
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

func gateMinLen(lits []string) int {
	m := 1 << 30
	for _, s := range lits {
		m = min(m, len(s))
	}
	return m
}

func litStronger(a, b *litInfo) bool {
	la, lb := gateMinLen(a.lits), gateMinLen(b.lits)
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
		if g := deriveGate(h.Pattern.String()); g != nil {
			h.Gate = g
		}
	}
}
