package main

import (
	"flag"
	"strings"
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
