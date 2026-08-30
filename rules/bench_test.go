package rules

import "testing"

// A realistic set: the worked examples, as one profile.
const benchRules = `
UHD T1: define if group in ["FraMeSToR", "W4NK3R", "HiFi", "Positive"]
Seeders: score min(grabs, 200) * 15 if grabs > 0
Freshness: score max(0, 500 - ageDays * 5) if true
Sweet spot: score 2000 - abs(sizeGB - 4) * 300 if sizeGB > 0
Trusted 4K: score 3000 if resolution == "2160p" and matched("UHD T1")
IMAX: score 2000 if releaseName matches "(?i)\bIMAX\b"
Atmos: score 800 if "atmos" in traits
Dual audio: score 1200 if "dual_audio" in traits
Oversized: reject if sizeGB > 12 and resolution != "2160p"
DV without fallback: reject if dolbyVision and not hdrFallback
Real 10-bit: score 400 if probed.bitDepth >= 10
Best 3 per flavour: keep 3 per resolution + " " + quality if true
`

func benchEngine(b *testing.B) *Engine {
	b.Helper()
	rs, err := ParseText(benchRules)
	if err != nil {
		b.Fatal(err)
	}
	eng, err := Compile(testRegistry(), rs)
	if err != nil {
		b.Fatal(err)
	}
	return eng
}

func benchFacts() MapFacts {
	return facts(map[string]Value{
		"resolution":      StrOf("2160p"),
		"quality":         StrOf("BluRay"),
		"group":           StrOf("FraMeSToR"),
		"releaseName":     StrOf("Movie.2020.2160p.UHD.BluRay.REMUX.IMAX.DV.HDR10.TrueHD.Atmos-FraMeSToR"),
		"traits":          StrListOf([]string{"remux", "hevc", "10bit", "atmos", "dolby_vision"}),
		"hdr":             StrListOf([]string{"DV", "HDR10"}),
		"dolbyVision":     BoolOf(true),
		"hdrFallback":     BoolOf(true),
		"sizeGB":          NumOf(58),
		"ageDays":         NumOf(12),
		"grabs":           NumOf(140),
		"probed.bitDepth": NumOf(10),
	})
}

func BenchmarkEvaluate(b *testing.B) {
	eng := benchEngine(b)
	f := benchFacts()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = eng.Evaluate(f, "movie", nil)
	}
}

// The condition alone, which is what runs for every rule that does not fire.
func BenchmarkCondition(b *testing.B) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "r", When: `resolution == "2160p" and "remux" in traits and sizeGB > 12`, Score: "100"},
	})
	if err != nil {
		b.Fatal(err)
	}
	f := benchFacts()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = eng.Evaluate(f, "", nil)
	}
}

func BenchmarkCompile(b *testing.B) {
	rs, err := ParseText(benchRules)
	if err != nil {
		b.Fatal(err)
	}
	reg := testRegistry()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := Compile(reg, rs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAggregates(b *testing.B) {
	eng, err := Compile(testRegistry(), []Rule{
		{Name: "r", When: `upscaled and exists(resolution == "2160p" and "remux" in traits)`, Action: ActionReject},
		{Name: "s", When: `count(resolution == "2160p") < 3`, Score: "500"},
	})
	if err != nil {
		b.Fatal(err)
	}
	set := make([]Facts, 100)
	for i := range set {
		set[i] = benchFacts()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = eng.ComputeAggregates(set)
	}
}
