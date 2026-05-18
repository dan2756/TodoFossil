package main

import (
	"fmt"
	"io"
	"time"

	"github.com/dan2756/todo-age/internal/git"
	"github.com/dan2756/todo-age/internal/scan"
)

// row is one marker together with the blame of its line.
type row struct {
	File        string
	Line        int
	Tag         string
	Text        string
	Owner       string
	Author      string
	Email       string
	Commit      string
	Date        string
	AgeDays     int
	Uncommitted bool

	time time.Time
	age  time.Duration
}

// collect scans the tracked files in dir and blames every line that holds
// a marker. Problems with single files are reported to warn and skipped.
func collect(dir string, m *scan.Matcher, now time.Time, warn io.Writer) ([]row, error) {
	if err := git.CheckRepo(dir); err != nil {
		return nil, err
	}
	files, err := git.LsFiles(dir, nil)
	if err != nil {
		return nil, err
	}
	var rows []row
	for _, f := range files {
		hits, err := scan.File(m, dir, f, scan.DefaultMaxSize)
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			continue
		}
		lines := make([]int, len(hits))
		for i, h := range hits {
			lines[i] = h.Line
		}
		blame, err := git.Blame(dir, f, lines)
		if err != nil {
			fmt.Fprintf(warn, "todo-age: warning: %s: %v\n", f, err)
			continue
		}
		for _, h := range hits {
			b, ok := blame[h.Line]
			if !ok {
				fmt.Fprintf(warn, "todo-age: warning: %s:%d: no blame for line\n", f, h.Line)
				continue
			}
			rows = append(rows, newRow(h, b, now))
		}
	}
	return rows, nil
}

func newRow(h scan.Hit, b git.Line, now time.Time) row {
	r := row{
		File:   h.File,
		Line:   h.Line,
		Tag:    h.Tag,
		Text:   h.Text,
		Owner:  h.Owner,
		Author: b.Author,
		Email:  b.AuthorMail,
		time:   b.AuthorTime,
	}
	if b.Uncommitted() {
		// git reports the current time and a placeholder address.
		r.Uncommitted = true
		r.Email = ""
		r.time = now
	} else {
		r.Commit = b.Commit[:min(len(b.Commit), 7)]
	}
	r.Date = r.time.Format(time.DateOnly)
	r.age = max(now.Sub(r.time), 0)
	r.AgeDays = int(r.age / (24 * time.Hour))
	return r
}
