package scan

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// DefaultMaxSize is the size in bytes above which files are not scanned.
const DefaultMaxSize = 1 << 20

// binarySniffLen is how much of a file is checked for NUL bytes, the same
// heuristic git uses to decide whether a file is binary.
const binarySniffLen = 8000

// Hit is a marker found in a file.
type Hit struct {
	File string // path as given to File
	Line int    // 1-based line number
	Match
}

// File scans the file name, relative to root, for markers. Binary files and
// files larger than maxSize bytes yield no hits; a maxSize of 0 means no
// limit. So do files that are missing (deleted but still tracked, also when
// a directory on their path has been replaced by a file) and anything that
// is not a regular file: symlinks, whose blame is that of the link rather
// than its target, and submodule directories.
func File(m *Matcher, root, name string, maxSize int64) ([]Hit, error) {
	p := filepath.Join(root, filepath.FromSlash(name))
	fi, err := os.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() || maxSize > 0 && fi.Size() > maxSize {
		return nil, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	if isBinary(data) || !m.mayContain(data) {
		return nil, nil
	}
	var hits []Hit
	n := 0
	for line := range bytes.Lines(data) {
		n++
		match, ok := m.Find(strings.TrimRight(string(line), "\r\n"))
		if ok {
			hits = append(hits, Hit{File: name, Line: n, Match: match})
		}
	}
	return hits, nil
}

func isBinary(data []byte) bool {
	return bytes.IndexByte(data[:min(len(data), binarySniffLen)], 0) >= 0
}
