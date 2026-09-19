package main

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dan2756/todo-age/internal/report"
)

// filter decides which rows are reported.
type filter struct {
	tags      []string       // only these tags, compared case-insensitively; empty means all
	author    *regexp.Regexp // matched against author name and email
	olderThan time.Duration  // zero means no limit
	newerThan time.Duration  // zero means no limit
}

func (f filter) keep(r report.Row) bool {
	if len(f.tags) > 0 && !containsFold(f.tags, r.Tag) {
		return false
	}
	if f.author != nil && !f.author.MatchString(r.Author) && !f.author.MatchString(r.Email) {
		return false
	}
	if f.olderThan > 0 && r.Age <= f.olderThan {
		return false
	}
	if f.newerThan > 0 && r.Age >= f.newerThan {
		return false
	}
	return true
}

// apply removes the rows f does not keep.
func (f filter) apply(rows []report.Row) []report.Row {
	return slices.DeleteFunc(rows, func(r report.Row) bool { return !f.keep(r) })
}

// scanTags returns the tags to look for: tags, plus any tag in only that is
// not among them yet, so that --tag BUG works without also passing --tags.
// only is compared case-insensitively, but tags are matched as written, so
// added tags are upper-cased: --tag bug looks for BUG markers, not for the
// word "bug" in prose.
func scanTags(tags, only []string) []string {
	out := slices.Clone(tags)
	for _, t := range only {
		if !containsFold(out, t) {
			out = append(out, strings.ToUpper(t))
		}
	}
	return out
}

func containsFold(list []string, s string) bool {
	return slices.ContainsFunc(list, func(v string) bool { return strings.EqualFold(v, s) })
}
