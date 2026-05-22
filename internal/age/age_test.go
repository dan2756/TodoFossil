package age

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := map[string]time.Duration{
		"90d":     90 * Day,
		"90":      90 * Day,
		"2w":      14 * Day,
		"6mo":     180 * Day,
		"1y":      365 * Day,
		"1y6mo":   545 * Day,
		"1y 2mo":  425 * Day,
		"12h":     12 * time.Hour,
		"3W":      21 * Day,
		"0d":      0,
		"1w2d12h": 9*Day + 12*time.Hour,
	}
	for in, want := range tests {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "d", "-3d", "3m", "1.5y", "2 fortnights", "999999999999y", "150y150y"} {
		if got, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %v; expected an error", in, got)
		}
	}
}
