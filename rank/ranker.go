package rank

// The Ranker: a compiled Profile plus the evaluation pipeline
// (parse → veto chain → score). Construction validates and compiles
// everything once; Rank/RankAll are then cheap and goroutine-safe.

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/dreulavelle/jhin/parser"
)

// Torrent is one evaluated release. Slices returned by RankAll preserve
// input order; use Sort for a ranked ordering.
type Torrent struct {
	// Raw is the original release name.
	Raw string `json:"raw_title"`
	// Infohash is carried through untouched when supplied.
	Infohash string `json:"infohash,omitempty"`
	// Data is the full parse result.
	Data *parser.Result `json:"data"`

	// Rank is the total score under the profile (always computed, even for
	// rejected releases).
	Rank int `json:"rank"`
	// Fetch reports whether the release passed every filter.
	Fetch bool `json:"fetch"`
	// Rejections lists which checks failed when Fetch is false.
	Rejections []string `json:"rejections,omitempty"`
	// TitleRatio is the similarity against the target title (0 when no
	// target was supplied).
	TitleRatio float64 `json:"title_ratio,omitempty"`
}

// Resolution returns the release's resolution bucket.
func (t *Torrent) Resolution() Resolution {
	return normalize_resolution(t.Data.Resolution)
}

// Ranker evaluates releases under a compiled Profile. Safe for concurrent
// use.
type Ranker struct {
	profile   Profile
	require   []*regexp.Regexp
	exclude   []*regexp.Regexp
	preferred []*regexp.Regexp

	requiredLangs  map[string]bool
	allowedLangs   map[string]bool
	excludeLangs   map[string]bool
	preferredLangs map[string]bool
}

// New compiles a profile into a Ranker.
func New(p Profile) (*Ranker, error) {
	r := &Ranker{profile: p}
	var err error
	if r.require, err = compile_patterns(p.Require); err != nil {
		return nil, fmt.Errorf("require: %w", err)
	}
	if r.exclude, err = compile_patterns(p.Exclude); err != nil {
		return nil, fmt.Errorf("exclude: %w", err)
	}
	if r.preferred, err = compile_patterns(p.Preferred); err != nil {
		return nil, fmt.Errorf("preferred: %w", err)
	}
	r.requiredLangs = expand_langs(p.Languages.Required)
	r.allowedLangs = expand_langs(p.Languages.Allowed)
	r.excludeLangs = expand_langs(p.Languages.Exclude)
	r.preferredLangs = expand_langs(p.Languages.Preferred)
	return r, nil
}

// compile_patterns follows the RTN convention: "/pat/" is case-sensitive,
// anything else case-insensitive.
func compile_patterns(patterns []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if p == "" {
			continue
		}
		src := p
		if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") && len(p) > 2 {
			src = p[1 : len(p)-1]
		} else {
			src = "(?i)" + p
		}
		re, err := regexp.Compile(src)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

// RankOptions tune a single Rank call.
type RankOptions struct {
	// TargetTitle, when set, computes TitleRatio and rejects releases whose
	// parsed title does not reach Options.TitleThreshold.
	TargetTitle string
	// Aliases are alternative correct titles for the target.
	Aliases []string
	// Infohash is carried through to the Torrent.
	Infohash string
}

// Rank parses and evaluates one release name.
func (r *Ranker) Rank(raw string, opts ...RankOptions) Torrent {
	var opt RankOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	data := parser.Parse(raw)
	t := Torrent{Raw: raw, Infohash: opt.Infohash, Data: data}
	r.evaluate(&t, &opt)
	return t
}

// RankAll evaluates a batch in parallel. The result slice is index-aligned
// with the input: out[i] is titles[i].
func (r *Ranker) RankAll(titles []string, opts ...RankOptions) []Torrent {
	var opt RankOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	out := make([]Torrent, len(titles))
	workers := min(runtime.GOMAXPROCS(0), max(1, len(titles)))
	var wg sync.WaitGroup
	ch := make(chan int, workers*2)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				data := parser.Parse(titles[i])
				out[i] = Torrent{Raw: titles[i], Data: data}
				r.evaluate(&out[i], &opt)
			}
		}()
	}
	for i := range titles {
		ch <- i
	}
	close(ch)
	wg.Wait()
	return out
}

// evaluate applies the veto chain and scoring to a parsed release.
func (r *Ranker) evaluate(t *Torrent, opt *RankOptions) {
	d := t.Data
	o := r.profile.Options
	attrs := attributes(d)

	// --- score (always computed) ---
	rank := 0
	for _, a := range attrs {
		rank += r.profile.policy(a).Rank
	}
	if len(r.preferred) > 0 && match_any(r.preferred, t.Raw) {
		rank += o.PreferredBonus
	}
	if len(r.preferredLangs) > 0 && lang_overlap(d.Languages, r.preferredLangs) {
		rank += o.PreferredBonus
	}
	t.Rank = rank

	// --- title similarity ---
	if opt.TargetTitle != "" {
		t.TitleRatio = BestSimilarity(opt.TargetTitle, d.Title, opt.Aliases...)
	}

	// --- veto chain ---
	reject := func(reason string) {
		t.Rejections = append(t.Rejections, reason)
	}

	if opt.TargetTitle != "" && t.TitleRatio < o.TitleThreshold {
		reject("title_mismatch")
	}

	if o.RemoveTrash {
		if d.Trash {
			reject("trash")
		} else {
			for _, a := range attrs {
				if trash_quality_attrs[a] || a == AttrCleanAudio {
					reject("trash:" + string(a))
					break
				}
			}
		}
	}
	if o.RemoveAdult && d.Adult {
		reject("adult")
	}

	// a require match short-circuits the remaining vetoes (trash/adult/title
	// still apply)
	if len(r.require) > 0 && match_any(r.require, t.Raw) {
		t.Fetch = len(t.Rejections) == 0
		return
	}

	if match_first(r.exclude, t.Raw, reject) {
		t.Fetch = false
		return
	}

	r.check_languages(d.Languages, reject)

	if res := normalize_resolution(d.Resolution); r.profile.Resolutions != nil {
		if enabled, ok := r.profile.Resolutions[res]; ok && !enabled {
			reject("resolution:" + string(res))
		}
	}

	for _, a := range attrs {
		if !r.profile.policy(a).Fetch {
			reject("attribute:" + string(a))
		}
	}

	if t.Rank < o.MinRank {
		reject("min_rank")
	}

	t.Fetch = len(t.Rejections) == 0
}

func (r *Ranker) check_languages(langs []string, reject func(string)) {
	if len(langs) == 0 {
		if r.profile.Options.RemoveUnknownLanguages {
			reject("language:unknown")
		} else if len(r.requiredLangs) > 0 {
			reject("language:missing_required")
		}
		return
	}
	if len(r.requiredLangs) > 0 && !lang_overlap(langs, r.requiredLangs) {
		reject("language:missing_required")
		return
	}
	if r.profile.Options.AllowEnglish {
		for _, l := range langs {
			if l == "en" {
				return
			}
		}
	}
	if len(r.allowedLangs) > 0 && lang_overlap(langs, r.allowedLangs) {
		return
	}
	for _, l := range langs {
		if r.excludeLangs[l] {
			reject("language:" + l)
		}
	}
}

func lang_overlap(langs []string, set map[string]bool) bool {
	for _, l := range langs {
		if set[l] {
			return true
		}
	}
	return false
}

func match_any(patterns []*regexp.Regexp, s string) bool {
	for _, re := range patterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func match_first(patterns []*regexp.Regexp, s string, reject func(string)) bool {
	for _, re := range patterns {
		if re.MatchString(s) {
			reject("exclude:" + re.String())
			return true
		}
	}
	return false
}
