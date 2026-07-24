package parser

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// Diagnostic + regression guard: how well does the prefilter gate the handler table on the
// golden corpus, and which ungated/hot handlers execute most often?
func TestGateStats(t *testing.T) {
	blob, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(blob, &entries); err != nil {
		t.Fatal(err)
	}
	titles := make([]string, len(entries))
	for i := range entries {
		titles[i] = entries[i].Title
	}
	ungated := 0
	for _, h := range handlers {
		if h.Gate == nil {
			ungated++
		}
	}
	execs := 0
	perHandler := make([]int, len(handlers))
	var hs haystack
	for _, title := range titles {
		hs.scan(title)
		for i, h := range handlers {
			if h.Gate == nil || h.Gate.hit(&hs) {
				execs++
				perHandler[i]++
			}
		}
	}
	avg := float64(execs) / float64(len(titles))
	t.Logf("handlers=%d ungated=%d avg-regex-execs/title=%.1f", len(handlers), ungated, avg)
	// regression guard: the prefilter currently gates the corpus down to ~50
	// regex executions per title; a broken derivation shows up as a jump here
	if avg > 70 {
		t.Errorf("prefilter coverage regressed: %.1f regex executions per title (want < 70)", avg)
	}
	type hot struct {
		field, pat string
		n          int
		gated      bool
	}
	var hots []hot
	for i, h := range handlers {
		if perHandler[i] > len(titles)/2 {
			p := "<nil pattern>"
			if h.Pattern != nil {
				p = h.Pattern.String()
			}
			if len(p) > 60 {
				p = p[:60]
			}
			hots = append(hots, hot{h.Field, p, perHandler[i], h.Gate != nil})
		}
	}
	t.Logf("handlers executing on >50%% of titles: %d", len(hots))
	for _, h := range hots {
		t.Logf("  %-12s gated=%-5v n=%-5d %s", h.field, h.gated, h.n, strings.ReplaceAll(h.pat, "\n", ""))
	}
}

// Diagnostic: rank handlers by raw regex cost over the corpus.
func TestHandlerCost(t *testing.T) {
	blob, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(blob, &entries); err != nil {
		t.Fatal(err)
	}
	type cost struct {
		i     int
		field string
		d     time.Duration
		gated bool
	}
	costs := make([]cost, 0, len(handlers))
	var total time.Duration
	for i := range handlers {
		h := &handlers[i]
		if h.Pattern == nil {
			continue
		}
		start := time.Now()
		for _, e := range entries {
			h.Pattern.FindStringSubmatchIndex(e.Title)
		}
		d := time.Since(start)
		total += d
		costs = append(costs, cost{i, h.Field, d, h.Gate != nil})
	}
	sort.Slice(costs, func(a, b int) bool { return costs[a].d > costs[b].d })
	t.Logf("total raw regex time over corpus (all handlers, ungated): %v", total)
	var top time.Duration
	for _, c := range costs[:40] {
		top += c.d
		p := handlers[c.i].Pattern.String()
		if len(p) > 55 {
			p = p[:55]
		}
		t.Logf("%8v gated=%-5v %-12s %s", c.d.Round(time.Microsecond), c.gated, c.field, strings.ReplaceAll(p, "\n", ""))
	}
	t.Logf("top-40 share: %v (%.0f%%)", top.Round(time.Millisecond), float64(top)/float64(total)*100)
}
