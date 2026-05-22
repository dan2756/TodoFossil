package report

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// SortKeys are the accepted sort orders; the first is the default.
var SortKeys = []string{"age", "file", "author"}

// Sort orders rows in place by key:
//
//	age     oldest first
//	file    by file, then line
//	author  by author name, ignoring case, then oldest first
//
// Ties fall back to file and line, so the order is deterministic.
func Sort(rows []Row, key string) error {
	var primary func(a, b Row) int
	switch key {
	case "age":
		primary = byAge
	case "file":
		primary = func(a, b Row) int { return 0 }
	case "author":
		primary = func(a, b Row) int {
			if c := cmp.Compare(strings.ToLower(a.Author), strings.ToLower(b.Author)); c != 0 {
				return c
			}
			return byAge(a, b)
		}
	default:
		return fmt.Errorf("unknown sort key %q (want %s)", key, strings.Join(SortKeys, ", "))
	}
	slices.SortStableFunc(rows, func(a, b Row) int {
		if c := primary(a, b); c != 0 {
			return c
		}
		if c := cmp.Compare(a.File, b.File); c != 0 {
			return c
		}
		return cmp.Compare(a.Line, b.Line)
	})
	return nil
}

func byAge(a, b Row) int {
	return a.Time.Compare(b.Time)
}
