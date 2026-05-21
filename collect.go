package main

import (
	"fmt"
	"io"
	"time"

	"github.com/dan2756/todo-age/internal/git"
	"github.com/dan2756/todo-age/internal/report"
	"github.com/dan2756/todo-age/internal/scan"
)

// scanOptions says which files to scan and what to look for.
type scanOptions struct {
	dir     string   // work tree directory to run in
	paths   []string // pathspecs limiting the scan; empty means everything
	exclude *scan.Excluder
	maxSize int64 // skip larger files
	matcher *scan.Matcher
}

// collect scans the tracked files selected by opts and blames every line
// that holds a marker. Problems with single files are reported to warn and
// skipped.
func collect(opts scanOptions, now time.Time, warn io.Writer) ([]report.Row, error) {
	dir := opts.dir
	if err := git.CheckRepo(dir); err != nil {
		return nil, err
	}
	files, err := git.LsFiles(dir, opts.paths)
	if err != nil {
		return nil, err
	}
	var rows []report.Row
	for _, f := range files {
		if opts.exclude.Match(f) {
			continue
		}
		hits, err := scan.File(opts.matcher, dir, f, opts.maxSize)
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

func newRow(h scan.Hit, b git.Line, now time.Time) report.Row {
	r := report.Row{
		File:   h.File,
		Line:   h.Line,
		Tag:    h.Tag,
		Text:   h.Text,
		Owner:  h.Owner,
		Author: b.Author,
		Email:  b.AuthorMail,
		Time:   b.AuthorTime,
	}
	if b.Uncommitted() {
		// git reports the current time and a placeholder address.
		r.Uncommitted = true
		r.Email = ""
		r.Time = now
	} else {
		r.Commit = b.Commit[:min(len(b.Commit), 7)]
	}
	r.Date = r.Time.Format(time.DateOnly)
	r.Age = max(now.Sub(r.Time), 0)
	r.AgeDays = int(r.Age / (24 * time.Hour))
	return r
}
