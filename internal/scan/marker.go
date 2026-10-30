// Package scan finds TODO-style markers in source files.
package scan

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DefaultTags are the markers looked for when none are configured.
var DefaultTags = []string{"TODO", "FIXME", "HACK", "XXX"}

// Match is a marker found on a line.
type Match struct {
	Tag   string // the marker, e.g. "TODO"
	Owner string // from TODO(owner), if present
	Text  string // the text after the marker
}

// Matcher finds markers in lines of text.
type Matcher struct {
	re *regexp.Regexp // any tag as a whole word
}

var (
	validTag = regexp.MustCompile(`^\w+$`)
	ownerRE  = regexp.MustCompile(`^\(([^()]*)\)`)
)

// commentClosers are stripped from the end of a marker's text.
var commentClosers = []string{"*/", "-->", "#}", "%>"}

// NewMatcher returns a Matcher for tags. Tags are case-sensitive and may
// contain only letters, digits and underscores.
func NewMatcher(tags []string) (*Matcher, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags to look for")
	}
	alts := make([]string, len(tags))
	for i, t := range tags {
		if !validTag.MatchString(t) {
			return nil, fmt.Errorf("invalid tag %q: use only letters, digits and underscores", t)
		}
		alts[i] = regexp.QuoteMeta(t)
	}
	return &Matcher{re: regexp.MustCompile(`\b(?:` + strings.Join(alts, "|") + `)\b`)}, nil
}

// Find returns the first marker on line, if any. A marker is a tag as a
// whole word, optionally followed by "(owner)" and a colon; its text is the
// rest of the line. Letters, digits and combining marks outside ASCII are
// part of the word, so "ÄTODO" is not a marker. A tag that is itself a
// quoted string, as in []string{"TODO"}, is not a marker.
func (m *Matcher) Find(line string) (Match, bool) {
	for _, loc := range m.re.FindAllStringIndex(line, -1) {
		start, end := loc[0], loc[1]
		if !wholeWord(line, start, end) || quoted(line, start, end) {
			continue
		}
		match := Match{Tag: line[start:end]}
		rest := line[end:]
		if sm := ownerRE.FindStringSubmatch(rest); sm != nil {
			match.Owner = strings.TrimSpace(sm[1])
			rest = rest[len(sm[0]):]
		}
		match.Text = cleanText(rest)
		return match, true
	}
	return Match{}, false
}

// mayContain reports whether data contains any tag at all, so that most
// files can be skipped without splitting them into lines.
func (m *Matcher) mayContain(data []byte) bool {
	return m.re.Match(data)
}

// identRune reports whether r continues a word. The matcher's \b is
// ASCII-only, so without this a tag glued to a non-ASCII letter or digit
// would look like a whole word.
func identRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.Is(unicode.M, r) || r == '_'
}

// wholeWord reports whether line[start:end] is bounded by a word break,
// counting Unicode letters, digits and combining marks as part of the word.
func wholeWord(line string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(line[:start])
		if identRune(r) {
			return false
		}
	}
	if end < len(line) {
		r, _ := utf8.DecodeRuneInString(line[end:])
		if identRune(r) {
			return false
		}
	}
	return true
}

// quoted reports whether line[start:end] is wrapped in matching quotes.
func quoted(line string, start, end int) bool {
	if start == 0 || end >= len(line) {
		return false
	}
	q := line[start-1]
	return (q == '"' || q == '\'' || q == '`') && line[end] == q
}

// cleanText strips the separator between a marker and its text, and any
// comment terminator at the end of the line.
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	for _, c := range commentClosers {
		if t, ok := strings.CutSuffix(s, c); ok {
			s = strings.TrimSpace(t)
			break
		}
	}
	if t, ok := strings.CutPrefix(s, ":"); ok {
		s = t
	} else if t, ok := strings.CutPrefix(s, "- "); ok {
		s = t
	}
	return strings.TrimSpace(s)
}
