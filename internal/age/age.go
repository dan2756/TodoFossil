// Package age parses ages such as "90d" or "1y6mo".
package age

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Units of age. Months and years are fixed lengths, which is precise
// enough for deciding how stale a comment is.
const (
	Day   = 24 * time.Hour
	Week  = 7 * Day
	Month = 30 * Day
	Year  = 365 * Day
)

var units = map[string]time.Duration{
	"h":  time.Hour,
	"d":  Day,
	"w":  Week,
	"mo": Month,
	"y":  Year,
}

// maxAge is the largest age Parse accepts, well below where a Duration
// overflows.
const maxAge = 200 * Year

// Parse parses an age made of one or more number-unit pairs, such as "90d",
// "2w", "6mo", "1y" or "1y6mo". The units are h, d, w, mo (30 days) and
// y (365 days). A bare number is a number of days.
func Parse(s string) (time.Duration, error) {
	in := strings.ToLower(strings.ReplaceAll(s, " ", ""))
	if in == "" {
		return 0, errors.New("empty age")
	}
	if n, err := strconv.Atoi(in); err == nil {
		in = strconv.Itoa(n) + "d"
	}
	var total time.Duration
	for in != "" {
		i := 0
		for i < len(in) && '0' <= in[i] && in[i] <= '9' {
			i++
		}
		j := i
		for j < len(in) && 'a' <= in[j] && in[j] <= 'z' {
			j++
		}
		if i == 0 || i == j {
			return 0, fmt.Errorf("invalid age %q: want a number and a unit, like 90d, 2w, 6mo or 1y", s)
		}
		n, err := strconv.Atoi(in[:i])
		unit, ok := units[in[i:j]]
		if !ok {
			hint := ""
			if in[i:j] == "m" {
				hint = " (use mo for months)"
			}
			return 0, fmt.Errorf("invalid age %q: unknown unit %q%s", s, in[i:j], hint)
		}
		if err != nil || time.Duration(n) > (maxAge-total)/unit {
			return 0, fmt.Errorf("invalid age %q: too large", s)
		}
		total += time.Duration(n) * unit
		in = in[j:]
	}
	return total, nil
}
