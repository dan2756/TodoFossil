package main

import (
	"regexp"
	"testing"

	"github.com/dan2756/todo-age/internal/age"
	"github.com/dan2756/todo-age/internal/report"
)

func TestFilter(t *testing.T) {
	old := report.Row{Tag: "TODO", Author: "Maria Lopez", Email: "maria@example.com", Age: 400 * age.Day}
	mid := report.Row{Tag: "FIXME", Author: "Sam Okafor", Email: "sam@corp.example", Age: 60 * age.Day}
	fresh := report.Row{Tag: "HACK", Author: "Not Committed Yet", Uncommitted: true}

	tests := []struct {
		name string
		f    filter
		want []bool // kept: old, mid, fresh
	}{
		{"none", filter{}, []bool{true, true, true}},
		{"tag", filter{tags: []string{"fixme", "HACK"}}, []bool{false, true, true}},
		{"author name", filter{author: regexp.MustCompile(`^Maria`)}, []bool{true, false, false}},
		{"author email", filter{author: regexp.MustCompile(`@corp\.`)}, []bool{false, true, false}},
		{"older than", filter{olderThan: 90 * age.Day}, []bool{true, false, false}},
		{"older than is strict", filter{olderThan: 60 * age.Day}, []bool{true, false, false}},
		{"newer than", filter{newerThan: 90 * age.Day}, []bool{false, true, true}},
		{"window", filter{olderThan: 30 * age.Day, newerThan: age.Year}, []bool{false, true, false}},
	}
	for _, tt := range tests {
		for i, r := range []report.Row{old, mid, fresh} {
			if got := tt.f.keep(r); got != tt.want[i] {
				t.Errorf("%s: keep(row %d) = %v, want %v", tt.name, i, got, tt.want[i])
			}
		}
	}
}
