package git

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParsePorcelain(t *testing.T) {
	f, err := os.Open("testdata/blame.porcelain")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines, err := ParsePorcelain(f)
	if err != nil {
		t.Fatal(err)
	}

	const (
		maria = "152579c5f0bdca2ad727172d6838b27f35134cab"
		sam   = "9e4f0c3b7a1d2e5f60718293a4b5c6d7e8f90a1b"
	)
	tests := []struct {
		line        int
		commit      string
		author      string
		mail        string
		time        string
		uncommitted bool
	}{
		{3, maria, "Maria Lopez", "maria@example.com", "2024-12-31T17:00:00-07:00", false},
		// Later lines of a commit carry only the header line.
		{4, maria, "Maria Lopez", "maria@example.com", "2024-12-31T17:00:00-07:00", false},
		{5, sam, "Sam Okafor", "sam@example.com", "2026-01-01T01:00:00+01:00", false},
		{9, maria, "Maria Lopez", "maria@example.com", "2024-12-31T17:00:00-07:00", false},
		{12, strings.Repeat("0", 40), "Not Committed Yet", "not.committed.yet", "2026-05-18T19:04:12Z", true},
	}
	if len(lines) != len(tests) {
		t.Errorf("got %d lines, want %d", len(lines), len(tests))
	}
	for _, tt := range tests {
		got, ok := lines[tt.line]
		if !ok {
			t.Errorf("line %d missing", tt.line)
			continue
		}
		if got.Commit != tt.commit || got.Author != tt.author || got.AuthorMail != tt.mail {
			t.Errorf("line %d = %+v", tt.line, got)
		}
		if ts := got.AuthorTime.Format(time.RFC3339); ts != tt.time {
			t.Errorf("line %d: time %s, want %s", tt.line, ts, tt.time)
		}
		if got.Uncommitted() != tt.uncommitted {
			t.Errorf("line %d: Uncommitted() = %v", tt.line, got.Uncommitted())
		}
	}
}

func TestParsePorcelainErrors(t *testing.T) {
	const hash = "152579c5f0bdca2ad727172d6838b27f35134cab"
	inputs := map[string]string{
		"content first": "\tline\n",
		"bad header":    "not a header\n",
		"short hash":    "152579c 1 1 1\n\tline\n",
		"bad time":      hash + " 1 1 1\nauthor-time soon\n\tline\n",
		"truncated":     hash + " 1 1 1\nauthor Someone\n",
	}
	for name, in := range inputs {
		if _, err := ParsePorcelain(strings.NewReader(in)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParsePorcelainEmpty(t *testing.T) {
	lines, err := ParsePorcelain(strings.NewReader(""))
	if err != nil || len(lines) != 0 {
		t.Errorf("got %v, %v; want no lines", lines, err)
	}
}

func TestRanges(t *testing.T) {
	tests := []struct {
		in   []int
		want [][2]int
	}{
		{nil, nil},
		{[]int{7}, [][2]int{{7, 7}}},
		{[]int{12, 3, 4, 10, 4, 5, 11}, [][2]int{{3, 5}, {10, 12}}},
		{[]int{1, 3, 5}, [][2]int{{1, 1}, {3, 3}, {5, 5}}},
	}
	for _, tt := range tests {
		if got := Ranges(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Ranges(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
