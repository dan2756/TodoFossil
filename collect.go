package main

import (
	"fmt"
	"io"
	"runtime"
	"slices"
	"sync"
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
//
// Each file costs a git blame process, so files are handled by a pool of
// workers. Results are put back in file order, which keeps warnings and
// ties in the output deterministic.
func collect(opts scanOptions, now time.Time, warn io.Writer) ([]report.Row, error) {
	dir := opts.dir
	if err := git.CheckRepo(dir); err != nil {
		return nil, err
	}
	files, err := git.LsFiles(dir, opts.paths)
	if err != nil {
		return nil, err
	}
	files = slices.DeleteFunc(files, opts.exclude.Match)
	// Before the first commit there is nothing to blame against.
	hasHead := git.HasHead(dir)

	results := make([]fileResult, len(files))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(files)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				results[i] = collectFile(opts, files[i], hasHead, now)
			}
		}()
	}
	for i := range files {
		next <- i
	}
	close(next)
	wg.Wait()

	var rows []report.Row
	for _, res := range results {
		for _, w := range res.warnings {
			fmt.Fprintf(warn, "todo-age: warning: %s\n", w)
		}
		rows = append(rows, res.rows...)
	}
	return rows, nil
}

type fileResult struct {
	rows     []report.Row
	warnings []string
}

// collectFile scans and blames a single file.
func collectFile(opts scanOptions, f string, hasHead bool, now time.Time) (res fileResult) {
	hits, err := scan.File(opts.matcher, opts.dir, f, opts.maxSize)
	if err != nil {
		// Such as permission denied; the error names the file.
		res.warnings = append(res.warnings, err.Error())
		return res
	}
	if len(hits) == 0 {
		return res
	}
	if !hasHead {
		for _, h := range hits {
			res.rows = append(res.rows, newRow(h, git.NotCommitted, now))
		}
		return res
	}
	lines := make([]int, len(hits))
	for i, h := range hits {
		lines[i] = h.Line
	}
	blame, err := git.Blame(opts.dir, f, lines)
	if err != nil {
		res.warnings = append(res.warnings, fmt.Sprintf("%s: %v", f, err))
		return res
	}
	for _, h := range hits {
		b, ok := blame[h.Line]
		if !ok {
			res.warnings = append(res.warnings, fmt.Sprintf("%s:%d: no blame for line", f, h.Line))
			continue
		}
		res.rows = append(res.rows, newRow(h, b, now))
	}
	return res
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
