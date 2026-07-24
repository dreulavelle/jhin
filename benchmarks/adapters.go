package benchmarks

// One adapter per library maps its native output into the Canonical record.
// Adapters are intentionally thin — no fixups beyond shape lifting (scalar
// season -> slice) and the shared normalizers. A panic inside a competitor
// is caught and reported as a parse error, not a crash of the harness.

import (
	"fmt"

	middelink "github.com/middelink/go-parse-torrent-name"
	goptn "github.com/razsteinmetz/go-ptn"

	torrentparser "github.com/ProfChaos/torrent-name-parser"

	"github.com/dreulavelle/jhin/parser"
)

type Library struct {
	Name    string
	Version string
	Repo    string
	Notes   string
	// Parse maps the library's output into the canonical record.
	Parse func(title string) (Canonical, error)
	// Bench invokes the library with no adapter overhead; results are
	// written to package-level sinks so the compiler cannot elide the call.
	Bench func(title string)
}

var (
	sinkJhin       *parser.Result
	sinkProfChaos  torrentparser.Torrent
	sinkGoPTN      *goptn.TorrentInfo
	sinkMiddelink  *middelink.TorrentInfo
	sinkBenchError error
)

// Libraries lists every benchmarked parser, jhin first.
var Libraries = []Library{
	{
		Name:    "jhin",
		Version: "(this repo)",
		Repo:    "github.com/dreulavelle/jhin",
		Parse: func(title string) (Canonical, error) {
			r := parser.Parse(title)
			if err := r.Error(); err != nil {
				return Canonical{}, err
			}
			return Canonical{
				Title:      r.Title,
				Year:       r.Year,
				Seasons:    r.Seasons,
				Episodes:   r.Episodes,
				Resolution: normResolution(r.Resolution),
				Source:     normSource(r.Quality),
				Codec:      normCodec(r.Codec),
				Group:      r.Group,
				Container:  normToken(r.Container),
			}, nil
		},
		Bench: func(title string) { sinkJhin = parser.Parse(title) },
	},
	{
		Name:    "ProfChaos/torrent-name-parser",
		Version: "v0.5.1",
		Repo:    "github.com/ProfChaos/torrent-name-parser",
		Parse: func(title string) (Canonical, error) {
			t, err := torrentparser.ParseName(title)
			if err != nil {
				return Canonical{}, err
			}
			seasons := t.Seasons
			if len(seasons) == 0 {
				seasons = intsFromScalar(t.Season)
			}
			return Canonical{
				Title:      t.Title,
				Year:       itoa(t.Year),
				Seasons:    seasons,
				Episodes:   intsFromScalar(t.Episode),
				Resolution: normResolution(string(t.Resolution)),
				Source:     normSource(t.Source),
				Codec:      normCodec(t.Codec),
				Group:      t.Group,
				Container:  normToken(t.Container),
			}, nil
		},
		Bench: func(title string) { sinkProfChaos, sinkBenchError = torrentparser.ParseName(title) },
	},
	{
		Name:    "razsteinmetz/go-ptn",
		Version: "v1.0.0",
		Repo:    "github.com/razsteinmetz/go-ptn",
		Notes:   "last commit 2024-12",
		Parse: func(title string) (Canonical, error) {
			t, err := goptn.Parse(title)
			if err != nil {
				return Canonical{}, err
			}
			return Canonical{
				Title:      t.Title,
				Year:       itoa(t.Year),
				Seasons:    intsFromScalar(t.Season),
				Episodes:   intsFromScalar(t.Episode),
				Resolution: normResolution(t.Resolution),
				Source:     normSource(t.Quality),
				Codec:      normCodec(t.Codec),
				Group:      t.Group,
				Container:  normToken(t.Container),
			}, nil
		},
		Bench: func(title string) { sinkGoPTN, sinkBenchError = goptn.Parse(title) },
	},
	{
		Name:    "middelink/go-parse-torrent-name",
		Version: "v0.0.0-20190301",
		Repo:    "github.com/middelink/go-parse-torrent-name",
		Notes:   "unmaintained since 2019",
		Parse: func(title string) (Canonical, error) {
			t, err := middelink.Parse(title)
			if err != nil {
				return Canonical{}, err
			}
			return Canonical{
				Title:      t.Title,
				Year:       itoa(t.Year),
				Seasons:    intsFromScalar(t.Season),
				Episodes:   intsFromScalar(t.Episode),
				Resolution: normResolution(t.Resolution),
				Source:     normSource(t.Quality),
				Codec:      normCodec(t.Codec),
				Group:      t.Group,
				Container:  normToken(t.Container),
			}, nil
		},
		Bench: func(title string) { sinkMiddelink, sinkBenchError = middelink.Parse(title) },
	},
}

// safeParse shields the harness from competitor panics.
func safeParse(lib *Library, title string) (c Canonical, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return lib.Parse(title)
}
