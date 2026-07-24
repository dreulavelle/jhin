package rank

// Title similarity: indel ratio over normalized titles (accent folding +
// punctuation stripping).

import (
	"strings"
	"unicode"
)

var accent_fold = map[rune]string{
	'ā': "a", 'ă': "a", 'ą': "a", 'ǎ': "a", 'ǻ': "a", 'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a",
	'ć': "c", 'č': "c", 'ç': "c", 'ĉ': "c", 'ċ': "c",
	'ď': "d", 'đ': "d",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ę': "e", 'ě': "e", 'ə': "e",
	'ĝ': "g", 'ğ': "g", 'ġ': "g", 'ģ': "g", 'ǧ': "g",
	'ĥ': "h",
	'î': "i", 'ï': "i", 'ì': "i", 'í': "i", 'ī': "i", 'ĩ': "i", 'ĭ': "i", 'ı': "i", 'ǐ': "i",
	'ĵ': "j", 'ķ': "k",
	'ĺ': "l", 'ļ': "l", 'ł': "l",
	'ń': "n", 'ň': "n", 'ñ': "n", 'ņ': "n", 'ŉ': "n", 'ǹ': "n",
	'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ő': "o", 'ǒ': "o", 'ǿ': "o", 'ò': "o",
	'œ': "oe", 'æ': "ae", 'ǽ': "ae",
	'ŕ': "r", 'ř': "r", 'ŗ': "r",
	'š': "s", 'ş': "s", 'ś': "s", 'ș': "s", 'ß': "ss",
	'ť': "t", 'ţ': "t",
	'ū': "u", 'ŭ': "u", 'ũ': "u", 'û': "u", 'ü': "u", 'ù': "u", 'ú': "u", 'ų': "u", 'ű': "u", 'ǔ': "u", 'ǚ': "u", 'ǜ': "u",
	'ŵ': "w",
	'ý': "y", 'ÿ': "y", 'ŷ': "y",
	'ž': "z", 'ż': "z", 'ź': "z",
	'ƒ': "f",
	'&': "and", '.': " ", '_': " ",
}

// Normalize lowers, folds accents, converts '&' to "and", and strips
// everything that is not alphanumeric or whitespace. Both sides of every
// similarity comparison go through this.
func Normalize(title string) string {
	var b strings.Builder
	b.Grow(len(title))
	for _, r := range strings.ToLower(title) {
		if repl, ok := accent_fold[r]; ok {
			b.WriteString(repl)
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// Similarity returns the indel ratio in [0,1] between two already-raw titles
// after normalization: 1 - indel_distance/(len(a)+len(b)).
func Similarity(a, b string) float64 {
	return indel_ratio([]rune(Normalize(a)), []rune(Normalize(b)))
}

// TitleMatch reports whether two titles are the same release title at the
// given threshold (0 uses the default 0.85). Aliases, when given, are
// alternative correct titles; the best ratio wins.
func TitleMatch(correctTitle, parsedTitle string, threshold float64, aliases ...string) bool {
	if threshold == 0 {
		threshold = 0.85
	}
	return BestSimilarity(correctTitle, parsedTitle, aliases...) >= threshold
}

// BestSimilarity returns the highest similarity between parsedTitle and the
// correct title or any alias.
func BestSimilarity(correctTitle, parsedTitle string, aliases ...string) float64 {
	parsed := []rune(Normalize(parsedTitle))
	best := indel_ratio([]rune(Normalize(correctTitle)), parsed)
	for _, alias := range aliases {
		if r := indel_ratio([]rune(Normalize(alias)), parsed); r > best {
			best = r
		}
	}
	return best
}

// indel_ratio computes 1 - D/(m+n) where D is the minimum number of
// insertions+deletions transforming a into b (substitution costs 2, i.e.
// LCS-based distance).
func indel_ratio(a, b []rune) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	total := len(a) + len(b)
	if len(a) == 0 || len(b) == 0 {
		return float64(total-total) / float64(total) // 0
	}
	// D = m + n - 2*LCS(a,b)
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else {
				cur[j] = max(prev[j], cur[j-1])
			}
		}
		prev, cur = cur, prev
	}
	lcs := prev[len(b)]
	dist := total - 2*lcs
	return float64(total-dist) / float64(total)
}
