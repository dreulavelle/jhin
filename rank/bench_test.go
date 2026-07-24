package rank

import (
	"encoding/json"
	"os"
	"testing"
)

func benchTitles(b testing.TB) []string {
	blob, err := os.ReadFile("../parser/testdata/golden.json")
	if err != nil {
		b.Skip("golden corpus unavailable")
	}
	var cases []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(blob, &cases); err != nil {
		b.Fatal(err)
	}
	titles := make([]string, len(cases))
	for i, c := range cases {
		titles[i] = c.Title
	}
	return titles
}

// BenchmarkRankAll measures the full pipeline (parse + evaluate) over the
// 1,133-title golden corpus in parallel.
func BenchmarkRankAll(b *testing.B) {
	titles := benchTitles(b)
	r, err := New(Default())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r.RankAll(titles)
	}
	b.ReportMetric(float64(len(titles)), "titles/op")
}

// BenchmarkEvaluate isolates evaluation (no parsing).
func BenchmarkEvaluate(b *testing.B) {
	r, err := New(Default())
	if err != nil {
		b.Fatal(err)
	}
	t := r.Rank("Movie.2020.2160p.BluRay.REMUX.DV.HDR.TrueHD.7.1.Atmos-GRP")
	opt := RankOptions{}
	b.ReportAllocs()
	for b.Loop() {
		tt := Torrent{Raw: t.Raw, Data: t.Data}
		r.evaluate(&tt, &opt)
	}
}
