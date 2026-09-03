package parser

// Hand-written Go ports of PTT's custom function handlers and the transformer
// helpers referenced by the generated table (parser/handlers_gen.go).
// Python source of truth: tools/upstream/ptt/handlers.py

import (
	_ "embed"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

//go:embed keywords/combined-keywords.txt
var adultKeywordsRaw string

// adultKeywords returns the cleaned keyword list; it doubles as the adult
// handler's prefilter gate (containment of any keyword is the exact
// necessary condition for the handler to fire).
func adultKeywords() []string {
	lines := strings.Split(adultKeywordsRaw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		kw := strings.ToLower(strings.TrimSpace(line))
		if kw == "" {
			continue
		}
		out = append(out, kw)
	}
	return out
}

func buildAdultPattern() *regexp.Regexp {
	kws := adultKeywords()
	escaped := make([]string, 0, len(kws))
	for _, kw := range kws {
		escaped = append(escaped, regexp.QuoteMeta(kw))
	}
	// substring containment (no word boundaries): any keyword occurring
	// anywhere in the lowercased title marks it adult
	return regexp.MustCompile(`(?i)` + strings.Join(escaped, "|"))
}

var adultKeywordsRegex = buildAdultPattern()

// def is_adult_content: keyword containment; sets adult without touching the
// title or match bookkeeping
var customAdult = handler{
	Field: "adult",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(bool); ok && v {
			return m
		}
		if adultKeywordsRegex.MatchString(title) {
			m.value = true
		}
		return m
	},
}

// ---------------------------------------------------------------------------
// Transformer helpers used by the generated table
// ---------------------------------------------------------------------------

// toValueSub implements PTT's value("... $1 ...") templates.
func toValueSub(template string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = strings.ReplaceAll(template, "$1", v)
		} else {
			m.value = template
		}
	}
}

var firstIntRegex = regexp.MustCompile(`\d+`)

// toTransformedResolution implements PTT's transform_resolution.
func toTransformedResolution() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			return
		}
		lower := strings.ToLower(v)
		switch {
		case strings.Contains(lower, "2160"), strings.Contains(lower, "4k"):
			m.value = "2160p"
		case strings.Contains(lower, "1440"), strings.Contains(lower, "2k"):
			m.value = "1440p"
		case strings.Contains(lower, "1080"):
			m.value = "1080p"
		case strings.Contains(lower, "720"):
			m.value = "720p"
		case strings.Contains(lower, "480"):
			m.value = "480p"
		case strings.Contains(lower, "360"):
			m.value = "360p"
		case strings.Contains(lower, "240"):
			m.value = "240p"
		}
	}
}

// ---------------------------------------------------------------------------
// Custom function handlers (Process-based, no Pattern)
// ---------------------------------------------------------------------------

var bitDepthCleanupRegex = regexp.MustCompile(`[ -]`)

// def handle_bit_depth: strip spaces/hyphens from the matched bit depth.
var customHandleBitDepth = handler{
	Field: "bitDepth",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && v != "" {
			m.value = bitDepthCleanupRegex.ReplaceAllString(v, "")
		}
		return m
	},
}

var codecCleanupRegex = regexp.MustCompile(`[ .-]`)

// def handle_space_in_codec: strip separators from the matched codec.
var customHandleSpaceInCodec = handler{
	Field: "codec",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && v != "" {
			m.value = codecCleanupRegex.ReplaceAllString(v, "")
		}
		return m
	},
}

var volumesFallbackRegex = regexp.MustCompile(`(?i)\bvol(?:ume)?[. -]*(\d{1,2})`)

// def handle_volumes: search from the year match onward for a single volume.
var customHandleVolumes = handler{
	Gate:  gate("vol"),
	Field: "volumes",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		startIndex := 0
		if ym, ok := result["year"]; ok {
			startIndex = min(ym.firstMIndex, len(title))
		}
		idxs := volumesFallbackRegex.FindStringSubmatchIndex(title[startIndex:])
		if idxs == nil {
			return m
		}
		if num, err := strconv.Atoi(title[startIndex:][idxs[2]:idxs[3]]); err == nil {
			m.value = []int{num}
			m.mIndex = startIndex + idxs[0]
			m.mValue = title[startIndex:][idxs[0]:idxs[1]]
			m.remove = true
			m.matchedNow = true
		}
		return m
	},
}

// def handle_episodes: positional fallback that looks for a bare episode
// number between the known landmarks (year/seasons start, resolution/quality/
// codec/audio end).
//
// Python patterns use variable-length lookarounds; RE2 cannot express them,
// so the candidate matches are validated in code:
//
//	beginning: (?<!movie\W*|film\W*|^)(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})(?:a|b|v\d|\.\d)?(?:\W|$)(?!movie|film|\d+)(?<!\[(?:480|720|1080)\])
//	middle:    ^(?:[([-][ .]?)?(\d{1,4})(?:a|b|v\d)?(?:\W|$)(?!movie|film)(?!\[(480|720|1080)\])
var (
	epsBeginningCore    = regexp.MustCompile(`(?i)(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})(?:a|b|v\d|\.\d)?(?:[^\p{L}\p{N}_]|$)`)
	epsMiddleCore       = regexp.MustCompile(`(?i)^(?:[([-][ .]?)?(\d{1,4})(?:a|b|v\d)?(?:[^\p{L}\p{N}_]|$)`)
	epsPrefixMoviefilm  = regexp.MustCompile(`(?i)(?:movie|film)\W*$`)
	epsSuffixMoviefilmD = regexp.MustCompile(`(?i)^(?:movie|film|\d)`)
	epsSuffixMoviefilm  = regexp.MustCompile(`(?i)^(?:movie|film)`)
	epsResBracketEnd    = regexp.MustCompile(`\[(?:480|720|1080)\]$`)
	epsResBracketNext   = regexp.MustCompile(`^\[(?:480|720|1080)\]`)
)

var customHandleEpisodes = handler{
	Field: "episodes",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}

		startIndex := 0
		for _, f := range []string{"year", "seasons"} {
			if fm, ok := result[f]; ok && fm.firstMIndex > 0 {
				if startIndex == 0 || fm.firstMIndex < startIndex {
					startIndex = fm.firstMIndex
				}
			}
		}
		endIndex := len(title)
		for _, f := range []string{"resolution", "quality", "codec", "audio"} {
			if fm, ok := result[f]; ok && fm.firstMIndex > 0 && fm.firstMIndex < endIndex {
				endIndex = fm.firstMIndex
			}
		}
		startIndex = min(startIndex, len(title))
		endIndex = min(endIndex, len(title))
		if startIndex > endIndex {
			startIndex = 0
		}

		beginningTitle := title[:endIndex]
		middleTitle := title[startIndex:endIndex]

		// beginning pattern with emulated lookarounds
		offset := 0
		for offset < len(beginningTitle) {
			idxs := epsBeginningCore.FindStringSubmatchIndex(beginningTitle[offset:])
			if idxs == nil {
				break
			}
			s, e := idxs[0]+offset, idxs[1]+offset
			ok := s != 0 &&
				!epsPrefixMoviefilm.MatchString(beginningTitle[:s]) &&
				!epsSuffixMoviefilmD.MatchString(beginningTitle[e:]) &&
				!epsResBracketEnd.MatchString(beginningTitle[:e])
			if ok {
				numStr := beginningTitle[idxs[2]+offset : idxs[3]+offset]
				if num, err := strconv.Atoi(numStr); err == nil {
					m.value = []int{num}
					m.mIndex = s
					m.mValue = ""
					m.matchedNow = true
					return m
				}
			}
			offset = e
			if e == s {
				offset++
			}
		}

		// middle pattern (anchored)
		idxs := epsMiddleCore.FindStringSubmatchIndex(middleTitle)
		if idxs != nil {
			e := idxs[1]
			if !epsSuffixMoviefilm.MatchString(middleTitle[e:]) && !epsResBracketNext.MatchString(middleTitle[e:]) {
				numStr := middleTitle[idxs[2]:idxs[3]]
				if num, err := strconv.Atoi(numStr); err == nil {
					m.value = []int{num}
					m.mIndex = startIndex + idxs[0]
					m.mValue = ""
					m.matchedNow = true
				}
			}
		}
		return m
	},
}

var (
	animeTitleRegex = regexp.MustCompile(`One.*?Piece|Bleach|Naruto`)
	animeEpnumRegex = regexp.MustCompile(`\b\d{1,4}\b`)
)

// def handle_anime_eps: One Piece / Bleach / Naruto single-episode fallback.
var customHandleAnimeEps = handler{
	Field: "episodes",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil || !animeTitleRegex.MatchString(title) {
			return m
		}
		idxs := animeEpnumRegex.FindStringIndex(title)
		if idxs == nil {
			return m
		}
		if num, err := strconv.Atoi(title[idxs[0]:idxs[1]]); err == nil {
			m.value = []int{num}
			m.mIndex = idxs[0]
			m.mValue = ""
			m.matchedNow = true
		}
		return m
	},
}

var (
	ptEpnameRegex  = regexp.MustCompile(`(?i)capitulo|ao`)
	ptDubladoRegex = regexp.MustCompile(`(?i)dublado`)
)

// def infer_language_based_on_naming: Portuguese naming conventions.
var customInferLanguageBasedOnNaming = handler{
	Field: "languages",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		vs, _ := m.value.(*valueSet[any])
		if vs != nil && (vs.exists("pt") || vs.exists("es")) {
			return m
		}
		matchedByNaming := false
		if em, ok := result["episodes"]; ok && ptEpnameRegex.MatchString(em.firstMValue) {
			matchedByNaming = true
		}
		if !matchedByNaming && ptDubladoRegex.MatchString(title) {
			matchedByNaming = true
		}
		if matchedByNaming {
			if vs == nil {
				vs = &valueSet[any]{existMap: map[any]struct{}{}, values: []any{}}
			}
			m.value = vs.append("pt")
		}
		return m
	},
}

// def handle_group: drop a bracketed group that overlaps other matches.
var customHandleGroup = handler{
	Field: "group",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		v, _ := m.value.(string)
		if v == "" || !strings.HasPrefix(m.firstMValue, "[") || !strings.HasSuffix(m.firstMValue, "]") {
			return m
		}
		endIndex := m.firstMIndex + len(m.firstMValue)
		for f, fm := range result {
			if f != "group" && fm.firstMIndex < endIndex {
				m.value = nil
				break
			}
		}
		return m
	},
}

// def handle_group_exclusion: drop "-" or empty groups.
var customHandleGroupExclusion = handler{
	Field: "group",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && (v == "-" || v == "") {
			m.value = nil
		}
		return m
	},
}

// ---------------------------------------------------------------------------
// Hand-translated handlers for mid-pattern lookarounds (see gen_overrides.py)
// ---------------------------------------------------------------------------

// scene: ^(?=.*(\b\d{3,4}p\b).*([_. ]WEB[_. ])(?!DL)\b)|\b(-CAKES|-GGEZ|...)
// Case-sensitive. Either a zero-width match at ^ when "<res>p ... WEB" (not
// WEB-DL) appears anywhere, or a known scene-group suffix.
var (
	sceneResRegex    = regexp.MustCompile(`\b\d{3,4}p\b`)
	sceneWebRegex    = regexp.MustCompile(`[_. ]WEB[_. ]`)
	sceneGroupsRegex = regexp.MustCompile(`\b(-CAKES|-GGEZ|-GGWP|-GLHF|-GOSSIP|-NAISU|-KOGI|-PECULATE|-SLOT|-EDITH|-ETHEL|-ELEANOR|-B2B|-SPAMnEGGS|-FTP|-DiRT|-SYNCOPY|-BAE|-SuccessfulCrab|-NHTFS|-SURCODE|-B0MBARDIERS)`)
)

var customScene = handler{
	Field: "scene",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		for _, web := range sceneWebRegex.FindAllStringIndex(title, -1) {
			if !sceneResRegex.MatchString(title[:web[0]]) {
				continue
			}
			if strings.HasPrefix(title[web[1]:], "DL") {
				continue
			}
			m.value = true
			m.mIndex = 0
			m.mValue = ""
			m.matchedNow = true
			return m
		}
		if idxs := sceneGroupsRegex.FindStringIndex(title); idxs != nil {
			m.value = true
			m.mIndex = idxs[0]
			m.mValue = title[idxs[0]:idxs[1]]
			m.matchedNow = true
		}
		return m
	},
}

// extras Featurette/Sample/Trailer:
// (?:(?<=\b(?:19\d{2}|20\d{2})\b.*)\bX\b|\bX\b(?!.*\b(?:19\d{2}|20\d{2})\b))
// Matches X when a year appears before it, or no year appears after it.
var extrasYearRegex = regexp.MustCompile(`\b(?:19\d{2}|20\d{2})\b`)

func extrasHandler(word *regexp.Regexp, after *regexp.Regexp, val string, g *gateLits) handler {
	// python: (?:(?<=\byear\b.*)\bWORD\b|\bWORD\b(?!.*\bAFTER\b))
	// The context regexes are evaluated against the whole title (so \b sees
	// real neighbors) and filtered by position.
	return handler{
		Field:         "extras",
		SkipFromTitle: true,
		Gate:          g,
		Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
			vs, _ := m.value.(*valueSet[any])
			for _, idxs := range word.FindAllStringIndex(title, -1) {
				yearBefore := false
				for _, y := range extrasYearRegex.FindAllStringIndex(title, -1) {
					if y[1] <= idxs[0] {
						yearBefore = true
						break
					}
				}
				afterHit := false
				for _, a := range after.FindAllStringIndex(title, -1) {
					if a[0] >= idxs[1] {
						afterHit = true
						break
					}
				}
				if yearBefore || !afterHit {
					if vs == nil {
						vs = &valueSet[any]{existMap: map[any]struct{}{}, values: []any{}}
					}
					m.value = vs.append(val)
					m.mIndex = idxs[0]
					m.mValue = title[idxs[0]:idxs[1]]
					m.matchedNow = true
					break
				}
			}
			return m
		},
	}
}

var (
	customExtrasFeaturette = extrasHandler(
		regexp.MustCompile(`(?i)\bFeaturettes?\b`), extrasYearRegex, "Featurette", gate("featurette"))
	customExtrasSample = extrasHandler(
		regexp.MustCompile(`(?i)\bSample\b`), extrasYearRegex, "Sample", gate("sample"))
	customExtrasTrailer = extrasHandler(
		regexp.MustCompile(`(?i)\bTrailers?\b`),
		regexp.MustCompile(`(?i)\b(?:19\d{2}|20\d{2}|.(Park|And))\b`), "Trailer", gate("trailer"))
)

// trash CAM: \b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\()\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b  (i)
var (
	trashCamCoreRegex   = regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?(CAM)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b`)
	trashCamRejectRegex = regexp.MustCompile(`(?i)^.?(?:S|E|\()\d+`)
)

var customTrashCam = handler{
	Gate:  gate("cam"),
	Field: "trash",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		offset := 0
		for offset < len(title) {
			idxs := trashCamCoreRegex.FindStringSubmatchIndex(title[offset:])
			if idxs == nil {
				return m
			}
			camEnd := offset + idxs[3]
			if !trashCamRejectRegex.MatchString(title[camEnd:]) {
				m.value = true
				m.mIndex = offset + idxs[0]
				m.mValue = title[offset+idxs[0] : offset+idxs[1]]
				m.matchedNow = true
				return m
			}
			offset += idxs[1]
			if idxs[1] == idxs[0] {
				offset++
			}
		}
		return m
	},
}

// group (dash form):
// - ?(?!\d+$|S\d+|\d+x|ep?\d+|[^[]+]$)([^\-. []+[^\-. [)\]\d][^\-. [)\]]*)(?:\[[\w.-]+])?(?=\.\w{2,4}$|$)  (i)
var (
	groupDashCoreRegex   = regexp.MustCompile(`(?i)- ?([^\-. []+[^\-. [)\]\d][^\-. [)\]]*)(?:\[[\w.-]+])?`)
	groupDashRejectRegex = regexp.MustCompile(`(?i)^(?:\d+$|S\d+|\d+x|ep?\d+|[^[]+]$)`)
	groupDashEndRegex    = regexp.MustCompile(`(?i)^\.\w{2,4}$`)
)

var customGroupDash = handler{
	Field: "group",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		offset := 0
		for offset < len(title) {
			idxs := groupDashCoreRegex.FindStringSubmatchIndex(title[offset:])
			if idxs == nil {
				return m
			}
			s, e := offset+idxs[0], offset+idxs[1]
			nameStart := offset + idxs[2]
			suffix := title[e:]
			if !groupDashRejectRegex.MatchString(title[nameStart:]) &&
				(suffix == "" || groupDashEndRegex.MatchString(suffix)) {
				m.value = title[offset+idxs[2] : offset+idxs[3]]
				m.mIndex = s
				m.mValue = title[s:e]
				m.matchedNow = true
				return m
			}
			offset = e
			if e == s {
				offset++
			}
		}
		return m
	},
}

// toIntString implements PTT's integer transformer for string-typed fields
// (year): the parsed integer is kept in decimal string form.
func toIntString() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if num, err := strconv.Atoi(stripNonDigits(v)); err == nil {
				m.value = strconv.Itoa(num)
				return
			}
		}
		m.value = nil
	}
}

func stripNonDigits(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}

// toFirstIntString implements PTT's first_integer for string-typed fields.
func toFirstIntString() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if s := firstIntRegex.FindString(v); s != "" {
				m.value = s
				return
			}
		}
		m.value = nil
	}
}

// ---------------------------------------------------------------------------
// scanValid: PTT-equivalent of a regex with embedded lookarounds — scans
// matches left to right until the validator accepts one (Python's regex
// engine does this natively; RE2 needs the explicit loop).
// ---------------------------------------------------------------------------

func scanValid(field string, re *regexp.Regexp, valid func(title string, idxs []int) bool, isSet bool, skipIfFirst bool, keepMatching bool) hProcessor {
	return func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		// python skipIfAlreadyFound: a found single-value field ends the handler
		if !keepMatching && m.value != nil {
			return m
		}
		offset := 0
		for offset <= len(title) {
			idxs := re.FindStringSubmatchIndex(title[offset:])
			if idxs == nil {
				return m
			}
			for i := range idxs {
				if idxs[i] >= 0 {
					idxs[i] += offset
				}
			}
			if valid == nil || valid(title, idxs) {
				if skipIfFirst {
					hasOther, hasBefore := false, false
					for f, fm := range result {
						if f != field {
							hasOther = true
							if idxs[0] >= fm.firstMIndex {
								hasBefore = true
							}
						}
					}
					if hasOther && !hasBefore {
						return m
					}
				}
				m.mIndex = idxs[0]
				m.mValue = title[idxs[0]:idxs[1]]
				if isSet {
					if _, ok := m.value.(*valueSet[any]); !ok {
						m.value = &valueSet[any]{existMap: map[any]struct{}{}, values: []any{}}
					}
				} else {
					if len(idxs) > 2 && idxs[2] >= 0 && idxs[3] > idxs[2] {
						m.value = title[idxs[2]:idxs[3]]
					} else {
						m.value = m.mValue
					}
				}
				m.matchedNow = true
				return m
			}
			if idxs[1] > idxs[0] {
				offset = idxs[1]
			} else {
				offset = idxs[0] + 1
			}
		}
		return m
	}
}

// shared context regexes for the language-code handlers
var (
	langWwwI         = regexp.MustCompile(`(?i)w{3}\.\w+\.$`)
	langWwwCs        = regexp.MustCompile(`w{3}\.\w+\.$`)
	langSciI         = regexp.MustCompile(`(?i)Sci-$`)
	langTeAviv       = regexp.MustCompile(`(?i)^\W*aviv`)
	langBnReject     = regexp.MustCompile(`(?i)^(?:.\bThe|and|of\b)`)
	langYtsPrefix    = regexp.MustCompile(`YTS\.$`)
	langExtSuffix    = regexp.MustCompile(`(?i)^[\]_)]?\.\w{2,4}$`)
	langSubsParen    = regexp.MustCompile(`(?i)subs?\([a-z,]+$`)
	langCodes2Suffix = regexp.MustCompile(`(?i)^[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,}`)
	langCodes2Prefix = regexp.MustCompile(`(?i)[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,}$`)
	langCode1Prefix  = regexp.MustCompile(`(?i)[ .,/-]+[A-Z]{2}[ .,/-]+$`)
	langCode1Suffix  = regexp.MustCompile(`(?i)^[ .,/-]+[A-Z]{2}[ .,/-]+`)
	langItCodesCs    = regexp.MustCompile(`^[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,}`)
	langHrSubSuffix  = regexp.MustCompile(`(?i)^[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub`)
	dubbedSubsAfter  = regexp.MustCompile(`(?i)\bsub(s|bed)?\b`)
)

// ---------------------------------------------------------------------------
// Bare uppercase DE as a German tag.
//
// GER, DEU and GERMAN are unambiguous; a bare "de" is also the commonest word
// in Romance titles ("La Casa de Papel", "Anatomia De Grey", "Espanol De
// Espana"), so PTT only accepted it inside a run of two-letter codes. The
// handlers below widen that in three directions, each pairing a CASE-SENSITIVE
// uppercase match with positive evidence that the token sits in metadata
// rather than prose. Lower- and title-case "de"/"De" never reach them.
//
// Note these run at handler ~286 of 432, by which point the metadata handlers
// have spliced their own matches out of the working title: resolution, source
// and codec are already gone, so only DL, MULTi, DUBBED and friends survive to
// serve as neighbours.
// ---------------------------------------------------------------------------

// deTagSurvivors are the release-metadata tokens still present in the working
// title when the language handlers run. Adjacency to one of them is evidence
// no Romance preposition can offer.
var deTagAdjacent = regexp.MustCompile(
	`\b(?:DE[ ._-](?i:DL|MULTI|DUBBED|SUBBED|SUBS|SYNC|LINE|AUDIO|TRUEDEF)` +
		`|(?i:DL|MULTI|DUBBED|SUBBED|SUBS|SYNC|LINE|AUDIO)[ ._-]DE)\b`)

// jaTagAdjacent is the same adjacency evidence for a bare uppercase JA
// (jhin #39); the validators below are code-agnostic and are reused as-is.
var jaTagAdjacent = regexp.MustCompile(
	`\b(?:JA[ ._-](?i:DL|MULTI|DUBBED|SUBBED|SUBS|SYNC|LINE|AUDIO)` +
		`|(?i:DL|MULTI|DUBBED|SUBBED|SUBS|SYNC|LINE|AUDIO)[ ._-]JA)\b`)

// deAdjacentWord returns the alphabetic token abutting off — searching back
// when dir is -1, forward when +1 — together with the byte offset it starts
// at. Separators are skipped; a digit run or a boundary yields "", because
// only a real word is evidence of prose.
func deAdjacentWord(title string, off, dir int) (string, int) {
	if dir < 0 {
		i := off
		for i > 0 {
			r, sz := utf8.DecodeLastRuneInString(title[:i])
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				break
			}
			i -= sz
		}
		end := i
		for i > 0 {
			r, sz := utf8.DecodeLastRuneInString(title[:i])
			if !unicode.IsLetter(r) {
				break
			}
			i -= sz
		}
		return title[i:end], i
	}
	i := off
	for i < len(title) {
		r, sz := utf8.DecodeRuneInString(title[i:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			break
		}
		i += sz
	}
	start := i
	for i < len(title) {
		r, sz := utf8.DecodeRuneInString(title[i:])
		if !unicode.IsLetter(r) {
			break
		}
		i += sz
	}
	return title[start:i], start
}

// deIsPreposition reports whether the span is flanked by words on both sides,
// as "Espanol DE Espana" is and a metadata token run is not.
//
// A neighbour inside the still-unconsumed title region counts only when the
// span itself is inside it too: past the title the word to the left is just
// the title's last word ("Movie.Name..DE.DL") and carries no signal, whereas
// a span sitting between two title words is prose by construction.
func deIsPreposition(title string, start, end, titleRegion int) bool {
	spanInTitle := start < titleRegion
	counts := func(word string, off int) bool {
		if utf8.RuneCountInString(word) < 3 {
			return false
		}
		return spanInTitle || off >= titleRegion
	}
	before, bOff := deAdjacentWord(title, start, -1)
	after, aOff := deAdjacentWord(title, end, 1)
	return counts(before, bOff) && counts(after, aOff)
}

// deIsLangCode reports whether word is an uppercase two-letter code the parser
// knows as a language.
func deIsLangCode(word string) bool {
	if len(word) != 2 || word != strings.ToUpper(word) {
		return false
	}
	_, ok := languageNames[strings.ToLower(word)]
	return ok
}

// deCodeRun expands the span across abutting uppercase two-letter language
// codes and returns the maximal run, reporting whether it caught any code
// besides the DE itself. "IT EN FR DE ES" is one run; the DE in
// "EL CLUB DE LA LUCHA" pairs only with LA.
func deCodeRun(title string, start, end int) (int, int, bool) {
	paired := false
	for {
		word, off := deAdjacentWord(title, start, -1)
		if !deIsLangCode(word) {
			break
		}
		start, paired = off, true
	}
	for {
		word, off := deAdjacentWord(title, end, 1)
		if !deIsLangCode(word) {
			break
		}
		end, paired = off+len(word), true
	}
	return start, end, paired
}

// validateDEPastTitle accepts an uppercase DE that the title no longer covers
// — everything before endOfTitle has been claimed by the metadata handlers —
// unless it reads as a preposition.
func validateDEPastTitle() *hMatchValidator {
	return &hMatchValidator{
		span: func(input string, match []int, ctx matchContext) bool {
			region := len(runePrefix(input, ctx.endOfTitle))
			if match[0] < region {
				return false
			}
			if langWwwI.MatchString(input[:match[0]]) {
				return false // a .DE domain, not a language tag
			}
			return !deIsPreposition(input, match[0], match[1], region)
		},
	}
}

// validateDELangCodePair accepts an uppercase DE abutting at least one other
// known two-letter language code, provided the run as a whole is not embedded
// in prose. This is the existing code-run gate loosened to runs of two.
func validateDELangCodePair() *hMatchValidator {
	return &hMatchValidator{
		span: func(input string, match []int, ctx matchContext) bool {
			if langWwwI.MatchString(input[:match[0]]) {
				return false
			}
			start, end, paired := deCodeRun(input, match[0], match[1])
			if !paired {
				return false
			}
			return !deIsPreposition(input, start, end, len(runePrefix(input, ctx.endOfTitle)))
		},
	}
}

// validateJALangCodePair is validateDELangCodePair for a bare uppercase JA
// (jhin #39), with one extra demand: the run must contain a language code
// other than JA itself. "JA EN" is a language pair; the "JA JA" that opens
// "JA JA DING DONG" is a lyric.
func validateJALangCodePair() *hMatchValidator {
	return &hMatchValidator{
		span: func(input string, match []int, ctx matchContext) bool {
			if langWwwI.MatchString(input[:match[0]]) {
				return false
			}
			start, end, paired := deCodeRun(input, match[0], match[1])
			if !paired {
				return false
			}
			other := false
			for _, word := range strings.FieldsFunc(input[start:end], func(r rune) bool { return !unicode.IsLetter(r) }) {
				if word != "JA" && deIsLangCode(word) {
					other = true
					break
				}
			}
			if !other {
				return false
			}
			return !deIsPreposition(input, start, end, len(runePrefix(input, ctx.endOfTitle)))
		},
	}
}

// langAbbrValid builds a validator for the PTT pattern family
// \b(?:(?<!w{3}\.\w+\.)ABBR|fullword)\b — the www-lookbehind (and optional
// extra reject) applies only when the abbreviated alternative matched.
func langAbbrValid(fullwords map[string]bool, extra func(title string, idxs []int) bool) func(string, []int) bool {
	return func(title string, idxs []int) bool {
		v := strings.ToLower(title[idxs[0]:idxs[1]])
		if fullwords[v] {
			return true
		}
		if langWwwI.MatchString(title[:idxs[0]]) {
			return false
		}
		if extra != nil && !extra(title, idxs) {
			return false
		}
		return true
	}
}

// ---------------------------------------------------------------------------
// toPttDate replicates PTT's transformers.date(): sanitize non-word runs to
// spaces, shorten the 4-letter month abbreviations (Janu/Febr/... — full month
// names are intentionally NOT shortened, matching Python), then try each
// arrow format's Go layout equivalent.
// ---------------------------------------------------------------------------

var (
	pttDateSanitizeRegex = regexp.MustCompile(`\W+`)
	pttDateMonth4Regex   = regexp.MustCompile(`(?i)\b(janu|febr|marc|apri|may|june|july|augu|sept|octo|nove|dece)\b`)
	pttDateOrdinalRegex  = regexp.MustCompile(`\b(\d{1,2})(?:St|Nd|Rd|Th|st|nd|rd|th)\b`)
	pttDateWordRegex     = regexp.MustCompile(`[A-Za-z]+`)
	pttMonthShort        = map[string]string{
		"janu": "Jan", "febr": "Feb", "marc": "Mar", "apri": "Apr",
		"may": "May", "june": "Jun", "july": "Jul", "augu": "Aug",
		"sept": "Sep", "octo": "Oct", "nove": "Nov", "dece": "Dec",
	}
)

func toPttDate(formats ...string) hTransformer {
	type layout struct {
		golayout string
		ordinal  bool
	}
	layouts := make([]layout, 0, len(formats))
	for _, f := range formats {
		l := layout{}
		switch f {
		case "YYYY MM DD":
			l.golayout = "2006 1 2"
		case "DD MM YYYY":
			l.golayout = "2 1 2006"
		case "MM DD YY":
			l.golayout = "1 2 06"
		case "DD MM YY":
			l.golayout = "2 1 06"
		case "YY MM DD":
			l.golayout = "06 1 2"
		case "DD MMM YYYY":
			l.golayout = "2 Jan 2006"
		case "Do MMM YYYY":
			l.golayout, l.ordinal = "2 Jan 2006", true
		case "Do MMMM YYYY":
			l.golayout, l.ordinal = "2 January 2006", true
		case "DD MMM YY":
			l.golayout = "2 Jan 06"
		case "YYYYMMDD":
			l.golayout = "20060102"
		}
		layouts = append(layouts, l)
	}
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			m.value = nil
			return
		}
		sanitized := strings.TrimSpace(pttDateSanitizeRegex.ReplaceAllString(v, " "))
		sanitized = pttDateMonth4Regex.ReplaceAllStringFunc(sanitized, func(s string) string {
			return pttMonthShort[strings.ToLower(s)]
		})
		// arrow parses month names case-insensitively; Go's time.Parse does
		// not, so normalize word casing
		norm := pttDateWordRegex.ReplaceAllStringFunc(sanitized, func(s string) string {
			return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		})
		for _, l := range layouts {
			cand := norm
			if l.ordinal {
				cand = pttDateOrdinalRegex.ReplaceAllString(cand, "$1")
			}
			if t, err := time.Parse(l.golayout, cand); err == nil {
				m.value = t.Format("2006-01-02")
				return
			}
		}
		m.value = nil
	}
}

var audioHrAfterRegex = regexp.MustCompile(`(?i)^.+HR`)

var (
	yearPrefixRejectRegex = regexp.MustCompile(`(?i)(?:\d|Cap[. ]?)$`)
	yearSuffixRejectRegex = regexp.MustCompile(`(?i)^(?:\d|kbps)`)
)
