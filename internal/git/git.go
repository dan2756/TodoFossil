// Package git wraps the few git commands todo-age needs and parses the
// porcelain output of git blame.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// ErrNotRepo is returned by CheckRepo when the directory is not inside a
// git work tree.
var ErrNotRepo = errors.New("not a git repository (or any of the parent directories)")

// Run runs git with args in dir and returns its standard output. When git
// fails, the returned error carries the first line of its standard error.
func Run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		msg = strings.TrimPrefix(msg, "fatal: ")
		msg = strings.TrimPrefix(msg, "error: ")
		if msg == "" {
			return nil, fmt.Errorf("git %s: %w", args[0], err)
		}
		return nil, fmt.Errorf("git %s: %s", args[0], msg)
	}
	return out, nil
}

// CheckRepo reports whether dir is inside a git work tree.
func CheckRepo(dir string) error {
	out, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return ErrNotRepo
		}
		return err
	}
	if strings.TrimSpace(string(out)) != "true" {
		return errors.New("not inside a git work tree")
	}
	return nil
}

// HasHead reports whether the repository in dir has a commit checked out.
// It does not in a freshly initialized repository.
func HasHead(dir string) bool {
	_, err := Run(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// LsFiles returns the tracked files under dir, relative to dir. When paths
// are given, only files matching those pathspecs are returned, and a
// pathspec that matches nothing is an error.
func LsFiles(dir string, paths []string) ([]string, error) {
	args := []string{"ls-files", "-z"}
	if len(paths) > 0 {
		args = append(args, "--error-unmatch")
	}
	args = append(args, "--")
	args = append(args, paths...)
	out, err := Run(dir, args...)
	if err != nil {
		return nil, err
	}
	var files []string
	seen := make(map[string]bool)
	for _, f := range strings.Split(string(out), "\x00") {
		// Unmerged paths are listed once per stage.
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		files = append(files, f)
	}
	return files, nil
}

// maxRanges is the most -L ranges passed to one git blame. Past it the
// whole file is blamed instead, as a command line with tens of thousands
// of ranges exceeds the system's limit on argument length.
const maxRanges = 1000

// Blame runs git blame --porcelain on path for the given line numbers,
// passing one -L range per run of consecutive lines so that a file is
// blamed once no matter how many lines are asked for. The result may also
// hold lines that were not asked for.
func Blame(dir, path string, lines []int) (map[int]Line, error) {
	args := []string{"blame", "--porcelain"}
	if ranges := Ranges(lines); len(ranges) <= maxRanges {
		for _, r := range ranges {
			args = append(args, "-L", strconv.Itoa(r[0])+","+strconv.Itoa(r[1]))
		}
	}
	args = append(args, "--", path)
	out, err := Run(dir, args...)
	if err != nil {
		return nil, err
	}
	return ParsePorcelain(bytes.NewReader(out))
}

// Ranges sorts and de-duplicates line numbers and merges consecutive ones
// into inclusive [start, end] ranges.
func Ranges(lines []int) [][2]int {
	sorted := slices.Clone(lines)
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)
	var out [][2]int
	for _, n := range sorted {
		if k := len(out) - 1; k >= 0 && out[k][1]+1 >= n {
			out[k][1] = n
			continue
		}
		out = append(out, [2]int{n, n})
	}
	return out
}
