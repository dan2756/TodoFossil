// Command todo-age lists TODO-style comments in a git repository together
// with the age and author of each line according to git blame.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/dan2756/todo-age/internal/report"
	"github.com/dan2756/todo-age/internal/scan"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `todo-age lists TODO, FIXME, HACK and XXX comments in the tracked files of
a git repository, oldest first, with the author and age of each line
according to git blame.

Usage:
  todo-age [flags] [path ...]

Paths limit the scan to tracked files under them.

Flags:
  -C dir            run as if todo-age was started in dir
  --format fmt      output format: table (default), json, csv or ndjson
  --tags list       markers to look for (default TODO,FIXME,HACK,XXX)
  --tag list        only show these markers; added to --tags if missing
  --author regex    only show lines whose author name or email matches
  --older-than age  only show markers older than age, e.g. 90d, 6mo, 1y
  --newer-than age  only show markers newer than age
  --sort key        age (oldest first, default), file or author
  --limit n         show at most n markers (default: all)
  --summary         print counts per tag and age bucket instead of markers
  --fail-older-than age
                    exit with status 1 if any marker is older than age
  --exclude list    skip files matching these globs, e.g. 'vendor/**,*.min.js'
  --max-size size   skip files larger than this, e.g. 512K or 4M (default 1M,
                    0 for no limit)
  --version         print the version and exit
  -h, --help        show this help

Lists are comma-separated, and list flags may be repeated. Ages are a
number and a unit: h, d, w, mo (30 days) or y (365 days); 1y6mo also works.

Exit status is 0 on success, 1 if --fail-older-than found an older marker,
and 2 on errors, such as not being inside a git repository.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now()))
}

// run is the whole program; it returns the process exit code.
func run(args []string, stdout, stderr io.Writer, now time.Time) int {
	fs := flag.NewFlagSet("todo-age", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	dir := fs.String("C", ".", "")
	format := fs.String("format", "table", "")
	tags := &listFlag{values: scan.DefaultTags}
	fs.Var(tags, "tags", "")
	var only listFlag
	fs.Var(&only, "tag", "")
	var exclude listFlag
	fs.Var(&exclude, "exclude", "")
	maxSize := sizeFlag(scan.DefaultMaxSize)
	fs.Var(&maxSize, "max-size", "")
	author := fs.String("author", "", "")
	var olderThan, newerThan ageFlag
	fs.Var(&olderThan, "older-than", "")
	fs.Var(&newerThan, "newer-than", "")
	sortKey := fs.String("sort", "age", "")
	limit := fs.Int("limit", 0, "")
	summary := fs.Bool("summary", false, "")
	var failOlder ageFlag
	fs.Var(&failOlder, "fail-older-than", "")
	showVersion := fs.Bool("version", false, "")
	paths, err := parseArgs(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return 0
		}
		return usageError(stderr, err)
	}
	if *showVersion {
		fmt.Fprintln(stdout, "todo-age", buildVersion())
		return 0
	}
	if !report.ValidFormat(*format) {
		return usageError(stderr, fmt.Errorf("unknown format %q (want %s)", *format, strings.Join(report.Formats, ", ")))
	}
	if !slices.Contains(report.SortKeys, *sortKey) {
		return usageError(stderr, fmt.Errorf("unknown sort key %q (want %s)", *sortKey, strings.Join(report.SortKeys, ", ")))
	}
	if *limit < 0 {
		return usageError(stderr, errors.New("--limit must not be negative"))
	}
	keep := filter{tags: only.values, olderThan: olderThan.d, newerThan: newerThan.d}
	if *author != "" {
		re, err := regexp.Compile(*author)
		if err != nil {
			return usageError(stderr, fmt.Errorf("bad --author pattern: %v", err))
		}
		keep.author = re
	}

	m, err := scan.NewMatcher(scanTags(tags.values, only.values))
	if err != nil {
		return usageError(stderr, err)
	}
	ex, err := scan.NewExcluder(exclude.values)
	if err != nil {
		return usageError(stderr, err)
	}
	opts := scanOptions{dir: *dir, paths: paths, exclude: ex, maxSize: int64(maxSize), matcher: m}
	rows, err := collect(opts, now, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "todo-age: %v\n", err)
		return 2
	}
	rows = keep.apply(rows)
	if err := report.Sort(rows, *sortKey); err != nil {
		return usageError(stderr, err)
	}
	// Count before --limit, so a short listing cannot hide an old marker.
	stale := 0
	if failOlder.d > 0 {
		for _, r := range rows {
			if r.Age > failOlder.d {
				stale++
			}
		}
	}
	if *summary {
		err = report.WriteSummary(stdout, *format, report.Summarize(rows))
	} else {
		if *limit > 0 && len(rows) > *limit {
			rows = rows[:*limit]
		}
		err = report.Write(stdout, *format, rows)
	}
	if err != nil {
		fmt.Fprintf(stderr, "todo-age: %v\n", err)
		return 2
	}
	if stale > 0 {
		fmt.Fprintf(stderr, "todo-age: %d %s older than %s\n", stale, plural(stale, "marker"), failOlder.s)
		return 1
	}
	return 0
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func usageError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "todo-age: %v\nRun 'todo-age --help' for usage.\n", err)
	return 2
}

// buildVersion prefers the version stamped by the linker, then the module
// version recorded by go install.
func buildVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}
