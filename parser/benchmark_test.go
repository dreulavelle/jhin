package parser

import "testing"

// benchCorpus is a mix of real-world release name shapes: plain scene
// releases, web/anime naming, site prefixes, alt-language titles, and
// multi-field UHD releases.
var benchCorpus = []string{
	"Deadpool 2016 1080p BluRay x264 DTS-JYK",
	"The Walking Dead S05E03 720p HDTV x264-ASAP[ettv]",
	"Marvels Agents of S H I E L D S02E05 HDTV x264-KILLERS [eztv]",
	"[SubsPlease] Sousou no Frieren - 28 (1080p) [F1FF71EB].mkv",
	"www.Torrenting.com - Anatomy Of A Fall (2023) 1080p WEBRip x265 10bit AAC",
	"The.Simpsons.S01E01.1080p.BluRay.x265.10bit.AAC5.1-Tigole",
	"Game.of.Thrones.S01-S08.COMPLETE.1080p.WEB-DL.DDP5.1.H.264-GoT",
	"Мстители: Война бесконечности / Avengers: Infinity War (2018) BDRip 1080p",
	"Spider-Man.No.Way.Home.2021.2160p.WEB-DL.DDP5.1.Atmos.HDR.HEVC-EVO[TGx]",
	"One Piece - 1071 (1080p) [Multi-Subs] [Dual-Audio] [x264]",
	"Oppenheimer.2023.IMAX.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.7.1.Atmos-FGT",
	"Friends S01E01",
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, title := range benchCorpus {
			Parse(title)
		}
	}
}

func BenchmarkParseSimple(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Parse("Deadpool 2016 1080p BluRay x264 DTS-JYK")
	}
}

func BenchmarkParseComplex(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Parse("Oppenheimer.2023.IMAX.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.7.1.Atmos-FGT")
	}
}

func BenchmarkPartialParser(b *testing.B) {
	parse := GetPartialParser([]string{"resolution", "year"})
	b.ReportAllocs()
	for b.Loop() {
		parse("Deadpool 2016 1080p BluRay x264 DTS-JYK")
	}
}
