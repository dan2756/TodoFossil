package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/a.go", "package a\r\n\r\n// TODO: crlf line\r\nfunc f() {} // FIXME(x): two\r\n")
	writeFile(t, dir, "blob.bin", "\x00\x01// TODO: inside a binary\n")
	writeFile(t, dir, "none.txt", "nothing to see\n")

	m, _ := NewMatcher(DefaultTags)
	hits, err := File(m, dir, "src/a.go", DefaultMaxSize)
	if err != nil {
		t.Fatal(err)
	}
	want := []Hit{
		{File: "src/a.go", Line: 3, Match: Match{"TODO", "", "crlf line"}},
		{File: "src/a.go", Line: 4, Match: Match{"FIXME", "x", "two"}},
	}
	if len(hits) != len(want) {
		t.Fatalf("got %d hits, want %d: %+v", len(hits), len(want), hits)
	}
	for i := range want {
		if hits[i] != want[i] {
			t.Errorf("hit %d = %+v, want %+v", i, hits[i], want[i])
		}
	}

	for _, name := range []string{"blob.bin", "none.txt"} {
		hits, err := File(m, dir, name, DefaultMaxSize)
		if err != nil || len(hits) != 0 {
			t.Errorf("File(%s) = %v, %v; want no hits", name, hits, err)
		}
	}

	hits, err = File(m, dir, "src/a.go", 10)
	if err != nil || len(hits) != 0 {
		t.Errorf("file over max size: got %v, %v; want no hits", hits, err)
	}
}

func TestFileSkipsNonRegular(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real.go", "// TODO: real\n")
	writeFile(t, dir, "sub/inner.go", "// TODO: inner\n")
	if err := os.Symlink("real.go", filepath.Join(dir, "link.go")); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	m, _ := NewMatcher(DefaultTags)
	for _, name := range []string{"link.go", "sub", "deleted.go"} {
		hits, err := File(m, dir, name, DefaultMaxSize)
		if err != nil || len(hits) != 0 {
			t.Errorf("File(%s) = %v, %v; want no hits and no error", name, hits, err)
		}
	}
}
