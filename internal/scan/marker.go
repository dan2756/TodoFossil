// Package scan finds TODO-style markers in source files.
package scan

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
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
// rest of the line. A tag that is itself a quoted string, as in
// []string{"TODO"}, is not a marker.
func (m *Matcher) Find(line string) (Match, bool) {
	for _, loc := range m.re.FindAllStringIndex(line, -1) {
		start, end := loc[0], loc[1]
		if quoted(line, start, end) {
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
