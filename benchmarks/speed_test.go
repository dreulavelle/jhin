package benchmarks

// Speed: every library parses the same 1,156-title corpus round-robin, so
// per-op cost is the mean over a realistic mix of easy and hostile titles,
// not a cherry-picked happy path. Panics are caught per call — uniformly for
// every library, so the (small, constant) defer cost cancels out.

import (
	"testing"

	"github.com/dreulavelle/jhin/parser"
)

var sinkParseAll []*parser.Result

func corpusTitles(tb testing.TB) []string {
	tb.Helper()
	entries, err := LoadCorpus("../parser/testdata/golden.json")
	if err != nil {
		tb.Fatal(err)
	}
	titles := make([]string, len(entries))
	for i := range entries {
		titles[i] = entries[i].Title
	}
	return titles
}

// BenchmarkParseAll measures jhin's batch API: wall-clock per title when the
// corpus is parsed through ParseAll's worker pool. No competitor offers a
// batch API, so this row is reported separately — any parser can be sharded
// across goroutines by the caller; this measures what jhin does for you.
func BenchmarkParseAll(b *testing.B) {
	titles := corpusTitles(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		sinkParseAll = parser.ParseAll(titles)
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(titles)), "ns/title")
}

func BenchmarkParse(b *testing.B) {
	titles := corpusTitles(b)
	for i := range Libraries {
		lib := &Libraries[i]
		b.Run(lib.Name, func(b *testing.B) {
			call := func(title string) {
				defer func() { _ = recover() }()
				lib.Bench(title)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				call(titles[i%len(titles)])
			}
		})
	}
}
