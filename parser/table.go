// The handler table: ordered handlers defining every field extracted from a
// release name. Ordering is semantically significant: earlier handlers may
// remove matched text and bound the title region for later ones. The
// accuracy contract is testdata/golden.json (TestGoldenCorpus).
//
// Each entry's comment preserves the pattern it implements; entries with
// Process functions emulate lookarounds RE2 cannot express (see
// handlers_custom.go).

package parser

import (
	"regexp"
	"strings"
)

var valueSetFieldMap = map[string]struct{}{
	"audio":     {},
	"channels":  {},
	"extras":    {},
	"hdr":       {},
	"languages": {},
}

var handlers = []handler{
	// year: \b19\d{2}\s?-\s?20\d{2}\b
	{
		Field:     "year",
		Pattern:   regexp.MustCompile(`\b19\d{2}\s?-\s?20\d{2}\b`),
		Transform: toFirstIntString(),
	},
	// title: 360.Degrees.of.Vision.The.Byakugan'?s.Blind.Spot
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)360.Degrees.of.Vision.The.Byakugan'?s.Blind.Spot`),
		Remove:  true,
	},
	// title: \b100[ .-]*years?[ .-]*quest\b
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)\b100[ .-]*years?[ .-]*quest\b`),
		Remove:  true,
	},
	// title: \[?(\+.)?Extras\]?
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)\[?(\+.)?Extras\]?`),
		Remove:  true,
	},
	// title: (\+Movies)?\+Specials
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)(\+Movies)?\+Specials`),
		Remove:  true,
	},
	// group: -?EDGE2020
	{
		Field:     "group",
		Pattern:   regexp.MustCompile(`-?EDGE2020`),
		Transform: toValue(`EDGE2020`),
		Remove:    true,
	},
	// title: TV Money
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)TV Money`),
		Remove:  true,
	},
	// container: \.?[\[(]?\b(MKV|AVI|MP4|WMV|MPG|MPEG)\b[\])]?
	{
		Field:     "container",
		Pattern:   regexp.MustCompile(`(?i)\.?[\[(]?\b(MKV|AVI|MP4|WMV|MPG|MPEG)\b[\])]?`),
		Transform: toLowercase(),
	},
	// torrent: \.torrent$
	{
		Field:     "torrent",
		Pattern:   regexp.MustCompile(`\.torrent$`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// adult: \b(XXX|xxx|Xxx)\b
	{
		Field:     "adult",
		Pattern:   regexp.MustCompile(`\b(XXX|xxx|Xxx)\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// adult: ['custom:create_adult_pattern']
	customAdult,
	// scene: ^(?=.*(\b\d{3,4}p\b).*([_. ]WEB[_. ])(?!DL)\b)|\b(-CAKES|-GGEZ|-GGWP|-GLHF|-GOSSIP|-NAISU|-KOGI|-PECULATE|-SLOT|-EDITH|-ETHEL|-ELEANOR|-B2B|-SPAMnEGGS|-FTP|-DiRT|-SYNCOPY|-BA
	customScene,
	// extras: \bNCED\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`\bNCED\b`),
		Transform: toValueSet(`NCED`),
		Remove:    true,
	},
	// extras: \bNCOP\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`\bNCOP\b`),
		Transform: toValueSet(`NCOP`),
		Remove:    true,
	},
	// extras: \bNC\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`\bNC\b`),
		Transform: toValueSet(`NC`),
		Remove:    true,
	},
	// extras: \bOVA\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`(?i)\bOVA\b`),
		Transform: toValueSet(`OVA`),
		Remove:    true,
	},
	// extras: \bED(\d?v?\d?)\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`(?i)\bED(\d?v?\d?)\b`),
		Transform: toValueSet(`ED`),
		Remove:    true,
	},
	// extras: \bOPv?(\d+)?\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`\bOPv?(\d+)?\b`),
		Transform: toValueSet(`OP`),
		Remove:    true,
	},
	// extras: \b(?:Deleted[ .-]*)?Scene(?:s)?\b
	{
		Field:     "extras",
		Pattern:   regexp.MustCompile(`(?i)\bDeleted.*Scenes?\b`),
		Transform: toValueSet(`Deleted Scene`),
	},
	// extras: (?:(?<=\b(?:19\d{2}|20\d{2})\b.*)\b(?:Featurettes?)\b|\bFeaturettes?\b(?!.*\b(?:19\d{2}|20\d{2})\b))
	customExtrasFeaturette,
	// extras: (?:(?<=\b(?:19\d{2}|20\d{2})\b.*)\b(?:Sample)\b|\b(?:Sample)\b(?!.*\b(?:19\d{2}|20\d{2})\b))
	customExtrasSample,
	// extras: (?:(?<=\b(?:19\d{2}|20\d{2})\b.*)\b(?:Trailers?)\b|\bTrailers?\b(?!.*\b(?:19\d{2}|20\d{2}|.(Park|And))\b))
	customExtrasTrailer,
	// ppv: \bPPV\b
	{
		Field:         "ppv",
		Pattern:       regexp.MustCompile(`(?i)\bPPV\b`),
		Transform:     toBoolean(),
		Remove:        true,
		SkipFromTitle: true,
	},
	// ppv: \b\W?Fight.?Nights?\W?\b
	{
		Field:         "ppv",
		Pattern:       regexp.MustCompile(`(?i)\b\W?Fight.?Nights?\W?\b`),
		Transform:     toBoolean(),
		SkipFromTitle: true,
	},
	// site: ^(www?[., ][\w-]+[. ][\w-]+(?:[. ][\w-]+)?)\s+-\s*
	{
		Field:         "site",
		Pattern:       regexp.MustCompile(`(?i)^(www?[., ][\w-]+[. ][\w-]+(?:[. ][\w-]+)?)\s+-\s*`),
		Remove:        true,
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// site: \bwww.+rodeo\b
	{
		Field:     "site",
		Pattern:   regexp.MustCompile(`(?i)\bwww.+rodeo\b`),
		Transform: toLowercase(),
		Remove:    true,
	},
	// resolution: \[?\]?3840x\d{4}[\])?]?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)\[?\]?3840x\d{4}[\])?]?`),
		Transform: toValue(`2160p`),
		Remove:    true,
	},
	// resolution: \[?\]?1920x\d{3,4}[\])?]?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)\[?\]?1920x\d{3,4}[\])?]?`),
		Transform: toValue(`1080p`),
		Remove:    true,
	},
	// resolution: \[?\]?1280x\d{3}[\])?]?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)\[?\]?1280x\d{3}[\])?]?`),
		Transform: toValue(`720p`),
		Remove:    true,
	},
	// resolution: \[?\]?(\d{3,4}x\d{3,4})[\])?]?p?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)\[?\]?(\d{3,4}x\d{3,4})[\])?]?p?`),
		Transform: toValueSub(`$1p`),
		Remove:    true,
	},
	// resolution: (480|720|1080)0[pi]
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(480|720|1080)0[pi]`),
		Transform: toValueSub(`$1p`),
		Remove:    true,
	},
	// resolution: (?:QHD|QuadHD|WQHD|2560(\d+)?x(\d+)?1440p?)
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:QHD|QuadHD|WQHD|2560(\d+)?x(\d+)?1440p?)`),
		Transform: toValue(`1440p`),
		Remove:    true,
	},
	// resolution: (?:Full HD|FHD|1920(\d+)?x(\d+)?1080p?)
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:Full HD|FHD|1920(\d+)?x(\d+)?1080p?)`),
		Transform: toValue(`1080p`),
		Remove:    true,
	},
	// resolution: (?:BD|HD|M)(2160p?|4k)
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|HD|M)(2160p?|4k)`),
		Transform: toValue(`2160p`),
		Remove:    true,
	},
	// resolution: (?:BD|HD|M)1080p?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|HD|M)1080p?`),
		Transform: toValue(`1080p`),
		Remove:    true,
	},
	// resolution: (?:BD|HD|M)720p?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|HD|M)720p?`),
		Transform: toValue(`720p`),
		Remove:    true,
	},
	// resolution: (?:BD|HD|M)480p?
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|HD|M)480p?`),
		Transform: toValue(`480p`),
		Remove:    true,
	},
	// resolution: \b(?:4k|2160p|1080p|720p|480p)(?!.*\b(?:4k|2160p|1080p|720p|480p)\b)
	{
		Field:         "resolution",
		Pattern:       regexp.MustCompile(`(?i)\b(?:4k|2160p|1080p|720p|480p)`),
		ValidateMatch: validateLookahead(`.*\b(?:4k|2160p|1080p|720p|480p)\b`, `i`, false),
		Transform:     toTransformedResolution(),
		Remove:        true,
	},
	// resolution: \b4k|21600?[pi]\b
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)\b4k|21600?[pi]\b`),
		Transform: toValue(`2160p`),
		Remove:    true,
	},
	// resolution: (\d{3,4})[pi]
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(\d{3,4}[pi])`),
		Transform: toLowercase(),
		Remove:    true,
	},
	// resolution: (240|360|480|576|720|1080|2160|3840)[pi]
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(240|360|480|576|720|1080|2160|3840)[pi]`),
		Transform: toLowercase(),
		Remove:    true,
	},
	// episode_code: [\[\()]([A-Za-f0-9]{8})[\]\)]
	{
		Field:     "episodeCode",
		Pattern:   regexp.MustCompile(`[\[\()]([A-Fa-f0-9]{8})[\]\)]`),
		Transform: toUppercase(),
		Remove:    true,
	},
	// episode_code: [\[\()]([0-9]{8})[\]\)]
	{
		Field:     "episodeCode",
		Pattern:   regexp.MustCompile(`[\[\()]([0-9]{8})[\]\)]`),
		Transform: toUppercase(),
		Remove:    true,
	},
	// trash: \b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\()\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b
	customTrashCam,
	// trash: \b(?:H[DQ][ .-]*)?S[ \.\-]print\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?S[ \.\-]print\b`),
		Transform: toBoolean(),
	},
	// trash: \b(?:HD[ .-]*)?T(?:ELE)?(C|S)(?:INE|YNC)?(?:Rip)?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\b(?:HD[ .-]*)?T(?:ELE)?(C|S)(?:INE|YNC)?(?:Rip)?\b`),
		Transform: toBoolean(),
		// every match contains T followed by ELE, C, or S
		Gate: gate("tele", "tc", "ts"),
	},
	// trash: \bPre.?DVD(?:Rip)?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bPre.?DVD(?:Rip)?\b`),
		Transform: toBoolean(),
	},
	// trash: \b(?:DVD?|BD|BR|HD)?[ .-]*Scr(?:eener)?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\b(?:DVD?|BD|BR|HD)?[ .-]*Scr(?:eener)?\b`),
		Transform: toBoolean(),
	},
	// trash: \bDVB[ .-]*(?:Rip)?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bDVB[ .-]*(?:Rip)?\b`),
		Transform: toBoolean(),
	},
	// trash: \bSAT[ .-]*Rips?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bSAT[ .-]*Rips?\b`),
		Transform: toBoolean(),
	},
	// trash: \bLeaked\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bLeaked\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// trash: threesixtyp
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)threesixtyp`),
		Transform: toBoolean(),
	},
	// trash: \bR5|R6\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bR5|R6\b`),
		Transform: toBoolean(),
	},
	// trash: \b(?:Deleted[ .-]*)?Scene(?:s)?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bDeleted.*Scenes?\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// trash: \bHQ.?(Clean)?.?(Aud(io)?)?\b
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)\bHQ.?(Clean)?.?(Aud(io)?)?\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// date: (?:\W|^)([[(]?(?:19[6-9]|20[012])[0-9]([. \-/\\])(?:0[1-9]|1[012])\2(?:0[1-9]|[12][0-9]|3[01])[])]?)(?:\W|$)
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W|^)([[(]?(?:19[6-9]|20[012])[0-9]([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])[])]?)(?:\W|$)`),
		ValidateMatch: validateMatchedGroupsAreSame(2, 3),
		Transform:     toPttDate(`YYYY MM DD`),
		Remove:        true,
	},
	// date: (?:\W|^)(\[?\]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])\2(?:19[6-9]|20[01])[0-9][\])]?)(?:\W|$)
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W|^)(\[?\]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:19[6-9]|20[01])[0-9][\])]?)(?:\W|$)`),
		ValidateMatch: validateMatchedGroupsAreSame(2, 3),
		Transform:     toPttDate(`DD MM YYYY`),
		Remove:        true,
	},
	// date: (?:\W)(\[?\]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])\2(?:[0][1-9]|[0126789][0-9])[\])]?)(?:\W|$)
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)(\[?\]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:[0][1-9]|[0126789][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validateMatchedGroupsAreSame(2, 3),
		Transform:     toPttDate(`MM DD YY`),
		Remove:        true,
	},
	// date: (?:\W)(\[?\]?(?:[0][1-9]|[12][0-9]|3[0-9])([. \-/\\])(?:0[1-9]|1[012])\2(?:0[1-9]|[12][0-9])[\])]?)(?:\W|$)
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)(\[?\]?(?:[0][1-9]|[12][0-9]|3[0-9])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validateMatchedGroupsAreSame(2, 3),
		Transform:     toPttDate(`YY MM DD`),
		Remove:        true,
	},
	// date: (?:\W)(\[?\]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])\2(?:[0][1-9]|[0126789][0-9])[\])]?)(?:\W|$)
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)(\[?\]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:[0][1-9]|[0126789][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validateMatchedGroupsAreSame(2, 3),
		Transform:     toPttDate(`DD MM YY`),
		Remove:        true,
	},
	// date: (?:\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?i)(?:\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)([. \-/\\])(?:19[7-9]|20[012])[0-9][)\]]?)`),
		ValidateMatch: validateAnd(validateMatchedGroupsAreSame(2, 3), validateLookahead(`\W|$`, ``, true)),
		Transform:     toPttDate(`DD MMM YYYY`, `Do MMM YYYY`, `Do MMMM YYYY`),
		Remove:        true,
	},
	// date: (?:\W|^)(\[?\]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-\/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?i)(?:\W|^)(\[?\]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)([. \-/\\])(?:0[1-9]|[0126789][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validateMatchedGroupsAreSame(2, 3),
		Transform:     toPttDate(`DD MMM YY`),
		Remove:        true,
	},
	// date: (?:\W|^)(\[?\]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[\])]?)(?:\W|$)
	{
		Field:     "date",
		Pattern:   regexp.MustCompile(`(?:\W|^)(\[?\]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[\])]?)(?:\W|$)`),
		Transform: toPttDate(`YYYYMMDD`),
		Remove:    true,
	},
	// complete: \b((?:19\d|20[012])\d[ .]?-[ .]?(?:19\d|20[012])\d)\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`\b((?:19\d|20[012])\d[ .]?-[ .]?(?:19\d|20[012])\d)\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: [([][ .]?((?:19\d|20[012])\d[ .]?-[ .]?\d{2})[ .]?[)\]]
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`[([][ .]?((?:19\d|20[012])\d[ .]?-[ .]?\d{2})[ .]?[)\]]`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// bitrate: \b\d+[kmg]bps\b
	{
		Field:     "bitrate",
		Pattern:   regexp.MustCompile(`(?i)\b\d+[kmg]bps\b`),
		Transform: toLowercase(),
		Remove:    true,
	},
	// year: \b(20[0-9]{2}|2100)(?!\D*\d{4}\b)
	{
		Field:         "year",
		Pattern:       regexp.MustCompile(`\b(20[0-9]{2}|2100)`),
		ValidateMatch: validateLookahead(`\D*\d{4}\b`, ``, false),
		Transform:     toIntString(),
		Remove:        true,
	},
	// year: [^SE][([]?(?!^)(?<!\d|Cap[. ]?)((?:19\d|20[012])\d)(?!\d|kbps)[)\]]?
	{
		Field: "year",
		Process: scanValid("year", regexp.MustCompile(`(?i)[^SE][([]?((?:19\d|20[012])\d)[)\]]?`), func(title string, idxs []int) bool {
			return !yearPrefixRejectRegex.MatchString(title[:idxs[2]]) && !yearSuffixRejectRegex.MatchString(title[idxs[3]:])
		}, false, false, false),
		Transform: toIntString(),
		Remove:    true,
	},
	// year: (?!^\w{4})^[([]?((?:19\d|20[012])\d)(?!\d|kbps)[)\]]?
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`^[(\[]?((?:19\d|20[012])\d)(?:\d|kbps)?[)\]]?`),
		ValidateMatch: validateFunc(func(input string, match []int) bool {
			mValue := input[match[0]:match[1]]
			if len(mValue) == 4 {
				return match[0] != 0
			}
			return len(strings.Trim(mValue, "()[]")) == 4
		}),
		Transform: toYear(),
		Remove:    true,
	},
	// edition: \b\d{2,3}(th)?[\.\s\-\+_\/(),]Anniversary[\.\s\-\+_\/(),](Edition|Ed)?\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\b\d{2,3}(th)?[\.\s\-\+_\/(),]Anniversary[\.\s\-\+_\/(),](Edition|Ed)?\b`),
		Transform: toValue(`Anniversary Edition`),
		Remove:    true,
	},
	// edition: \bUltimate[\.\s\-\+_\/(),]Edition\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bUltimate[\.\s\-\+_\/(),]Edition\b`),
		Transform: toValue(`Ultimate Edition`),
		Remove:    true,
	},
	// edition: \bExtended[\.\s\-\+_\/(),]Director(\')?s\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bExtended[\.\s\-\+_\/(),]Director(\')?s\b`),
		Transform: toValue(`Directors Cut`),
		Remove:    true,
	},
	// edition: \b(custom.?)?Extended\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\b(custom.?)?Extended\b`),
		Transform: toValue(`Extended Edition`),
		Remove:    true,
	},
	// edition: \bDirector(\')?s.?Cut\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bDirector(\')?s.?Cut\b`),
		Transform: toValue(`Directors Cut`),
		Remove:    true,
	},
	// edition: \bCollector(\')?s\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bCollector(\')?s\b`),
		Transform: toValue(`Collectors Edition`),
		Remove:    true,
	},
	// edition: \bTheatrical\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bTheatrical\b`),
		Transform: toValue(`Theatrical`),
		Remove:    true,
	},
	// edition: \buncut(?!.gems)\b
	{
		Field:         "edition",
		Pattern:       regexp.MustCompile(`(?i)\buncut(?:.gems)?\b`),
		ValidateMatch: validateNotMatch(regexp.MustCompile(`(?i)(?:.gems)`)),
		Transform:     toValue("Uncut"),
		Remove:        true,
	},
	// edition: \bIMAX\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bIMAX\b`),
		Transform: toValue(`IMAX`),
		Remove:    true,
	},
	// edition: \b\.Diamond\.\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\b\.Diamond\.\b`),
		Transform: toValue(`Diamond Edition`),
		Remove:    true,
	},
	// edition: \bRemaster(?:ed)?\b
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bRemaster(?:ed)?\b`),
		Transform: toValue(`Remastered`),
		Remove:    true,
	},
	// upscaled: \b(?:AI.?)?(Upscal(ed?|ing)|Enhanced?)\b
	{
		Field:     "upscaled",
		Pattern:   regexp.MustCompile(`(?i)\b(?:AI.?)?(Upscal(ed?|ing)|Enhanced?)\b`),
		Transform: toBoolean(),
	},
	// upscaled: \b(?:iris2|regrade|ups(uhd|fhd|hd|4k))\b
	{
		Field:     "upscaled",
		Pattern:   regexp.MustCompile(`(?i)\b(?:iris2|regrade|ups(uhd|fhd|hd|4k))\b`),
		Transform: toBoolean(),
	},
	// upscaled: \b\.AI\.\b
	{
		Field:     "upscaled",
		Pattern:   regexp.MustCompile(`(?i)\b\.AI\.\b`),
		Transform: toBoolean(),
	},
	// convert: \bCONVERT\b
	{
		Field:     "convert",
		Pattern:   regexp.MustCompile(`\bCONVERT\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// hardcoded: \b(HC|HARDCODED)\b
	{
		Field:     "hardcoded",
		Pattern:   regexp.MustCompile(`\b(HC|HARDCODED)\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// proper: \b(?:REAL.)?PROPER\b
	{
		Field:     "proper",
		Pattern:   regexp.MustCompile(`(?i)\b(?:REAL.)?PROPER\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// repack: \bREPACK|RERIP\b
	{
		Field:     "repack",
		Pattern:   regexp.MustCompile(`(?i)\bREPACK|RERIP\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// retail: \bRetail\b
	{
		Field:     "retail",
		Pattern:   regexp.MustCompile(`(?i)\bRetail\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// remastered: \bRemaster(?:ed)?\b
	{
		Field:     "remastered",
		Pattern:   regexp.MustCompile(`(?i)\bRemaster(?:ed)?\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// documentary: \bDOCU(?:menta?ry)?\b
	{
		Field:         "documentary",
		Pattern:       regexp.MustCompile(`(?i)\bDOCU(?:menta?ry)?\b`),
		Transform:     toBoolean(),
		SkipFromTitle: true,
	},
	// unrated: \bunrated\b
	{
		Field:     "unrated",
		Pattern:   regexp.MustCompile(`(?i)\bunrated\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// uncensored: \buncensored\b
	{
		Field:     "uncensored",
		Pattern:   regexp.MustCompile(`(?i)\buncensored\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// commentary: \bcommentary\b
	{
		Field:     "commentary",
		Pattern:   regexp.MustCompile(`(?i)\bcommentary\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// region: R\dJ?\b
	{
		Field:     "region",
		Pattern:   regexp.MustCompile(`R\dJ?\b`),
		Transform: toUppercase(),
		Remove:    true,
	},
	// region: \b(PAL|NTSC|SECAM)\b
	{
		Field:     "region",
		Pattern:   regexp.MustCompile(`(?i)\b(PAL|NTSC|SECAM)\b`),
		Transform: toUppercase(),
		Remove:    true,
	},
	// quality: \b(?:HD[ .-]*)?T(?:ELE)?S(?:YNC)?(?:Rip)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:HD[ .-]*)?T(?:ELE)?S(?:YNC)?(?:Rip)?\b`),
		Gate:      gate("tele", "ts"),
		Transform: toValue(`TeleSync`),
		Remove:    true,
	},
	// quality: \b(?:HD[ .-]*)?T(?:ELE)?C(?:INE)?(?:Rip)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`\b(?:HD[ .-]*)?T(?:ELE)?C(?:INE)?(?:Rip)?\b`),
		Gate:      gate("tele", "tc"),
		Transform: toValue(`TeleCine`),
		Remove:    true,
	},
	// quality: \b(?:DVD?|BD|BR|HD)?[ .-]*Scr(?:eener)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:DVD?|BD|BR|HD)?[ .-]*Scr(?:eener)?\b`),
		Transform: toValue(`SCR`),
		Remove:    true,
	},
	// quality: \bP(?:RE)?-?(HD|DVD)(?:Rip)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bP(?:RE)?-?(HD|DVD)(?:Rip)?\b`),
		Transform: toValue(`SCR`),
		Remove:    true,
	},
	// quality: \bBlu[ .-]*Ray\b(?=.*remux)
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\bBlu[ .-]*Ray\b`),
		ValidateMatch: validateLookahead(`.*remux`, `i`, true),
		Transform:     toValue(`BluRay REMUX`),
		Remove:        true,
	},
	// quality: (?:BD|BR|UHD)[- ]?remux
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|BR|UHD)[- ]?remux`),
		Transform: toValue(`BluRay REMUX`),
		Remove:    true,
	},
	// quality: (?<=remux.*)\bBlu[ .-]*Ray\b
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\bBlu[ .-]*Ray\b`),
		ValidateMatch: validateLookbehind(`remux.*`, `i`, true),
		Transform:     toValue(`BluRay REMUX`),
		Remove:        true,
	},
	// quality: \bremux\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bremux\b`),
		Transform: toValue(`REMUX`),
		Remove:    true,
	},
	// quality: \bBlu[ .-]*Ray\b(?![ .-]*Rip)
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\bBlu[ .-]*Ray\b`),
		ValidateMatch: validateLookahead(`[ .-]*Rip`, `i`, false),
		Transform:     toValue(`BluRay`),
		Remove:        true,
	},
	// quality: \bUHD[ .-]*Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bUHD[ .-]*Rip\b`),
		Transform: toValue(`UHDRip`),
		Remove:    true,
	},
	// quality: \bHD[ .-]*Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bHD[ .-]*Rip\b`),
		Transform: toValue(`HDRip`),
		Remove:    true,
	},
	// quality: \bMicro[ .-]*HD\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bMicro[ .-]*HD\b`),
		Transform: toValue(`HDRip`),
		Remove:    true,
	},
	// quality: \b(?:BR|Blu[ .-]*Ray)[ .-]*Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:BR|Blu[ .-]*Ray)[ .-]*Rip\b`),
		Transform: toValue(`BRRip`),
		Remove:    true,
	},
	// quality: \bBD[ .-]*Rip\b|\bBDR\b|\bBD-RM\b|[[(]BD[\]) .,-]
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bBD[ .-]*Rip\b|\bBDR\b|\bBD-RM\b|[[(]BD[\]) .,-]`),
		Transform: toValue(`BDRip`),
		Remove:    true,
	},
	// quality: \b(?:HD[ .-]*)?DVD[ .-]*Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:HD[ .-]*)?DVD[ .-]*Rip\b`),
		Transform: toValue(`DVDRip`),
		Remove:    true,
	},
	// quality: \bVHS[ .-]*Rip?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bVHS[ .-]*Rip?\b`),
		Transform: toValue(`VHSRip`),
		Remove:    true,
	},
	// quality: \bDVD(?:R\d?|.*Mux)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bDVD(?:R\d?|.*Mux)?\b`),
		Transform: toValue(`DVD`),
		Remove:    true,
	},
	// quality: \bVHS\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bVHS\b`),
		Transform: toValue(`VHS`),
		Remove:    true,
	},
	// quality: \bPPVRip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bPPVRip\b`),
		Transform: toValue(`PPVRip`),
		Remove:    true,
	},
	// quality: \bHD.?TV.?Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bHD.?TV.?Rip\b`),
		Transform: toValue(`HDTVRip`),
		Remove:    true,
	},
	// quality: \bDVB[ .-]*(?:Rip)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bDVB[ .-]*(?:Rip)?\b`),
		Transform: toValue(`HDTV`),
		Remove:    true,
	},
	// quality: \bSAT[ .-]*Rips?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bSAT[ .-]*Rips?\b`),
		Transform: toValue(`SATRip`),
		Remove:    true,
	},
	// quality: \bTVRips?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bTVRips?\b`),
		Transform: toValue(`TVRip`),
		Remove:    true,
	},
	// quality: \bR5\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bR5\b`),
		Transform: toValue(`R5`),
		Remove:    true,
	},
	// quality: \b(?:DL|WEB|BD|BR)MUX\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:DL|WEB|BD|BR)MUX\b`),
		Transform: toValue(`WEBMux`),
		Remove:    true,
	},
	// quality: \bWEB[ .-]*Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]*Rip\b`),
		Transform: toValue(`WEBRip`),
		Remove:    true,
	},
	// quality: \bWEB[ .-]?DL[ .-]?Rip\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]?DL[ .-]?Rip\b`),
		Transform: toValue(`WEB-DLRip`),
		Remove:    true,
	},
	// quality: \bWEB[ .-]*(DL|.BDrip|.DLRIP)\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]*(DL|.BDrip|.DLRIP)\b`),
		Transform: toValue(`WEB-DL`),
		Remove:    true,
	},
	// quality: \b(?<!\w.)WEB\b|\bWEB(?!([ \.\-\(\],]+\d))\b
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\b(?:\w.)?WEB\b|\bWEB(?:(?:[ \.\-\(\],]+\d))?\b`),
		ValidateMatch: validateNotMatch(regexp.MustCompile(`(?i)\b(?:\w.)WEB\b|\bWEB(?:(?:[ \.\-\(\],]+\d))\b`)),
		Transform:     toValue("WEB"),
		Remove:        true,
		SkipFromTitle: true,
	},
	// quality: \b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\()\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b
	{
		Gate:  gate("cam"),
		Field: "quality",
		Process: scanValid("quality", regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?(CAM)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b`), func(title string, idxs []int) bool {
			return !trashCamRejectRegex.MatchString(title[idxs[3]:])
		}, false, false, false),
		Transform:     toValue(`CAM`),
		Remove:        true,
		SkipFromTitle: true,
	},
	// quality: \b(?:H[DQ][ .-]*)?S[ \.\-]print
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?S[ \.\-]print`),
		Transform:     toValue(`CAM`),
		Remove:        true,
		SkipFromTitle: true,
	},
	// quality: \bPDTV\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bPDTV\b`),
		Transform: toValue(`PDTV`),
		Remove:    true,
	},
	// quality: \bHD(.?TV)?\b
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bHD(.?TV)?\b`),
		Transform: toValue(`HDTV`),
		Remove:    true,
	},
	// bit_depth: \bhevc\s?10\b
	{
		Field:     "bitDepth",
		Pattern:   regexp.MustCompile(`(?i)\bhevc\s?10\b`),
		Transform: toValue(`10bit`),
	},
	// bit_depth: (?:8|10|12)[-\.]?(?=bit\b)
	{
		Field:         "bitDepth",
		Pattern:       regexp.MustCompile(`(?i)(?:8|10|12)[-\.]?`),
		ValidateMatch: validateLookahead(`bit\b`, `i`, true),
		Transform:     toValueSub(`$1bit`),
		Remove:        true,
	},
	// bit_depth: \bhdr10\b
	{
		Field:     "bitDepth",
		Pattern:   regexp.MustCompile(`(?i)\bhdr10\b`),
		Transform: toValue(`10bit`),
	},
	// bit_depth: \bhi10\b
	{
		Field:     "bitDepth",
		Pattern:   regexp.MustCompile(`(?i)\bhi10\b`),
		Transform: toValue(`10bit`),
	},
	// bit_depth: ['custom:handle_bit_depth']
	customHandleBitDepth,
	// hdr: \bDV\b|dolby.?vision|\bDoVi\b
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)\bDV\b|dolby.?vision|\bDoVi\b`),
		Transform:    toValueSet(`DV`),
		Remove:       true,
		KeepMatching: true,
	},
	// hdr: HDR10(?:\+|[-\.\s]?plus)
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)HDR10(?:\+|[-\.\s]?plus)`),
		Transform:    toValueSet(`HDR10+`),
		Remove:       true,
		KeepMatching: true,
	},
	// hdr: \bHDR(?:10)?\b
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)\bHDR(?:10)?\b`),
		Transform:    toValueSet(`HDR`),
		Remove:       true,
		KeepMatching: true,
	},
	// hdr: \bSDR\b
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)\bSDR\b`),
		Transform:    toValueSet(`SDR`),
		Remove:       true,
		KeepMatching: true,
	},
	// codec: \b[hx][\. \-]?264\b
	{
		Field:     "codec",
		Pattern:   regexp.MustCompile(`(?i)\b[hx][\. \-]?264\b`),
		Transform: toValue(`avc`),
		Remove:    true,
	},
	// codec: \b[hx][\. \-]?265\b
	{
		Field:     "codec",
		Pattern:   regexp.MustCompile(`(?i)\b[hx][\. \-]?265\b`),
		Transform: toValue(`hevc`),
		Remove:    true,
	},
	// codec: \b\W264\W\b
	{
		Field:         "codec",
		Pattern:       regexp.MustCompile(`\b\W264\W\b`),
		Transform:     toValue(`avc`),
		Remove:        true,
		SkipFromTitle: true,
	},
	// codec: \b\W265\W\b
	{
		Field:         "codec",
		Pattern:       regexp.MustCompile(`\b\W265\W\b`),
		Transform:     toValue(`hevc`),
		Remove:        true,
		SkipFromTitle: true,
	},
	// codec: \bHEVC10(bit)?\b|\b[xh][\. \-]?265\b
	{
		Field:     "codec",
		Pattern:   regexp.MustCompile(`(?i)\bHEVC10(bit)?\b|\b[xh][\. \-]?265\b`),
		Transform: toValue(`hevc`),
		Remove:    true,
	},
	// codec: \bhevc(?:\s?10)?\b
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\bhevc(?:\s?10)?\b`),
		Transform:    toValue(`hevc`),
		Remove:       true,
		KeepMatching: true,
	},
	// codec: \bdivx|xvid\b
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\bdivx|xvid\b`),
		Transform:    toValue(`xvid`),
		Remove:       true,
		KeepMatching: true,
	},
	// codec: \bavc\b
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\bavc\b`),
		Transform:    toValue(`avc`),
		Remove:       true,
		KeepMatching: true,
	},
	// codec: \bav1\b
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\bav1\b`),
		Transform:    toValue(`av1`),
		Remove:       true,
		KeepMatching: true,
	},
	// codec: \b(?:mpe?g\d*)\b
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\b(?:mpe?g\d*)\b`),
		Transform:    toValue(`mpeg`),
		Remove:       true,
		KeepMatching: true,
	},
	// codec: ['custom:handle_space_in_codec']
	customHandleSpaceInCodec,
	// channels: 5[\.\s]1(?:ch|-S\d+)?\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)5[\.\s]1(?:ch|-S\d+)?\b`),
		Transform:    toValueSet(`5.1`),
		Remove:       true,
		KeepMatching: true,
	},
	// channels: \b(?:x[2-4]|5[\W]1(?:x[2-4])?)\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)\b(?:x[2-4]|5[\W]1(?:x[2-4])?)\b`),
		Transform:    toValueSet(`5.1`),
		Remove:       true,
		KeepMatching: true,
	},
	// channels: \b7[\.\- ]1(.?ch(annel)?)?\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)\b7[\.\- ]1(.?ch(annel)?)?\b`),
		Transform:    toValueSet(`7.1`),
		Remove:       true,
		KeepMatching: true,
	},
	// channels: \+?2[\.\s]0(?:x[2-4])?\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)\+?2[\.\s]0(?:x[2-4])?\b`),
		Transform:    toValueSet(`2.0`),
		Remove:       true,
		KeepMatching: true,
	},
	// channels: \b2\.0\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)\b2\.0\b`),
		Transform:    toValueSet(`2.0`),
		Remove:       true,
		KeepMatching: true,
	},
	// channels: \bstereo\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)\b(?:24-bit\s)?stereo\b`),
		Transform:    toValueSet(`stereo`),
		KeepMatching: true,
	},
	// channels: \bmono\b
	{
		Field:        "channels",
		Pattern:      regexp.MustCompile(`(?i)\bmono\b`),
		Transform:    toValueSet(`mono`),
		KeepMatching: true,
	},
	// audio: \b(?!.+HR)(DTS.?HD.?Ma(ster)?|DTS.?X)\b
	{
		Gate:  gate("dts"),
		Field: "audio",
		Process: scanValid("audio", regexp.MustCompile(`(?i)\b(DTS.?HD.?Ma(?:ster)?|DTS.?X)\b`), func(title string, idxs []int) bool {
			return !audioHrAfterRegex.MatchString(title[idxs[0]:])
		}, true, false, true),
		Transform:    toValueSet(`DTS Lossless`),
		KeepMatching: true,
		Remove:       true,
	},
	// audio: \bDTS(?!(.?HD.?Ma(ster)?|.X)).?(HD.?HR|HD)?\b
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\bDTS(?:(?:.?HD.?Ma(?:ster)?|.X))?.?(?:HD.?HR|HD)?\b`),
		ValidateMatch: validateNotMatch(regexp.MustCompile(`(?i)DTS(?:.?HD.?Ma(?:ster)?|.X)`)),
		Transform:     toValueSet("DTS Lossy"),
		Remove:        true,
		KeepMatching:  true,
	},
	// audio: \b(Dolby.?)?Atmos\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b(Dolby.?)?Atmos\b`),
		Transform:    toValueSet(`Atmos`),
		Remove:       true,
		KeepMatching: true,
	},
	// audio: \b(True[ .-]?HD|\.True\.)\b
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\b(True[ .-]?HD|\.True\.)\b`),
		Transform:     toValueSet(`TrueHD`),
		Remove:        true,
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// audio: \bTRUE\b
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`\bTRUE\b`),
		Transform:     toValueSet(`TrueHD`),
		Remove:        true,
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// audio: \bFLAC(?:\d\.\d)?(?:x\d+)?\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bFLAC(?:\d\.\d)?(?:x\d+)?\b`),
		Transform:    toValueSet(`FLAC`),
		Remove:       true,
		KeepMatching: true,
	},
	// audio: DD2?[\+p]|DD Plus|Dolby Digital Plus|DDP(5[ \.\_]1)?|E-?AC-?3(?:-S\d+)?
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`DD2?[\+p]|DD Plus|Dolby Digital Plus|DDP(5[ \.\_]1)?|E-?AC-?3(?:-S\d+)?`),
		Transform:    toValueSet(`Dolby Digital Plus`),
		Remove:       true,
		KeepMatching: true,
	},
	// audio: \bddp(5.1)?
	{
		Field:     "audio",
		Pattern:   regexp.MustCompile(`(?i)\bddp(5.1)?`),
		Transform: toValueSet(`Dolby Digital Plus`),
		Remove:    true,
	},
	// audio: \b(DD|Dolby.?Digital|DolbyD|AC-?3(x2)?(?:-S\d+)?)\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b(DD|Dolby.?Digital|DolbyD|AC-?3(x2)?(?:-S\d+)?)\b`),
		Transform:    toValueSet(`Dolby Digital`),
		Remove:       true,
		KeepMatching: true,
	},
	// audio: \bQ?Q?AAC(x?2)?\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bQ?Q?AAC(x?2)?\b`),
		Transform:    toValueSet(`AAC`),
		Remove:       true,
		KeepMatching: true,
	},
	// audio: \bL?PCM\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bL?PCM\b`),
		Transform:    toValueSet(`PCM`),
		Remove:       true,
		KeepMatching: true,
	},
	// audio: \bOPUS(\b|\d)(?!.*[ ._-](\d{3,4}p))
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`\bOPUS(\b|\d)`),
		ValidateMatch: validateLookahead(`.*[ ._-](\d{3,4}p)`, ``, false),
		Transform:     toValueSet(`OPUS`),
		Remove:        true,
		KeepMatching:  true,
	},
	// audio: \b(H[DQ])?.?(Clean.?Aud(io)?)\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b(H[DQ])?.?(Clean.?Aud(io)?)\b`),
		Transform:    toValueSet(`HQ Clean Audio`),
		Remove:       true,
		KeepMatching: true,
	},
	// group: - ?(?!\d+$|S\d+|\d+x|ep?\d+|[^[]+]$)([^\-. []+[^\-. [)\]\d][^\-. [)\]]*)(?:\[[\w.-]+])?(?=\.\w{2,4}$|$)
	customGroupDash,
	// volumes: \bvol(?:s|umes?)?[. -]*(?:\d{1,2}[., +/\\&-]+)+\d{1,2}\b
	{
		Field:     "volumes",
		Pattern:   regexp.MustCompile(`(?i)\bvol(?:s|umes?)?[. -]*(?:\d{1,2}[., +/\\&-]+)+\d{1,2}\b`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// volumes: ['custom:handle_volumes']
	customHandleVolumes,
	// languages: \b(temporadas?|completa)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(temporadas?|completa)\b`),
		Transform:    toValueSet(`es`),
		KeepMatching: true,
	},
	// languages: \b(?:INT[EÉ]GRALE?)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:INT[EÉ]GRALE?)\b`),
		Transform:    toValueSet(`fr`),
		KeepMatching: true,
	},
	// languages: \b(?:Saison)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:Saison)\b`),
		Transform:    toValueSet(`fr`),
		KeepMatching: true,
	},
	// complete: \b(?:INTEGRALE?|INTÉGRALE?)\b
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`(?i)\b(?:INTEGRALE?|INTÉGRALE?)\b`),
		Transform:    toBoolean(),
		Remove:       true,
		KeepMatching: true,
	},
	// complete: (Movie|Complete).Collection
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`(Movie|Complete).Collection`),
		Transform:    toBoolean(),
		Remove:       true,
		KeepMatching: true,
	},
	// complete: Complete(.\d{1,2})
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`Complete(.\d{1,2})`),
		Transform:    toBoolean(),
		Remove:       true,
		KeepMatching: true,
	},
	// complete: (?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bbox[ .-]?set\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bbox[ .-]?set\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: (?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bmini[ .-]?series\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bmini[ .-]?series\b`),
		Transform: toBoolean(),
	},
	// complete: (?:\bthe\W)?(?:\bcomplete|full|all)\b.*\b(?:series|seasons|collection|episodes|set|pack|movies)\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:\bthe\W)?(?:\bcomplete\b|\bfull\b|\ball\b)\b.*\b(?:series|seasons|collection|episodes|set|pack|movies)\b`),
		Transform: toBoolean(),
	},
	// complete: (Top\W+)?\d+\W+(movies?|series|seasons?)\W+Collection
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(Top\W+)?\d+\W+(movies?|series|seasons?)\W+Collection`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: (?:\bthe\W)?\bultimate\b[ .]\bcollection\b
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`(?i)(?:\bthe\W)?\bultimate\b[ .]\bcollection\b`),
		Transform:    toBoolean(),
		KeepMatching: true,
	},
	// complete: \bcollection\b.*\b(?:set|pack|movies)\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bcollection\b.*\b(?:set|pack|movies)\b`),
		Transform: toBoolean(),
	},
	// complete: \bcollection(?:(\s\[|\s\())
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bcollection(?:(\s\[|\s\())`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: duology|trilogy|quadr[oi]logy|tetralogy|pentalogy|hexalogy|heptalogy|anthology
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`(?i)duology|trilogy|quadr[oi]logy|tetralogy|pentalogy|hexalogy|heptalogy|anthology`),
		Transform:    toBoolean(),
		KeepMatching: true,
	},
	// complete: \bcompleta\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bcompleta\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: \bsaga\b
	{
		Field:         "complete",
		Pattern:       regexp.MustCompile(`(?i)\bsaga\b`),
		Transform:     toBoolean(),
		SkipFromTitle: true,
	},
	// complete: \b\[Complete\]\b
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\b\[Complete\]\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: (?<!A.?|The.?)\bComplete\b
	{
		Field:         "complete",
		Pattern:       regexp.MustCompile(`(?i)\bComplete\b`),
		ValidateMatch: validateLookbehind(`A.?|The.?`, `i`, false),
		Transform:     toBoolean(),
		Remove:        true,
	},
	// complete: COMPLETE
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`COMPLETE`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// complete: \bkolekcja\b(?:\Wfilm(?:y|ów|ow)?)?
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`\bkolekcja\b(?:\Wfilm(?:y|ów|ow)?)?`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// seasons: (?:complete\W|seasons?\W|\W|^)((?:s\d{1,2}[., +/\\&-]+)+s\d{1,2}\b)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:complete\W|seasons?\W|\W|^)((?:s\d{1,2}[., +/\\&-]+)+s\d{1,2}\b)`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:complete\W|seasons?\W|\W|^)[([]?(s\d{2,}-\d{2,}\b)[)\]]?
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:complete\W|seasons?\W|\W|^)[([]?(s\d{2,}-\d{2,}\b)[)\]]?`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:complete\W|seasons?\W|\W|^)[([]?(s[1-9]-[2-9])[)\]]?
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:complete\W|seasons?\W|\W|^)[([]?(s[1-9]-[2-9])[)\]]?`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: \d+ª(?:.+)?(?:a.?)?\d+ª(?:(?:.+)?(?:temporadas?))
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\d+ª(?:.+)?(?:a.?)?\d+ª(?:(?:.+)?(?:temporadas?))`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|temporadas?)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[., /\\&]+)+\d{1,2}\b)[)\]]?
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|temporadas?)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[., /\\&]+)+\d{1,2}\b)[)\]]?`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|temporadas?)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[.-]+)+[1-9]\d?\b)[)\]]?
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|temporadas?)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[.-]+)+[1-9]\d?\b)[)\]]?`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete\W)?season[. ]?[([]?((?:\d{1,2}[. -]+)+[1-9]\d?\b)[)\]]?(?!.*\.\w{2,4}$)
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?season[. ]?[([]?((?:\d{1,2}[. -]+)+[1-9]\d?\b)[)\]]?`),
		ValidateMatch: validateLookahead(`.*\.\w{2,4}$`, `i`, false),
		Transform:     toIntRange(),
		Remove:        true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete\W)?\bseasons?\b[. -]?(\d{1,2}[. -]?(?:to|thru|and|\+|:)[. -]?\d{1,2})\b
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?\bseasons?\b[. -]?(\d{1,2}[. -]?(?:to|thru|and|\+|:)[. -]?\d{1,2})\b`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete\W)?(?:saison|seizoen|season|series|temp(?:orada)?):?[. ]?(\d{1,2})\b
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:saison|seizoen|season|series|temp(?:orada)?):?[. ]?(\d{1,2})\b`),
		Transform: toIntArray(),
	},
	// seasons: (\d{1,2})(?:-?й)?[. _]?(?:[Сс]езон|sez(?:on)?)(?:\W?\D|$)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(\d{1,2})(?:-?й)?[. _]?(?:[Сс]езон|sez(?:on)?)(?:\W?\D|$)`),
		Transform: toIntArray(),
		Remove:    true,
	},
	// seasons: [Сс]езон:?[. _]?№?(\d{1,2})(?!\d)
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?i)[Сс]езон:?[. _]?№?(\d{1,2})`),
		ValidateMatch: validateLookahead(`\d`, `i`, false),
		Transform:     toIntArray(),
		Remove:        true,
	},
	// seasons: (?:\D|^)(\d{1,2})Â?[°ºªa]?[. ]*temporada
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:\D|^)(\d{1,2})Â?[°ºªa]?[. ]*temporada`),
		Transform: toIntArray(),
		Remove:    true,
	},
	// seasons: t(\d{1,3})(?:[ex]+|$)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)t(\d{1,3})(?:[ex]+|$)`),
		Transform: toIntArray(),
		Remove:    true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete)?(?<![a-z])\bs(\d{1,3})(?:[\Wex]|\d{2}\b|$)
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete)?(?:[a-z])?\bs(\d{1,3})(?:[\Wex]|\d{2}\b|$)`),
		ValidateMatch: validateNotMatch(regexp.MustCompile(`(?i)(?:[a-z])\bs\d{1,3}`)),
		Transform:     toIntArray(),
		KeepMatching:  true,
	},
	// seasons: (?:(?:\bthe\W)?\bcomplete\W)?(?:\W|^)(\d{1,2})[. ]?(?:st|nd|rd|th)[. ]*season
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:\W|^)(\d{1,2})[. ]?(?:st|nd|rd|th)[. ]*season`),
		Transform: toIntArray(),
	},
	// seasons: (?<=S)\d{2}(?=E\d+)
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`\d{2}`),
		ValidateMatch: validateAnd(validateLookbehind(`S`, ``, true), validateLookahead(`E\d+`, ``, true)),
		Transform:     toIntArray(),
		Remove:        true,
	},
	// seasons: (?:\D|^)(\d{1,2})[xх]\d{1,3}(?:\D|$)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?:\D|^)(\d{1,2})[xх]\d{1,3}(?:\D|$)`),
		Transform: toIntArray(),
	},
	// seasons: \bSn([1-9])(?:\D|$)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`\bSn([1-9])(?:\D|$)`),
		Transform: toIntArray(),
	},
	// seasons: [[(](\d{1,2})\.\d{1,3}[)\]]
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`[[(](\d{1,2})\.\d{1,3}[)\]]`),
		Transform: toIntArray(),
	},
	// seasons: -\s?(\d{1,2})\.\d{2,3}\s?-
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`-\s?(\d{1,2})\.\d{2,3}\s?-`),
		Transform: toIntArray(),
	},
	// seasons: (?:^|\/)(\d{1,2})-\d{2}\b(?!-\d)
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?:^|\/)(\d{1,2})-\d{2}\b`),
		ValidateMatch: validateLookahead(`-\d`, ``, false),
		Transform:     toIntArray(),
	},
	// seasons: [^\w-](\d{1,2})-\d{2}(?=\.\w{2,4}$)
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`[^\w-](\d{1,2})-\d{2}`),
		ValidateMatch: validateLookahead(`\.\w{2,4}$`, ``, true),
		Transform:     toIntArray(),
	},
	// seasons: (?<!\bEp?(?:isode)? ?\d+\b.*)\b(\d{2})[ ._]\d{2}(?:.F)?\.\w{2,4}$
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`\b(\d{2})[ ._]\d{2}(?:.F)?\.\w{2,4}$`),
		ValidateMatch: validateLookbehind(`\bEp?(?:isode)? ?\d+\b.*`, ``, false),
		Transform:     toIntArray(),
	},
	// seasons: \bEp(?:isode)?\W+(\d{1,2})\.\d{1,3}\b
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\bEp(?:isode)?\W+(\d{1,2})\.\d{1,3}\b`),
		Transform: toIntArray(),
	},
	// seasons: \bSeasons?\b.*\b(\d{1,2}-\d{1,2})\b
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\bSeasons?\b.*\b(\d{1,2}-\d{1,2})\b`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// seasons: (?:\W|^)(\d{1,2})(?:e|ep)\d{1,3}(?:\W|$)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:\W|^)(\d{1,2})(?:e|ep)\d{1,3}(?:\W|$)`),
		Transform: toIntArray(),
	},
	// seasons: \bs(\d{1,4})
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\bs(\d{1,4})`),
		Transform: toIntArray(),
		Remove:    true,
	},
	// seasons: \bТВ-(\d{1,2})\b  (Go \b is ASCII-only; use unicode context)
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}_])ТВ-(\d{1,2})(?:[^\p{L}\p{N}_]|$)`),
		Transform: toIntArray(),
	},
	// episodes: (?:[\W\d]|^)e[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|e){1,2}[ .]?\d{1,3})+)(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)e[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|e){1,2}[ .]?\d{1,3})+)(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: (?:[\W\d]|^)ep[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|ep){1,2}[ .]?\d{1,3})+)(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)ep[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|ep){1,2}[ .]?\d{1,3})+)(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: (?:[\W\d]|^)\d+[xх][ .]?[([]?(\d{1,3}(?:[ .]?[xх][ .]?\d{1,3})+)(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)\d+[xх][ .]?[([]?(\d{1,3}(?:[ .]?[xх][ .]?\d{1,3})+)(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: Серии:\s+(\d+)\s+(?:of|из|iz)\s+\d+\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)Серии:\s+(\d+)\s+(?:of|из|iz)\s+\d+\b`),
		Transform: toIntRangeTill(),
	},
	// episodes: (?:[\W\d]|^)(?:episodes?|[Сс]ерии:?)[ .]?[([]?(\d{1,3}(?:[ .+]*[&+][ .]?\d{1,3})+)(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)(?:episodes?|[Сс]ерии:?)[ .]?[([]?(\d{1,3}(?:[ .+]*[&+][ .]?\d{1,3})+)(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: [([]?(?:\D|^)(\d{1,3}[ .]?ao[ .]?\d{1,3})[)\]]?(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)[([]?(?:\D|^)(\d{1,3}[ .]?ao[ .]?\d{1,3})[)\]]?(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: (?:[\W\d]|^)(?:e|eps?|episodes?|[Сс]ерии:?|\d+[xх])[ .]*[([]?(\d{1,3}(?:-\d{1,3})+)(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)(?:e|eps?|episodes?|[Сс]ерии:?|\d+[xх])[ .]*[([]?(\d{1,3}(?:-\d{1,3})+)(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: (?:\W|^)(\d{1,3}(?:[ .]*~[ .]*\d{1,3})+)(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\W|^)(\d{1,3}(?:[ .]*~[ .]*\d{1,3})+)(?:\W|$)`),
		Transform: toIntRange(),
	},
	// episodes: \bE\d{1,4}\s*à\s*E\d{1,4}\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\bE\d{1,4}\s*à\s*E\d{1,4}\b`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// episodes: [st]\d{1,2}[. ]?[xх-]?[. ]?(?:e|x|х|ep|-|\.)[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\D|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)[st]\d{1,2}[. ]?[xх-]?[. ]?(?:e|x|х|ep|-|\.)[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\D|$)`),
		Transform: toIntArray(),
		Remove:    true,
	},
	// episodes: \b[st]\d{2}(\d{2})\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b[st]\d{2}(\d{2})\b`),
		Transform: toIntArray(),
	},
	// episodes: -\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?!-\d)(?:\W|$)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?:-\d*)?(?:\W|$)`),
		ValidateMatch: validateNotMatch(regexp.MustCompile(`(?i)-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?:-\d*)`)),
		Transform:     toIntRange(),
	},
	// episodes: s\d{1,2}\s?\((\d{1,3}[ .]*-[ .]*\d{1,3})\)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)s\d{1,2}\s?\((\d{1,3}[ .]*-[ .]*\d{1,3})\)`),
		Transform: toIntRange(),
	},
	// episodes: (?:^|\/)\d{1,2}-(\d{2})\b(?!-\d)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?:^|\/)\d{1,2}-(\d{2})\b`),
		ValidateMatch: validateLookahead(`-\d`, ``, false),
		Transform:     toIntArray(),
	},
	// episodes: (?<!\d-)\b\d{1,2}-(\d{2})(?=\.\w{2,4}$)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`\b\d{1,2}-(\d{2})`),
		ValidateMatch: validateAnd(validateLookbehind(`\d-`, ``, false), validateLookahead(`\.\w{2,4}$`, ``, true)),
		Transform:     toIntArray(),
	},
	// episodes: (?<=^\[.+].+)[. ]+-[. ]+(\d{1,4})[. ]+(?=\W)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)[. ]+-[. ]+(\d{1,4})[. ]+`),
		ValidateMatch: validateAnd(validateLookbehind(`^\[.+].+`, `i`, true), validateLookahead(`\W`, `i`, true)),
		Transform:     toIntArray(),
		Remove:        true,
	},
	// episodes: (?<!(?:seasons?|[Сс]езони?)\W*)(?:[ .([-]|^)(\d{1,3}(?:[ .]?[,&+~][ .]?\d{1,3})+)(?:[ .)\]-]|$)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)(?:[ .([-]|^)(\d{1,3}(?:[ .]?[,&+~][ .]?\d{1,3})+)(?:[ .)\]-]|$)`),
		ValidateMatch: validateLookbehind(`(?:seasons?|[Сс]езони?)\W*`, `i`, false),
		Transform:     toIntRange(),
	},
	// episodes: (?<!(?:seasons?|[Сс]езони?)\W*)(?:[ .([-]|^)(\d{1,3}(?:-\d{1,3})+)(?:[ .)(\]]|-\D|$)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)(?:[ .([-]|^)(\d{1,3}(?:-\d{1,3})+)(?:[ .)(\]]|-\D|$)`),
		ValidateMatch: validateLookbehind(`(?:seasons?|[Сс]езони?)\W*`, `i`, false),
		Transform:     toIntRange(),
	},
	// episodes: \bEp(?:isode)?\W+\d{1,2}\.(\d{1,3})\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\bEp(?:isode)?\W+\d{1,2}\.(\d{1,3})\b`),
		Transform: toIntArray(),
	},
	// episodes: Ep.\d+.-.\d+
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)Ep.\d+.-.\d+`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// episodes: (?:\b[ée]p?(?:isode)?|[Ээ]пизод|[Сс]ер(?:ии|ия|\.)?|cap(?:itulo)?|epis[oó]dio)[. ]?[-:#№]?[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\b[ée]p?(?:isode)?|[Ээ]пизод|[Сс]ер(?:ии|ия|\.)?|cap(?:itulo)?|epis[oó]dio)[. ]?[-:#№]?[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\W|$)`),
		Transform: toIntArray(),
	},
	// episodes: \b(\d{1,3})(?:-?я)?[ ._-]*(?:ser(?:i?[iyj]a|\b)|[Сс]ер(?:ии|ия|\.)?)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b(\d{1,3})(?:-?я)?[ ._-]*(?:ser(?:i?[iyj]a|\b)|[Сс]ер(?:ии|ия|\.)?)`),
		Transform: toIntArray(),
	},
	// episodes: (?:\D|^)\d{1,2}[. ]?[xх][. ]?(\d{1,3})(?:[abc]|v0?[1-4]|\D|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?:\D|^)\d{1,2}[. ]?[xх][. ]?(\d{1,3})(?:[abc]|v0?[1-4]|\D|$)`),
		Transform: toIntArray(),
	},
	// episodes: (?<=S\d{2}E)\d+
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)\d+`),
		ValidateMatch: validateLookbehind(`S\d{2}E`, `i`, true),
		Transform:     toIntArray(),
	},
	// episodes: [[(]\d{1,2}\.(\d{1,3})[)\]]
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`[[(]\d{1,2}\.(\d{1,3})[)\]]`),
		Transform: toIntArray(),
	},
	// episodes: \b[Ss]\d{1,2}[ .](\d{1,2})\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`\b[Ss]\d{1,2}[ .](\d{1,2})\b`),
		Transform: toIntArray(),
	},
	// episodes: -\s?\d{1,2}\.(\d{2,3})\s?-
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`-\s?\d{1,2}\.(\d{2,3})\s?-`),
		Transform: toIntArray(),
	},
	// episodes: (?:\[|\()(\d+)\s(?:of|из|iz)\s\d+(?:\]|\))
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\[|\()(\d+)\s(?:of|из|iz)\s\d+(?:\]|\))`),
		Transform: toIntRangeTill(),
	},
	// episodes: (?<=\D|^)(\d{1,3})[. ]?(?:of|из|iz)[. ]?\d{1,3}(?=\D|$)
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)(\d{1,3})[. ]?(?:of|из|iz)[. ]?\d{1,3}`),
		ValidateMatch: validateAnd(validateLookbehind(`\D|^`, `i`, true), validateLookahead(`\D|$`, `i`, true)),
		Transform:     toIntArray(),
	},
	// episodes: \b\d{2}[ ._-](\d{2})(?:.F)?\.\w{2,4}$
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`\b\d{2}[ ._-](\d{2})(?:.F)?\.\w{2,4}$`),
		Transform: toIntArray(),
	},
	// episodes: (?<!^)\[(?!720|1080)([\.\-\s\W]\d{2,3}[\.\-\s\W])](?!(?:\.\w{2,4})?$)
	{
		Field:   "episodes",
		Pattern: regexp.MustCompile(`\[([^\w\p{L}\p{N}]\d{2,3}[^\w\p{L}\p{N}])]`),
		ValidateMatch: validateAnd(
			validateNotAtStart(),
			validateLookahead(`(?:\.\w{2,4})?$`, ``, false),
		),
		Transform: toIntArray(),
	},
	// episodes: (\d+)(?=.?\[([A-Z0-9]{8})])
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)(\d+)`),
		ValidateMatch: validateLookahead(`.?\[([A-Z0-9]{8})]`, `i`, true),
		Transform:     toIntArray(),
	},
	// episodes: (?<![xh])\b264\b|\b265\b
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)\b264\b|\b265\b`),
		ValidateMatch: validateLookbehind(`[xh]`, `i`, false),
		Transform:     toIntArray(),
		Remove:        true,
	},
	// episodes: (?<!\bMovie\s-\s)(?<=\s-\s)\d+(?=\s[-(\s])
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`\s-\s(\d+)`),
		ValidateMatch: validateAnd(validateLookbehind(`\bMovie`, ``, false), validateLookahead(`\s[-(\s]`, ``, true)),
		MatchGroup:    1,
		Transform:     toIntArray(),
		Remove:        true,
	},
	// episodes: (?:\W|^)(?:\d+)?(?:e|ep)(\d{1,3})(?:\W|$)
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\W|^)(?:\d+)?(?:e|ep)(\d{1,3})(?:\W|$)`),
		Transform: toIntArray(),
		Remove:    true,
	},
	// episodes: \d+.-.\d+TV
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\d+.-.\d+TV`),
		Transform: toIntRange(),
		Remove:    true,
	},
	// episodes: E(\d+)\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)E(\d+)\b`),
		Transform: toIntArray(),
	},
	// episodes: \b\d{1,4}-\d{1,4}\b
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b\d{1,4}-\d{1,4}\b`),
		Transform: toIntRange(),
	},
	// episodes: ['custom:handle_episodes']
	customHandleEpisodes,
	// episodes: ['custom:handle_anime_eps']
	customHandleAnimeEps,
	// country: \b(US|UK|AU|NZ|CA)\b
	{
		Field:     "country",
		Pattern:   regexp.MustCompile(`\b(US|UK|AU|NZ|CA)\b`),
		Transform: toValueSub(`$1`),
	},
	// languages: \bengl?(?:sub[A-Z]*)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bengl?(?:sub[A-Z]*)?\b`),
		Transform:    toValueSet(`en`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \beng?sub[A-Z]*\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\beng?sub[A-Z]*\b`),
		Transform:    toValueSet(`en`),
		KeepMatching: true,
	},
	// languages: \bing(?:l[eéê]s)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bing(?:l[eéê]s)?\b`),
		Transform:    toValueSet(`en`),
		KeepMatching: true,
	},
	// languages: \besub\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\besub\b`),
		Transform:    toValueSet(`en`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \benglish\W+(?:subs?|sdh|hi)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\benglish\W+(?:subs?|sdh|hi)\b`),
		Transform:    toValueSet(`en`),
		KeepMatching: true,
	},
	// languages: \beng?\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\beng?\b`),
		Transform:     toValueSet(`en`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \benglish?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\benglish?\b`),
		Transform:    toValueSet(`en`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(?:JP|JAP|JPN)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:JP|JAP|JPN)\b`),
		Transform:    toValueSet(`ja`),
		KeepMatching: true,
	},
	// languages: \b(japanese|japon[eê]s)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(japanese|japon[eê]s)\b`),
		Transform:    toValueSet(`ja`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(?:KOR|kor[ .-]?sub)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:KOR|kor[ .-]?sub)\b`),
		Transform:    toValueSet(`ko`),
		KeepMatching: true,
	},
	// languages: \b(korean|coreano)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(korean|coreano)\b`),
		Transform:    toValueSet(`ko`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(?:traditional\W*chinese|chinese\W*traditional)(?:\Wchi)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:traditional\W*chinese|chinese\W*traditional)(?:\Wchi)?\b`),
		Transform:    toValueSet(`zh`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \bzh-hant\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bzh-hant\b`),
		Transform:    toValueSet(`zh`),
		KeepMatching: true,
	},
	// languages: \b(?:mand[ae]rin|ch[sn])\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:mand[ae]rin|ch[sn])\b`),
		Transform:    toValueSet(`zh`),
		KeepMatching: true,
	},
	// languages: (?<!shang-?)\bCH(?:I|T)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bCH(?:I|T)\b`),
		ValidateMatch: validateLookbehind(`shang-?`, `i`, false),
		Transform:     toValueSet(`zh`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(chinese|chin[eê]s)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(chinese|chin[eê]s)\b`),
		Transform:    toValueSet(`zh`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bzh-hans\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bzh-hans\b`),
		Transform:    toValueSet(`zh`),
		KeepMatching: true,
	},
	// languages: \bFR(?:a|e|anc[eê]s|VF[FQIB2]?)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bFR(?:a|e|anc[eê]s|VF[FQIB2]?)\b`),
		Transform:     toValueSet(`fr`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b\[?(VF[FQRIB2]?\]?\b|(VOST)?FR2?)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`\b\[?(VF[FQRIB2]?\]?\b|(VOST)?FR2?)\b`),
		Transform:    toValueSet(`fr`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \b(TRUE|SUB).?FRENCH\b|\bFRENCH\b|\bFre?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`\b(TRUE|SUB).?FRENCH\b|\bFRENCH\b|\bFre?\b`),
		Transform:    toValueSet(`fr`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \b(VOST(?:FR?|A)?)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(VOST(?:FR?|A)?)\b`),
		Transform:    toValueSet(`fr`),
		KeepMatching: true,
	},
	// languages: \b(VF[FQIB2]?|(TRUE|SUB).?FRENCH|(VOST)?FR2?)\b
	{
		Field:     "languages",
		Pattern:   regexp.MustCompile(`(?i)\b(VF[FQIB2]?|(TRUE|SUB).?FRENCH|(VOST)?FR2?)\b`),
		Transform: toValueSet(`fr`),
		Remove:    true,
	},
	// languages: \bspanish\W?latin|american\W*(?:spa|esp?)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bspanish\W?latin|american\W*(?:spa|esp?)`),
		Transform:     toValueSet(`la`),
		Remove:        true,
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:\bla\b.+(?:cia\b))
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:\bla\b.+(?:cia\b))`),
		Transform:     toValueSet(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:audio.)?lat(?:in?|ino)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:audio.)?lat(?:in?|ino)?\b`),
		Transform:    toValueSet(`la`),
		KeepMatching: true,
	},
	// languages: \b(?:audio.)?(?:ESP?|spa|(en[ .]+)?espa[nñ]ola?|castellano)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:audio.)?(?:ESP?|spa|(en[ .]+)?espa[nñ]ola?|castellano)\b`),
		Transform:    toValueSet(`es`),
		KeepMatching: true,
	},
	// languages: \bes(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b
	{
		Gate:  gate("es"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bes\b`), func(title string, idxs []int) bool {
			return langCodes2Suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     toValueSet(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})es\b
	{
		Gate:  gate("es"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bes\b`), func(title string, idxs []int) bool {
			return langCodes2Prefix.MatchString(title[:idxs[0]])
		}, true, false, true),
		Transform:     toValueSet(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)es(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\b
	{
		Gate:  gate("es"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bes\b`), func(title string, idxs []int) bool {
			return langCode1Prefix.MatchString(title[:idxs[0]]) && langCode1Suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     toValueSet(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bes(?=\.(?:ass|ssa|srt|sub|idx)$)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bes`),
		ValidateMatch: validateLookahead(`\.(?:ass|ssa|srt|sub|idx)$`, `i`, true),
		Transform:     toValueSet(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bspanish\W+subs?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bspanish\W+subs?\b`),
		Transform:    toValueSet(`es`),
		KeepMatching: true,
	},
	// languages: \b(spanish|espanhol)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(spanish|espanhol)\b`),
		Transform:    toValueSet(`es`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b[\.\s\[]?Sp[\.\s\]]?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b[\.\s\[]?Sp[\.\s\]]?\b`),
		Transform:    toValueSet(`es`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \b(?:p[rt]|en|port)[. (\\/-]*BR\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:p[rt]|en|port)[. (\\/-]*BR\b`),
		Transform:    toValueSet(`pt`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \bbr(?:a|azil|azilian)\W+(?:pt|por)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bbr(?:a|azil|azilian)\W+(?:pt|por)\b`),
		Transform:    toValueSet(`pt`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \b(?:leg(?:endado|endas?)?|dub(?:lado)?|portugu[eèê]se?)[. -]*BR\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:leg(?:endado|endas?)?|dub(?:lado)?|portugu[eèê]se?)[. -]*BR\b`),
		Transform:    toValueSet(`pt`),
		KeepMatching: true,
	},
	// languages: \bleg(?:endado|endas?)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bleg(?:endado|endas?)\b`),
		Transform:    toValueSet(`pt`),
		KeepMatching: true,
	},
	// languages: \bportugu[eèê]s[ea]?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bportugu[eèê]s[ea]?\b`),
		Transform:    toValueSet(`pt`),
		KeepMatching: true,
	},
	// languages: \bPT[. -]*(?:PT|ENG?|sub(?:s|titles?))\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bPT[. -]*(?:PT|ENG?|sub(?:s|titles?))\b`),
		Transform:    toValueSet(`pt`),
		KeepMatching: true,
	},
	// languages: \bpt(?=\.(?:ass|ssa|srt|sub|idx)$)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bpt`),
		ValidateMatch: validateLookahead(`\.(?:ass|ssa|srt|sub|idx)$`, `i`, true),
		Transform:     toValueSet(`pt`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bPT\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bPT\b`),
		Transform:    toValueSet(`pt`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \bpor\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bpor\b`),
		Transform:     toValueSet(`pt`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b-?ITA\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b-?ITA\b`),
		Transform:    toValueSet(`it`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \b(?<!w{3}\.\w+\.)IT(?=[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,})\b
	{
		Gate:  gate("it"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`\bIT\b`), func(title string, idxs []int) bool {
			return !langWwwCs.MatchString(title[:idxs[0]]) && langItCodesCs.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     toValueSet(`it`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bit(?=\.(?:ass|ssa|srt|sub|idx)$)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bit`),
		ValidateMatch: validateLookahead(`\.(?:ass|ssa|srt|sub|idx)$`, `i`, true),
		Transform:     toValueSet(`it`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bitaliano?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bitaliano?\b`),
		Transform:    toValueSet(`it`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bgreek[ .-]*(?:audio|lang(?:uage)?|subs?(?:titles?)?)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bgreek[ .-]*(?:audio|lang(?:uage)?|subs?(?:titles?)?)?\b`),
		Transform:    toValueSet(`el`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(?:GER|DEU)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:GER|DEU)\b`),
		Transform:     toValueSet(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b
	{
		Gate:  gate("de"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bde\b`), func(title string, idxs []int) bool {
			return langCodes2Suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     toValueSet(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})de\b
	{
		Gate:  gate("de"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bde\b`), func(title string, idxs []int) bool {
			return langCodes2Prefix.MatchString(title[:idxs[0]])
		}, true, false, true),
		Transform:     toValueSet(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)de(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\b
	{
		Gate:  gate("de"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bde\b`), func(title string, idxs []int) bool {
			return langCode1Prefix.MatchString(title[:idxs[0]]) && langCode1Suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     toValueSet(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bde(?=\.(?:ass|ssa|srt|sub|idx)$)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bde`),
		ValidateMatch: validateLookahead(`\.(?:ass|ssa|srt|sub|idx)$`, `i`, true),
		Transform:     toValueSet(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(german|alem[aã]o)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(german|alem[aã]o)\b`),
		Transform:    toValueSet(`de`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bRUS?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bRUS?\b`),
		Transform:    toValueSet(`ru`),
		KeepMatching: true,
	},
	// languages: \b(russian|russo)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(russian|russo)\b`),
		Transform:    toValueSet(`ru`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bUKR\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bUKR\b`),
		Transform:    toValueSet(`uk`),
		KeepMatching: true,
	},
	// languages: \bukrainian\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bukrainian\b`),
		Transform:    toValueSet(`uk`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bhin(?:di)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bhin(?:di)?\b`),
		Transform:    toValueSet(`hi`),
		KeepMatching: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)tel(?!\W*aviv)|telugu)\b
	{
		Gate:  gate("tel", "telugu"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\b(?:tel|telugu)\b`), langAbbrValid(map[string]bool{"telugu": true}, func(title string, idxs []int) bool {
			return !langTeAviv.MatchString(title[idxs[1]:])
		}), true, false, true),
		Transform:    toValueSet(`te`),
		KeepMatching: true,
		Remove:       true,
	},
	// languages: \bt[aâ]m(?:il)?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bt[aâ]m(?:il)?\b`),
		Transform:    toValueSet(`ta`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)MAL(?:ay)?|malayalam)\b
	{
		Gate:         gate("mal", "malayalam"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:MAL(?:ay)?|malayalam)\b`), langAbbrValid(map[string]bool{"malayalam": true}, nil), true, true, true),
		Transform:    toValueSet(`ml`),
		KeepMatching: true,
		Remove:       true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)KAN(?:nada)?|kannada)\b
	{
		Gate:         gate("kan", "kannada"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:KAN(?:nada)?|kannada)\b`), langAbbrValid(map[string]bool{"kannada": true}, nil), true, false, true),
		Transform:    toValueSet(`kn`),
		KeepMatching: true,
		Remove:       true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)MAR(?:a(?:thi)?)?|marathi)\b
	{
		Gate:         gate("mar", "marathi"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:MAR(?:a(?:thi)?)?|marathi)\b`), langAbbrValid(map[string]bool{"marathi": true}, nil), true, false, true),
		Transform:    toValueSet(`mr`),
		KeepMatching: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)GUJ(?:arati)?|gujarati)\b
	{
		Gate:         gate("guj", "gujarati"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:GUJ(?:arati)?|gujarati)\b`), langAbbrValid(map[string]bool{"gujarati": true}, nil), true, false, true),
		Transform:    toValueSet(`gu`),
		KeepMatching: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)PUN(?:jabi)?|punjabi)\b
	{
		Gate:         gate("pun", "punjabi"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:PUN(?:jabi)?|punjabi)\b`), langAbbrValid(map[string]bool{"punjabi": true}, nil), true, false, true),
		Transform:    toValueSet(`pa`),
		KeepMatching: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)BEN(?!.\bThe|and|of\b)(?:gali)?|bengali)\b
	{
		Gate:  gate("ben", "bengali"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\b(?:BEN(?:gali)?|bengali)\b`), langAbbrValid(map[string]bool{"bengali": true}, func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "ben") {
				return !langBnReject.MatchString(title[idxs[1]:])
			}
			return true
		}), true, true, true),
		Transform:    toValueSet(`bn`),
		KeepMatching: true,
	},
	// languages: \b(?<!YTS\.)LT\b
	{
		Gate:  gate("lt"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`\bLT\b`), func(title string, idxs []int) bool {
			return !langYtsPrefix.MatchString(title[:idxs[0]])
		}, true, false, true),
		Transform:     toValueSet(`lt`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \blithuanian\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\blithuanian\b`),
		Transform:    toValueSet(`lt`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \blatvian\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\blatvian\b`),
		Transform:    toValueSet(`lv`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bestonian\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bestonian\b`),
		Transform:    toValueSet(`et`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)PL|pol)\b
	{
		Gate:         gate("pl", "pol"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:PL|pol)\b`), langAbbrValid(map[string]bool{"pol": true}, nil), true, false, true),
		Transform:    toValueSet(`pl`),
		KeepMatching: true,
	},
	// languages: \b(polish|polon[eê]s|polaco)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(polish|polon[eê]s|polaco)\b`),
		Transform:    toValueSet(`pl`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(PLDUB|PLSUB|DUBPL|DubbingPL|LekPL|LektorPL)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(PLDUB|PLSUB|DUBPL|DubbingPL|LekPL|LektorPL)\b`),
		Transform:    toValueSet(`pl`),
		Remove:       true,
		KeepMatching: true,
	},
	// languages: \bCZ[EH]?\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bCZ[EH]?\b`),
		Transform:    toValueSet(`cs`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bczech\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bczech\b`),
		Transform:    toValueSet(`cs`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \bslo(?:vak|vakian|subs|[\]_)]?\.\w{2,4}$)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bslo(?:vak|vakian|subs|[\]_)]?\.\w{2,4}$)\b`),
		Transform:     toValueSet(`sk`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bHU\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`\bHU\b`),
		Transform:     toValueSet(`hu`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bHUN(?:garian)?\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bHUN(?:garian)?\b`),
		Transform:     toValueSet(`hu`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bROM(?:anian)?\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bROM(?:anian)?\b`),
		Transform:     toValueSet(`ro`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bRO(?=[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bRO`),
		ValidateMatch: validateLookahead(`[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub`, `i`, true),
		Transform:     toValueSet(`ro`),
		KeepMatching:  true,
	},
	// languages: \bbul(?:garian)?\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bbul(?:garian)?\b`),
		Transform:     toValueSet(`bg`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:srp|serbian)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:srp|serbian)\b`),
		Transform:    toValueSet(`sr`),
		KeepMatching: true,
	},
	// languages: \b(?:HRV|croatian)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:HRV|croatian)\b`),
		Transform:    toValueSet(`hr`),
		KeepMatching: true,
	},
	// languages: \bHR(?=[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)\b
	{
		Gate:  gate("hr"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\bHR\b`), func(title string, idxs []int) bool {
			return langHrSubSuffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:    toValueSet(`hr`),
		KeepMatching: true,
	},
	// languages: \bslovenian\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bslovenian\b`),
		Transform:     toValueSet(`sl`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)NL|dut|holand[eê]s)\b
	{
		Gate:         gate("nl", "dut", "holand"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:NL|dut|holand[eê]s)\b`), langAbbrValid(map[string]bool{"dut": true, "holandes": true, "holandês": true}, nil), true, false, true),
		Transform:    toValueSet(`nl`),
		KeepMatching: true,
	},
	// languages: \bdutch\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bdutch\b`),
		Transform:     toValueSet(`nl`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bflemish\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bflemish\b`),
		Transform:    toValueSet(`nl`),
		KeepMatching: true,
	},
	// languages: \b(?:DK|danska|dansub|nordic)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:DK|danska|dansub|nordic)\b`),
		Transform:    toValueSet(`da`),
		KeepMatching: true,
	},
	// languages: \b(danish|dinamarqu[eê]s)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(danish|dinamarqu[eê]s)\b`),
		Transform:     toValueSet(`da`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bdan\b(?=.*\.(?:srt|vtt|ssa|ass|sub|idx)$)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bdan\b`),
		ValidateMatch: validateLookahead(`.*\.(?:srt|vtt|ssa|ass|sub|idx)$`, `i`, true),
		Transform:     toValueSet(`da`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.|Sci-)FI|finsk|finsub|nordic)\b
	{
		Gate:  gate("fi", "finsk", "finsub", "nordic"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\b(?:FI|finsk|finsub|nordic)\b`), langAbbrValid(map[string]bool{"finsk": true, "finsub": true, "nordic": true}, func(title string, idxs []int) bool {
			return !langSciI.MatchString(title[:idxs[0]])
		}), true, false, true),
		Transform:    toValueSet(`fi`),
		KeepMatching: true,
	},
	// languages: \bfinnish\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bfinnish\b`),
		Transform:     toValueSet(`fi`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:(?<!w{3}\.\w+\.)SE|swe|swesubs?|sv(?:ensk)?|nordic)\b
	{
		Gate:         gate("se", "swe", "sv", "nordic"),
		Field:        "languages",
		Process:      scanValid("languages", regexp.MustCompile(`(?i)\b(?:SE|swe|swesubs?|sv(?:ensk)?|nordic)\b`), langAbbrValid(map[string]bool{"swe": true, "swesub": true, "swesubs": true, "sv": true, "svensk": true, "nordic": true}, nil), true, false, true),
		Transform:    toValueSet(`sv`),
		KeepMatching: true,
	},
	// languages: \b(swedish|sueco)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(swedish|sueco)\b`),
		Transform:     toValueSet(`sv`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:NOR|norsk|norsub|nordic)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:NOR|norsk|norsub|nordic)\b`),
		Transform:    toValueSet(`no`),
		KeepMatching: true,
	},
	// languages: \b(norwegian|noruegu[eê]s|bokm[aå]l|nob|nor(?=[\]_)]?\.\w{2,4}$))\b
	{
		Gate:  gate("nor", "noruegu", "bokm", "nob"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\b(?:norwegian|noruegu[eê]s|bokm[aå]l|nob|nor)\b`), func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "nor") {
				return langExtSuffix.MatchString(title[idxs[1]:])
			}
			return true
		}, true, false, true),
		Transform:     toValueSet(`no`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:arabic|[aá]rabe|ara)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:arabic|[aá]rabe|ara)\b`),
		Transform:    toValueSet(`ar`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \barab.*(?:audio|lang(?:uage)?|sub(?:s|titles?)?)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\barab.*(?:audio|lang(?:uage)?|sub(?:s|titles?)?)\b`),
		Transform:     toValueSet(`ar`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bar(?=\.(?:ass|ssa|srt|sub|idx)$)
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bar`),
		ValidateMatch: validateLookahead(`\.(?:ass|ssa|srt|sub|idx)$`, `i`, true),
		Transform:     toValueSet(`ar`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:turkish|tur(?:co)?)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:turkish|tur(?:co)?)\b`),
		Transform:     toValueSet(`tr`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(TİVİBU|tivibu|bitturk(.net)?|turktorrent)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(TİVİBU|tivibu|bitturk(.net)?|turktorrent)\b`),
		Transform:     toValueSet(`tr`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bvietnamese\b|\bvie(?=[\]_)]?\.\w{2,4}$)
	{
		Gate:  gate("vie"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\b(?:vietnamese\b|vie)`), func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "vie") {
				return langExtSuffix.MatchString(title[idxs[1]:])
			}
			return true
		}, true, false, true),
		Transform:     toValueSet(`vi`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \bind(?:onesian)?\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bind(?:onesian)?\b`),
		Transform:     toValueSet(`id`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(thai|tailand[eê]s)\b
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(thai|tailand[eê]s)\b`),
		Transform:    toValueSet(`th`),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// languages: \b(THA|tha)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`\b(THA|tha)\b`),
		Transform:     toValueSet(`th`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(?:malay|may(?=[\]_)]?\.\w{2,4}$)|(?<=subs?\([a-z,]+)may)\b
	{
		Gate:  gate("malay", "may"),
		Field: "languages",
		Process: scanValid("languages", regexp.MustCompile(`(?i)\b(?:malay|may)\b`), func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "may") {
				return langExtSuffix.MatchString(title[idxs[1]:]) || langSubsParen.MatchString(title[:idxs[0]])
			}
			return true
		}, true, true, true),
		Transform:    toValueSet(`ms`),
		KeepMatching: true,
	},
	// languages: \bheb(?:rew|raico)?\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bheb(?:rew|raico)?\b`),
		Transform:     toValueSet(`he`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: \b(persian|persa)\b
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(persian|persa)\b`),
		Transform:     toValueSet(`fa`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u3040-\u30ff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{3040}-\x{30ff}]+`),
		Transform:     toValueSet(`ja`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u3400-\u4dbf]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{3400}-\x{4dbf}]+`),
		Transform:     toValueSet(`zh`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u4e00-\u9fff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{4e00}-\x{9fff}]+`),
		Transform:     toValueSet(`zh`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\uf900-\ufaff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{f900}-\x{faff}]+`),
		Transform:     toValueSet(`zh`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\uff66-\uff9f]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{ff66}-\x{ff9f}]+`),
		Transform:     toValueSet(`ja`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0400-\u04ff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0400}-\x{04ff}]+`),
		Transform:     toValueSet(`ru`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0600-\u06ff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0600}-\x{06ff}]+`),
		Transform:     toValueSet(`ar`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0750-\u077f]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0750}-\x{077f}]+`),
		Transform:     toValueSet(`ar`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0c80-\u0cff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0c80}-\x{0cff}]+`),
		Transform:     toValueSet(`kn`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0d00-\u0d7f]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0d00}-\x{0d7f}]+`),
		Transform:     toValueSet(`ml`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0e00-\u0e7f]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0e00}-\x{0e7f}]+`),
		Transform:     toValueSet(`th`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0900-\u097f]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0900}-\x{097f}]+`),
		Transform:     toValueSet(`hi`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0980-\u09ff]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0980}-\x{09ff}]+`),
		Transform:     toValueSet(`bn`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: [\u0a00-\u0a7f]+
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0a00}-\x{0a7f}]+`),
		Transform:     toValueSet(`gu`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// languages: ['custom:infer_language_based_on_naming']
	customInferLanguageBasedOnNaming,
	// subbed: \bmulti(?:ple)?[ .-]*(?:su?$|sub\w*|dub\w*)\b|msub
	{
		Field:     "subbed",
		Pattern:   regexp.MustCompile(`(?i)\bmulti(?:ple)?[ .-]*(?:su?$|sub\w*|dub\w*)\b|msub`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// subbed: \b(?:Official.*?|Dual-?)?sub(s|bed)?\b
	{
		Field:     "subbed",
		Pattern:   regexp.MustCompile(`(?i)\b(?:Official.*?|Dual-?)?sub(s|bed)?\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// dubbed: [\[(\s]?\bmulti(?:ple)?[ .-]*(?:lang(?:uages?)?|audio|VF2)\b\][\[(\s]?
	{
		Field:        "dubbed",
		Pattern:      regexp.MustCompile(`(?i)[\[(\s]?\bmulti(?:ple)?[ .-]*(?:lang(?:uages?)?|audio|VF2)\b\][\[(\s]?`),
		Transform:    toBoolean(),
		Remove:       true,
		KeepMatching: true,
	},
	// dubbed: \btri(?:ple)?[ .-]*(?:audio|dub\w*)\b
	{
		Field:        "dubbed",
		Pattern:      regexp.MustCompile(`(?i)\btri(?:ple)?[ .-]*(?:audio|dub\w*)\b`),
		Transform:    toBoolean(),
		KeepMatching: true,
	},
	// dubbed: \bdual[ .-]*(?:au?$|[aá]udio|line)\b
	{
		Field:        "dubbed",
		Pattern:      regexp.MustCompile(`(?i)\bdual[ .-]*(?:au?$|[aá]udio|line)\b`),
		Transform:    toBoolean(),
		KeepMatching: true,
	},
	// dubbed: \bdual\b(?![ .-]*sub)
	{
		Field:         "dubbed",
		Pattern:       regexp.MustCompile(`(?i)\bdual\b`),
		ValidateMatch: validateLookahead(`[ .-]*sub`, `i`, false),
		Transform:     toBoolean(),
		KeepMatching:  true,
	},
	// dubbed: \b(fan\s?dub)\b
	{
		Field:         "dubbed",
		Pattern:       regexp.MustCompile(`(?i)\b(fan\s?dub)\b`),
		Transform:     toBoolean(),
		Remove:        true,
		SkipFromTitle: true,
	},
	// dubbed: \b(Fan.*)?(?:DUBBED|dublado|dubbing|DUBS?)\b
	{
		Field:     "dubbed",
		Pattern:   regexp.MustCompile(`(?i)\b(Fan.*)?(?:DUBBED|dublado|dubbing|DUBS?)\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// dubbed: \b(?!.*\bsub(s|bed)?\b)([ _\-\[(\.])?(dual|multi)([ _\-\[(\.])?(audio)\b
	{
		Gate:  gate("audio"),
		Field: "dubbed",
		Process: scanValid("dubbed", regexp.MustCompile(`(?i)(?:[ _\-\[(\.])?(?:dual|multi)(?:[ _\-\[(\.])?audio\b`), func(title string, idxs []int) bool {
			return !dubbedSubsAfter.MatchString(title[idxs[0]:])
		}, false, false, false),
		Transform: toBoolean(),
		Remove:    true,
	},
	// dubbed: \b(JAP?(anese)?|ZH)\+ENG?(lish)?|ENG?(lish)?\+(JAP?(anese)?|ZH)\b
	{
		Field:     "dubbed",
		Pattern:   regexp.MustCompile(`(?i)\b(JAP?(anese)?|ZH)\+ENG?(lish)?|ENG?(lish)?\+(JAP?(anese)?|ZH)\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// dubbed: \bMULTi\b
	{
		Field:     "dubbed",
		Pattern:   regexp.MustCompile(`(?i)\bMULTi\b`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// group: ['custom:handle_group']
	customHandleGroup,
	// 3d: (?<=\b[12]\d{3}\b).*\b(3d|sbs|half[ .-]ou|half[ .-]sbs)\b
	{
		Field:         "threeD",
		Pattern:       regexp.MustCompile(`(?i).*\b(3d|sbs|half[ .-]ou|half[ .-]sbs)\b`),
		ValidateMatch: validateLookbehind(`\b[12]\d{3}\b`, `i`, true),
		Transform:     toBoolean(),
		SkipIfFirst:   true,
	},
	// 3d: \b((Half.)?SBS|HSBS)\b
	{
		Field:       "threeD",
		Pattern:     regexp.MustCompile(`(?i)\b((Half.)?SBS|HSBS)\b`),
		Transform:   toBoolean(),
		SkipIfFirst: true,
	},
	// 3d: \bBluRay3D\b
	{
		Field:       "threeD",
		Pattern:     regexp.MustCompile(`(?i)\bBluRay3D\b`),
		Transform:   toBoolean(),
		SkipIfFirst: true,
	},
	// 3d: \bBD3D\b
	{
		Field:       "threeD",
		Pattern:     regexp.MustCompile(`(?i)\bBD3D\b`),
		Transform:   toBoolean(),
		SkipIfFirst: true,
	},
	// 3d: \b3D\b
	{
		Field:       "threeD",
		Pattern:     regexp.MustCompile(`(?i)\b3D\b`),
		Transform:   toBoolean(),
		SkipIfFirst: true,
	},
	// size: \b(\d+(\.\d+)?\s?(MB|GB|TB))\b
	{
		Field:   "size",
		Pattern: regexp.MustCompile(`(?i)\b(\d+(\.\d+)?\s?(MB|GB|TB))\b`),
		Remove:  true,
	},
	// site: \b(?:www?.?)?(?:\w+\-)?\w+[\.\s](?:com|org|net|ms|tv|mx|co|party|vip|nu|pics)\b
	{
		Field:         "site",
		Pattern:       regexp.MustCompile(`(?i)\b(?:www?.?)?(?:\w+\-)?\w+[\.\s](?:com|org|net|ms|tv|mx|co|\.party|vip|nu|pics)\b`),
		ValidateMatch: validateSiteLeavesTitle(),
		Transform:     toValueSub(`$1`),
		Remove:        true,
	},
	// site: rarbg|torrentleech|(?:the)?piratebay
	{
		Field:     "site",
		Pattern:   regexp.MustCompile(`(?i)rarbg|torrentleech|(?:the)?piratebay`),
		Transform: toValueSub(`$1`),
		Remove:    true,
	},
	// site: \[([^\]]+\.[^\]]+)\](?=\.\w{2,4}$|\s)
	{
		Field:         "site",
		Pattern:       regexp.MustCompile(`(?i)\[([^\]]+\.[^\]]+)\]`),
		ValidateMatch: validateLookahead(`\.\w{2,4}$|\s`, `i`, true),
		Transform:     toValueSub(`$1`),
		Remove:        true,
	},
	// network: \bATVP?\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bATVP?\b`),
		Transform: toValue(`Apple TV`),
		Remove:    true,
	},
	// network: \bAMZN\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAMZN\b`),
		Transform: toValue(`Amazon`),
		Remove:    true,
	},
	// network: \bNF|Netflix\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bNF|Netflix\b`),
		Transform: toValue(`Netflix`),
		Remove:    true,
	},
	// network: \bNICK(elodeon)?\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bNICK(elodeon)?\b`),
		Transform: toValue(`Nickelodeon`),
		Remove:    true,
	},
	// network: \bDSNY?P?\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bDSNY?P?\b`),
		Transform: toValue(`Disney`),
		Remove:    true,
	},
	// network: \bH(MAX|BO)\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bH(MAX|BO)\b`),
		Transform: toValue(`HBO`),
		Remove:    true,
	},
	// network: \bHULU\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bHULU\b`),
		Transform: toValue(`Hulu`),
		Remove:    true,
	},
	// network: \bCBS\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bCBS\b`),
		Transform: toValue(`CBS`),
		Remove:    true,
	},
	// network: \bNBC\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bNBC\b`),
		Transform: toValue(`NBC`),
		Remove:    true,
	},
	// network: \bAMC\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAMC\b`),
		Transform: toValue(`AMC`),
		Remove:    true,
	},
	// network: \bPBS\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bPBS\b`),
		Transform: toValue(`PBS`),
		Remove:    true,
	},
	// network: \b(Crunchyroll|[. -]CR[. -])\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\b(Crunchyroll|[. -]CR[. -])\b`),
		Transform: toValue(`Crunchyroll`),
		Remove:    true,
	},
	// network: \bVICE\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`\bVICE\b`),
		Transform: toValue(`VICE`),
		Remove:    true,
	},
	// network: \bSony\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bSony\b`),
		Transform: toValue(`Sony`),
		Remove:    true,
	},
	// network: \bHallmark\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bHallmark\b`),
		Transform: toValue(`Hallmark`),
		Remove:    true,
	},
	// network: \bAdult.?Swim\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAdult.?Swim\b`),
		Transform: toValue(`Adult Swim`),
		Remove:    true,
	},
	// network: \bAnimal.?Planet|ANPL\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAnimal.?Planet|ANPL\b`),
		Transform: toValue(`Animal Planet`),
		Remove:    true,
	},
	// network: \bCartoon.?Network(.TOONAMI.BROADCAST)?\b
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bCartoon.?Network(.TOONAMI.BROADCAST)?\b`),
		Transform: toValue(`Cartoon Network`),
		Remove:    true,
	},
	// extension: \.(3g2|3gp|avi|flv|mkv|mk3d|mov|mp2|mp4|m4v|mpe|mpeg|mpg|mpv|webm|wmv|ogm|divx|ts|m2ts|iso|vob|sub|idx|ttxt|txt|smi|srt|ssa|ass|vtt|nfo|html)$
	{
		Field:     "extension",
		Pattern:   regexp.MustCompile(`(?i)\.(3g2|3gp|avi|flv|mkv|mk3d|mov|mp2|mp4|m4v|mpe|mpeg|mpg|mpv|webm|wmv|ogm|divx|ts|m2ts|iso|vob|sub|idx|ttxt|txt|smi|srt|ssa|ass|vtt|nfo|html)$`),
		Transform: toLowercase(),
		Remove:    true,
	},
	// audio: \bMP3\b
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bMP3\b`),
		Transform:    toValueSet(`MP3`),
		Remove:       true,
		KeepMatching: true,
	},
	// group: \(([\w-]+)\)(?:$|\.\w{2,4}$)
	{
		Field:   "group",
		Pattern: regexp.MustCompile(`\(([\w-]+)\)(?:$|\.\w{2,4}$)`),
	},
	// group: \b(INFLATE|DEFLATE)\b
	{
		Field:     "group",
		Pattern:   regexp.MustCompile(`\b(INFLATE|DEFLATE)\b`),
		Transform: toValueSub(`$1`),
		Remove:    true,
	},
	// group: \b(?:Erai-raws|Erai-raws\.com)\b
	{
		Field:     "group",
		Pattern:   regexp.MustCompile(`(?i)\b(?:Erai-raws|Erai-raws\.com)\b`),
		Transform: toValue(`Erai-raws`),
		Remove:    true,
	},
	// group: ^\[([^[\]]+)]
	{
		Field:   "group",
		Pattern: regexp.MustCompile(`^\[([^[\]]+)]`),
	},
	// group: ['custom:handle_group_exclusion']
	customHandleGroupExclusion,
	// trash: acesse o original
	{
		Field:     "trash",
		Pattern:   regexp.MustCompile(`(?i)acesse o original`),
		Transform: toBoolean(),
		Remove:    true,
	},
	// title: \bHigh.?Quality\b
	{
		Field:         "title",
		Pattern:       regexp.MustCompile(`(?i)\bHigh.?Quality\b`),
		Remove:        true,
		SkipFromTitle: true,
	},
}
