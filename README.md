[![CI](https://img.shields.io/github/actions/workflow/status/dreulavelle/jhin/ci.yml?branch=main&label=CI&style=for-the-badge)](https://github.com/dreulavelle/jhin/actions/workflows/ci.yml)
[![Go Reference](https://img.shields.io/badge/go-reference-%23007d9c?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/dreulavelle/jhin)
[![License](https://img.shields.io/github/license/dreulavelle/jhin?style=for-the-badge)](https://github.com/dreulavelle/jhin/blob/main/LICENSE)

# Jhin

All-in-one Go library for torrent release names: **parsing**, **ranking**,
**filtering**, and **sorting** — built for high accuracy and high throughput.

Jhin is the Go successor to [PTT](https://github.com/dreulavelle/PTT) (parsing)
and [rank-torrent-name](https://github.com/dreulavelle/rank-torrent-name)
(ranking/filtering/sorting), unified into a single dependency-light library.

## Status

- `jhin` / `jhin/parser` — torrent title parsing (100% parity with PTT v1.6.16, golden-verified)
- `jhin/rank` — ranking engine (in progress)
- `jhin/filter` — filtering engine (in progress)

## Install

```sh
go get github.com/dreulavelle/jhin
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/dreulavelle/jhin"
)

func main() {
	result := jhin.Parse("The.Matrix.1999.1080p.BluRay.x264")
	fmt.Println(result.Title, result.Year, result.Resolution, result.Codec)
}
```

Only need a few fields? A partial parser skips every other handler:

```go
parse := jhin.GetPartialParser([]string{"resolution", "year"})
result := parse("The.Matrix.1999.1080p.BluRay.x264")
```

### CLI

```sh
go install github.com/dreulavelle/jhin/cmd/jhin@latest

jhin parse --pretty "The.Matrix.1999.1080p.BluRay.x264"
```

## `Result` Reference

Field semantics match [PTT](https://github.com/dreulavelle/PTT) v1.6.16 exactly
(verified by a 1,133-title golden corpus generated from the Python parser).

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
