package rules

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokKind uint8

const (
	tEOF tokKind = iota
	tIdent
	tNum
	tStr
	tOp
	tHash
)

type token struct {
	kind tokKind
	text string
	val  Value // literals only
	pos  int
}

func (t token) is(op string) bool { return t.kind == tOp && t.text == op }

// Word operators — and, or, not, in, matches, contains, startsWith, endsWith —
// are lexed as plain identifiers. The parser decides whether one is an
// operator by where it sits, which is what lets `contains` be both a field
// name and an operator without the lexer having to guess.

// punctuation, longest first so that ">=" is never read as ">" then "=".
var punct = []string{"==", "!=", "<=", ">=", "&&", "||", "(", ")", "[", "]", ",", ".", "?", ":", "<", ">", "+", "-", "*", "/", "%"}

type lexer struct {
	src  string
	pos  int
	toks []token
}

// lex turns source into tokens. It never panics on malformed input; every
// rejection comes back as an error naming a byte offset.
func lex(src string) ([]token, error) {
	l := &lexer{src: src}
	for {
		l.skipSpace()
		if l.pos >= len(l.src) {
			l.toks = append(l.toks, token{kind: tEOF, pos: l.pos})
			return l.toks, nil
		}
		if len(l.toks) > maxTokens {
			return nil, fmt.Errorf("expression has more than %d tokens", maxTokens)
		}
		start := l.pos
		c := l.src[l.pos]
		// The rune is decoded up front so that a byte which is not the start
		// of an identifier cannot reach lexIdent. One that did would match no
		// identifier character, consume nothing, and leave this loop spinning
		// on the same position forever.
		r, rsize := utf8.DecodeRuneInString(l.src[l.pos:])
		switch {
		case c == '"' || c == '\'' || c == '`':
			s, err := l.lexString()
			if err != nil {
				return nil, err
			}
			l.toks = append(l.toks, token{kind: tStr, text: s, val: StrOf(s), pos: start})
		case c >= '0' && c <= '9':
			n, err := l.lexNumber()
			if err != nil {
				return nil, err
			}
			l.toks = append(l.toks, token{kind: tNum, text: l.src[start:l.pos], val: NumOf(n), pos: start})
		case c == '#':
			l.pos++
			l.toks = append(l.toks, token{kind: tHash, text: "#", pos: start})
		case isIdentStart(r):
			name := l.lexIdent()
			l.toks = append(l.toks, token{kind: tIdent, text: name, pos: start})
		default:
			op := l.lexPunct()
			if op == "" {
				l.pos += rsize
				return nil, fmt.Errorf("unexpected character %q at %d", string(r), start)
			}
			// && and || are accepted as spellings of and/or so a condition
			// pasted from another engine compiles rather than half-compiling.
			switch op {
			case "&&":
				op = "and"
			case "||":
				op = "or"
			}
			l.toks = append(l.toks, token{kind: tOp, text: op, pos: start})
		}
		if l.pos == start {
			return nil, fmt.Errorf("cannot read the expression at %d", start)
		}
	}
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.src) {
		switch l.src[l.pos] {
		case ' ', '\t', '\n', '\r':
			l.pos++
		default:
			return
		}
	}
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (l *lexer) lexIdent() string {
	start := l.pos
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		if !isIdentPart(r) {
			break
		}
		l.pos += size
	}
	return l.src[start:l.pos]
}

func (l *lexer) lexPunct() string {
	for _, p := range punct {
		if strings.HasPrefix(l.src[l.pos:], p) {
			l.pos += len(p)
			return p
		}
	}
	return ""
}

func (l *lexer) lexNumber() (float64, error) {
	start := l.pos
	seenDot := false
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c >= '0' && c <= '9' {
			l.pos++
			continue
		}
		// a dot is part of the number only when a digit follows, so that
		// `1.5` lexes as a number and `seasons.0` could never try to.
		if c == '.' && !seenDot && l.pos+1 < len(l.src) && l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9' {
			seenDot = true
			l.pos++
			continue
		}
		if c == '_' {
			l.pos++
			continue
		}
		break
	}
	text := strings.ReplaceAll(l.src[start:l.pos], "_", "")
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("bad number %q at %d", text, start)
	}
	return n, nil
}

// stringEscapes are the continuations the string layer still interprets,
// beyond \\ and the active quote. Deliberately without 'b'.
const stringEscapes = "afnrtv"

// lexString reads a quoted string, treating a backslash before anything that
// is not a real string escape as a literal backslash.
//
// Conditions are written by people pasting regular expressions out of regex
// tools and community lists, and those are written in raw notation: \+ for a
// literal plus, \d for a digit, \b for a word boundary. Requiring every
// backslash to be doubled would make the string layer's rules everyone's
// problem. \b is taken literally too — the string layer reads it as a
// backspace, but no release name contains one and every regex author means a
// word boundary, so the string meaning is always the bug.
func (l *lexer) lexString() (string, error) {
	quote := l.src[l.pos]
	start := l.pos
	l.pos++
	var b strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == quote {
			l.pos++
			return b.String(), nil
		}
		if quote == '`' { // raw: no escapes at all
			b.WriteByte(c)
			l.pos++
			continue
		}
		if c != '\\' || l.pos+1 >= len(l.src) {
			b.WriteByte(c)
			l.pos++
			continue
		}
		next := l.src[l.pos+1]
		switch {
		case next == quote:
			b.WriteByte(next)
		case next == '\\':
			b.WriteByte('\\')
		case strings.IndexByte(stringEscapes, next) >= 0:
			b.WriteByte(map[byte]byte{'a': 7, 'f': 12, 'n': '\n', 'r': '\r', 't': '\t', 'v': 11}[next])
		case next == 'x' || next == 'u' || next == 'U' || (next >= '0' && next <= '7'):
			// numeric escapes name the same character in a string as in a
			// regex, so passing them through changes nothing either way
			b.WriteByte('\\')
			b.WriteByte(next)
		default:
			// raw regex notation: \b, \d, \+, \.
			b.WriteByte('\\')
			b.WriteByte(next)
		}
		l.pos += 2
	}
	return "", fmt.Errorf("unterminated string at %d", start)
}
