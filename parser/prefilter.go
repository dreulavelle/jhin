package parser

// Literal prefiltering: most handlers' regexes require some literal substring
// (e.g. "webrip", "atmos", "telugu") or, failing that, a character from some
// script range (e.g. Cyrillic season markers) or a short digram (e.g. "s" or
// "e" followed by a digit). Running ~430 compiled regexes per title costs
// ~1µs each in fixed overhead; gating skips the ones that provably cannot
// match.
//
// All gate literals across all handlers are compiled into a single
// Aho-Corasick automaton. One scan of the case-folded title produces a bitset
// of every literal present; each handler's gate is then a couple of bitset
// probes instead of per-handler strings.Contains loops.
//
// Correctness invariants:
//   - gate literals are derived from the handler's own pattern via
//     regexp/syntax as REQUIRED literals: if none occur (case-insensitively)
//     in the title, the regex cannot match. Literals are folded with the same
//     simple case folding RE2 uses for (?i) — so the gate stays a necessary
//     condition for both case-sensitive and case-folded matches.
//   - script gates are derived from character classes whose every rune is
//     non-ASCII: if the title contains no rune from the class, the regex
//     cannot match. Script presence is computed from the ORIGINAL title.
//   - a POSITIVE lookaround in ValidateMatch contributes its own required
//     literals: every accepted match implies that pattern matches somewhere
//     in the title, and a handler whose candidates all fail validation is
//     skipped with no side effects. Negative lookarounds contribute nothing.
//   - regexes always run against the original (current) title; the folded
//     copy exists only for the automaton scan.
//   - the haystack is rescanned whenever a handler mutates the title, since
//     a removal can splice two fragments into a new substring.
//
// TestPrefilterEquivalence and FuzzPrefilterEquivalence verify that gating
// never changes any parse result.

import (
	"regexp/syntax"
	"sort"
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
	gateMaxAlternatives = 64 // max literals per gate group
	gateMinLiteralLen   = 2  // bytes; a multibyte rune alone qualifies
	gateMergeCap        = 64 // max literals in a cross-product merge
	gateMaxGroups       = 4  // max AND-ed factor groups per gate
	classExpandCap      = 14 // max runes when expanding a character class
	maxScriptClasses    = 32 // script presence is a uint32 bitmask
)

// ---------------------------------------------------------------------------
// Script classes: character classes with only non-ASCII runes, gated by
// "title contains at least one rune from the class".
// ---------------------------------------------------------------------------

type scriptClass struct {
	pairs []rune // flattened [lo, hi, lo, hi, ...]
}

var scriptClasses []scriptClass

func (c *scriptClass) contains(r rune) bool {
	for i := 0; i+1 < len(c.pairs); i += 2 {
		if c.pairs[i] <= r && r <= c.pairs[i+1] {
			return true
		}
	}
	return false
}

// registerScript interns a range list and returns its presence bit, or 0 when
// the registry is full.
func registerScript(pairs []rune) uint32 {
	for i, c := range scriptClasses {
		if len(c.pairs) == len(pairs) {
			same := true
			for j := range pairs {
				if c.pairs[j] != pairs[j] {
					same = false
					break
				}
			}
			if same {
				return 1 << uint(i)
			}
		}
	}
	if len(scriptClasses) >= maxScriptClasses {
		return 0
	}
	scriptClasses = append(scriptClasses, scriptClass{pairs: append([]rune(nil), pairs...)})
	return 1 << uint(len(scriptClasses)-1)
}

// scriptPresence reports which registered script classes occur in the
// original title. Pure-ASCII titles short-circuit to 0.
func scriptPresence(title string) uint32 {
	ascii := true
	for i := 0; i < len(title); i++ {
		if title[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return 0
	}
	var mask uint32
	all := uint32(1<<uint(len(scriptClasses))) - 1
	for _, r := range title {
		if r < 0x80 {
			continue
		}
		for i := range scriptClasses {
			bit := uint32(1) << uint(i)
			if mask&bit == 0 && scriptClasses[i].contains(r) {
				mask |= bit
			}
		}
		if mask == all {
			break
		}
	}
	return mask
}

// ---------------------------------------------------------------------------
// Haystack: per-parse scan state shared by every gate.
// ---------------------------------------------------------------------------

type haystack struct {
	folded  []byte // reused across the rescans one parse triggers
	scripts uint32
	hits    []uint64 // literal-ID bitset filled by the automaton scan
}

// appendFoldLower writes foldLower(s) into dst. Pure-ASCII input — the large
// majority of titles — folds byte-wise with no intermediate allocation;
// anything else defers to foldLower so the fold orbit rules stay in one place.
func appendFoldLower(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return append(dst, foldLower(s)...)
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 32
		}
		dst = append(dst, c)
	}
	return dst
}

func (h *haystack) scan(title string) {
	h.folded = appendFoldLower(h.folded[:0], title)
	h.scripts = scriptPresence(title)
	if h.hits == nil {
		h.hits = make([]uint64, acMatcher.bitsetWords())
	} else {
		clear(h.hits)
	}
	acMatcher.scan(h.folded, h.hits)
}

// ---------------------------------------------------------------------------
// Gates
// ---------------------------------------------------------------------------

// gateGroup is one necessary condition: at least one literal (or a rune from
// the script classes) must occur in the title.
type gateGroup struct {
	lits       []string // folded; interned into the automaton at init
	bit        int32    // set by the automaton when any literal occurs
	scriptMask uint32   // alternative: any of these scripts present
}

// gateLits is the AND of its groups: every group's condition must hold, or
// the handler cannot match. Factors of a concatenation are each individually
// required, so AND-ing them is sound and strictly stronger than any single
// factor.
//
// groups is the definition; compile() lowers it into the fixed probe arrays
// the hot path reads, so hit() costs a couple of array loads instead of
// walking a slice of 40-byte structs.
type gateLits struct {
	groups []gateGroup

	n      int32
	word   [gateMaxGroups]int32 // index into haystack.hits; -1 without literals
	mask   [gateMaxGroups]uint64
	script [gateMaxGroups]uint32
}

// compile resolves each group's automaton bit into a word/mask pair. Groups
// past gateMaxGroups are dropped, which only makes the gate more permissive.
func (g *gateLits) compile() {
	g.n = int32(min(len(g.groups), gateMaxGroups))
	for i := 0; i < int(g.n); i++ {
		grp := &g.groups[i]
		if len(grp.lits) > 0 {
			g.word[i] = grp.bit >> 6
			g.mask[i] = 1 << (uint(grp.bit) & 63)
		} else {
			g.word[i] = -1
		}
		g.script[i] = grp.scriptMask
	}
}

// hit reports whether the handler could possibly match.
func (g *gateLits) hit(h *haystack) bool {
	for i := int32(0); i < g.n; i++ {
		if w := g.word[i]; w >= 0 && h.hits[w]&g.mask[i] != 0 {
			continue
		}
		if s := g.script[i]; s != 0 && h.scripts&s != 0 {
			continue
		}
		return false
	}
	return true
}

func gate(lits ...string) *gateLits {
	return &gateLits{groups: []gateGroup{{lits: lits}}}
}

// groupOK vets one factor as a usable gate group.
func groupOK(info *litInfo) bool {
	if info == nil || (len(info.lits) == 0 && info.scriptMask == 0) {
		return false
	}
	if len(info.lits) > gateMaxAlternatives {
		return false
	}
	for _, l := range info.lits {
		if len(l) < gateMinLiteralLen {
			return false
		}
	}
	return true
}

// deriveGate computes required-literal factors for a pattern, or nil when
// the pattern cannot be gated safely. A top-level concatenation yields up to
// gateMaxGroups independent factors, AND-ed together.
func deriveGate(pattern string) *gateLits {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil
	}
	root := re
	for root.Op == syntax.OpCapture {
		root = root.Sub[0]
	}
	var factors []*litInfo
	if root.Op == syntax.OpConcat {
		_, cands := concatLiterals(root.Sub)
		for _, c := range cands {
			if groupOK(c) {
				factors = append(factors, c)
			}
		}
	}
	if len(factors) == 0 {
		if info := requiredLiterals(re); groupOK(info) {
			factors = []*litInfo{info}
		}
	}
	if len(factors) == 0 {
		return nil
	}
	// keep the strongest distinct factors
	sort.SliceStable(factors, func(a, b int) bool { return litStronger(factors[a], factors[b]) })
	g := &gateLits{}
	seen := map[string]bool{}
	for _, f := range factors {
		key := strings.Join(f.lits, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		g.groups = append(g.groups, gateGroup{lits: f.lits, scriptMask: f.scriptMask})
		if len(g.groups) == gateMaxGroups {
			break
		}
	}
	return g
}

// ---------------------------------------------------------------------------
// Literal derivation
// ---------------------------------------------------------------------------

// litInfo describes a requirement every match of a subexpression satisfies:
// at least one of lits occurs in it (folded), or — when scriptMask is set —
// at least one rune from a registered script class occurs.
type litInfo struct {
	lits []string
	// exact: lits enumerate the complete text this subexpression can match,
	// so adjacent exact parts may be concatenated via cross-product
	exact bool
	// head/tail: every match BEGINS/ENDS with an element of the set (nil =
	// unknown). Lets an exact neighbor extend into a variable part: "s"
	// followed by \d{1,4} yields the digrams s0..s9.
	head, tail []string
	scriptMask uint32
}

func isZeroWidth(op syntax.Op) bool {
	switch op {
	case syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary, syntax.OpEmptyMatch:
		return true
	}
	return false
}

func foldString(runes []rune) string {
	var b strings.Builder
	for _, r := range runes {
		b.WriteRune(foldLowerRune(r))
	}
	return b.String()
}

// expandClass enumerates a small character class as folded single-rune
// literals, or nil if the class is too large or infinite.
func expandClass(pairs []rune) []string {
	n := 0
	for i := 0; i+1 < len(pairs); i += 2 {
		n += int(pairs[i+1]-pairs[i]) + 1
		if n > classExpandCap {
			return nil
		}
	}
	seen := make(map[string]bool, n)
	out := make([]string, 0, n)
	for i := 0; i+1 < len(pairs); i += 2 {
		for r := pairs[i]; r <= pairs[i+1]; r++ {
			s := string(foldLowerRune(r))
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

// requiredLiterals returns a requirement for the regex to match, or nil if
// none can be derived.
func requiredLiterals(re *syntax.Regexp) *litInfo {
	switch re.Op {
	case syntax.OpLiteral:
		s := foldString(re.Rune)
		return &litInfo{lits: []string{s}, exact: true, head: []string{s}, tail: []string{s}}
	case syntax.OpCharClass:
		if lits := expandClass(re.Rune); lits != nil {
			return &litInfo{lits: lits, exact: true, head: lits, tail: lits}
		}
		// non-ASCII-only classes gate on script presence
		nonASCII := len(re.Rune) > 0
		for i := 0; i+1 < len(re.Rune); i += 2 {
			if re.Rune[i] < 0x80 {
				nonASCII = false
				break
			}
		}
		if nonASCII {
			if bit := registerScript(re.Rune); bit != 0 {
				return &litInfo{scriptMask: bit}
			}
		}
		return nil
	case syntax.OpCapture:
		return requiredLiterals(re.Sub[0])
	case syntax.OpPlus:
		return repeatLiterals(re.Sub[0], 1, -1)
	case syntax.OpRepeat:
		if re.Min >= 1 {
			return repeatLiterals(re.Sub[0], re.Min, re.Max)
		}
		return nil
	case syntax.OpConcat:
		best, _ := concatLiterals(re.Sub)
		return best
	case syntax.OpAlternate:
		// every branch must contribute, else the gate is not a necessary
		// condition
		out := &litInfo{exact: true}
		heads, tails := true, true
		for _, sub := range re.Sub {
			info := requiredLiterals(sub)
			if info == nil {
				return nil
			}
			out.lits = append(out.lits, info.lits...)
			out.scriptMask |= info.scriptMask
			out.exact = out.exact && info.exact && info.scriptMask == 0
			if info.head == nil {
				heads = false
			} else {
				out.head = append(out.head, info.head...)
			}
			if info.tail == nil {
				tails = false
			} else {
				out.tail = append(out.tail, info.tail...)
			}
			if len(out.lits) > gateMergeCap || len(out.head) > gateMergeCap || len(out.tail) > gateMergeCap {
				return nil
			}
		}
		if !heads {
			out.head = nil
		}
		if !tails {
			out.tail = nil
		}
		return out
	default:
		return nil
	}
}

// repeatLiterals handles sub{min,max} with min >= 1: the requirement of one
// iteration holds, and every match starts/ends with an iteration's start/end.
func repeatLiterals(sub *syntax.Regexp, mn, mx int) *litInfo {
	info := requiredLiterals(sub)
	if info == nil {
		return nil
	}
	out := &litInfo{
		lits:       info.lits,
		scriptMask: info.scriptMask,
		exact:      info.exact && info.scriptMask == 0 && mn == 1 && mx == 1,
	}
	if info.scriptMask == 0 {
		if info.exact {
			out.head, out.tail = info.lits, info.lits
		} else {
			out.head, out.tail = info.head, info.tail
		}
	}
	return out
}

// cross concatenates every pair, or nil past the merge cap.
func cross(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 || len(a)*len(b) > gateMergeCap {
		return nil
	}
	out := make([]string, 0, len(a)*len(b))
	for _, x := range a {
		for _, y := range b {
			out = append(out, x+y)
		}
	}
	return out
}

// concatLiterals returns the strongest single factor plus every candidate
// factor found. Each candidate is independently a necessary condition of the
// whole concatenation, so callers may AND them.
func concatLiterals(subs []*syntax.Regexp) (*litInfo, []*litInfo) {
	var best *litInfo
	var cands []*litInfo
	consider := func(c *litInfo) {
		if c == nil || (len(c.lits) == 0 && c.scriptMask == 0) {
			return
		}
		cands = append(cands, c)
		if best == nil || litStronger(c, best) {
			best = c
		}
	}

	// merge maximal runs of adjacent exact parts via cross-product
	// (zero-width assertions are transparent: they consume no text)
	var run *litInfo
	var pendingTail []string // tail set of the non-exact part just before run
	var headSet, tailSet []string
	firstChild := true
	runFromStart := false // current run began at the first non-zero-width child
	runAll := true        // run covers every non-zero-width child so far
	flush := func(atEnd bool) {
		if run != nil {
			consider(run)
			if merged := cross(pendingTail, run.lits); merged != nil {
				consider(&litInfo{lits: merged})
			}
			if runFromStart {
				headSet = run.lits
			}
			if atEnd && runAll {
				tailSet = run.lits
			} else if atEnd {
				tailSet = run.lits
			}
		}
		run = nil
		pendingTail = nil
		runFromStart = false
	}
	for _, sub := range subs {
		if isZeroWidth(sub.Op) {
			continue
		}
		info := requiredLiterals(sub)
		if info != nil && info.exact && info.scriptMask == 0 {
			if run == nil {
				run = &litInfo{lits: info.lits, exact: true}
				runFromStart = firstChild
			} else if merged := cross(run.lits, info.lits); merged != nil {
				run = &litInfo{lits: merged, exact: true}
			} else {
				// the run no longer covers the whole concatenation; its
				// literals stay valid as a gate but are no longer exact text
				flush(false)
				runAll = false
				run = &litInfo{lits: info.lits, exact: true}
			}
			firstChild = false
			continue
		}
		// non-exact child: try extending the run into its head
		if run != nil && info != nil && len(info.head) > 0 {
			if merged := cross(run.lits, info.head); merged != nil {
				consider(&litInfo{lits: merged})
			}
		}
		flush(false)
		runAll = false
		consider(info)
		if info != nil {
			pendingTail = info.tail
			if firstChild {
				headSet = info.head
			}
			tailSet = info.tail
		} else {
			pendingTail = nil
			tailSet = nil
		}
		firstChild = false
	}
	flush(true)

	if best == nil {
		return nil, nil
	}
	// exactness survives only when a single merged run covered everything
	out := &litInfo{lits: best.lits, scriptMask: best.scriptMask, exact: best.exact && runAll}
	if best.scriptMask == 0 {
		out.head, out.tail = headSet, tailSet
		if out.exact {
			out.head, out.tail = out.lits, out.lits
		}
	}
	return out, cands
}

func gateMinLen(lits []string) int {
	m := 1 << 30
	for _, s := range lits {
		m = min(m, len(s))
	}
	return m
}

func litStronger(a, b *litInfo) bool {
	// a pure script requirement is the weakest possible gate
	if (len(a.lits) == 0) != (len(b.lits) == 0) {
		return len(a.lits) > 0
	}
	la, lb := gateMinLen(a.lits), gateMinLen(b.lits)
	if la != lb {
		return la > lb
	}
	return len(a.lits) < len(b.lits)
}

// ---------------------------------------------------------------------------
// Aho-Corasick automaton over every gate literal
// ---------------------------------------------------------------------------

type acAutomaton struct {
	// root transitions are dense for speed; other nodes are sparse ranges
	// into the flat edge arrays, sorted by byte
	root      [256]int32
	edgeStart []int32 // per node: first edge index
	edgeEnd   []int32
	edgeByte  []byte
	edgeNext  []int32
	fail      []int32
	// gate indexes satisfied at each node (suffix-merged), CSR-packed:
	// node u owns outGates[outOff[u]:outOff[u+1]]. Most nodes own none, and
	// the two offsets sit adjacent in one array — the per-byte check is two
	// neighbouring loads instead of a slice header.
	outOff   []int32
	outGates []int32
	numGates int
}

var acMatcher *acAutomaton

func (a *acAutomaton) bitsetWords() int { return (a.numGates + 63) / 64 }

// scan folds nothing — the haystack must already be folded — and sets the bit
// of every literal that occurs.
func (a *acAutomaton) scan(s []byte, hits []uint64) {
	state := int32(0)
	for _, b := range s {
		for {
			var next int32 = -1
			if state == 0 {
				next = a.root[b]
			} else {
				for e := a.edgeStart[state]; e < a.edgeEnd[state]; e++ {
					if a.edgeByte[e] == b {
						next = a.edgeNext[e]
						break
					}
				}
			}
			if next >= 0 {
				state = next
				break
			}
			if state == 0 {
				break
			}
			state = a.fail[state]
		}
		for e, end := a.outOff[state], a.outOff[state+1]; e < end; e++ {
			id := a.outGates[e]
			hits[id>>6] |= 1 << (uint(id) & 63)
		}
	}
}

// buildAutomaton interns each unique literal, maps it to every gate that
// lists it, and constructs the trie with BFS failure links and suffix-merged
// outputs. Matching a literal sets the bit of each gate it belongs to, so a
// gate probe is O(1) no matter how many literals it has.
func buildAutomaton(gates []*gateLits) *acAutomaton {
	ids := map[string]int32{}
	var lits []string
	var litGates [][]int32
	nextBit := int32(0)
	for _, g := range gates {
		for gi := range g.groups {
			grp := &g.groups[gi]
			grp.bit = nextBit
			nextBit++
			for _, l := range grp.lits {
				f := foldLower(l)
				id, ok := ids[f]
				if !ok {
					id = int32(len(lits))
					ids[f] = id
					lits = append(lits, f)
					litGates = append(litGates, nil)
				}
				litGates[id] = append(litGates[id], grp.bit)
			}
		}
	}

	// trie construction with temporary per-node maps
	type node struct {
		next map[byte]int32
		out  []int32
	}
	nodes := []*node{{next: map[byte]int32{}}}
	for id, lit := range lits {
		cur := int32(0)
		for i := 0; i < len(lit); i++ {
			b := lit[i]
			nxt, ok := nodes[cur].next[b]
			if !ok {
				nxt = int32(len(nodes))
				nodes[cur].next[b] = nxt
				nodes = append(nodes, &node{next: map[byte]int32{}})
			}
			cur = nxt
		}
		nodes[cur].out = append(nodes[cur].out, int32(id))
	}

	a := &acAutomaton{
		numGates:  int(nextBit),
		fail:      make([]int32, len(nodes)),
		edgeStart: make([]int32, len(nodes)),
		edgeEnd:   make([]int32, len(nodes)),
	}
	// per-node outputs during construction; the BFS reads a node's failure
	// target randomly, so they stay indexable until the final CSR pack
	nodeOut := make([][]int32, len(nodes))
	for i := range a.root {
		a.root[i] = -1
	}

	// BFS: failure links and suffix-merged outputs
	queue := make([]int32, 0, len(nodes))
	for b, n := range nodes[0].next {
		a.root[b] = n
		a.fail[n] = 0
		queue = append(queue, n)
	}
	toGates := func(litIDs []int32, inherited []int32) []int32 {
		if len(litIDs) == 0 && len(inherited) == 0 {
			return nil
		}
		seen := map[int32]bool{}
		var out []int32
		for _, g := range inherited {
			if !seen[g] {
				seen[g] = true
				out = append(out, g)
			}
		}
		for _, id := range litIDs {
			for _, g := range litGates[id] {
				if !seen[g] {
					seen[g] = true
					out = append(out, g)
				}
			}
		}
		return out
	}
	nodeOut[0] = toGates(nodes[0].out, nil)
	for qi := 0; qi < len(queue); qi++ {
		u := queue[qi]
		nodeOut[u] = toGates(nodes[u].out, nodeOut[a.fail[u]])
		for b, v := range nodes[u].next {
			f := a.fail[u]
			for {
				if f == 0 {
					if r := a.root[b]; r >= 0 && r != v {
						a.fail[v] = r
					} else {
						a.fail[v] = 0
					}
					break
				}
				found := int32(-1)
				for c, w := range nodes[f].next {
					if c == b {
						found = w
						break
					}
				}
				if found >= 0 {
					a.fail[v] = found
					break
				}
				f = a.fail[f]
			}
			queue = append(queue, v)
		}
	}

	// flatten sparse edges, sorted for deterministic layout
	for u := 1; u < len(nodes); u++ {
		a.edgeStart[u] = int32(len(a.edgeByte))
		bs := make([]int, 0, len(nodes[u].next))
		for b := range nodes[u].next {
			bs = append(bs, int(b))
		}
		sort.Ints(bs)
		for _, b := range bs {
			a.edgeByte = append(a.edgeByte, byte(b))
			a.edgeNext = append(a.edgeNext, nodes[u].next[byte(b)])
		}
		a.edgeEnd[u] = int32(len(a.edgeByte))
	}

	// pack outputs into CSR; outOff has one trailing entry so the scan can
	// read outOff[state+1] without a bounds special case
	a.outOff = make([]int32, len(nodes)+1)
	for u := 0; u < len(nodes); u++ {
		a.outOff[u] = int32(len(a.outGates))
		a.outGates = append(a.outGates, nodeOut[u]...)
	}
	a.outOff[len(nodes)] = int32(len(a.outGates))
	return a
}

func mustParse(pattern string) *syntax.Regexp {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return &syntax.Regexp{Op: syntax.OpNoMatch}
	}
	return re
}

// andImplied strengthens a handler's gate with the required literals of its
// positive lookarounds. A handler whose every candidate match fails
// validation is skipped without side effects, so a lookaround's literals are
// as necessary as the pattern's own — and for patterns too generic to gate
// (bit depth's `(?:8|10|12)`), the lookaround is the only usable condition.
func andImplied(g *gateLits, implied []string) *gateLits {
	for _, pattern := range implied {
		lg := deriveGate(pattern)
		if lg == nil {
			continue
		}
		if g == nil {
			g = &gateLits{}
		}
		for _, grp := range lg.groups {
			if len(g.groups) >= gateMaxGroups {
				break
			}
			g.groups = append(g.groups, grp)
		}
	}
	return g
}

func init() {
	var gates []*gateLits
	for i := range handlers {
		h := &handlers[i]
		if h.Gate == nil && h.Pattern != nil {
			h.Gate = deriveGate(h.Pattern.String())
		}
		// lookaround literals apply only where a Pattern gates the handler;
		// ValidateMatch is never consulted on the Process-only path
		if h.Pattern != nil && h.ValidateMatch != nil {
			h.Gate = andImplied(h.Gate, h.ValidateMatch.implied)
		}
		if h.Gate == nil && h.Field == "adult" && h.Process != nil {
			// the adult handler is pure keyword containment; its keyword
			// list is the exact necessary condition
			h.Gate = gate(adultKeywords()...)
		}
		if h.Gate == nil && h.Field == "scene" && h.Process != nil {
			// customScene fires only on a sceneWebRegex or sceneGroupsRegex
			// match; the union of their required literals is a necessary
			// condition
			iw := requiredLiterals(mustParse(sceneWebRegex.String()))
			ig := requiredLiterals(mustParse(sceneGroupsRegex.String()))
			if groupOK(iw) && groupOK(ig) && iw.scriptMask == 0 && ig.scriptMask == 0 {
				h.Gate = gate(append(iw.lits, ig.lits...)...)
			}
		}
		if h.Gate != nil {
			gates = append(gates, h.Gate)
		}
	}
	acMatcher = buildAutomaton(gates)
	// buildAutomaton assigns each group its bit; only then can a gate be
	// lowered to word/mask probes
	for _, g := range gates {
		g.compile()
	}
}
