package scan

import (
	"fmt"
	"path"
	"strings"
)

// Excluder matches file paths against exclude patterns.
//
// A pattern without a slash, such as "*.min.js" or "vendor", matches any
// single path element, so it excludes matching files and everything under
// matching directories at any depth. A pattern with a slash, such as
// "vendor/**" or "web/*.js", is matched against the path from its start;
// "**" matches any number of path elements. A pattern that matches a
// directory excludes everything below it.
type Excluder struct {
	patterns []pattern
}

type pattern struct {
	elems    []string
	anywhere bool // no slash: match any single element
}

// NewExcluder compiles patterns. Empty patterns are ignored.
func NewExcluder(patterns []string) (*Excluder, error) {
	e := &Excluder{}
	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		p = strings.TrimPrefix(p, "./")
		p = strings.Trim(p, "/")
		if p == "" {
			continue
		}
		elems := strings.Split(p, "/")
		for _, el := range elems {
			if _, err := path.Match(el, ""); err != nil {
				return nil, fmt.Errorf("bad exclude pattern %q: %w", raw, err)
			}
		}
		e.patterns = append(e.patterns, pattern{elems: elems, anywhere: len(elems) == 1})
	}
	return e, nil
}

// Match reports whether the slash-separated path name is excluded.
func (e *Excluder) Match(name string) bool {
	if e == nil {
		return false
	}
	elems := strings.Split(path.Clean(name), "/")
	for _, p := range e.patterns {
		if p.anywhere {
			for _, el := range elems {
				if ok, _ := path.Match(p.elems[0], el); ok {
					return true
				}
			}
		} else if matchPrefix(p.elems, elems) {
			return true
		}
	}
	return false
}

// matchPrefix reports whether pat matches a leading run of elems.
func matchPrefix(pat, elems []string) bool {
	if len(pat) == 0 {
		return true
	}
	if pat[0] == "**" {
		for i := 0; i <= len(elems); i++ {
			if matchPrefix(pat[1:], elems[i:]) {
				return true
			}
		}
		return false
	}
	if len(elems) == 0 {
		return false
	}
	ok, _ := path.Match(pat[0], elems[0])
	return ok && matchPrefix(pat[1:], elems[1:])
}
