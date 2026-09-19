package main

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args      []string
		wantPos   []string
		wantTag   []string
		wantFmt   string
		wantError bool
	}{
		{[]string{}, nil, nil, "table", false},
		{[]string{"src/", "--tag", "FIXME", "cmd/"}, []string{"src/", "cmd/"}, []string{"FIXME"}, "table", false},
		{[]string{"--format=json", "--tag", "TODO,HACK", "--tag", "XXX"}, nil, []string{"TODO", "HACK", "XXX"}, "json", false},
		{[]string{"a", "--", "--format", "b"}, []string{"a", "--format", "b"}, nil, "table", false},
		{[]string{"--nope"}, nil, nil, "", true},
	}
	for _, tt := range tests {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		format := fs.String("format", "table", "")
		var tag listFlag
		fs.Var(&tag, "tag", "")
		pos, err := parseArgs(fs, tt.args)
		if tt.wantError {
			if err == nil {
				t.Errorf("parseArgs(%q): expected an error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseArgs(%q): %v", tt.args, err)
			continue
		}
		if !reflect.DeepEqual(pos, tt.wantPos) || !reflect.DeepEqual(tag.values, tt.wantTag) || *format != tt.wantFmt {
			t.Errorf("parseArgs(%q) = %q, tag %q, format %q", tt.args, pos, tag.values, *format)
		}
	}
}

func TestListFlagReplacesDefault(t *testing.T) {
	defaults := []string{"TODO", "FIXME"}
	l := &listFlag{values: defaults}
	if err := l.Set("BUG, NOTE,"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"BUG", "NOTE"}; !reflect.DeepEqual(l.values, want) {
		t.Errorf("values = %q, want %q", l.values, want)
	}
	if defaults[0] != "TODO" {
		t.Error("Set modified the default slice")
	}
}

func TestScanTags(t *testing.T) {
	got := scanTags([]string{"TODO", "FIXME"}, []string{"fixme", "BUG"})
	if want := []string{"TODO", "FIXME", "BUG"}; !reflect.DeepEqual(got, want) {
		t.Errorf("scanTags = %q, want %q", got, want)
	}
	// Tags are matched case-sensitively, so --tag bug must look for BUG.
	got = scanTags([]string{"TODO"}, []string{"bug", "Bug"})
	if want := []string{"TODO", "BUG"}; !reflect.DeepEqual(got, want) {
		t.Errorf("scanTags with lowercase --tag = %q, want %q", got, want)
	}
	got = scanTags([]string{"note"}, []string{"NOTE"})
	if want := []string{"note"}; !reflect.DeepEqual(got, want) {
		t.Errorf("scanTags with a tag already in --tags = %q, want %q", got, want)
	}
}

func TestSizeFlag(t *testing.T) {
	tests := map[string]int64{
		"0":       0,
		"1048576": 1 << 20,
		"512K":    512 << 10,
		"2m":      2 << 20,
		"1MiB":    1 << 20,
		"3 GB":    3 << 30,
		"100b":    100,
	}
	for in, want := range tests {
		var s sizeFlag
		if err := s.Set(in); err != nil || int64(s) != want {
			t.Errorf("Set(%q) = %d, %v; want %d", in, s, err, want)
		}
	}
	for _, in := range []string{"", "big", "-1", "1.5M", "2T", "10iB", "99999999999999999999"} {
		var s sizeFlag
		if err := s.Set(in); err == nil {
			t.Errorf("Set(%q): expected an error", in)
		}
	}
}
