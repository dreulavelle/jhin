[![CI](https://img.shields.io/github/actions/workflow/status/dreulavelle/jhin/ci.yml?branch=main&label=CI&style=for-the-badge)](https://github.com/dreulavelle/jhin/actions/workflows/ci.yml)
[![Go Reference](https://img.shields.io/badge/go-reference-%23007d9c?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/dreulavelle/jhin)
[![License](https://img.shields.io/github/license/dreulavelle/jhin?style=for-the-badge)](https://github.com/dreulavelle/jhin/blob/main/LICENSE)

# Jhin

All-in-one Go library for torrent release names: **parsing**, **ranking**,
**filtering**, and **sorting** — built for high accuracy and high throughput,
with **zero runtime dependencies**.

Jhin is the Go successor to [PTT](https://github.com/dreulavelle/PTT) and
[rank-torrent-name](https://github.com/dreulavelle/rank-torrent-name),
unified into one library:

- **`jhin/parser`** extracts 46 metadata fields from a release name in ~90µs.
  Accuracy is contract-tested: a 1,156-title golden corpus verifies
  byte-identical output against the Python PTT parser.
- **`jhin/rank`** scores, filters, and sorts releases against a declarative
  user profile — per-attribute policies, regex gates, resolution and language
  rules — evaluated in parallel.

## Install

```sh
go get github.com/dreulavelle/jhin
```

## Quick start: parsing

```go
package main

import (
	"fmt"

	"github.com/dreulavelle/jhin"
)

func main() {
	r := jhin.Parse("The.Witcher.S01-S03.COMPLETE.2160p.NF.WEB-DL.DDP5.1.Atmos.DV.HDR.HEVC.10bit.MULTi-Kitsune[TGx]")

	fmt.Println(r.Title)      // "The Witcher"
	fmt.Println(r.Seasons)    // [1 2 3]
	fmt.Println(r.Resolution) // "2160p"
	fmt.Println(r.Quality)    // "WEB-DL"
	fmt.Println(r.HDR)        // ["DV" "HDR"]
	fmt.Println(r.Audio)      // ["Atmos" "Dolby Digital Plus"]
	fmt.Println(r.Network)    // "Netflix"
	fmt.Println(r.Group)      // "Kitsune"
}
```

`Result` has ~46 fields (see the [reference](#result-reference) below) and
marshals straight to JSON. Useful extras:

```go
r.Normalize()                 // canonical forms: 2160p→4k, codec avc→AVC, ...
r.LanguageNames()             // ["fr"] → ["French"]

parser.ParseAll(titles)       // parallel batch, index-aligned with input
parser.ExtractSeasons(title)  // just the season numbers
parser.GetPartialParser([]string{"resolution", "year"}) // few-field fast path
```

## Quick start: ranking, filtering & sorting

The `rank` package answers three questions for every release: *how good is
it* (rank), *am I allowed to grab it* (fetch + rejection reasons), and *in
what order to output them* (sort) — releases are bucketed by resolution with
higher resolutions on top, ranked descending within each bucket.

```go
package main

import (
	"fmt"

	"github.com/dreulavelle/jhin/rank"
)

func main() {
	ranker, err := rank.New(rank.Default())
	if err != nil {
		panic(err)
	}

	titles := []string{
		"Movie.2020.2160p.BluRay.REMUX.DV.TrueHD.7.1-GRP",
		"Movie.2020.1080p.WEB-DL.DDP5.1.H.264-GRP",
		"Movie.2020.CAM.x264-TRASH",
	}

	// Index-aligned with the input — nothing is reordered or dropped.
	// (rank.Entry + ranker.RankEntries carries per-release infohashes.)
	torrents := ranker.RankAll(titles)
	for _, t := range torrents {
		fmt.Printf("%6d fetch=%-5v %v %s\n", t.Rank, t.Fetch, t.Rejections, t.Raw)
	}

	// Sorting is separate and explicit: resolution bucket, then rank by
	// default — or compose your own chain via SortOptions.Criteria.
	best := ranker.Sort(torrents, rank.SortOptions{
		FetchableOnly: true,
		BucketLimit:   5, // top 5 per resolution bucket
	})
	fmt.Println("best:", best[0].Raw)
}
```

### Profiles

Everything tunable lives in one declarative, JSON-serializable `Profile`:

```go
p := rank.Default()

// Per-attribute policy: may it be fetched, and what is it worth?
p.Attributes = map[rank.Attr]rank.Policy{
	rank.AttrRemux:       {Fetch: true, Rank: 25000},
	rank.AttrDolbyVision: {Fetch: false},           // veto DV entirely
}

// Regex gates against the raw title ("/pat/" = case-sensitive).
p.Require = []string{`\b(2160p|1080p)\b`}
p.Exclude = []string{`\bHDCAM\b`}
p.Preferred = []string{`\bIMAX\b`} // matching adds Options.PreferredBonus

// Resolution and language rules. Default enables 4K/1440p/1080p/720p;
// disable what you don't want, or reorder preference without banning:
p.Resolutions[rank.Res2160p] = false
p.ResolutionOrder = []rank.Resolution{rank.Res1080p, rank.Res2160p, rank.Res720p}
p.Languages.Exclude = []string{"ru"}       // codes or groups: anime/common/all
p.Languages.Preferred = []string{"en"}

// Weighted keywords: additive scores without vetoes.
p.PatternRanks = []rank.PatternRank{
	{Pattern: `\bIMAX\b`, Rank: 500},
	{Pattern: `\bHDCAM\b`, Rank: -2000},
}

p.Save("profile.json")                      // and rank.Load("profile.json")

ranker, _ := rank.New(p)
```

### Pinning to a specific movie or show

```go
t := ranker.Rank(raw, rank.RankOptions{
	TargetTitle: "The Matrix",
	Aliases:     []string{"Matrix"},
})
// t.TitleRatio holds the similarity; below Options.TitleThreshold (0.85)
// the release is rejected with "title_mismatch".
```

Standalone helpers: `rank.TitleMatch(a, b, threshold, aliases...)`,
`rank.Similarity(a, b)`, `rank.Normalize(title)`. For debugging a score,
`ranker.Explain(&torrent)` returns the per-clause breakdown — every point
traces to an attribute, pattern, or preference in the profile.

## CLI

```sh
go install github.com/dreulavelle/jhin/cmd/jhin@latest

jhin parse --pretty "The.Matrix.1999.1080p.BluRay.x264"
```

## Performance

Benchmarked on a Ryzen 9 5900HX (see `benchmarks/`):

| Operation | Time | Notes |
|---|---|---|
| Parse (simple title) | ~92µs | 68 allocs |
| Parse (corpus mean, 1,156 mixed titles) | ~181µs | easy and hostile titles alike |
| Batch parse | ~43µs/title | `ParseAll` across 8 threads — 100k titles in ~4.5s |

The parser runs an ordered table of 424 handlers behind a literal prefilter:
each handler's regex is analyzed at init to derive required substrings, and
handlers that provably cannot match are skipped. Equivalence is enforced by
tests and fuzzing — the prefilter can never change a result.

## Accuracy

`parser/testdata/golden.json` pins the expected output for 1,156 real-world
release names across every field — the corpus was generated by the original
Python PTT parser and jhin reproduces it byte-for-byte. Any behavioral
regression fails CI.

## How it compares

Measured 2026-07-24 on the 1,156-title corpus; full methodology, raw
results, disclosure, and reproduction steps in
[`benchmarks/`](benchmarks/README.md). Accuracy is scored per field, only on
the 9 fields every library claims, after neutral vocabulary normalization.

| Library | Accuracy (9 shared fields) | Speed (per title, serial) | Fields |
|---|---|---|---|
| **jhin** (Go) | **100%**¹ | 181µs (43µs batched) | 46 |
| [ProfChaos/torrent-name-parser](https://github.com/ProfChaos/torrent-name-parser) (Go) | 78.8% | 64µs | 28 |
| [middelink/go-parse-torrent-name](https://github.com/middelink/go-parse-torrent-name) (Go, unmaintained) | 70.1% | 38µs | 22 |
| [razsteinmetz/go-ptn](https://github.com/razsteinmetz/go-ptn) (Go) | 67.4% | 45µs | 26 |
| [parse-torrent-title](https://github.com/clement-escolano/parse-torrent-title) (JS) | not scored² | 17µs | ~20 |
| [PTT](https://github.com/dreulavelle/PTT) (Python) | 100%¹ | 595µs | 46 |
| [guessit](https://github.com/guessit-io/guessit) (Python) | not scored² | 4,430µs | ~30 |

¹ The gold labels are generated by Python PTT, and jhin is contract-tested
byte-identical to it — so both score 100% by construction. The table's real
content is the other columns and the other rows.
² Different output schema; no honest cross-vocabulary accuracy score without
a shared gold standard, so speed only.

The other Go parsers are faster per call because they do less: ~30 regexes
filling ~20 scalar fields versus jhin's 424 handlers and 46 fields
(multi-season packs, episode ranges, 60+ languages, HDR, editions, trash
detection). The accuracy column is what the extra microseconds buy.

## `Result` Reference

Field semantics match [PTT](https://github.com/dreulavelle/PTT) 1.8.5
(main@88429bb90ace) exactly, golden-verified.

- **Adult** (`bool`): adult-content detection (keyword list)
- **Audio** (`[]string`): `DTS Lossless`, `DTS Lossy`, `Atmos`, `TrueHD`, `FLAC`, `Dolby Digital Plus`, `Dolby Digital`, `AAC`, `PCM`, `OPUS`, `MP3`, `HQ Clean Audio`
- **BitDepth** (`string`): `8bit`, `10bit`, `12bit`
- **Bitrate** (`string`): e.g. `448kbps`
- **Channels** (`[]string`): `2.0`, `5.1`, `7.1`, `stereo`, `mono`
- **Codec** (`string`): `avc`, `hevc`, `av1`, `xvid`, `mpeg` (normalized: `AVC`, `HEVC`, ...)
- **Commentary** / **Complete** / **Convert** / **Documentary** / **Dubbed** (`bool`)
- **Container** (`string`): `mkv`, `avi`, `mp4`, ...
- **Country** (`string`): `US`, `UK`, `AU`, `NZ`, `CA`
- **Date** (`string`): `YYYY-MM-DD`
- **Edition** (`string`): `Anniversary Edition`, `Director's Cut`, `Extended Edition`, `IMAX`, ...
- **EpisodeCode** (`string`): 8-char CRC code
- **Episodes** / **Seasons** / **Volumes** (`[]int`)
- **Extension** (`string`): file extension
- **Extras** (`[]string`): `Featurette`, `Sample`, `Trailer`, `NCED`, `NCOP`, ...
- **Group** (`string`): release group
- **HDR** (`[]string`): `DV`, `HDR10+`, `HDR`, `SDR`
- **Hardcoded** (`bool`)
- **Languages** (`[]string`): ISO 639-1 codes (`en`, `ja`, `zh`, ...) plus `multi subs`, `multi audio`, `dual audio`
- **Network** (`string`): `Netflix`, `Amazon`, `HBO`, ...
- **PPV** / **Proper** / **Remastered** / **Repack** / **Retail** (`bool`)
- **Quality** (`string`): `WEB`, `WEB-DL`, `WEBRip`, `BluRay`, `BluRay REMUX`, `HDTV`, `CAM`, `TeleSync`, `DVDRip`, ...
- **Region** (`string`): `R0`-`R9`
- **Resolution** (`string`): `2160p`, `1440p`, `1080p`, `720p`, `480p`, ... (Normalize() maps to `4k`/`2k`)
- **Scene** (`bool`): scene-release detection
- **Site** (`string`): source website
- **Size** (`string`): e.g. `2.3GB`
- **Subbed** (`bool`)
- **ThreeD** (`bool`): 3D release
- **Title** (`string`): cleaned title
- **Torrent** / **Trash** / **Uncensored** / **Unrated** / **Upscaled** (`bool`)
- **Year** (`string`): `YYYY` or `YYYY-YYYY`

## Acknowledgement

- [dreulavelle/PTT](https://github.com/dreulavelle/PTT)
- [dreulavelle/rank-torrent-name](https://github.com/dreulavelle/rank-torrent-name)
- [MunifTanjim/go-ptt](https://github.com/MunifTanjim/go-ptt) — the original Go port this library builds on
- [TheBeastLT/parse-torrent-title](https://github.com/TheBeastLT/parse-torrent-title)

## License

Licensed under the MIT License. Check the [LICENSE](./LICENSE) file for details.
