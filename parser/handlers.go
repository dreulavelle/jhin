package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type hProcessor func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta
type hTransformer func(title string, m *parseMeta, result map[string]*parseMeta)
type hMatchValidator func(input string, idxs []int) bool

type handler struct {
	Field         string
	Pattern       *regexp.Regexp
	ValidateMatch hMatchValidator
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

func validate_or(validators ...hMatchValidator) hMatchValidator {
	return func(input string, idxs []int) bool {
		for _, validator := range validators {
			if validator(input, idxs) {
				return true
			}
		}
		return false
	}
}

func validate_and(validators ...hMatchValidator) hMatchValidator {
	return func(input string, idxs []int) bool {
		for _, validator := range validators {
			if !validator(input, idxs) {
				return false
			}
		}
		return true
	}
}

func validate_not_at_start() hMatchValidator {
	return func(input string, match []int) bool {
		return match[0] != 0
	}
}
func validate_not_at_end() hMatchValidator {
	return func(input string, match []int) bool {
		return match[1] != len(input)
	}
}

func validate_lookbehind(pattern, flags string, polarity bool) hMatchValidator {
	re := regexp.MustCompile("(?" + flags + ")(?:" + pattern + ")$")
	return func(input string, match []int) bool {
		rv := input[:match[0]]
		if polarity {
			return re.MatchString(rv)
		}
		return !re.MatchString(rv)
	}
}

func validate_lookahead(pattern, flags string, polarity bool) hMatchValidator {
	re := regexp.MustCompile("(?" + flags + ")^(?:" + pattern + ")")
	return func(input string, match []int) bool {
		rv := input[match[1]:]
		if polarity {
			return re.MatchString(rv)
		}
		return !re.MatchString(rv)
	}
}

func validate_not_match(re *regexp.Regexp) hMatchValidator {
	return func(input string, match []int) bool {
		rv := input[match[0]:match[1]]
		return !re.MatchString(rv)
	}
}

func validate_match(re *regexp.Regexp) hMatchValidator {
	return func(input string, match []int) bool {
		rv := input[match[0]:match[1]]
		return re.MatchString(rv)
	}
}

func validate_matched_groups_are_same(indices ...int) hMatchValidator {
	return func(input string, match []int) bool {
		first := input[match[indices[0]*2]:match[indices[0]*2+1]]
		for _, index := range indices[1:] {
			other := input[match[index*2]:match[index*2+1]]
			if other != first {
				return false
			}
		}
		return true
	}
}

func to_value(value string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = value
	}
}

func to_lowercase() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.ToLower(m.value.(string))
	}
}

func to_uppercase() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.ToUpper(m.value.(string))
	}
}

func to_trimmed() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.TrimSpace(m.value.(string))
	}
}

func to_clean_date() hTransformer {
	re := regexp.MustCompile(`(\d+)(?:st|nd|rd|th)`)
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = re.ReplaceAllString(v, "$1")
		}
	}
}

func to_clean_month() hTransformer {
	re := regexp.MustCompile(`(?i)(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)`)
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = re.ReplaceAllStringFunc(v, func(str string) string {
				return str[0:3]
			})
		}
	}
}

func to_date(format string) hTransformer {
	seperatorRe := regexp.MustCompile(`[.\-/\\]`)
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if t, err := time.Parse(format, seperatorRe.ReplaceAllString(v, " ")); err == nil {
				m.value = t.Format("2006-01-02")
				return
			}
		}
		m.value = ""
	}
}

func to_year() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		vstr, ok := m.value.(string)
		if !ok {
			m.value = ""
			return
		}
		parts := non_digits_regex.Split(vstr, -1)
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

func to_int_range() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			m.value = nil
			return
		}
		parts := strings.Split(strings.Trim(non_digits_regex.ReplaceAllString(v, " "), " "), " ")
		nums := make([]int, len(parts))
		for i, part := range parts {
			if num, err := strconv.Atoi(part); err == nil {
				nums[i] = num
			}
		}
		if len(nums) == 2 && nums[0] < nums[1] {
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

func to_int_range_till() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			m.value = nil
			return
		}
		parts := strings.Split(strings.Trim(non_digits_regex.ReplaceAllString(v, " "), " "), " ")
		if len(parts) == 0 {
			m.value = nil
			return
		}
		if num, err := strconv.Atoi(parts[0]); err == nil {
			nums := make([]int, num)
			for i := range num {
				nums[i] = i + 1
			}
			m.value = nums
			return
		}
	}
}

func to_with_suffix(suffix string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = v + suffix
		} else {
			m.value = ""
		}
	}
}

func to_boolean() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = true
	}
}

type value_set[T comparable] struct {
	existMap map[T]struct{}
	values   []T
}

func (vs *value_set[any]) append(v any) *value_set[any] {
	if _, found := vs.existMap[v]; !found {
		vs.existMap[v] = struct{}{}
		vs.values = append(vs.values, v)
	}
	return vs
}

func (vs *value_set[any]) exists(v any) bool {
	_, found := vs.existMap[v]
	return found
}

func to_value_set(v any) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if val, ok := m.value.(*value_set[any]); ok {
			m.value = val.append(v)
		}
	}
}

func to_value_set_with_transform(to_v func(v string) any) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if val, ok := m.value.(*value_set[any]); ok {
			m.value = val.append(to_v(m.mValue))
		}
	}
}

func to_value_set_multi_with_transform(to_v func(v string) []any) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if val, ok := m.value.(*value_set[any]); ok {
			for _, v := range to_v(m.mValue) {
				m.value = val.append(v)
			}
		}
	}
}

func to_int_array() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if num, err := strconv.Atoi(v); err == nil {
				m.value = []int{num}
				return
			}
		}
		m.value = []int{}
	}
}

func remove_from_value(re *regexp.Regexp) hProcessor {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && v != "" {
			m.value = re.ReplaceAllString(v, "")
		}
		return m
	}
}

func regex_match_until_valid(re *regexp.Regexp, validator hMatchValidator) hProcessor {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) *parseMeta {
		offset := 0
		for offset < len(title) {
			idxs := re.FindStringSubmatchIndex(title[offset:])
			if idxs == nil {
				return m
			}
			for i := range idxs {
				if idxs[i] >= 0 {
					idxs[i] += offset
				}
			}
			if validator(title, idxs) {
				m.mIndex = idxs[0]
				m.mValue = title[idxs[0]:idxs[1]]
				if len(idxs) >= 4 && idxs[2] >= 0 && idxs[3] >= 0 {
					m.value = title[idxs[2]:idxs[3]]
				} else {
					m.value = m.mValue
				}
				return m
			}
			offset = idxs[1]
			if offset == idxs[0] {
				offset++
			}
		}
		return m
	}
}
