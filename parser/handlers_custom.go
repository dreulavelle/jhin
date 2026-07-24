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
)

//go:embed keywords/combined-keywords.txt
var adult_keywords_raw string

func build_adult_pattern() *regexp.Regexp {
	lines := strings.Split(adult_keywords_raw, "\n")
	escaped := make([]string, 0, len(lines))
	for _, line := range lines {
		kw := strings.TrimSpace(line)
		if kw == "" {
			continue
		}
		escaped = append(escaped, regexp.QuoteMeta(kw))
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(escaped, "|") + `)\b`)
}

// custom_adult: parser.add_handler("adult", create_adult_pattern(), boolean, ...)
var custom_adult = handler{
	Field:         "adult",
	Pattern:       build_adult_pattern(),
	Transform:     to_boolean(),
	Remove:        true,
	SkipFromTitle: true,
}

// ---------------------------------------------------------------------------
// Transformer helpers used by the generated table
// ---------------------------------------------------------------------------

// to_value_sub implements PTT's value("... $1 ...") templates.
func to_value_sub(template string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = strings.ReplaceAll(template, "$1", v)
		} else {
			m.value = template
		}
	}
}

func to_int_value(v int) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = v
	}
}

func to_string_array() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = []string{v}
		}
	}
}

var first_int_regex = regexp.MustCompile(`\d+`)

// to_first_int_array implements PTT's first_integer transformer (list-typed
// fields like seasons receive it, so the value is an int slice).
func to_first_int_array() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if s := first_int_regex.FindString(v); s != "" {
				if num, err := strconv.Atoi(s); err == nil {
					m.value = []int{num}
					return
				}
			}
		}
		m.value = nil
	}
}

// to_transformed_resolution implements PTT's transform_resolution.
func to_transformed_resolution() hTransformer {
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

var bit_depth_cleanup_regex = regexp.MustCompile(`[ -]`)

// def handle_bit_depth: strip spaces/hyphens from the matched bit depth.
var custom_handle_bit_depth = handler{
	Field: "bitDepth",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && v != "" {
			m.value = bit_depth_cleanup_regex.ReplaceAllString(v, "")
		}
		return m
	},
}

var codec_cleanup_regex = regexp.MustCompile(`[ .-]`)

// def handle_space_in_codec: strip separators from the matched codec.
var custom_handle_space_in_codec = handler{
	Field: "codec",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && v != "" {
			m.value = codec_cleanup_regex.ReplaceAllString(v, "")
		}
		return m
	},
}

var volumes_fallback_regex = regexp.MustCompile(`(?i)\bvol(?:ume)?[. -]*(\d{1,2})`)

// def handle_volumes: search from the year match onward for a single volume.
var custom_handle_volumes = handler{
	Field: "volumes",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		startIndex := 0
		if ym, ok := result["year"]; ok {
			startIndex = min(ym.firstMIndex, len(title))
		}
		idxs := volumes_fallback_regex.FindStringSubmatchIndex(title[startIndex:])
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
	eps_beginning_core     = regexp.MustCompile(`(?i)(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})(?:a|b|v\d|\.\d)?(?:[^\p{L}\p{N}_]|$)`)
	eps_middle_core        = regexp.MustCompile(`(?i)^(?:[([-][ .]?)?(\d{1,4})(?:a|b|v\d)?(?:[^\p{L}\p{N}_]|$)`)
	eps_prefix_moviefilm   = regexp.MustCompile(`(?i)(?:movie|film)\W*$`)
	eps_suffix_moviefilm_d = regexp.MustCompile(`(?i)^(?:movie|film|\d)`)
	eps_suffix_moviefilm   = regexp.MustCompile(`(?i)^(?:movie|film)`)
	eps_res_bracket_end    = regexp.MustCompile(`\[(?:480|720|1080)\]$`)
	eps_res_bracket_next   = regexp.MustCompile(`^\[(?:480|720|1080)\]`)
)

var custom_handle_episodes = handler{
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
			idxs := eps_beginning_core.FindStringSubmatchIndex(beginningTitle[offset:])
			if idxs == nil {
				break
			}
			s, e := idxs[0]+offset, idxs[1]+offset
			ok := s != 0 &&
				!eps_prefix_moviefilm.MatchString(beginningTitle[:s]) &&
				!eps_suffix_moviefilm_d.MatchString(beginningTitle[e:]) &&
				!eps_res_bracket_end.MatchString(beginningTitle[:e])
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
		idxs := eps_middle_core.FindStringSubmatchIndex(middleTitle)
		if idxs != nil {
			e := idxs[1]
			if !eps_suffix_moviefilm.MatchString(middleTitle[e:]) && !eps_res_bracket_next.MatchString(middleTitle[e:]) {
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
	anime_title_regex = regexp.MustCompile(`One.*?Piece|Bleach|Naruto`)
	anime_epnum_regex = regexp.MustCompile(`\b\d{1,4}\b`)
)

// def handle_anime_eps: One Piece / Bleach / Naruto single-episode fallback.
var custom_handle_anime_eps = handler{
	Field: "episodes",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil || !anime_title_regex.MatchString(title) {
			return m
		}
		idxs := anime_epnum_regex.FindStringIndex(title)
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
	pt_epname_regex  = regexp.MustCompile(`(?i)capitulo|ao`)
	pt_dublado_regex = regexp.MustCompile(`(?i)dublado`)
)

// def infer_language_based_on_naming: Portuguese naming conventions.
var custom_infer_language_based_on_naming = handler{
	Field: "languages",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		vs, _ := m.value.(*value_set[any])
		if vs != nil && (vs.exists("pt") || vs.exists("es")) {
			return m
		}
		matchedByNaming := false
		if em, ok := result["episodes"]; ok && pt_epname_regex.MatchString(em.firstMValue) {
			matchedByNaming = true
		}
		if !matchedByNaming && pt_dublado_regex.MatchString(title) {
			matchedByNaming = true
		}
		if matchedByNaming {
			if vs == nil {
				vs = &value_set[any]{existMap: map[any]struct{}{}, values: []any{}}
			}
			m.value = vs.append("pt")
		}
		return m
	},
}

// def handle_group: drop a bracketed group that overlaps other matches.
var custom_handle_group = handler{
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
var custom_handle_group_exclusion = handler{
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
	scene_res_regex    = regexp.MustCompile(`\b\d{3,4}p\b`)
	scene_web_regex    = regexp.MustCompile(`[_. ]WEB[_. ]`)
	scene_groups_regex = regexp.MustCompile(`\b(-CAKES|-GGEZ|-GGWP|-GLHF|-GOSSIP|-NAISU|-KOGI|-PECULATE|-SLOT|-EDITH|-ETHEL|-ELEANOR|-B2B|-SPAMnEGGS|-FTP|-DiRT|-SYNCOPY|-BAE|-SuccessfulCrab|-NHTFS|-SURCODE|-B0MBARDIERS)`)
)

var custom_scene = handler{
	Field: "scene",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		for _, web := range scene_web_regex.FindAllStringIndex(title, -1) {
			if !scene_res_regex.MatchString(title[:web[0]]) {
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
		if idxs := scene_groups_regex.FindStringIndex(title); idxs != nil {
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
var extras_year_regex = regexp.MustCompile(`\b(?:19\d{2}|20\d{2})\b`)

func extras_handler(word *regexp.Regexp, after *regexp.Regexp, val string) handler {
	// python: (?:(?<=\byear\b.*)\bWORD\b|\bWORD\b(?!.*\bAFTER\b))
	// The context regexes are evaluated against the whole title (so \b sees
	// real neighbors) and filtered by position.
	return handler{
		Field:         "extras",
		SkipFromTitle: true,
		Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
			vs, _ := m.value.(*value_set[any])
			for _, idxs := range word.FindAllStringIndex(title, -1) {
				yearBefore := false
				for _, y := range extras_year_regex.FindAllStringIndex(title, -1) {
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
						vs = &value_set[any]{existMap: map[any]struct{}{}, values: []any{}}
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
	custom_extras_featurette = extras_handler(
		regexp.MustCompile(`(?i)\bFeaturettes?\b`), extras_year_regex, "Featurette")
	custom_extras_sample = extras_handler(
		regexp.MustCompile(`(?i)\bSample\b`), extras_year_regex, "Sample")
	custom_extras_trailer = extras_handler(
		regexp.MustCompile(`(?i)\bTrailers?\b`),
		regexp.MustCompile(`(?i)\b(?:19\d{2}|20\d{2}|.(Park|And))\b`), "Trailer")
)

// trash CAM: \b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\()\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b  (i)
var (
	trash_cam_core_regex   = regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?(CAM)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b`)
	trash_cam_reject_regex = regexp.MustCompile(`(?i)^.?(?:S|E|\()\d+`)
)

var custom_trash_cam = handler{
	Field: "trash",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		offset := 0
		for offset < len(title) {
			idxs := trash_cam_core_regex.FindStringSubmatchIndex(title[offset:])
			if idxs == nil {
				return m
			}
			camEnd := offset + idxs[3]
			if !trash_cam_reject_regex.MatchString(title[camEnd:]) {
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
	group_dash_core_regex   = regexp.MustCompile(`(?i)- ?([^\-. []+[^\-. [)\]\d][^\-. [)\]]*)(?:\[[\w.-]+])?`)
	group_dash_reject_regex = regexp.MustCompile(`(?i)^(?:\d+$|S\d+|\d+x|ep?\d+|[^[]+]$)`)
	group_dash_end_regex    = regexp.MustCompile(`(?i)^\.\w{2,4}$`)
)

var custom_group_dash = handler{
	Field: "group",
	Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
		if m.value != nil {
			return m
		}
		offset := 0
		for offset < len(title) {
			idxs := group_dash_core_regex.FindStringSubmatchIndex(title[offset:])
			if idxs == nil {
				return m
			}
			s, e := offset+idxs[0], offset+idxs[1]
			nameStart := offset + idxs[2]
			suffix := title[e:]
			if !group_dash_reject_regex.MatchString(title[nameStart:]) &&
				(suffix == "" || group_dash_end_regex.MatchString(suffix)) {
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

// to_int_string implements PTT's integer transformer for string-typed fields
// (year): the parsed integer is kept in decimal string form.
func to_int_string() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if num, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				m.value = strconv.Itoa(num)
				return
			}
		}
		m.value = nil
	}
}

// to_first_int_string implements PTT's first_integer for string-typed fields.
func to_first_int_string() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if s := first_int_regex.FindString(v); s != "" {
				m.value = s
				return
			}
		}
		m.value = nil
	}
}

// ---------------------------------------------------------------------------
// scan_valid: PTT-equivalent of a regex with embedded lookarounds — scans
// matches left to right until the validator accepts one (Python's regex
// engine does this natively; RE2 needs the explicit loop).
// ---------------------------------------------------------------------------

func scan_valid(field string, re *regexp.Regexp, valid func(title string, idxs []int) bool, isSet bool, skipIfFirst bool, keepMatching bool) hProcessor {
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
					if _, ok := m.value.(*value_set[any]); !ok {
						m.value = &value_set[any]{existMap: map[any]struct{}{}, values: []any{}}
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
	lang_www_i         = regexp.MustCompile(`(?i)w{3}\.\w+\.$`)
	lang_www_cs        = regexp.MustCompile(`w{3}\.\w+\.$`)
	lang_sci_i         = regexp.MustCompile(`(?i)Sci-$`)
	lang_te_aviv       = regexp.MustCompile(`(?i)^\W*aviv`)
	lang_bn_reject     = regexp.MustCompile(`(?i)^(?:.\bThe|and|of\b)`)
	lang_yts_prefix    = regexp.MustCompile(`YTS\.$`)
	lang_ext_suffix    = regexp.MustCompile(`(?i)^[\]_)]?\.\w{2,4}$`)
	lang_subs_paren    = regexp.MustCompile(`(?i)subs?\([a-z,]+$`)
	lang_codes2_suffix = regexp.MustCompile(`(?i)^[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,}`)
	lang_codes2_prefix = regexp.MustCompile(`(?i)[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,}$`)
	lang_code1_prefix  = regexp.MustCompile(`(?i)[ .,/-]+[A-Z]{2}[ .,/-]+$`)
	lang_code1_suffix  = regexp.MustCompile(`(?i)^[ .,/-]+[A-Z]{2}[ .,/-]+`)
	lang_it_codes_cs   = regexp.MustCompile(`^[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,}`)
	lang_hr_sub_suffix = regexp.MustCompile(`(?i)^[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub`)
	dubbed_subs_after  = regexp.MustCompile(`(?i)\bsub(s|bed)?\b`)
)

// lang_abbr_valid builds a validator for the PTT pattern family
// \b(?:(?<!w{3}\.\w+\.)ABBR|fullword)\b — the www-lookbehind (and optional
// extra reject) applies only when the abbreviated alternative matched.
func lang_abbr_valid(fullwords map[string]bool, extra func(title string, idxs []int) bool) func(string, []int) bool {
	return func(title string, idxs []int) bool {
		v := strings.ToLower(title[idxs[0]:idxs[1]])
		if fullwords[v] {
			return true
		}
		if lang_www_i.MatchString(title[:idxs[0]]) {
			return false
		}
		if extra != nil && !extra(title, idxs) {
			return false
		}
		return true
	}
}

// ---------------------------------------------------------------------------
// to_ptt_date replicates PTT's transformers.date(): sanitize non-word runs to
// spaces, shorten the 4-letter month abbreviations (Janu/Febr/... — full month
// names are intentionally NOT shortened, matching Python), then try each
// arrow format's Go layout equivalent.
// ---------------------------------------------------------------------------

var (
	ptt_date_sanitize_regex = regexp.MustCompile(`\W+`)
	ptt_date_month4_regex   = regexp.MustCompile(`(?i)\b(janu|febr|marc|apri|may|june|july|augu|sept|octo|nove|dece)\b`)
	ptt_date_ordinal_regex  = regexp.MustCompile(`\b(\d{1,2})(?:St|Nd|Rd|Th|st|nd|rd|th)\b`)
	ptt_date_word_regex     = regexp.MustCompile(`[A-Za-z]+`)
	ptt_month_short         = map[string]string{
		"janu": "Jan", "febr": "Feb", "marc": "Mar", "apri": "Apr",
		"may": "May", "june": "Jun", "july": "Jul", "augu": "Aug",
		"sept": "Sep", "octo": "Oct", "nove": "Nov", "dece": "Dec",
	}
)

func to_ptt_date(formats ...string) hTransformer {
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
		sanitized := strings.TrimSpace(ptt_date_sanitize_regex.ReplaceAllString(v, " "))
		sanitized = ptt_date_month4_regex.ReplaceAllStringFunc(sanitized, func(s string) string {
			return ptt_month_short[strings.ToLower(s)]
		})
		// arrow parses month names case-insensitively; Go's time.Parse does
		// not, so normalize word casing
		norm := ptt_date_word_regex.ReplaceAllStringFunc(sanitized, func(s string) string {
			return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		})
		for _, l := range layouts {
			cand := norm
			if l.ordinal {
				cand = ptt_date_ordinal_regex.ReplaceAllString(cand, "$1")
			}
			if t, err := time.Parse(l.golayout, cand); err == nil {
				m.value = t.Format("2006-01-02")
				return
			}
		}
		m.value = nil
	}
}

var audio_hr_after_regex = regexp.MustCompile(`(?i)^.+HR`)

var (
	year_prefix_reject_regex = regexp.MustCompile(`(?i)(?:\d|Cap[. ]?)$`)
	year_suffix_reject_regex = regexp.MustCompile(`(?i)^(?:\d|kbps)`)
)
