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

func TestHuman(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "today"},
		{23 * time.Hour, "today"},
		{Day, "1d"},
		{13 * Day, "13d"},
		{14 * Day, "2w"},
		{21 * Day, "3w"},
		{59 * Day, "8w"},
		{60 * Day, "2mo"},
		{200 * Day, "6mo"},
		{359 * Day, "11mo"},
		{360 * Day, "11mo"},
		{364 * Day, "11mo"},
		{365 * Day, "1y"},
		{729 * Day, "1y 11mo"},
		{730 * Day, "2y"},
		{425 * Day, "1y 2mo"},
		{3*Year + 100*Day, "3y 3mo"},
	}
	for _, tt := range tests {
		if got := Human(tt.in); got != tt.want {
			t.Errorf("Human(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
