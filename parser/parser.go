package parser

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// PTT parse.py NON_ENGLISH_CHARS: Japanese, Chinese, CJK compat,
	// halfwidth Katakana, Cyrillic, Arabic, Kannada, Malayalam, Thai
	non_english_chars                                    = `\x{3040}-\x{30ff}\x{3400}-\x{4dbf}\x{4e00}-\x{9fff}\x{f900}-\x{faff}\x{ff66}-\x{ff9f}\x{0400}-\x{04ff}\x{0600}-\x{06ff}\x{0750}-\x{077f}\x{0c80}-\x{0cff}\x{0d00}-\x{0d7f}\x{0e00}-\x{0e7f}`
	russian_cast_regex                                   = regexp.MustCompile(`(\([^)]*[\x{0400}-\x{04ff}][^)]*\))$|(?:\/.*?)(\(.*\))$`)
	alt_titles_regex                                     = regexp.MustCompile(`[^/|(]*[` + non_english_chars + `][^/|]*[/|]|[/|][^/|(]*[` + non_english_chars + `][^/|]*`)
	not_only_non_english_regex                           = regexp.MustCompile(`(?:[a-zA-Z][^` + non_english_chars + `]+)([` + non_english_chars + `].*[` + non_english_chars + `])|([` + non_english_chars + `].*[` + non_english_chars + `])(?:[^` + non_english_chars + `]+[a-zA-Z])`)
	not_allowed_symbols_at_start_and_end_regex           = regexp.MustCompile(`^[^\w\p{L}\p{N}` + non_english_chars + `#[【★]+|[ \-:/\\[|{(#$&^]+$`)
	remaining_not_allowed_symbols_at_start_and_end_regex = regexp.MustCompile(`^[^\w\p{L}\p{N}` + non_english_chars + `#]+|]$`)
	empty_brackets_regex                                 = regexp.MustCompile(`\(\s*\)|\[\s*\]|\{\s*\}`)
	parentheses_without_content_regex                    = regexp.MustCompile(`\([^\w\p{L}\p{N}]*\)|\[[^\w\p{L}\p{N}]*\]|\{[^\w\p{L}\p{N}]*\}`)
	mp3_at_end_regex                                     = regexp.MustCompile(`\bmp3$`)
	special_char_spacing_regex                           = regexp.MustCompile(`[\-\+\_\{\}\[\]][^\w\p{L}\p{N}]{2,}`)

	movie_indicator_regex                = regexp.MustCompile(`(?i)[[(]movie[)\]]`)
	release_group_marking_at_start_regex = regexp.MustCompile(`^[[【★].*[\]】★][ .]?(.+)`)
	release_group_marking_at_end_regex   = regexp.MustCompile(`(.+)[ .]?[[【★].*[\]】★]$`)

	before_title_regex = regexp.MustCompile(`^\[([^[\]]+)]`)
	non_digit_regex    = regexp.MustCompile(`\D`)
	non_digits_regex   = regexp.MustCompile(`\D+`)
	non_alphas_regex   = regexp.MustCompile(`\W+`)
	underscores_regex  = regexp.MustCompile(`_+`)
	whitespaces_regex  = regexp.MustCompile(`\s+`)

	redundant_symbols_at_end = regexp.MustCompile(`[ \-:./\\]+$`)

	curly_brackets  = []string{"{", "}"}
	square_brackets = []string{"[", "]"}
	parentheses     = []string{"(", ")"}
	brackets        = [][]string{curly_brackets, square_brackets, parentheses}
)

func clean_title(rawTitle string) string {
	title := strings.TrimSpace(rawTitle)

	title = strings.ReplaceAll(title, "_", " ")
	title = movie_indicator_regex.ReplaceAllString(title, "") // clear movie indication flag
	title = not_allowed_symbols_at_start_and_end_regex.ReplaceAllString(title, "")
	for _, parts := range russian_cast_regex.FindAllStringSubmatch(title, -1) {
		for i, mStr := range parts {
			if i != 0 {
				// clear russian cast information
				title = strings.Replace(title, mStr, "", 1)
			}
		}
	}
	title = release_group_marking_at_start_regex.ReplaceAllString(title, "$1") // remove release group markings sections from the start
	title = release_group_marking_at_end_regex.ReplaceAllString(title, "$1")   // remove unneeded markings section at the end if present
	title = alt_titles_regex.ReplaceAllString(title, "")                       // remove alt language titles
	for i, mStr := range not_only_non_english_regex.FindStringSubmatch(title) {
		if i != 0 {
			// remove non english chars if they are not the only ones left
			title = strings.Replace(title, mStr, "", 1)
		}
	}
	title = remaining_not_allowed_symbols_at_start_and_end_regex.ReplaceAllString(title, "")
	title = empty_brackets_regex.ReplaceAllString(title, "")
	title = mp3_at_end_regex.ReplaceAllString(title, "")
	title = parentheses_without_content_regex.ReplaceAllString(title, "")
	title = special_char_spacing_regex.ReplaceAllString(title, "")

	for _, b := range brackets {
		if strings.Count(title, b[0]) != strings.Count(title, b[1]) {
			title = strings.ReplaceAll(strings.ReplaceAll(title, b[0], ""), b[1], "")
		}
	}

	if !strings.Contains(title, " ") && strings.Contains(title, ".") {
		title = strings.ReplaceAll(title, ".", " ")
	}

	title = redundant_symbols_at_end.ReplaceAllString(title, "")
	title = whitespaces_regex.ReplaceAllString(title, " ")

	return strings.TrimSpace(title)
}

type Result struct {
	Adult       bool     `json:"adult"`
	Audio       []string `json:"audio"`
	BitDepth    string   `json:"bit_depth"`
	Bitrate     string   `json:"bitrate"`
	Channels    []string `json:"channels"`
	Codec       string   `json:"codec"`
	Commentary  bool     `json:"commentary"`
	Complete    bool     `json:"complete"`
	Container   string   `json:"container"`
	Convert     bool     `json:"convert"`
	Country     string   `json:"country"`
	Date        string   `json:"date"`
	Documentary bool     `json:"documentary"`
	Dubbed      bool     `json:"dubbed"`
	Edition     string   `json:"edition"`
	EpisodeCode string   `json:"episode_code"`
	Episodes    []int    `json:"episodes"`
	Extension   string   `json:"extension"`
	Extras      []string `json:"extras"`
	Group       string   `json:"group"`
	HDR         []string `json:"hdr"`
	Hardcoded   bool     `json:"hardcoded"`
	Languages   []string `json:"languages"`
	Network     string   `json:"network"`
	PPV         bool     `json:"ppv"`
	Proper      bool     `json:"proper"`
	Quality     string   `json:"quality"`
	Region      string   `json:"region"`
	Remastered  bool     `json:"remastered"`
	Repack      bool     `json:"repack"`
	Resolution  string   `json:"resolution"`
	Retail      bool     `json:"retail"`
	Scene       bool     `json:"scene"`
	Seasons     []int    `json:"seasons"`
	Site        string   `json:"site"`
	Size        string   `json:"size"`
	Subbed      bool     `json:"subbed"`
	ThreeD      bool     `json:"3d"`
	Title       string   `json:"title"`
	Torrent     bool     `json:"torrent"`
	Trash       bool     `json:"trash"`
	Uncensored  bool     `json:"uncensored"`
	Unrated     bool     `json:"unrated"`
	Upscaled    bool     `json:"upscaled"`
	Volumes     []int    `json:"volumes"`
	Year        string   `json:"year"`

	err           error `json:"-"`
	is_normalized bool  `json:"-"`
}

func (r *Result) Error() error {
	if r.err == nil {
		return nil
	}
	return r.err
}

type parseMeta struct {
	mIndex int
	mValue string
	// first match position/text for the field — PTT's `matched[name]` keeps
	// the first occurrence even when later handlers overwrite the value
	firstMIndex int
	firstMValue string
	value       any
	remove      bool
	processed   bool
	// matchedNow mirrors PTT's per-iteration match dict: side effects (remove,
	// end-of-title) apply only when the current handler actually matched.
	matchedNow bool
}

func value_set_strings(v any) []string {
	vs := v.(*value_set[any])
	values := make([]string, len(vs.values))
	for i, v := range vs.values {
		values[i] = v.(string)
	}
	return values
}

func has_value_set(field string) bool {
	_, ok := value_set_field_map[field]
	return ok
}

func parse(title string, handlers []handler) (r *Result) {
	r = &Result{}

	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(error); ok {
				r.err = e
			} else {
				panic(err)
			}
		}
	}()

	title = whitespaces_regex.ReplaceAllString(title, " ")
	title = underscores_regex.ReplaceAllString(title, " ")
	result := map[string]*parseMeta{}
	// endOfTitle is tracked in RUNES to mirror Python's character indexing —
	// multibyte titles would otherwise slice at different boundaries
	endOfTitle := utf8.RuneCountInString(title)

	lowerTitle := strings.ToLower(title)
	prevTitle := title
	for hi, handler := range handlers {
		if title != prevTitle {
			// title mutated: refresh the prefilter haystack (removals can
			// splice fragments into new substrings)
			lowerTitle = strings.ToLower(title)
			if debug_hook != nil {
				debug_hook(hi-1, title)
			}
			prevTitle = title
		}
		field := handler.Field
		skipFromTitle := handler.SkipFromTitle

		if prefilter_enabled && handler.Gate != nil && !handler.Gate.hit(lowerTitle) {
			continue
		}

		m, mFound := result[field]

		if handler.Pattern != nil {
			if mFound && !handler.KeepMatching {
				continue
			}

			idxs := handler.Pattern.FindStringSubmatchIndex(title)
			if len(idxs) == 0 {
				continue
			}
			if handler.ValidateMatch != nil && !handler.ValidateMatch(title, idxs) {
				// Python's regex engine keeps scanning when an embedded
				// lookaround rejects a position; emulate by advancing past
				// rejected candidates (unless the pattern is ^-anchored,
				// where no later position can match).
				pat := strings.TrimPrefix(handler.Pattern.String(), "(?i)")
				if strings.HasPrefix(pat, "^") {
					continue
				}
				for {
					next := idxs[1]
					if next <= idxs[0] {
						next = idxs[0] + 1
					}
					if next > len(title) {
						idxs = nil
						break
					}
					sub := handler.Pattern.FindStringSubmatchIndex(title[next:])
					if sub == nil {
						idxs = nil
						break
					}
					for i := range sub {
						if sub[i] >= 0 {
							sub[i] += next
						}
					}
					idxs = sub
					if handler.ValidateMatch(title, idxs) {
						break
					}
				}
				if idxs == nil {
					continue
				}
			}
			shouldSkip := false
			if handler.SkipIfFirst {
				hasOther := false
				hasBefore := false
				for f, fm := range result {
					if f != field {
						hasOther = true
						if idxs[0] >= fm.firstMIndex {
							hasBefore = true
							break
						}
					}
				}
				shouldSkip = hasOther && !hasBefore
			}
			if shouldSkip {
				continue
			}

			if len(handler.SkipIfBefore) > 0 {
				for _, skipField := range handler.SkipIfBefore {
					if fm, ok := result[skipField]; ok && idxs[0] < fm.mIndex {
						shouldSkip = true
						break
					}
				}
				if shouldSkip {
					continue
				}
			}

			rawMatchedPart := title[idxs[0]:idxs[1]]
			matchedPart := rawMatchedPart
			// PTT: clean_match = group(1) `or` raw_match — an absent or empty
			// capture group falls back to the raw match
			if len(idxs) > 2 {
				if handler.ValueGroup == 0 {
					if idxs[2] >= 0 && idxs[3] > idxs[2] {
						matchedPart = title[idxs[2]:idxs[3]]
					}
				} else if len(idxs) > handler.ValueGroup*2 {
					g := handler.ValueGroup * 2
					if idxs[g] >= 0 && idxs[g+1] > idxs[g] {
						matchedPart = title[idxs[g]:idxs[g+1]]
					}
				}
			}

			// PTT compares against the bracket CONTENTS (group 1), not the
			// full bracketed match
			if bt := before_title_regex.FindStringSubmatch(title); bt != nil && strings.Contains(bt[1], rawMatchedPart) {
				skipFromTitle = true
			}

			if !mFound {
				m = &parseMeta{}
				if has_value_set(field) {
					m.value = &value_set[any]{existMap: map[any]struct{}{}, values: []any{}}
				}
				mFound = true
				result[field] = m
			}

			fresh := m.firstMValue == "" && m.firstMIndex == 0 && !m.processed

			m.mIndex = idxs[0]
			m.mValue = rawMatchedPart
			if !has_value_set(field) {
				m.value = matchedPart
			}

			if handler.MatchGroup != 0 {
				m.mIndex = idxs[handler.MatchGroup*2]
				m.mValue = title[idxs[handler.MatchGroup*2]:idxs[handler.MatchGroup*2+1]]
			}

			if fresh {
				m.firstMIndex = m.mIndex
				m.firstMValue = m.mValue
			}

			m.matchedNow = true
		}

		if handler.Process != nil {
			if mFound {
				m = handler.Process(title, m, result)
			} else {
				m = handler.Process(title, &parseMeta{}, result)
				if m.value != nil {
					result[field] = m
					mFound = true
				}
			}
			// Process-created matches must also record the first-match
			// position for later skipIfFirst comparisons
			if m != nil && m.matchedNow && !m.processed && m.firstMValue == "" && m.firstMIndex == 0 {
				m.firstMIndex = m.mIndex
				m.firstMValue = m.mValue
			}
		}

		// PTT runs the transformer only when the handler matched this iteration
		if m.value != nil && m.matchedNow && handler.Transform != nil {
			handler.Transform(title, m, result)
		}

		// PTT strips whitespace from every string value after transforming
		if m.matchedNow {
			if s, ok := m.value.(string); ok {
				m.value = strings.TrimSpace(s)
			}
		}

		if m.value == nil {
			delete(result, field)
			mFound = false
		}

		if !mFound || !m.matchedNow {
			continue
		}
		m.matchedNow = false

		if handler.Remove || m.remove {
			m.remove = true
			title = title[:m.mIndex] + title[m.mIndex+len(m.mValue):]
		}

		// PTT: `1 < match_index < end_of_title` (rune-based like Python)
		mRuneIndex := utf8.RuneCountInString(title[:min(m.mIndex, len(title))])
		if !skipFromTitle && mRuneIndex > 1 && mRuneIndex < endOfTitle {
			endOfTitle = mRuneIndex
			if debug_eot_hook != nil {
				debug_eot_hook(hi, endOfTitle)
			}
		}

		if m.remove && skipFromTitle && mRuneIndex < endOfTitle {
			// adjust title index in case part of it should be removed and skipped
			endOfTitle -= utf8.RuneCountInString(m.mValue)
		}

		m.remove = false
		m.processed = true
	}

	for field, fieldMeta := range result {
		v := fieldMeta.value
		switch field {
		case "adult":
			r.Adult = v.(bool)
		case "audio":
			r.Audio = value_set_strings(v)
		case "bitDepth":
			r.BitDepth = v.(string)
		case "bitrate":
			r.Bitrate = v.(string)
		case "channels":
			r.Channels = value_set_strings(v)
		case "codec":
			r.Codec = v.(string)
		case "country":
			r.Country = v.(string)
		case "commentary":
			r.Commentary = v.(bool)
		case "complete":
			r.Complete = v.(bool)
		case "container":
			r.Container = v.(string)
		case "convert":
			r.Convert = v.(bool)
		case "date":
			r.Date = v.(string)
		case "documentary":
			r.Documentary = v.(bool)
		case "dubbed":
			r.Dubbed = v.(bool)
		case "edition":
			r.Edition = v.(string)
		case "episodeCode":
			r.EpisodeCode = v.(string)
		case "episodes":
			r.Episodes = v.([]int)
		case "extension":
			r.Extension = v.(string)
		case "extras":
			r.Extras = value_set_strings(v)
		case "group":
			r.Group = v.(string)
		case "hardcoded":
			r.Hardcoded = v.(bool)
		case "hdr":
			r.HDR = value_set_strings(v)
		case "languages":
			r.Languages = value_set_strings(v)
		case "network":
			r.Network = v.(string)
		case "ppv":
			r.PPV = v.(bool)
		case "proper":
			r.Proper = v.(bool)
		case "region":
			r.Region = v.(string)
		case "remastered":
			r.Remastered = v.(bool)
		case "repack":
			r.Repack = v.(bool)
		case "resolution":
			r.Resolution = v.(string)
		case "retail":
			r.Retail = v.(bool)
		case "seasons":
			r.Seasons = v.([]int)
		case "size":
			r.Size = v.(string)
		case "site":
			r.Site = v.(string)
		case "quality":
			r.Quality = v.(string)
		case "scene":
			r.Scene = v.(bool)
		case "torrent":
			r.Torrent = v.(bool)
		case "trash":
			r.Trash = v.(bool)
		case "subbed":
			r.Subbed = v.(bool)
		case "threeD":
			r.ThreeD = v.(bool)
		case "uncensored":
			r.Uncensored = v.(bool)
		case "unrated":
			r.Unrated = v.(bool)
		case "upscaled":
			r.Upscaled = v.(bool)
		case "volumes":
			r.Volumes = v.([]int)
		case "year":
			r.Year = v.(string)
		}
	}

	// PTT defaults episodes/seasons/languages to empty arrays
	if r.Episodes == nil {
		r.Episodes = []int{}
	}
	if r.Seasons == nil {
		r.Seasons = []int{}
	}
	if r.Languages == nil {
		r.Languages = []string{}
	}

	r.Title = clean_title(rune_prefix(title, endOfTitle))

	return r
}

func Parse(title string) *Result {
	return parse(title, handlers)
}

func GetPartialParser(fieldNames []string) func(title string) *Result {
	selectedFieldMap := map[string]struct{}{}
	for _, fieldName := range fieldNames {
		selectedFieldMap[fieldName] = struct{}{}
	}

	selectedHandlers := []handler{}
	for _, h := range handlers {
		if _, ok := selectedFieldMap[h.Field]; ok {
			selectedHandlers = append(selectedHandlers, h)
		}
	}

	return func(title string) *Result {
		return parse(title, selectedHandlers)
	}
}

// debug_hook, when set (tests only), is called with the index of the handler
// that just mutated the working title.
var debug_hook func(handlerIdx int, title string)

// debug_eot_hook (tests only) reports endOfTitle updates.
var debug_eot_hook func(handlerIdx int, endOfTitle int)

// rune_prefix returns the first n runes of s (Python-style slicing).
func rune_prefix(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
