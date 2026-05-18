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
	"text/tabwriter"
	"time"

	"github.com/dan2756/todo-age/internal/scan"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `todo-age lists TODO, FIXME, HACK and XXX comments in the tracked files of
a git repository, oldest first, with the author and age of each line
according to git blame.

Usage:
  todo-age [flags]

Flags:
  -C dir       run as if todo-age was started in dir
  --version    print the version and exit
  -h, --help   show this help
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
	showVersion := fs.Bool("version", false, "")
	if err := fs.Parse(args); err != nil {
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
	if fs.NArg() > 0 {
		return usageError(stderr, fmt.Errorf("unexpected argument %q", fs.Arg(0)))
	}

	m, err := scan.NewMatcher(scan.DefaultTags)
	if err != nil {
		return usageError(stderr, err)
	}
	rows, err := collect(*dir, m, now, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "todo-age: %v\n", err)
		return 2
	}
	slices.SortStableFunc(rows, func(a, b row) int {
		if c := a.time.Compare(b.time); c != 0 {
			return c
		}
		if c := strings.Compare(a.File, b.File); c != 0 {
			return c
		}
		return a.Line - b.Line
	})
	if err := writeTable(stdout, rows); err != nil {
		fmt.Fprintf(stderr, "todo-age: %v\n", err)
		return 2
	}
	return 0
}

func usageError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "todo-age: %v\nRun 'todo-age --help' for usage.\n", err)
	return 2
}

func writeTable(w io.Writer, rows []row) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "AGE\tDATE\tAUTHOR\tTAG\tLOCATION\tTEXT")
	for _, r := range rows {
		tag := r.Tag
		if r.Owner != "" {
			tag += "(" + r.Owner + ")"
		}
		fmt.Fprintf(tw, "%dd\t%s\t%s\t%s\t%s:%d\t%s\n",
			r.AgeDays, r.Date, r.Author, tag, r.File, r.Line, strings.ReplaceAll(r.Text, "\t", " "))
	}
	return tw.Flush()
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
