package rank

// Profile: everything a user can tune, in one declarative structure.

import (
	"encoding/json"
	"os"

	"github.com/dreulavelle/jhin/parser"
)

// Policy controls a single attribute: whether releases carrying it may be
// fetched at all, and how it contributes to the score.
type Policy struct {
	Fetch bool `json:"fetch"`
	Rank  int  `json:"rank"`
}

// PatternRank scores a regex match on the raw title. Patterns wrapped in
// slashes ("/X/") are case-sensitive; anything else is case-insensitive.
type PatternRank struct {
	Pattern string `json:"pattern"`
	Rank    int    `json:"rank"`
}

// Resolution buckets, finest-grained; sorting and per-resolution gating both
// use these.
type Resolution string

const (
	Res2160p   Resolution = "2160p"
	Res1440p   Resolution = "1440p"
	Res1080p   Resolution = "1080p"
	Res720p    Resolution = "720p"
	Res576p    Resolution = "576p"
	Res480p    Resolution = "480p"
	Res360p    Resolution = "360p"
	Res240p    Resolution = "240p"
	ResUnknown Resolution = "unknown"
)

// resolutionBucket orders resolutions for sorting (higher = better).
var resolutionBucket = map[Resolution]int{
	Res2160p: 9, Res1440p: 8, Res1080p: 7, Res720p: 6,
	Res576p: 5, Res480p: 4, Res360p: 3, Res240p: 2, ResUnknown: 1,
}

// normalizeResolution maps parser output (either raw "2160p" style or
// Normalize()d "4k"/"2k") onto a Resolution. Full WxH values are reduced to
// their height first, so a 480p release that spelled out both dimensions is
// gated as 480p rather than escaping into ResUnknown.
func normalizeResolution(s string) Resolution {
	switch parser.ResolutionHeight(s) {
	case "2160p", "4k":
		return Res2160p
	case "1440p", "2k":
		return Res1440p
	case "1080p":
		return Res1080p
	case "720p":
		return Res720p
	case "576p":
		return Res576p
	case "480p":
		return Res480p
	case "360p":
		return Res360p
	case "240p":
		return Res240p
	default:
		return ResUnknown
	}
}

// Languages configures language-based filtering and preference.
// Group aliases may be used anywhere a code is accepted: "anime", "common",
// "non_anime", "all".
type Languages struct {
	// Required: at least one must be present, else the release is rejected.
	Required []string `json:"required,omitempty"`
	// Allowed: bypass exclusion when present.
	Allowed []string `json:"allowed,omitempty"`
	// Exclude: reject releases containing any of these.
	Exclude []string `json:"exclude,omitempty"`
	// Preferred: releases containing any of these get the preferred bonus.
	Preferred []string `json:"preferred,omitempty"`
}

// Options are the global knobs.
type Options struct {
	// TitleThreshold is the similarity ratio [0,1] a parsed title must reach
	// against a target title (when one is supplied to Rank).
	TitleThreshold float64 `json:"title_threshold"`
	// RemoveTrash rejects trash sources (CAM/TeleSync/...), trash-flagged
	// titles and clean-audio releases.
	RemoveTrash bool `json:"remove_trash"`
	// RemoveAdult rejects adult content.
	RemoveAdult bool `json:"remove_adult"`
	// RemoveUnknownLanguages rejects releases with no detected language.
	RemoveUnknownLanguages bool `json:"remove_unknown_languages"`
	// AllowEnglish accepts releases containing English regardless of the
	// language exclusion rules.
	AllowEnglish bool `json:"allow_english"`
	// MinRank rejects releases scoring below it (use math.MinInt to
	// disable). Unlike rank-torrent-name, this gate always applies rather
	// than only when trash removal is requested.
	MinRank int `json:"min_rank"`
	// PreferredBonus is added once when any preferred pattern or preferred
	// language matches.
	PreferredBonus int `json:"preferred_bonus"`
}

// Profile is a complete, serializable ranking/filtering configuration.
//
// Patterns in Require/Exclude/Preferred wrapped in slashes ("/X/") are
// case-sensitive; anything else is compiled case-insensitively.
type Profile struct {
	Name string `json:"name,omitempty"`

	Require   []string `json:"require,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
	Preferred []string `json:"preferred,omitempty"`

	// PatternRanks add their Rank (positive or negative) to every release
	// whose raw title matches Pattern — weighted keywords without vetoes.
	PatternRanks []PatternRank `json:"pattern_ranks,omitempty"`

	// ResolutionOrder, when set, replaces the built-in best-to-worst
	// resolution ordering used by Ranker.Sort: first entry sorts highest.
	// Unlisted resolutions fall below all listed ones, keeping their
	// default relative order.
	ResolutionOrder []Resolution `json:"resolution_order,omitempty"`

	Resolutions map[Resolution]bool `json:"resolutions,omitempty"`
	Languages   Languages           `json:"languages,omitempty"`
	Options     Options             `json:"options"`

	// Attributes overrides the base policy per attribute; unset attributes
	// fall back to the profile's base (DefaultPolicies unless replaced).
	Attributes map[Attr]Policy `json:"attributes,omitempty"`
}

// Default returns a sensible starting profile: 4K/1440p/1080p/720p enabled
// with higher quality favored, SD tiers disabled, and trash, CAMs, 3D, and
// adult content removed. Disable tiers you don't want (for example
// Resolutions[Res2160p] = false) or replace the maps wholesale — consuming
// apps are expected to build their own profiles on top of these defaults.
func Default() Profile {
	return Profile{
		Name: "default",
		Resolutions: map[Resolution]bool{
			Res2160p: true, Res1440p: true, Res1080p: true, Res720p: true,
			Res576p: false, Res480p: false, Res360p: false, Res240p: false,
			ResUnknown: true,
		},
		Options: Options{
			TitleThreshold: 0.85,
			RemoveTrash:    true,
			RemoveAdult:    true,
			AllowEnglish:   true,
			MinRank:        -10000,
			PreferredBonus: 10000,
		},
	}
}

// DefaultPolicies is the base Policy table; profiles override per
// attribute via Profile.Attributes.
var DefaultPolicies = map[Attr]Policy{
	// sources
	AttrRemux:    {Fetch: true, Rank: 10000},
	AttrWebDL:    {Fetch: true, Rank: 200},
	AttrWeb:      {Fetch: true, Rank: 100},
	AttrBluRay:   {Fetch: true, Rank: 100},
	AttrWebRip:   {Fetch: true, Rank: -1000},
	AttrDVD:      {Fetch: false, Rank: -5000},
	AttrHDTV:     {Fetch: true, Rank: -5000},
	AttrBDRip:    {Fetch: false, Rank: -5000},
	AttrDVDRip:   {Fetch: false, Rank: -5000},
	AttrUHDRip:   {Fetch: false, Rank: -5000},
	AttrVHS:      {Fetch: false, Rank: -10000},
	AttrWebMux:   {Fetch: false, Rank: -10000},
	AttrBRRip:    {Fetch: false, Rank: -10000},
	AttrHDRip:    {Fetch: true, Rank: -10000},
	AttrPPVRip:   {Fetch: false, Rank: -10000},
	AttrTVRip:    {Fetch: false, Rank: -10000},
	AttrVHSRip:   {Fetch: false, Rank: -10000},
	AttrWebDLRip: {Fetch: false, Rank: -10000},
	AttrSATRip:   {Fetch: false, Rank: -10000},

	// trash sources
	AttrCam:      {Fetch: false, Rank: -10000},
	AttrTeleCine: {Fetch: false, Rank: -10000},
	AttrTeleSync: {Fetch: false, Rank: -10000},
	AttrScreener: {Fetch: false, Rank: -10000},
	AttrR5:       {Fetch: false, Rank: -10000},
	AttrPDTV:     {Fetch: false, Rank: -10000},

	// codecs
	AttrAVC:  {Fetch: true, Rank: 500},
	AttrHEVC: {Fetch: true, Rank: 500},
	AttrAV1:  {Fetch: true, Rank: 500},
	AttrXvid: {Fetch: false, Rank: -10000},
	AttrMPEG: {Fetch: false, Rank: -1000},

	// hdr / depth
	AttrDolbyVision: {Fetch: true, Rank: 3000},
	AttrHDR10Plus:   {Fetch: true, Rank: 2100},
	AttrHDR:         {Fetch: true, Rank: 2000},
	AttrSDR:         {Fetch: true, Rank: 0},
	Attr10Bit:       {Fetch: true, Rank: 100},

	// audio
	AttrDTSLossless:      {Fetch: true, Rank: 2000},
	AttrTrueHD:           {Fetch: true, Rank: 2000},
	AttrAtmos:            {Fetch: true, Rank: 1000},
	AttrDolbyDigitalPlus: {Fetch: true, Rank: 150},
	AttrDTSLossy:         {Fetch: true, Rank: 100},
	AttrAAC:              {Fetch: true, Rank: 100},
	AttrDolbyDigital:     {Fetch: true, Rank: 50},
	AttrFLAC:             {Fetch: true, Rank: 0},
	AttrOPUS:             {Fetch: true, Rank: 0},
	AttrPCM:              {Fetch: true, Rank: 0},
	AttrMP3:              {Fetch: false, Rank: -1000},
	AttrCleanAudio:       {Fetch: false, Rank: -10000},

	// channels
	AttrSurround: {Fetch: true, Rank: 0},
	AttrStereo:   {Fetch: true, Rank: 0},
	AttrMono:     {Fetch: false, Rank: 0},

	// extras
	AttrEdition:     {Fetch: true, Rank: 100},
	AttrProper:      {Fetch: true, Rank: 20},
	AttrRepack:      {Fetch: true, Rank: 20},
	AttrNetwork:     {Fetch: true, Rank: 0},
	AttrRetail:      {Fetch: true, Rank: 0},
	AttrSubbed:      {Fetch: true, Rank: 0},
	AttrScene:       {Fetch: true, Rank: 0},
	AttrUncensored:  {Fetch: true, Rank: 0},
	AttrHardcoded:   {Fetch: true, Rank: 0},
	AttrDocumentary: {Fetch: false, Rank: -250},
	AttrConverted:   {Fetch: false, Rank: -1000},
	AttrDubbed:      {Fetch: true, Rank: -1000},
	Attr3D:          {Fetch: false, Rank: -10000},
	AttrUpscaled:    {Fetch: false, Rank: -10000},
	AttrSite:        {Fetch: false, Rank: -10000},
	AttrSize:        {Fetch: false, Rank: -10000},
}

// Save writes the profile as JSON.
func (p Profile) Save(path string) error {
	blob, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0o644)
}

// Load reads a profile from JSON.
func Load(path string) (Profile, error) {
	var p Profile
	blob, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(blob, &p)
	return p, err
}

// language groups usable as aliases in Languages lists.
var languageGroups = map[string][]string{
	"anime": {"ja", "zh", "ko"},
	"non_anime": {
		"de", "es", "hi", "ta", "ru", "ua", "th", "it", "ar", "pt", "fr",
		"pa", "mr", "gu", "te", "kn", "ml", "vi", "id", "tr", "he", "fa",
		"el", "lt", "lv", "et", "pl", "cs", "sk", "hu", "ro", "bg", "sr",
		"hr", "sl", "nl", "da", "fi", "sv", "no", "ms",
	},
	"common": {"de", "es", "hi", "ta", "ru", "ua", "th", "it", "zh", "ar", "fr"},
}

// expandLangs resolves group aliases into a lookup set.
func expandLangs(codes []string) map[string]bool {
	out := make(map[string]bool, len(codes))
	for _, c := range codes {
		if c == "all" {
			for _, g := range [2]string{"anime", "non_anime"} {
				for _, l := range languageGroups[g] {
					out[l] = true
				}
			}
			continue
		}
		if group, ok := languageGroups[c]; ok {
			for _, l := range group {
				out[l] = true
			}
			continue
		}
		out[c] = true
	}
	return out
}
