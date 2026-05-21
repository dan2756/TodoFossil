// Command todo-age lists TODO-style comments in a git repository together
// with the age and author of each line according to git blame.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
  --exclude list    skip files matching these globs, e.g. 'vendor/**,*.min.js'
  --max-size size   skip files larger than this, e.g. 512K or 4M (default 1M,
                    0 for no limit)
  --version         print the version and exit
  -h, --help        show this help

Lists are comma-separated, and list flags may be repeated.
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
	rows = filter{tags: only.values}.apply(rows)
	slices.SortStableFunc(rows, func(a, b report.Row) int {
		if c := a.Time.Compare(b.Time); c != 0 {
			return c
		}
		if c := strings.Compare(a.File, b.File); c != 0 {
			return c
		}
		return a.Line - b.Line
	})
	if err := report.Write(stdout, *format, rows); err != nil {
		fmt.Fprintf(stderr, "todo-age: %v\n", err)
		return 2
	}
	return 0
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
