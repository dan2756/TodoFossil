package scan

import "testing"

func TestFind(t *testing.T) {
	m, err := NewMatcher(DefaultTags)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		line  string
		want  Match
		found bool
	}{
		{"// TODO: fix this", Match{"TODO", "", "fix this"}, true},
		{"# FIXME(alice): handle nil", Match{"FIXME", "alice", "handle nil"}, true},
		{"/* HACK work around the vendor bug */", Match{"HACK", "", "work around the vendor bug"}, true},
		{"<!-- XXX: remove before release -->", Match{"XXX", "", "remove before release"}, true},
		{"<!-- TODO: send -->", Match{"TODO", "", "send"}, true},
		{"// TODO: send -->", Match{"TODO", "", "send -->"}, true},
		{"// TODO: write <!-- comment -->", Match{"TODO", "", "write <!-- comment -->"}, true},
		{"	x := f() // TODO(bob) no colon", Match{"TODO", "bob", "no colon"}, true},
		{"-- TODO - dash separator", Match{"TODO", "", "dash separator"}, true},
		{"; FIXME:no space", Match{"FIXME", "", "no space"}, true},
		{"// TODO( carol ): padded owner", Match{"TODO", "carol", "padded owner"}, true},
		{"// TODO(): empty owner", Match{"TODO", "", "empty owner"}, true},
		{"// TODO", Match{"TODO", "", ""}, true},
		{"/* TODO */", Match{"TODO", "", ""}, true},
		{"// TODO and FIXME on one line", Match{"TODO", "", "and FIXME on one line"}, true},
		{"tags := []string{\"TODO\", \"FIXME\"} // HACK: keep in sync", Match{"HACK", "", "keep in sync"}, true},
		{"// TODOS are not markers", Match{}, false},
		{"// todo: lowercase is not a marker", Match{}, false},
		{"var TODO_LIST = nil", Match{}, false},
		{"const mask = 0xXXX", Match{}, false},
		{"// ÄTODO: glued to a letter", Match{}, false},
		{"// TODOα: glued to a letter", Match{}, false},
		{"// TODO²: glued to a digit", Match{}, false},
		{"// TODO\u0301: combining mark", Match{}, false},
		{"// ÄTODO then TODO: real", Match{"TODO", "", "real"}, true},
		{"// café TODO: keep", Match{"TODO", "", "keep"}, true},
		{"// TODO: über", Match{"TODO", "", "über"}, true},
		{"tags := []string{\"TODO\", 'FIXME', `XXX`}", Match{}, false},
		{"// see “TODO” here", Match{}, false},
		{"// see ‘FIXME’ here", Match{}, false},
		{"// see «XXX» here", Match{}, false},
		{"// “TODO” then TODO: real", Match{"TODO", "", "real"}, true},
		{"// “TODO: inside quotes”", Match{"TODO", "", "inside quotes”"}, true},
		{"", Match{}, false},
	}
	for _, tt := range tests {
		got, found := m.Find(tt.line)
		if found != tt.found || got != tt.want {
			t.Errorf("Find(%q) = %+v, %v; want %+v, %v", tt.line, got, found, tt.want, tt.found)
		}
	}
}

func TestCustomTags(t *testing.T) {
	m, err := NewMatcher([]string{"BUG", "NOTE"})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := m.Find("// BUG(dan): off by one"); !ok || got != (Match{"BUG", "dan", "off by one"}) {
		t.Errorf("BUG marker: got %+v, %v", got, ok)
	}
	if _, ok := m.Find("// TODO: not configured"); ok {
		t.Error("TODO matched although only BUG and NOTE were configured")
	}
}

func TestNewMatcherErrors(t *testing.T) {
	for _, tags := range [][]string{nil, {"TO DO"}, {"TODO", ""}, {"@todo"}} {
		if _, err := NewMatcher(tags); err == nil {
			t.Errorf("NewMatcher(%q): expected an error", tags)
		}
	}
}
