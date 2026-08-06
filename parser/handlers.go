package parser

import (
	"regexp"
	"strconv"
	"strings"
)

type hProcessor func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta
type hTransformer func(title string, m *parseMeta, result map[string]*parseMeta)

// hMatchValidator decides whether a candidate match is acceptable.
//
// implied holds patterns that every ACCEPTED match implies occur somewhere in
// the title — the positive lookarounds. Because a handler whose matches all
// fail validation is skipped outright, those patterns' required literals are
// necessary conditions for the handler to do anything, and the prefilter ANDs
// them into its gate. A negative lookaround implies nothing and contributes
// none.
// matchContext is the parse state a validator may consult beyond the match
// itself. endOfTitle is a rune index into the working title; stripped reports
// whether an earlier handler has already removed text from it.
type matchContext struct {
	endOfTitle int
	stripped   bool
}

// span is consulted in addition to fn for the rare validator that cannot
// decide from the match alone.
type hMatchValidator struct {
	fn      func(input string, idxs []int) bool
	span    func(input string, idxs []int, ctx matchContext) bool
	implied []string
}

// validateFunc wraps a bare predicate that carries no gating information.
func validateFunc(fn func(input string, idxs []int) bool) *hMatchValidator {
	return &hMatchValidator{fn: fn}
}

func (v *hMatchValidator) accepts(input string, idxs []int, ctx matchContext) bool {
	if v.fn != nil && !v.fn(input, idxs) {
		return false
	}
	return v.span == nil || v.span(input, idxs, ctx)
}

type handler struct {
	Field         string
	Pattern       *regexp.Regexp
	ValidateMatch *hMatchValidator
	Transform     hTransformer
	Process       hProcessor

	Remove        bool     // remove
	KeepMatching  bool     // !skipIfAlreadyFound
	SkipIfFirst   bool     // skipIfFirst
	SkipIfBefore  []string // skipIfBefore
	SkipFromTitle bool     // skipFromTitle

	MatchGroup int // capture group to use as match
	ValueGroup int // capture group to use as value

	// Gate holds required literals for prefiltering: when none occur in the
	// title the handler is skipped without running its regex. Derived
	// automatically from Pattern (see prefilter.go); set explicitly for
	// Process-based handlers.
	Gate *gateLits
}

// validateAnd requires every validator to accept, so each one's implied
// patterns stay individually necessary.
func validateAnd(validators ...*hMatchValidator) *hMatchValidator {
	var implied []string
	for _, v := range validators {
		implied = append(implied, v.implied...)
	}
	return &hMatchValidator{
		implied: implied,
		span: func(input string, idxs []int, ctx matchContext) bool {
			for _, validator := range validators {
				if !validator.accepts(input, idxs, ctx) {
					return false
				}
			}
			return true
		},
	}
}

func validateNotAtStart() *hMatchValidator {
	return validateFunc(func(input string, match []int) bool {
		return match[0] != 0
	})
}

func validateLookbehind(pattern, flags string, polarity bool) *hMatchValidator {
	re := regexp.MustCompile("(?" + flags + ")(?:" + pattern + ")$")
	v := &hMatchValidator{
		fn: func(input string, match []int) bool {
			rv := input[:match[0]]
			if polarity {
				return re.MatchString(rv)
			}
			return !re.MatchString(rv)
		},
	}
	if polarity {
		v.implied = []string{pattern}
	}
	return v
}

func validateLookahead(pattern, flags string, polarity bool) *hMatchValidator {
	re := regexp.MustCompile("(?" + flags + ")^(?:" + pattern + ")")
	v := &hMatchValidator{
		fn: func(input string, match []int) bool {
			rv := input[match[1]:]
			if polarity {
				return re.MatchString(rv)
			}
			return !re.MatchString(rv)
		},
	}
	if polarity {
		v.implied = []string{pattern}
	}
	return v
}

func validateNotMatch(re *regexp.Regexp) *hMatchValidator {
	return validateFunc(func(input string, match []int) bool {
		rv := input[match[0]:match[1]]
		return !re.MatchString(rv)
	})
}

// validateSiteLeavesTitle rejects a domain-shaped match that would swallow the
// whole title. "<word><separator><tld>" is ambiguous — "The Net", "The.Net"
// and "Rutracker.org" all fit it — so where the match would leave no title it
// is only trusted for a name nothing has yet been stripped from: a bare site
// stands alone, whereas anything that carried removable metadata is a release,
// and a release has a title.
func validateSiteLeavesTitle() *hMatchValidator {
	return &hMatchValidator{
		span: func(input string, match []int, ctx matchContext) bool {
			titleRegion := runePrefix(input, ctx.endOfTitle)
			if match[0] >= len(titleRegion) {
				return true // the match sits past the title entirely
			}
			var after string
			if match[1] < len(titleRegion) {
				after = titleRegion[match[1]:]
			}
			if strings.TrimSpace(titleRegion[:match[0]]) != "" || strings.TrimSpace(after) != "" {
				return true
			}
			return !ctx.stripped
		},
	}
}

func validateMatchedGroupsAreSame(indices ...int) *hMatchValidator {
	return validateFunc(func(input string, match []int) bool {
		first := input[match[indices[0]*2]:match[indices[0]*2+1]]
		for _, index := range indices[1:] {
			other := input[match[index*2]:match[index*2+1]]
			if other != first {
				return false
			}
		}
		return true
	})
}

func toValue(value string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = value
	}
}

func toLowercase() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.ToLower(m.value.(string))
	}
}

func toUppercase() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.ToUpper(m.value.(string))
	}
}

func toYear() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		vstr, ok := m.value.(string)
		if !ok {
			m.value = ""
			return
		}
		parts := nonDigitsRegex.Split(vstr, -1)
		if len(parts) == 1 {
			m.value = parts[0]
			return
		}
		start, end := parts[0], parts[1]
		endYear, err := strconv.Atoi(end)
		if err != nil {
			m.value = start
			return
		}
		startYear, err := strconv.Atoi(start)
		if err != nil {
			m.value = ""
			return
		}
		if endYear < 100 {
			endYear = endYear + startYear - startYear%100
		}
		if endYear <= startYear {
			m.value = ""
			return
		}
		m.value = strconv.Itoa(startYear) + "-" + strconv.Itoa(endYear)
	}
}

func toIntRange() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			m.value = nil
			return
		}
		parts := strings.Split(strings.Trim(nonDigitsRegex.ReplaceAllString(v, " "), " "), " ")
		nums := make([]int, len(parts))
		for i, part := range parts {
			if num, err := strconv.Atoi(part); err == nil {
				nums[i] = num
			}
		}
		if len(nums) == 2 && nums[0] < nums[1] {
			if nums[1]-nums[0]+1 > maxRangeTill {
				m.value = nil
				return
			}
			seq := make([]int, nums[1]-nums[0]+1)
			for i := range seq {
				seq[i] = nums[0] + i
			}
			nums = seq
		}
		for i, num := range nums {
			if i != len(nums)-1 && num+1 != nums[i+1] {
				m.value = nil // not in sequence and ascending order
				return
			}
		}
		m.value = nums
	}
}

// maxRangeTill bounds 1..N episode expansion; titles are untrusted input
// and an uncapped N is a memory-exhaustion vector. The largest legitimate
// episode count in the wild is ~2000.
const maxRangeTill = 10000

func toIntRangeTill() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			m.value = nil
			return
		}
		parts := strings.Split(strings.Trim(nonDigitsRegex.ReplaceAllString(v, " "), " "), " ")
		if len(parts) == 0 {
			m.value = nil
			return
		}
		if num, err := strconv.Atoi(parts[0]); err == nil {
			if num > maxRangeTill {
				m.value = nil
				return
			}
			nums := make([]int, num)
			for i := range num {
				nums[i] = i + 1
			}
			m.value = nums
			return
		}
	}
}

func toBoolean() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = true
	}
}

type valueSet[T comparable] struct {
	existMap map[T]struct{}
	values   []T
}

func (vs *valueSet[any]) append(v any) *valueSet[any] {
	if _, found := vs.existMap[v]; !found {
		vs.existMap[v] = struct{}{}
		vs.values = append(vs.values, v)
	}
	return vs
}

func (vs *valueSet[any]) exists(v any) bool {
	_, found := vs.existMap[v]
	return found
}

func toValueSet(v any) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if val, ok := m.value.(*valueSet[any]); ok {
			m.value = val.append(v)
		}
	}
}

func toIntArray() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if num, err := strconv.Atoi(stripNonDigits(v)); err == nil {
				m.value = []int{num}
				return
			}
		}
		m.value = []int{}
	}
}
