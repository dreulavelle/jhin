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
//
// An indented line continues the one above it, because a condition worth
// writing is often too long to read on one:
//
//	Untrusted UHD encode: reject if
//	    resolution == "2160p" and "bluray" in traits
//	    and not (matched("UHD T1") or matched("UHD T2"))
//	    and exists(resolution == "2160p" and "remux" in traits)
//
// Rule names start at the left margin, so nothing else has to distinguish a
// continuation from a new rule.
func ParseText(src string) ([]Rule, error) {
	var out []Rule
	var pending string
	var pendingLine int

	flush := func() error {
		if pending == "" {
			return nil
		}
		r, err := ParseLine(pending)
		pending = ""
		if err != nil {
			return fmt.Errorf("line %d: %w", pendingLine, err)
		}
		out = append(out, r)
		return nil
	}

	for i, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		if indented(line) && pending != "" {
			// Joined with a newline rather than a space: the expression lexer
			// treats both as whitespace, and keeping the break means an error
			// names the line the author actually wrote it on.
			pending += "\n" + trimmed
			continue
		}
		if err := flush(); err != nil {
			return nil, err
		}
		pending, pendingLine = trimmed, i+1
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return out, nil
}

// isSpace covers the ASCII whitespace strings.TrimSpace strips. The two must
// agree: a byte one layer folds and another keeps would shift where a line
// splits between one parse and the next.
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// stringSpansLines rejects a string literal carrying a raw line break. The
// text form is line-oriented — canonical output is one rule per line — so a
// break inside a string could not survive a round trip through fmt. The
// escape spells it instead.
func stringSpansLines(s string) error {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote == 0 {
			if c == '"' || c == '\'' || c == '`' {
				quote = c
			}
			continue
		}
		if c == '\\' && quote != '`' && i+1 < len(s) && s[i+1] != '\n' && s[i+1] != '\r' {
			i++
			continue
		}
		if c == '\n' || c == '\r' {
			return fmt.Errorf("a string cannot continue on the next line; write \\n for a line break")
		}
		if c == quote {
			quote = 0
		}
	}
	return nil
}

// cutSpace splits at the first whitespace of any kind, so a tab between an
// action and its value reads the same as a space.
func cutSpace(s string) (head, tail string) {
	for i := 0; i < len(s); i++ {
		if isSpace(s[i]) {
			return s[:i], strings.TrimSpace(s[i+1:])
		}
	}
	return s, ""
}

func indented(line string) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
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
	action, tail := cutSpace(rest)
	// Normalized exactly as EffectiveAction normalizes, so the two can never
	// disagree about what the action is — TrimSpace also covers the unicode
	// spaces the byte-level split does not.
	action = strings.ToLower(strings.TrimSpace(action))
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
	// Validated on the final fields rather than the raw line, because the
	// splits above are not quote-aware: what ends up stored is what has to
	// survive being written back onto one line.
	for _, part := range []string{r.When, r.Score, r.GroupBy} {
		if err := stringSpansLines(part); err != nil {
			return r, err
		}
	}
	return r, nil
}

// parseHead reads `Name [scope] [off]`, in either order and both optional.
func parseHead(head string) (name string, scope []string, enabled bool, err error) {
	head = strings.TrimSpace(head)
	// The name is what starts a rule, so it lives on the first line by
	// definition; a break in here means a continuation line is carrying
	// what should have been a rule of its own.
	if strings.ContainsAny(head, "\n\r") {
		return "", nil, false, fmt.Errorf("a rule's name does not continue across lines")
	}
	enabled = true
	// Brackets are read right to left, so the groups land reversed; the
	// scopes inside one group are already in written order and stay so.
	var groups [][]string
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
			var group []string
			for _, s := range strings.Split(tag, ",") {
				switch s = strings.TrimSpace(s); s {
				case "":
				case "off", "disabled":
					// reserved by the format: a scope under this name would
					// write back as the flag and mean something else
					enabled = false
				default:
					group = append(group, s)
				}
			}
			if len(group) > 0 {
				groups = append(groups, group)
			}
		}
	}
	if head == "" {
		return "", nil, false, fmt.Errorf("the rule has no name")
	}
	for i := len(groups) - 1; i >= 0; i-- {
		scope = append(scope, groups[i]...)
	}
	return head, scope, enabled, nil
}

// parseKeep reads `N` or `N per <expression>`.
func parseKeep(body string) (int, string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0, "", fmt.Errorf("keep needs a number, e.g. `keep 3`")
	}
	countText, rest := cutSpace(body)
	n, err := strconv.Atoi(countText)
	if err != nil {
		return 0, "", fmt.Errorf("keep needs a number, got %q", countText)
	}
	if rest == "" {
		return n, "", nil
	}
	per, group := cutSpace(rest)
	if strings.ToLower(per) != "per" || group == "" {
		return 0, "", fmt.Errorf("after the number, keep takes `per <grouping>`, got %q", rest)
	}
	return n, group, nil
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
			// the separator is `if` at the start of the tail — which is where
			// a bodiless action like `reject` leaves it — or after a space,
			// and never inside a string or a bracketed group. What follows it
			// may be a newline as well as a space: a condition continued
			// across lines keeps its breaks so errors can name them.
			if depth != 0 || (i > 0 && !isSpace(s[i-1])) {
				continue
			}
			if i+2 < len(s) && s[i] == 'i' && s[i+1] == 'f' && isSpace(s[i+2]) {
				return strings.TrimSpace(s[:i]), s[i+3:], nil
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
			b.WriteString(foldSpace(r.GroupBy))
		}
	case ActionScore:
		// a score rule that never had points recorded is written as score 0,
		// so the line still parses back to the same rule
		score := foldSpace(r.Score)
		if score == "" {
			score = "0"
		}
		b.WriteString("score ")
		b.WriteString(score)
	default:
		b.WriteString(r.EffectiveAction())
		if s := foldSpace(r.Score); s != "" {
			b.WriteByte(' ')
			b.WriteString(s)
		}
	}
	b.WriteString(" if ")
	b.WriteString(foldSpace(r.When))
	return b.String()
}

// foldSpace collapses runs of whitespace to one space — which is how a
// condition written across continuation lines becomes one canonical line —
// while leaving string literals untouched: the spacing inside "two  spaces"
// is the value, not layout.
func foldSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var quote byte
	pending := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			b.WriteByte(c)
			if c == '\\' && quote != '`' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if isSpace(c) {
			pending = true
			continue
		}
		if pending && b.Len() > 0 {
			b.WriteByte(' ')
		}
		pending = false
		if c == '"' || c == '\'' || c == '`' {
			quote = c
		}
		b.WriteByte(c)
	}
	return b.String()
}
