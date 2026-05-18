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
	re *regexp.Regexp
}

var validTag = regexp.MustCompile(`^\w+$`)

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
	// A tag as a whole word, an optional (owner), then the rest of the line.
	re := regexp.MustCompile(`\b(` + strings.Join(alts, "|") + `)\b(?:\(([^()]*)\))?(.*)`)
	return &Matcher{re: re}, nil
}

// Find returns the first marker on line, if any.
func (m *Matcher) Find(line string) (Match, bool) {
	sm := m.re.FindStringSubmatch(line)
	if sm == nil {
		return Match{}, false
	}
	return Match{
		Tag:   sm[1],
		Owner: strings.TrimSpace(sm[2]),
		Text:  cleanText(sm[3]),
	}, true
}

// mayContain reports whether data contains any marker at all, so that most
// files can be skipped without splitting them into lines.
func (m *Matcher) mayContain(data []byte) bool {
	return m.re.Match(data)
}

// cleanText strips the separator between a marker and its text.
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, ":")
	return strings.TrimSpace(s)
}
