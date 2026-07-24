// Package parser extracts release metadata from torrent names using an
// ordered handler table with a literal prefilter.
package parser

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// PTT parse.py NON_ENGLISH_CHARS: Japanese, Chinese, CJK compat,
	// halfwidth Katakana, Cyrillic, Arabic, Kannada, Malayalam, Thai
	nonEnglishChars                              = `\x{3040}-\x{30ff}\x{3400}-\x{4dbf}\x{4e00}-\x{9fff}\x{f900}-\x{faff}\x{ff66}-\x{ff9f}\x{0400}-\x{04ff}\x{0600}-\x{06ff}\x{0750}-\x{077f}\x{0c80}-\x{0cff}\x{0d00}-\x{0d7f}\x{0e00}-\x{0e7f}`
	russianCastRegex                             = regexp.MustCompile(`(\([^)]*[\x{0400}-\x{04ff}][^)]*\))$|(?:\/.*?)(\(.*\))$`)
	altTitlesRegex                               = regexp.MustCompile(`[^/|(]*[` + nonEnglishChars + `][^/|]*[/|]|[/|][^/|(]*[` + nonEnglishChars + `][^/|]*`)
	notOnlyNonEnglishRegex                       = regexp.MustCompile(`(?:[a-zA-Z][^` + nonEnglishChars + `]+)([` + nonEnglishChars + `].*[` + nonEnglishChars + `])|([` + nonEnglishChars + `].*[` + nonEnglishChars + `])(?:[^` + nonEnglishChars + `]+[a-zA-Z])`)
	notAllowedSymbolsAtStartAndEndRegex          = regexp.MustCompile(`^[^\w\p{L}\p{N}` + nonEnglishChars + `#[【★]+|[ \-:/\\[|{(#$&^]+$`)
	remainingNotAllowedSymbolsAtStartAndEndRegex = regexp.MustCompile(`^[^\w\p{L}\p{N}` + nonEnglishChars + `#]+|]$`)
	emptyBracketsRegex                           = regexp.MustCompile(`\(\s*\)|\[\s*\]|\{\s*\}`)
	parenthesesWithoutContentRegex               = regexp.MustCompile(`\([^\w\p{L}\p{N}]*\)|\[[^\w\p{L}\p{N}]*\]|\{[^\w\p{L}\p{N}]*\}`)
	mp3AtEndRegex                                = regexp.MustCompile(`\bmp3$`)
	specialCharSpacingRegex                      = regexp.MustCompile(`[\-\+\_\{\}\[\]][^\w\p{L}\p{N}]{2,}`)

	movieIndicatorRegex             = regexp.MustCompile(`(?i)[[(]movie[)\]]`)
	releaseGroupMarkingAtStartRegex = regexp.MustCompile(`^[[【★].*[\]】★][ .]?(.+)`)
	releaseGroupMarkingAtEndRegex   = regexp.MustCompile(`(.+)[ .]?[[【★].*[\]】★]$`)

	beforeTitleRegex = regexp.MustCompile(`^\[([^[\]]+)]`)
	nonDigitsRegex   = regexp.MustCompile(`\D+`)
	underscoresRegex = regexp.MustCompile(`_+`)
	whitespacesRegex = regexp.MustCompile(`\s+`)

	redundantSymbolsAtEnd = regexp.MustCompile(`[ \-:./\\]+$`)

	curlyBrackets  = []string{"{", "}"}
	squareBrackets = []string{"[", "]"}
	parentheses    = []string{"(", ")"}
	brackets       = [][]string{curlyBrackets, squareBrackets, parentheses}
)

// replaceAll is ReplaceAllString with an allocation-free fast path: most
// cleanup regexes match nothing on most titles.
func replaceAll(re *regexp.Regexp, s, repl string) string {
	if !re.MatchString(s) {
		return s
	}
	return re.ReplaceAllString(s, repl)
}

func cleanTitle(rawTitle string) string {
	title := strings.TrimSpace(rawTitle)

	title = strings.ReplaceAll(title, "_", " ")
	title = replaceAll(movieIndicatorRegex, title, "") // clear movie indication flag
	title = replaceAll(notAllowedSymbolsAtStartAndEndRegex, title, "")
	for _, parts := range russianCastRegex.FindAllStringSubmatch(title, -1) {
		for i, mStr := range parts {
			if i != 0 {
				// clear russian cast information
				title = strings.Replace(title, mStr, "", 1)
			}
		}
	}
	title = replaceAll(releaseGroupMarkingAtStartRegex, title, "$1") // remove release group markings sections from the start
	title = replaceAll(releaseGroupMarkingAtEndRegex, title, "$1")   // remove unneeded markings section at the end if present
	title = replaceAll(altTitlesRegex, title, "")                    // remove alt language titles
	for i, mStr := range notOnlyNonEnglishRegex.FindStringSubmatch(title) {
		if i != 0 {
			// remove non english chars if they are not the only ones left
			title = strings.Replace(title, mStr, "", 1)
		}
	}
	title = replaceAll(remainingNotAllowedSymbolsAtStartAndEndRegex, title, "")
	title = replaceAll(emptyBracketsRegex, title, "")
	title = replaceAll(mp3AtEndRegex, title, "")
	title = replaceAll(parenthesesWithoutContentRegex, title, "")
	title = replaceAll(specialCharSpacingRegex, title, "")

	for _, b := range brackets {
		if strings.Count(title, b[0]) != strings.Count(title, b[1]) {
			title = strings.ReplaceAll(strings.ReplaceAll(title, b[0], ""), b[1], "")
		}
	}

	if !strings.Contains(title, " ") && strings.Contains(title, ".") {
		title = strings.ReplaceAll(title, ".", " ")
	}

	title = replaceAll(redundantSymbolsAtEnd, title, "")
	title = replaceAll(whitespacesRegex, title, " ")

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

	err          error `json:"-"`
	isNormalized bool  `json:"-"`
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

func valueSetStrings(v any) []string {
	vs := v.(*valueSet[any])
	values := make([]string, len(vs.values))
	for i, v := range vs.values {
		values[i] = v.(string)
	}
	return values
}

func hasValueSet(field string) bool {
	_, ok := valueSetFieldMap[field]
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

	title = replaceAll(whitespacesRegex, title, " ")
	title = replaceAll(underscoresRegex, title, " ")
	result := make(map[string]*parseMeta, 24)
	// endOfTitle is tracked in RUNES to mirror Python's character indexing —
	// multibyte titles would otherwise slice at different boundaries
	endOfTitle := utf8.RuneCountInString(title)

	lowerTitle := foldLower(title)
	prevTitle := title
	for hi, handler := range handlers {
		if title != prevTitle {
			// title mutated: refresh the prefilter haystack (removals can
			// splice fragments into new substrings)
			lowerTitle = foldLower(title)
			if debugHook != nil {
				debugHook(hi-1, title)
			}
			prevTitle = title
		}
		field := handler.Field
		skipFromTitle := handler.SkipFromTitle

		if prefilterEnabled && handler.Gate != nil && !handler.Gate.hit(lowerTitle) {
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
			if bt := beforeTitleRegex.FindStringSubmatch(title); bt != nil && strings.Contains(bt[1], rawMatchedPart) {
				skipFromTitle = true
			}

			if !mFound {
				m = &parseMeta{}
				if hasValueSet(field) {
					m.value = &valueSet[any]{existMap: map[any]struct{}{}, values: []any{}}
				}
				mFound = true
				result[field] = m
			}

			fresh := m.firstMValue == "" && m.firstMIndex == 0 && !m.processed

			m.mIndex = idxs[0]
			m.mValue = rawMatchedPart
			if !hasValueSet(field) {
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
			if debugEotHook != nil {
				debugEotHook(hi, endOfTitle)
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
			r.Audio = valueSetStrings(v)
		case "bitDepth":
			r.BitDepth = v.(string)
		case "bitrate":
			r.Bitrate = v.(string)
		case "channels":
			r.Channels = valueSetStrings(v)
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
			r.Extras = valueSetStrings(v)
		case "group":
			r.Group = v.(string)
		case "hardcoded":
			r.Hardcoded = v.(bool)
		case "hdr":
			r.HDR = valueSetStrings(v)
		case "languages":
			r.Languages = valueSetStrings(v)
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

	r.Title = cleanTitle(runePrefix(title, endOfTitle))

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

// debugHook, when set (tests only), is called with the index of the handler
// that just mutated the working title. Setting it while parses run
// concurrently is a data race.
var debugHook func(handlerIdx int, title string)

// debugEotHook (tests only) reports endOfTitle updates.
var debugEotHook func(handlerIdx int, endOfTitle int)

// runePrefix returns the first n runes of s (Python-style slicing).
func runePrefix(s string, n int) string {
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
