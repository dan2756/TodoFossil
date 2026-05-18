package git

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Line is the blame information for a single line of a file.
type Line struct {
	Commit     string    // full object name; all zeros if not committed yet
	Author     string    // author name
	AuthorMail string    // author email, without angle brackets
	AuthorTime time.Time // author date, in the author's time zone
}

// Uncommitted reports whether the line has local changes that are not
// committed yet. git blame reports such lines with an all-zero commit.
func (l Line) Uncommitted() bool {
	return strings.Trim(l.Commit, "0") == ""
}

// commitInfo collects the headers of one commit. git blame --porcelain
// prints them only the first time a commit appears, so later lines share
// the same commitInfo.
type commitInfo struct {
	hash   string
	author string
	mail   string
	time   int64
	tz     string
}

// ParsePorcelain parses the output of git blame --porcelain and returns the
// blame information keyed by line number in the final file.
func ParsePorcelain(r io.Reader) (map[int]Line, error) {
	br := bufio.NewReader(r)
	commits := make(map[string]*commitInfo)
	byLine := make(map[int]*commitInfo)
	var cur *commitInfo
	var final int
	for n := 1; ; n++ {
		s, err := br.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if s == "" && err != nil {
			break
		}
		s = strings.TrimSuffix(s, "\n")
		switch {
		case strings.HasPrefix(s, "\t"):
			// The line's content ends its entry.
			if cur == nil {
				return nil, fmt.Errorf("blame output line %d: content without header", n)
			}
			byLine[final] = cur
			cur = nil
		case cur == nil:
			hash, line, ok := parseHeader(s)
			if !ok {
				return nil, fmt.Errorf("blame output line %d: unexpected %q", n, s)
			}
			if commits[hash] == nil {
				commits[hash] = &commitInfo{hash: hash}
			}
			cur, final = commits[hash], line
		default:
			key, val, _ := strings.Cut(s, " ")
			switch key {
			case "author":
				cur.author = val
			case "author-mail":
				cur.mail = strings.TrimSuffix(strings.TrimPrefix(val, "<"), ">")
			case "author-time":
				t, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("blame output line %d: bad author-time %q", n, val)
				}
				cur.time = t
			case "author-tz":
				cur.tz = val
			}
		}
		if err != nil {
			break
		}
	}
	if cur != nil {
		return nil, errors.New("blame output ends in the middle of an entry")
	}
	lines := make(map[int]Line, len(byLine))
	for n, c := range byLine {
		lines[n] = Line{
			Commit:     c.hash,
			Author:     c.author,
			AuthorMail: c.mail,
			AuthorTime: time.Unix(c.time, 0).In(zone(c.tz)),
		}
	}
	return lines, nil
}

// parseHeader parses an entry header of the form
// "<hash> <orig-line> <final-line> [<group-size>]".
func parseHeader(s string) (hash string, final int, ok bool) {
	f := strings.Fields(s)
	if len(f) != 3 && len(f) != 4 {
		return "", 0, false
	}
	if !isHash(f[0]) {
		return "", 0, false
	}
	final, err := strconv.Atoi(f[2])
	if err != nil || final < 1 {
		return "", 0, false
	}
	return f[0], final, true
}

func isHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

// zone turns a git time zone offset such as "+0130" into a location.
func zone(tz string) *time.Location {
	if len(tz) != 5 || (tz[0] != '+' && tz[0] != '-') {
		return time.UTC
	}
	hh, err1 := strconv.Atoi(tz[1:3])
	mm, err2 := strconv.Atoi(tz[3:5])
	if err1 != nil || err2 != nil {
		return time.UTC
	}
	secs := hh*3600 + mm*60
	if tz[0] == '-' {
		secs = -secs
	}
	return time.FixedZone(tz, secs)
}
