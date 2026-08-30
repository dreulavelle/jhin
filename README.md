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

- **`jhin/parser`** extracts 46 metadata fields from a release name in ~55µs.
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

`Result` has 46 fields (see the [reference](#result-reference) below) and
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
	// (Use rank.Entry with ranker.RankEntries to carry infohashes along.)
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
// Require is conjunctive — every pattern must match, so use alternation
// within one pattern for "any of these".
p.Require = []string{`\b(2160p|1080p)\b`}
p.Exclude = []string{`\bHDCAM\b`}   // any match rejects
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

## Rules

Everything derivable from a release name is already profile-configurable.
Rules are how an application folds in what only it knows — size, age, grab
count, what a probe measured — from configuration rather than from Go:

```
Seeders:             score min(grabs, 200) * 15  if grabs > 0
Freshness:           score max(0, 500 - ageDays * 5)  if true
Oversized:           reject if sizeGB > 12 and resolution != "2160p"
DV without fallback: reject if dolbyVision and not hdrFallback
Bad upscale:         reject if upscaled and exists(resolution == "2160p" and "remux" in traits)
Best 2 per flavour:  keep 2 per resolution + " " + quality if true
UHD T1:              define if group in ["FraMeSToR", "W4NK3R"]
```

Points are an expression rather than a constant, so an attribute can be scored
*by* a value instead of at a flat rate. `exists`, `count` and `none` ask about
the whole result set, which is what turns "reject upscales" into "reject an
upscale only when something better turned up". `define` names a condition for
`matched("UHD T1")` to reuse, so a release-group tier list is written once.

An application declares its own attributes against a registry — that, not the
grammar, is where new capability goes:

```go
reg := rules.Core()                     // everything a release name says
reg.Tier("measured", "a probed file")
reg.Namespace("probed", "measured").Num("height").Bool("dolbyVision")
reg.Field("sizeGB", rules.Num, "reported")
reg.Func("imdbRating", nil, rules.Num, fetchRating)   // anything the language can't say
reg.Effect("tag", rules.Str)                          // an action jhin doesn't interpret

profile.Rules, _ = rules.ParseText(ruleFile)
eng, _ := profile.CompileRules(reg)
ranker, _ := rank.New(profile, rank.WithRules(eng))

torrents := ranker.RankEntries(entries)   // Entry.Facts carries your data
best := ranker.Sort(torrents, rank.SortOptions{FetchableOnly: true})
rank.ApplyLimits(best)                    // caps need the final order
```

A field belongs to a confidence tier, and a rule reading a tier the release
carries nothing in is **skipped and reported** rather than judged against zero
— otherwise one `probed.height < 1080` rule would empty every result list of
everything nothing had opened. `Explain` reports rule contributions alongside
attribute ones, and `Torrent.RuleSkipped` says what did not run and why.

Conditions are checked when the profile is compiled: an unknown attribute, a
type mismatch or a bad pattern names the rule it came from —
`jhin rules check <file>` reports it before a search ever runs. Full reference:
[`docs/rules.md`](docs/rules.md).

## CLI

```sh
go install github.com/dreulavelle/jhin/cmd/jhin@latest

jhin parse --pretty "The.Matrix.1999.1080p.BluRay.x264"   # parse a title
jhin parse --long "The.Matrix.1999.1080p.BluRay.x264"     # ...including unset fields
jhin rank --target "The Matrix" < titles.txt              # rank/filter/sort a list
jhin rank --rules my-rules.txt < titles.txt               # ...with a rule file
jhin rules check my-rules.txt                             # compile a rule file
jhin rules fields                                         # what a rule can name
jhin version                                              # installed version
```

`parse` prints only the fields it actually set, so the output is what the
title carried:

```json
{
  "codec": "AVC",
  "quality": "BluRay",
  "resolution": "1080p",
  "title": "The Matrix",
  "year": "1999"
}
```

`--long` prints all 45 fields, unset ones included, for a stable shape to
script against.

## Performance

Benchmarked on a Ryzen 9 5900HX (see [`docs/benchmark.md`](docs/benchmark.md)):

| Operation | Time | Notes |
|---|---|---|
| Parse (simple title) | ~35µs | 58 allocs |
| Parse (corpus mean, 1,156 mixed titles) | ~55µs | easy and hostile titles alike |
| Batch parse | ~14µs/title | `ParseAll` across 8 threads — 100k titles in ~1.4s |

How it stays fast: the parser is an ordered table of ~430 regex handlers,
but most never run. At startup, every handler's pattern is analyzed to find
substrings it cannot match without — down to two-character sets like
`s0`…`s9` for `S01`-style patterns — and all of them are compiled into a
single Aho-Corasick automaton. One scan per title then decides which
handlers could possibly match, cutting ~430 potential regex executions to
about 42. The same reasoning guards title cleanup, where a byte-level check
replaces each cleanup regex that provably cannot match. Neither can ever do
more than *skip* work: equivalence is enforced by tests and continuous
fuzzing.

## Accuracy

`parser/testdata/golden.json` pins the expected output for 1,156 real-world
release names across every field — the corpus was generated by the original
Python PTT parser and jhin reproduces it byte-for-byte. Any behavioral
regression fails CI.

## How it compares

Measured 2026-07-25 on the 1,156-title corpus; full methodology, disclosure,
and reproduction steps in [`docs/benchmark.md`](docs/benchmark.md). Accuracy
is scored per field, only on the 9 fields every library claims, after neutral
vocabulary normalization.

| Library | Accuracy (9 shared fields) | Speed (per title, serial) | Fields |
|---|---|---|---|
| **jhin** (Go) | **100%** ¹ | 55µs (14µs batched) | 46 |
| [ProfChaos/torrent-name-parser](https://github.com/ProfChaos/torrent-name-parser) (Go) | 78.8% | 60µs | 28 |
| [middelink/go-parse-torrent-name](https://github.com/middelink/go-parse-torrent-name) (Go, unmaintained) | 70.1% | 38µs | 22 |
| [razsteinmetz/go-ptn](https://github.com/razsteinmetz/go-ptn) (Go) | 67.4% | 43µs | 26 |
| [parse-torrent-title](https://github.com/clement-escolano/parse-torrent-title) (JS) | not scored ² | 17µs | ~20 |
| [PTT](https://github.com/dreulavelle/PTT) (Python) | 100% ¹ | 576µs | 46 |
| [guessit](https://github.com/guessit-io/guessit) (Python) | not scored ² | 4,298µs | ~30 |

¹ The gold labels are generated by Python PTT, and jhin is contract-tested
byte-identical to it — so both score 100% by construction. The table's real
content is the other columns and the other rows.

² Different output schema; there is no honest cross-vocabulary accuracy
score without a shared gold standard, so these are compared on speed only.

The lighter Go parsers run ~30 regexes filling ~20 scalar fields; jhin runs
432 handlers behind an Aho-Corasick prefilter to extract 46 fields
(multi-season packs, episode ranges, 60+ languages, HDR, editions, trash
detection) — and still lands within ~1.4x of the fastest of them, which
fills less than half as many fields. The accuracy column is what the
remaining microseconds buy.

## `Result` Reference

Field semantics match [PTT](https://github.com/dreulavelle/PTT) 1.8.5
(commit `88429bb`) exactly, verified by the golden corpus.

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

## Acknowledgements

- [dreulavelle/PTT](https://github.com/dreulavelle/PTT)
- [dreulavelle/rank-torrent-name](https://github.com/dreulavelle/rank-torrent-name)
- [MunifTanjim/go-ptt](https://github.com/MunifTanjim/go-ptt) — the original Go port this library builds on
- [TheBeastLT/parse-torrent-title](https://github.com/TheBeastLT/parse-torrent-title)

## License

Licensed under the MIT License. Check the [LICENSE](./LICENSE) file for details.
