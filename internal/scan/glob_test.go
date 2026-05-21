package scan

import "testing"

func TestExcluder(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"vendor/**", "vendor/a.go", true},
		{"vendor/**", "vendor/x/y/z.go", true},
		{"vendor/**", "src/vendor/a.go", false},
		{"vendor/**", "vendorized/a.go", false},
		{"*.min.js", "app.min.js", true},
		{"*.min.js", "static/js/app.min.js", true},
		{"*.min.js", "static/js/app.js", false},
		{"vendor", "vendor/a.go", true},
		{"vendor", "third_party/vendor/a.go", true},
		{"docs/", "docs/guide.md", true},
		{"./docs", "docs/guide.md", true},
		{"/build", "build/out.txt", true},
		{"**/testdata/**", "internal/scan/testdata/x.txt", true},
		{"**/testdata/**", "testdata/x.txt", true},
		{"**/testdata/**", "internal/testdata.go", false},
		{"src/*.go", "src/a.go", true},
		{"src/*.go", "src/sub/a.go", false},
		{"web/**/*.css", "web/a/b/site.css", true},
		{"web/**/*.css", "web/site.css", true},
		{"web/**/*.css", "web/site.js", false},
	}
	for _, tt := range tests {
		e, err := NewExcluder([]string{tt.pattern})
		if err != nil {
			t.Fatalf("NewExcluder(%q): %v", tt.pattern, err)
		}
		if got := e.Match(tt.name); got != tt.want {
			t.Errorf("pattern %q, path %q: got %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

func TestExcluderSeveralPatterns(t *testing.T) {
	e, err := NewExcluder([]string{"vendor/**", "", "*.min.js"})
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{"vendor/x.go": true, "a.min.js": true, "main.go": false} {
		if got := e.Match(name); got != want {
			t.Errorf("Match(%q) = %v, want %v", name, got, want)
		}
	}
	var none *Excluder
	if none.Match("anything") {
		t.Error("nil Excluder matched")
	}
}

func TestExcluderBadPattern(t *testing.T) {
	if _, err := NewExcluder([]string{"src/[a-"}); err == nil {
		t.Error("expected an error for a malformed pattern")
	}
}
