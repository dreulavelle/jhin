package rules

import (
	"fmt"
	"strconv"
	"strings"
)

// The text form: one rule per line, which is the shape that suits writing ten
// in a sitting, reordering them, or pasting a set someone shared.
//
//	Atmos: score -800 if "atmos" in traits
//	DV without fallback: reject if dolbyVision and not hdrFallback
//	At most 3 in 4K [movie]: keep 3 if resolution == "2160p"
//	Best 3 of each flavour: keep 3 per resolution + " " + quality if true
//	UHD T1: define if group in ["FraMeSToR", "W4NK3R"]
//	Old experiment [off]: score 100 if "remux" in traits
//
// A line is `Name: action if condition`. Brackets before the colon carry the
// scope and `off` for a disabled rule, in either order and both optional.
//
// This grammar wraps conditions; it never parses them. Everything after `if`
// is handed to the expression language exactly as written, so a condition
// containing a colon, a bracket or the word "if" inside a string survives.

// ParseText reads the text form. It reports the line number with any error,
// so a half-typed rule names itself rather than the file.
func ParseText(src string) ([]Rule, error) {
	var out []Rule
	for i, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		r, err := ParseLine(trimmed)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// ParseLine reads one rule.
func ParseLine(line string) (Rule, error) {
	var r Rule
	head, rest, ok := strings.Cut(line, ":")
	if !ok {
		return r, fmt.Errorf("expected `Name: action if condition`")
	}

	name, scope, enabled, err := parseHead(head)
	if err != nil {
		return r, err
	}
	r.Name, r.Scope = name, scope
	if !enabled {
		off := false
		r.Enabled = &off
	}

	rest = strings.TrimSpace(rest)
	action, tail, _ := strings.Cut(rest, " ")
	action = strings.ToLower(strings.TrimSpace(action))
	tail = strings.TrimSpace(tail)
	if action == "" || action == "if" {
		return r, fmt.Errorf("expected an action after the colon, e.g. `score 100 if ...`")
	}

	body, condition, err := splitCondition(tail)
	if err != nil {
		return r, err
	}
	if strings.TrimSpace(condition) == "" {
		return r, fmt.Errorf("no condition: a rule needs `if <condition>`")
	}
	r.When = strings.TrimSpace(condition)

	switch action {
	case "reject":
		r.Action = ActionReject
		if body != "" {
			return r, fmt.Errorf("reject takes nothing before `if`, got %q", body)
		}
	case "define":
		r.Action = ActionDefine
		if body != "" {
			return r, fmt.Errorf("define takes nothing before `if`, got %q", body)
		}
	case "score":
		r.Action = ActionScore
		r.Score = body
	case "keep":
		r.Action = ActionLimit
		count, group, err := parseKeep(body)
		if err != nil {
			return r, err
		}
		r.Count, r.GroupBy = count, group
	default:
		// an application-registered effect; its value is the body
		r.Action = action
		r.Score = body
	}
	return r, nil
}

// parseHead reads `Name [scope] [off]`, in either order and both optional.
func parseHead(head string) (name string, scope []string, enabled bool, err error) {
	head = strings.TrimSpace(head)
	enabled = true
	for strings.HasSuffix(head, "]") {
		open := strings.LastIndexByte(head, '[')
		if open < 0 {
			return "", nil, false, fmt.Errorf("unbalanced \"]\" in the rule name")
		}
		tag := strings.ToLower(strings.TrimSpace(head[open+1 : len(head)-1]))
		head = strings.TrimSpace(head[:open])
		switch tag {
		case "off", "disabled":
			enabled = false
		case "":
			return "", nil, false, fmt.Errorf("empty [] in the rule name")
		default:
			for _, s := range strings.Split(tag, ",") {
				if s = strings.TrimSpace(s); s != "" {
					scope = append(scope, s)
				}
			}
		}
	}
	if head == "" {
		return "", nil, false, fmt.Errorf("the rule has no name")
	}
	// brackets were read right to left; put the scopes back in written order
	for i, j := 0, len(scope)-1; i < j; i, j = i+1, j-1 {
		scope[i], scope[j] = scope[j], scope[i]
	}
	return head, scope, enabled, nil
}

// parseKeep reads `N` or `N per <expression>`.
func parseKeep(body string) (int, string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0, "", fmt.Errorf("keep needs a number, e.g. `keep 3`")
	}
	countText, rest, _ := strings.Cut(body, " ")
	n, err := strconv.Atoi(strings.TrimSpace(countText))
	if err != nil {
		return 0, "", fmt.Errorf("keep needs a number, got %q", countText)
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return n, "", nil
	}
	per, group, found := strings.Cut(rest, " ")
	if strings.ToLower(strings.TrimSpace(per)) != "per" || !found {
		return 0, "", fmt.Errorf("after the number, keep takes `per <grouping>`, got %q", rest)
	}
	return n, strings.TrimSpace(group), nil
}

// splitCondition finds the ` if ` that separates a rule's body from its
// condition, ignoring any inside a string or a bracketed group — so
// `score 1 if title contains " if "` splits in the one right place.
func splitCondition(s string) (body, condition string, err error) {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == '\\' && quote != '`' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			quote = c
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case 'i':
			// the separator is `if ` at the start of the tail — which is
			// where a bodiless action like `reject` leaves it — or after a
			// space, and never inside a string or a bracketed group
			if depth != 0 || (i > 0 && s[i-1] != ' ') {
				continue
			}
			if strings.HasPrefix(s[i:], "if ") {
				return strings.TrimSpace(s[:max(i-1, 0)]), s[i+3:], nil
			}
		}
	}
	if quote != 0 {
		return "", "", fmt.Errorf("unterminated string")
	}
	return "", "", fmt.Errorf("expected `if <condition>`")
}

// FormatText writes rules back as lines. Round-tripping is exact for
// everything the text form can express.
func FormatText(rules []Rule) string {
	var b strings.Builder
	for _, r := range rules {
		b.WriteString(FormatLine(r))
		b.WriteByte('\n')
	}
	return b.String()
}

// FormatLine writes one rule as a line.
func FormatLine(r Rule) string {
	var b strings.Builder
	b.WriteString(r.Name)
	if len(r.Scope) > 0 {
		b.WriteString(" [")
		b.WriteString(strings.Join(r.Scope, ", "))
		b.WriteByte(']')
	}
	if !r.IsEnabled() {
		b.WriteString(" [off]")
	}
	b.WriteString(": ")

	switch r.EffectiveAction() {
	case ActionReject:
		b.WriteString("reject")
	case ActionDefine:
		b.WriteString("define")
	case ActionLimit:
		fmt.Fprintf(&b, "keep %d", r.Count)
		if r.GroupBy != "" {
			b.WriteString(" per ")
			b.WriteString(r.GroupBy)
		}
	case ActionScore:
		// a score rule that never had points recorded is written as score 0,
		// so the line still parses back to the same rule
		score := strings.TrimSpace(r.Score)
		if score == "" {
			score = "0"
		}
		b.WriteString("score ")
		b.WriteString(score)
	default:
		b.WriteString(r.EffectiveAction())
		if s := strings.TrimSpace(r.Score); s != "" {
			b.WriteByte(' ')
			b.WriteString(s)
		}
	}
	b.WriteString(" if ")
	b.WriteString(strings.Join(strings.Fields(r.When), " "))
	return b.String()
}
