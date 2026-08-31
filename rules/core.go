package rules

import (
	"strconv"
	"strings"

	"github.com/dreulavelle/jhin/parser"
)

// Everything jhin itself can answer about a release: what its name says.
//
// These carry no tier. A release always has a name, so a rule reading only
// these can always be judged — which is what makes the tier machinery a
// property of what an application adds rather than a tax on the basics.

// Core returns a registry holding every attribute readable from a release
// name. Applications build on it:
//
//	reg := rules.Core()
//	reg.Tier("measured", "a probed file")
//	reg.Namespace("probed", "measured").Num("height")
func Core() *Registry {
	r := NewRegistry()
	for path, t := range coreFields {
		r.Field(path, t, "")
	}
	return r
}

// coreFields is the schema. Names are what a rule author writes, so this map
// is user-facing surface: renaming an entry breaks profiles.
var coreFields = map[string]Type{
	// what it is
	"title":       Str,
	"year":        Num,
	"releaseName": Str,

	// picture
	"resolution": Str,
	"quality":    Str,
	"codec":      Str,
	"bitDepth":   Num,
	"hdr":        StrList,
	// dolbyVision and hdrFallback say in one word what no single regex can:
	// whether a device that cannot decode DV still gets an HDR picture.
	"dolbyVision": Bool,
	"hdrFallback": Bool,

	// sound
	"audio":     StrList,
	"channels":  StrList,
	"languages": StrList,

	// numbering
	"seasons":     NumList,
	"episodes":    NumList,
	"volumes":     NumList,
	"episodeCode": Str,
	"seasonPack":  Bool,
	"complete":    Bool,

	// provenance
	"group":     Str,
	"edition":   Str,
	"container": Str,
	"extension": Str,
	"network":   Str,
	"region":    Str,
	"site":      Str,
	"date":      Str,
	"size":      Str,
	"bitrate":   Str,
	"country":   Str,
	"extras":    StrList,

	// traits is every attribute the ranker scores, by the same keys — so a
	// rule can reach anything the baseline has an opinion about without a
	// separate field for each one.
	"traits": StrList,

	// flags
	"proper":      Bool,
	"repack":      Bool,
	"remastered":  Bool,
	"upscaled":    Bool,
	"threeD":      Bool,
	"dubbed":      Bool,
	"subbed":      Bool,
	"hardcoded":   Bool,
	"documentary": Bool,
	"adult":       Bool,
	"trash":       Bool,
	"scene":       Bool,
	"retail":      Bool,
	"uncensored":  Bool,
	"unrated":     Bool,
	"convert":     Bool,
	"commentary":  Bool,
	"ppv":         Bool,
	"torrent":     Bool,
}

// CoreFields lists the core attribute names, sorted.
func CoreFields() []string {
	out := make([]string, 0, len(coreFields))
	for k := range coreFields {
		out = append(out, k)
	}
	return out
}

// ResultFacts answers the core schema from a parse result.
//
// traits is supplied rather than derived because the trait vocabulary lives
// in the rank package, and rank builds on rules rather than the other way
// round. Pass rank.Attributes(result); pass nil for none.
type ResultFacts struct {
	raw    string
	r      *parser.Result
	traits []string
}

// FromResult wraps a parse result as facts. A nil result answers every
// attribute with its zero, which is what an unparseable name has always meant.
func FromResult(raw string, r *parser.Result, traits []string) *ResultFacts {
	return &ResultFacts{raw: raw, r: r, traits: traits}
}

// TierPresent reports true only for the always-present tier. An application
// layering its own facts on top answers for its own tiers — see Layer.
func (f *ResultFacts) TierPresent(tier string) bool { return tier == "" }

func (f *ResultFacts) Lookup(path string) (Value, bool) {
	if f.r == nil {
		return Value{}, false
	}
	r := f.r
	switch path {
	case "title":
		return StrOf(r.Title), true
	case "releaseName":
		return StrOf(f.raw), true
	case "year":
		return NumOf(atoiSafe(r.Year)), true
	case "resolution":
		return StrOf(r.Resolution), true
	case "quality":
		return StrOf(r.Quality), true
	case "codec":
		return StrOf(r.Codec), true
	case "bitDepth":
		return NumOf(leadingInt(r.BitDepth)), true
	case "hdr":
		return StrListOf(r.HDR), true
	case "dolbyVision":
		dv, _ := dynamicRange(r.HDR)
		return BoolOf(dv), true
	case "hdrFallback":
		_, fb := dynamicRange(r.HDR)
		return BoolOf(fb), true
	case "audio":
		return StrListOf(r.Audio), true
	case "channels":
		return StrListOf(r.Channels), true
	case "languages":
		return StrListOf(r.Languages), true
	case "seasons":
		return NumListOf(r.Seasons), true
	case "episodes":
		return NumListOf(r.Episodes), true
	case "volumes":
		return NumListOf(r.Volumes), true
	case "episodeCode":
		return StrOf(r.EpisodeCode), true
	case "seasonPack":
		return BoolOf(len(r.Seasons) > 0 && len(r.Episodes) == 0), true
	case "complete":
		return BoolOf(r.Complete), true
	case "group":
		return StrOf(r.Group), true
	case "edition":
		return StrOf(r.Edition), true
	case "container":
		return StrOf(r.Container), true
	case "extension":
		return StrOf(r.Extension), true
	case "network":
		return StrOf(r.Network), true
	case "region":
		return StrOf(r.Region), true
	case "site":
		return StrOf(r.Site), true
	case "date":
		return StrOf(r.Date), true
	case "size":
		return StrOf(r.Size), true
	case "bitrate":
		return StrOf(r.Bitrate), true
	case "country":
		return StrOf(r.Country), true
	case "extras":
		return StrListOf(r.Extras), true
	case "traits":
		return StrListOf(f.traits), true
	case "proper":
		return BoolOf(r.Proper), true
	case "repack":
		return BoolOf(r.Repack), true
	case "remastered":
		return BoolOf(r.Remastered), true
	case "upscaled":
		return BoolOf(r.Upscaled), true
	case "threeD":
		return BoolOf(r.ThreeD), true
	case "dubbed":
		return BoolOf(r.Dubbed), true
	case "subbed":
		return BoolOf(r.Subbed), true
	case "hardcoded":
		return BoolOf(r.Hardcoded), true
	case "documentary":
		return BoolOf(r.Documentary), true
	case "adult":
		return BoolOf(r.Adult), true
	case "trash":
		return BoolOf(r.Trash), true
	case "scene":
		return BoolOf(r.Scene), true
	case "retail":
		return BoolOf(r.Retail), true
	case "uncensored":
		return BoolOf(r.Uncensored), true
	case "unrated":
		return BoolOf(r.Unrated), true
	case "convert":
		return BoolOf(r.Convert), true
	case "commentary":
		return BoolOf(r.Commentary), true
	case "ppv":
		return BoolOf(r.PPV), true
	case "torrent":
		return BoolOf(r.Torrent), true
	}
	return Value{}, false
}

// dynamicRange reads the parser's HDR vocabulary. "DV" is Dolby Vision; a
// release carrying any other tag alongside it still shows HDR on a device
// that cannot decode DV, which is the distinction "block DV without a
// fallback" needs and the one no single regex can express.
func dynamicRange(tags []string) (dolbyVision, fallback bool) {
	for _, t := range tags {
		if strings.EqualFold(t, "DV") || strings.EqualFold(t, "DolbyVision") {
			dolbyVision = true
			continue
		}
		if t != "" {
			fallback = true
		}
	}
	return dolbyVision, fallback
}

func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

// leadingInt reads the number off the front of a value, because the parser
// spells bit depth "10bit" — fed to Atoi whole, every release would read as
// zero bits and a rule like bitDepth >= 10 could never fire.
func leadingInt(s string) int {
	n, any := 0, false
	for i := 0; i < len(s) && i < 9; i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n, any = n*10+int(s[i]-'0'), true
	}
	if !any {
		return 0
	}
	return n
}

// MapFacts answers from a map, for an application whose facts are already
// dynamic. Absent keys report the field's zero.
type MapFacts struct {
	Values map[string]Value
	// Tiers lists the tiers this release carries. A tier not listed skips
	// every rule that reads it.
	Tiers map[string]bool
}

func (m MapFacts) Lookup(path string) (Value, bool) {
	v, ok := m.Values[path]
	return v, ok
}

func (m MapFacts) TierPresent(tier string) bool {
	if tier == "" {
		return true
	}
	return m.Tiers[tier]
}

// Layer combines fact sources into one. A lookup takes the first answer, so
// earlier sources shadow later ones; a tier is present when any source says
// it is. This is how an application stacks its own facts onto Core's.
func Layer(sources ...Facts) Facts { return layered(sources) }

type layered []Facts

func (l layered) Lookup(path string) (Value, bool) {
	for _, s := range l {
		if s == nil {
			continue
		}
		if v, ok := s.Lookup(path); ok {
			return v, true
		}
	}
	return Value{}, false
}

func (l layered) TierPresent(tier string) bool {
	if tier == "" {
		return true
	}
	for _, s := range l {
		if s != nil && s.TierPresent(tier) {
			return true
		}
	}
	return false
}
