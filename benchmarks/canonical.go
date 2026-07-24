package benchmarks

// Canonical is the neutral record every library's output is mapped into
// before scoring. Normalization lives here — in the harness, on neutral
// ground — so no library is scored against another library's vocabulary.
// Only fields every benchmarked library claims to extract are included.

import (
	"strconv"
	"strings"
)

type Canonical struct {
	Title      string
	Year       string
	Seasons    []int
	Episodes   []int
	Resolution string
	Source     string
	Codec      string
	Group      string
	Container  string
}

// normToken lowercases and strips separators so "WEB-DL", "WEBDL" and
// "web dl" compare equal.
func normToken(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normTitle folds case and punctuation: comparisons are on words, not
// separator or casing conventions.
func normTitle(s string) string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127)
	})
	return strings.Join(fields, " ")
}

// normResolution buckets by vertical line count: "4k"/"UHD"/"2160p" are one
// class, and interlaced/progressive suffixes are ignored.
func normResolution(s string) string {
	switch t := normToken(s); t {
	case "4k", "uhd", "2160p", "2160i", "2160":
		return "2160p"
	case "8k", "4320p", "4320":
		return "4320p"
	case "2k", "1440p", "1440":
		return "1440p"
	case "fhd", "1080p", "1080i", "1080":
		return "1080p"
	case "hd", "720p", "720i", "720":
		return "720p"
	case "480p", "480i", "480", "sd":
		return "480p"
	case "576p", "576i", "576":
		return "576p"
	case "540p", "540":
		return "540p"
	case "360p", "360":
		return "360p"
	case "240p", "240":
		return "240p"
	default:
		return t
	}
}

// normCodec maps encoder spellings onto the codec family: x264/H.264/AVC are
// all AVC, x265/H.265/HEVC are all HEVC.
func normCodec(s string) string {
	switch t := normToken(s); t {
	case "x264", "h264", "avc":
		return "avc"
	case "x265", "h265", "hevc":
		return "hevc"
	case "mpeg", "mpeg2", "mpeg4":
		return "mpeg"
	default:
		return t // xvid, divx, av1, ...
	}
}

// normSource maps rip-type spellings onto equivalence classes. Distinct
// quality tiers (WEB-DL vs WEBRip, BluRay vs BDRip) stay distinct — that
// difference is the point of the field.
func normSource(s string) string {
	switch t := normToken(s); t {
	case "bluray", "bdmv", "bd", "bdiso":
		return "bluray"
	case "blurayremux", "remux", "bdremux", "uhdremux":
		return "remux"
	case "web", "webdl":
		return "webdl"
	case "webrip":
		return "webrip"
	case "webmux":
		return "webmux"
	case "hdtv", "hdtvrip":
		return "hdtv"
	case "cam", "camrip", "hdcam":
		return "cam"
	case "ts", "telesync", "hdts":
		return "telesync"
	case "tc", "telecine", "hdtc":
		return "telecine"
	case "scr", "screener", "dvdscr", "dvdscreener", "bdscr", "webscreener":
		return "screener"
	case "dvdrip", "dvdmux":
		return "dvdrip"
	case "dvd", "dvd5", "dvd9", "dvdiso":
		return "dvd"
	case "hdrip", "webdlrip", "dvbrip":
		return "hdrip"
	case "pdtv", "dtvrip", "tvrip", "satrip":
		return "tvrip"
	case "vhs", "vhsrip":
		return "vhs"
	case "ppv", "ppvrip":
		return "ppv"
	case "workprint", "wp":
		return "workprint"
	default:
		return t
	}
}

func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// intsFromScalar lifts a scalar season/episode into the slice shape;
// 0 means "not extracted".
func intsFromScalar(n int) []int {
	if n == 0 {
		return nil
	}
	return []int{n}
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
