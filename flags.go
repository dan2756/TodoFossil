package main

import (
	"errors"
	"flag"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dan2756/todo-age/internal/age"
)

// listFlag is a comma-separated list that may also be given more than once.
// The first use replaces the default value.
type listFlag struct {
	values []string
	set    bool
}

func (l *listFlag) String() string { return strings.Join(l.values, ",") }

func (l *listFlag) Set(s string) error {
	if !l.set {
		l.values, l.set = nil, true
	}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			l.values = append(l.values, v)
		}
	}
	return nil
}

// parseArgs parses flags that may appear before, between or after
// positional arguments, which it returns. Everything after "--" is
// positional.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		if used := len(args) - len(rest); used > 0 && args[used-1] == "--" {
			return append(pos, rest...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// sizeFlag is a size in bytes, written as a number with an optional unit:
// 500K, 2M, 1MiB, 1GB. Units are powers of 1024.
type sizeFlag int64

var sizeRE = regexp.MustCompile(`^(\d+)\s*([kmg]?)(i?b)?$`)

func (s *sizeFlag) String() string { return strconv.FormatInt(int64(*s), 10) }

func (s *sizeFlag) Set(v string) error {
	m := sizeRE.FindStringSubmatch(strings.ToLower(strings.TrimSpace(v)))
	if m == nil || (m[2] == "" && m[3] == "ib") {
		return errors.New("want a size such as 500K, 2M or 1048576")
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return errors.New("size out of range")
	}
	shift := map[string]uint{"": 0, "k": 10, "m": 20, "g": 30}[m[2]]
	if n > (1<<62)>>shift {
		return errors.New("size out of range")
	}
	*s = sizeFlag(n << shift)
	return nil
}

// ageFlag is an age such as 90d or 6mo. Zero means the flag was not given.
type ageFlag struct {
	d time.Duration
	s string
}

func (a *ageFlag) String() string { return a.s }

func (a *ageFlag) Set(v string) error {
	d, err := age.Parse(v)
	if err != nil {
		return err
	}
	if d <= 0 {
		return errors.New("age must be greater than zero")
	}
	a.d, a.s = d, v
	return nil
}
