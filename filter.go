package main

import (
	"slices"
	"strings"

	"github.com/dan2756/todo-age/internal/report"
)

// filter decides which rows are reported.
type filter struct {
	tags []string // only these tags, compared case-insensitively; empty means all
}

func (f filter) keep(r report.Row) bool {
	if len(f.tags) > 0 && !containsFold(f.tags, r.Tag) {
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
func scanTags(tags, only []string) []string {
	out := slices.Clone(tags)
	for _, t := range only {
		if !containsFold(out, t) {
			out = append(out, t)
		}
	}
	return out
}

func containsFold(list []string, s string) bool {
	return slices.ContainsFunc(list, func(v string) bool { return strings.EqualFold(v, s) })
}
