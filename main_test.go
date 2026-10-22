package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dan2756/todo-age/internal/report"
)

// testNow is the current time as far as the end-to-end tests are concerned.
var testNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	// Keep the machine's git configuration (signing, hooks, aliases) out of
	// the throwaway repositories.
	os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Exit(m.Run())
}

type testRepo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *testRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	r := &testRepo{t: t, dir: t.TempDir()}
	r.git(nil, "init", "-q", "-b", "main")
	r.git(nil, "config", "user.name", "Test User")
	r.git(nil, "config", "user.email", "test@example.com")
	r.git(nil, "config", "commit.gpgsign", "false")
	return r
}

func (r *testRepo) git(env []string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *testRepo) write(name, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// commit commits all changes with a backdated author and committer date.
func (r *testRepo) commit(date, author, email, msg string) {
	r.t.Helper()
	r.git(nil, "add", "-A")
	r.git([]string{
		"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date,
		"GIT_AUTHOR_NAME=" + author, "GIT_AUTHOR_EMAIL=" + email,
	}, "commit", "-q", "-m", msg)
}

func (r *testRepo) run(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(append([]string{"-C", r.dir}, args...), &out, &errOut, testNow)
	return code, out.String(), errOut.String()
}

func (r *testRepo) rows(args ...string) []report.Row {
	r.t.Helper()
	code, out, errOut := r.run(append(args, "--format", "json")...)
	if code != 0 {
		r.t.Fatalf("todo-age %q: exit %d\n%s", args, code, errOut)
	}
	var rows []report.Row
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		r.t.Fatalf("todo-age %q: %v\n%s", args, err, out)
	}
	return rows
}

// historyRepo builds a repository whose markers were written by two people
// at known dates, plus some local changes.
func historyRepo(t *testing.T) *testRepo {
	r := newRepo(t)
	r.write("server/handler.go", "package server\n\n// TODO(maria): validate the redirect target\nfunc handle() {}\n")
	r.commit("2024-12-01T10:00:00+0000", "Maria Lopez", "maria@example.com", "add handler")

	r.write("store/cache.go", "package store\n\n// FIXME: eviction ignores TTL\n// HACK work around the old client\n")
	r.write("vendor/lib/lib.go", "package lib // TODO: vendored code\n")
	r.write("web/app.min.js", "/* TODO minified */\n")
	r.commit("2025-10-01T09:30:00+0200", "Sam Okafor", "sam@example.com", "add cache")

	r.write("server/handler.go", "package server\n\n// TODO(maria): validate the redirect target\nfunc handle() {}\n\n// XXX: drop after the v2 cutover\n")
	r.commit("2025-12-20T08:00:00+0000", "Maria Lopez", "maria@example.com", "mark cutover")

	// Tracked but not committed: a local edit and a staged new file.
	r.write("store/cache.go", "package store\n\n// FIXME: eviction ignores TTL\n// HACK work around the old client\n// TODO: not committed yet\n")
	r.write("notes/plan.md", "TODO: staged only\n")
	r.git(nil, "add", "notes/plan.md")
	// Untracked files are not scanned.
	r.write("scratch.go", "// TODO: untracked\n")
	return r
}

func loc(r report.Row) string { return fmt.Sprintf("%s:%d", r.File, r.Line) }

func TestEndToEnd(t *testing.T) {
	r := historyRepo(t)
	rows := r.rows()

	want := []struct {
		loc, tag, owner, text, author, email, date string
		ageDays                                    int
		uncommitted                                bool
	}{
		{"server/handler.go:3", "TODO", "maria", "validate the redirect target", "Maria Lopez", "maria@example.com", "2024-12-01", 396, false},
		{"store/cache.go:3", "FIXME", "", "eviction ignores TTL", "Sam Okafor", "sam@example.com", "2025-10-01", 92, false},
		{"store/cache.go:4", "HACK", "", "work around the old client", "Sam Okafor", "sam@example.com", "2025-10-01", 92, false},
		{"vendor/lib/lib.go:1", "TODO", "", "vendored code", "Sam Okafor", "sam@example.com", "2025-10-01", 92, false},
		{"web/app.min.js:1", "TODO", "", "minified", "Sam Okafor", "sam@example.com", "2025-10-01", 92, false},
		{"server/handler.go:6", "XXX", "", "drop after the v2 cutover", "Maria Lopez", "maria@example.com", "2025-12-20", 12, false},
		{"notes/plan.md:1", "TODO", "", "staged only", "Not Committed Yet", "", "2026-01-01", 0, true},
		{"store/cache.go:5", "TODO", "", "not committed yet", "Not Committed Yet", "", "2026-01-01", 0, true},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		g := rows[i]
		if loc(g) != w.loc || g.Tag != w.tag || g.Owner != w.owner || g.Text != w.text ||
			g.Author != w.author || g.Email != w.email || g.Date != w.date ||
			g.AgeDays != w.ageDays || g.Uncommitted != w.uncommitted {
			t.Errorf("row %d = %+v\nwant %+v", i, g, w)
		}
		if w.uncommitted && g.Commit != "" || !w.uncommitted && len(g.Commit) != 7 {
			t.Errorf("row %d: commit %q", i, g.Commit)
		}
	}
}

func TestAgeUsesAuthorDate(t *testing.T) {
	r := newRepo(t)
	r.write("a.go", "// TODO: original\n")
	r.git(nil, "add", "-A")
	r.git([]string{
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00+0000",
		"GIT_COMMITTER_DATE=2024-06-01T00:00:00+0000",
		"GIT_AUTHOR_NAME=Original Author",
		"GIT_AUTHOR_EMAIL=orig@example.com",
		"GIT_COMMITTER_NAME=Later Committer",
		"GIT_COMMITTER_EMAIL=later@example.com",
	}, "commit", "-q", "-m", "add")

	rows := r.rows()
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	got := rows[0]
	authorAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	committerAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	wantDays := int(testNow.Sub(authorAt) / (24 * time.Hour))
	committerDays := int(testNow.Sub(committerAt) / (24 * time.Hour))
	if wantDays == committerDays {
		t.Fatal("fixture dates are the same number of days")
	}
	if got.Author != "Original Author" || got.Email != "orig@example.com" ||
		got.Date != "2020-01-01" || got.AgeDays != wantDays {
		t.Fatalf("row = %+v; want author date %s (%d days, not committer %d)", got, authorAt.Format(time.DateOnly), wantDays, committerDays)
	}
}

func TestEndToEndFilters(t *testing.T) {
	r := historyRepo(t)
	tests := []struct {
		args []string
		want []string
	}{
		{[]string{"--older-than", "90d"}, []string{"server/handler.go:3", "store/cache.go:3", "store/cache.go:4", "vendor/lib/lib.go:1", "web/app.min.js:1"}},
		{[]string{"--older-than", "90d", "--exclude", "vendor/**,*.min.js"}, []string{"server/handler.go:3", "store/cache.go:3", "store/cache.go:4"}},
		{[]string{"--newer-than", "1mo", "--tag", "XXX,todo"}, []string{"server/handler.go:6", "notes/plan.md:1", "store/cache.go:5"}},
		{[]string{"--author", "^Sam", "--tag", "FIXME"}, []string{"store/cache.go:3"}},
		{[]string{"server/", "notes"}, []string{"server/handler.go:3", "server/handler.go:6", "notes/plan.md:1"}},
		{[]string{"--sort", "file", "--limit", "2"}, []string{"notes/plan.md:1", "server/handler.go:3"}},
		{[]string{"--tags", "HACK"}, []string{"store/cache.go:4"}},
	}
	for _, tt := range tests {
		var got []string
		for _, row := range r.rows(tt.args...) {
			got = append(got, loc(row))
		}
		if strings.Join(got, " ") != strings.Join(tt.want, " ") {
			t.Errorf("todo-age %q:\n got %v\nwant %v", tt.args, got, tt.want)
		}
	}
}

func TestEndToEndTable(t *testing.T) {
	r := historyRepo(t)
	code, out, errOut := r.run("--tag", "TODO", "--limit", "1")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	want := "AGE     DATE        AUTHOR       TAG          LOCATION             TEXT\n" +
		"1y 1mo  2024-12-01  Maria Lopez  TODO(maria)  server/handler.go:3  validate the redirect target\n"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestEndToEndSummary(t *testing.T) {
	r := historyRepo(t)
	code, out, errOut := r.run("--summary", "--format", "json", "--limit", "1")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var s report.Summary
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	// --limit does not apply to the summary.
	if s.Total != 8 || s.ByTag["TODO"] != 5 || s.ByTag["FIXME"] != 1 {
		t.Errorf("summary = %+v", s)
	}
	counts := map[string]int{}
	for _, b := range s.ByAge {
		counts[b.Bucket] = b.Count
	}
	if counts["<1w"] != 2 || counts["1w-1mo"] != 1 || counts["3mo-6mo"] != 4 || counts["1y-2y"] != 1 {
		t.Errorf("age buckets = %v", s.ByAge)
	}
}

func TestFailOlderThan(t *testing.T) {
	r := historyRepo(t)
	tests := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"--fail-older-than", "1y"}, 1, "1 marker older than 1y"},
		{[]string{"--fail-older-than", "60d", "--limit", "1"}, 1, "5 markers older than 60d"},
		{[]string{"--fail-older-than", "2y"}, 0, ""},
		{[]string{"--fail-older-than", "1y", "--exclude", "server"}, 0, ""},
	}
	for _, tt := range tests {
		code, _, errOut := r.run(tt.args...)
		if code != tt.code || !strings.Contains(errOut, tt.msg) {
			t.Errorf("todo-age %q: exit %d, stderr %q; want exit %d, %q", tt.args, code, errOut, tt.code, tt.msg)
		}
	}
}

func TestManyFiles(t *testing.T) {
	r := newRepo(t)
	const n = 60
	for i := range n {
		r.write(fmt.Sprintf("pkg%02d/f.go", i), fmt.Sprintf("package p\n\n// TODO: item %d\n// FIXME: other %d\n", i, i))
	}
	r.commit("2025-03-01T00:00:00+0000", "Test User", "test@example.com", "many")
	rows := r.rows("--sort", "age")
	if len(rows) != 2*n {
		t.Fatalf("got %d rows, want %d", len(rows), 2*n)
	}
	// Same age everywhere, so the order falls back to file and line.
	for i, row := range rows {
		if want := fmt.Sprintf("pkg%02d/f.go:%d", i/2, 3+i%2); loc(row) != want {
			t.Fatalf("row %d is %s, want %s", i, loc(row), want)
		}
	}
}

func TestNoCommitsYet(t *testing.T) {
	r := newRepo(t)
	r.write("main.go", "package main\n\n// TODO: write main\n")
	r.git(nil, "add", "main.go")
	code, out, errOut := r.run("--format", "csv")
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	want := "file,line,tag,text,owner,author,email,commit,date,age_days,uncommitted\n" +
		"main.go,3,TODO,write main,,Not Committed Yet,,,2026-01-01,0,true\n"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestOddTrackedFiles(t *testing.T) {
	r := newRepo(t)
	r.write("a.go", "// TODO: real one\n")
	r.write("gone.go", "// TODO: deleted locally\n")
	r.write("dir/b.go", "// FIXME: in a directory\n")
	for link, target := range map[string]string{"link.go": "a.go", "dirlink": "dir"} {
		if err := os.Symlink(target, filepath.Join(r.dir, link)); err != nil {
			t.Skip("symlinks not supported:", err)
		}
	}
	r.commit("2025-07-01T00:00:00+0000", "Test User", "test@example.com", "init")
	// A submodule is a gitlink entry whose work tree is a directory.
	head := r.git(nil, "rev-parse", "HEAD")
	r.git(nil, "update-index", "--add", "--cacheinfo", "160000,"+head+",sub")
	if err := os.Mkdir(filepath.Join(r.dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(r.dir, "gone.go")); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := r.run("--format", "csv", "--sort", "file")
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "a.go,1,TODO,") || !strings.HasPrefix(lines[2], "dir/b.go,1,FIXME,") {
		t.Errorf("got:\n%s", out)
	}
}

func TestUnreadableFiles(t *testing.T) {
	r := newRepo(t)
	r.write("a.go", "// TODO: readable\n")
	r.write("dir/x.go", "// TODO: directory became a file\n")
	r.write("loop/y.go", "// TODO: behind a symlink loop\n")
	r.commit("2025-07-01T00:00:00+0000", "Test User", "test@example.com", "init")
	// git status lists dir/x.go as deleted.
	if err := os.RemoveAll(filepath.Join(r.dir, "dir")); err != nil {
		t.Fatal(err)
	}
	r.write("dir", "now a file\n")
	// loop/y.go cannot be opened, not even by root.
	if err := os.RemoveAll(filepath.Join(r.dir, "loop")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("loop", filepath.Join(r.dir, "loop")); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	code, out, errOut := r.run("--format", "csv")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "a.go,1,TODO,") {
		t.Errorf("got:\n%s", out)
	}
	if !strings.Contains(errOut, "warning:") || !strings.Contains(errOut, "y.go") || strings.Contains(errOut, "x.go") {
		t.Errorf("stderr %q; want a warning about loop/y.go only", errOut)
	}
}

func TestManyMarkerRuns(t *testing.T) {
	r := newRepo(t)
	// As many -L ranges as separate marked lines would not fit on a command
	// line; the file is still under the default --max-size.
	const n = 90000
	r.write("gen.txt", strings.Repeat("// TODO x\n\n", n))
	r.commit("2020-01-01T00:00:00+0000", "Test User", "test@example.com", "generated")
	r.write("gen.txt", strings.Repeat("// TODO x\n\n", n)+"// TODO new\n")
	code, out, errOut := r.run("--summary", "--format", "json", "--fail-older-than", "1y")
	if code != 1 || !strings.Contains(errOut, fmt.Sprintf("%d markers older than 1y", n)) || strings.Contains(errOut, "warning") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	var s report.Summary
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	if s.Total != n+1 {
		t.Errorf("total = %d, want %d", s.Total, n+1)
	}
}

func TestErrors(t *testing.T) {
	notRepo := t.TempDir()
	// Do not let git find a repository above the temp dir.
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(notRepo))
	var out, errOut bytes.Buffer
	if code := run([]string{"-C", notRepo}, &out, &errOut, testNow); code != 2 || !strings.Contains(errOut.String(), "not a git repository") {
		t.Errorf("outside a repository: exit %d, stderr %q", code, errOut.String())
	}

	r := newRepo(t)
	r.write("a.go", "// TODO: x\n")
	r.commit("2025-06-01T00:00:00+0000", "Test User", "test@example.com", "init")
	for _, args := range [][]string{
		{"--format", "xml"},
		{"--older-than", "soon"},
		{"--author", "("},
		{"--tags", "TO DO"},
		{"missing/"},
		{"--bogus"},
	} {
		if code, _, _ := r.run(args...); code != 2 {
			t.Errorf("todo-age %q: exit %d, want 2", args, code)
		}
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "--version"} {
		var out, errOut bytes.Buffer
		if code := run([]string{arg}, &out, &errOut, testNow); code != 0 || out.Len() == 0 {
			t.Errorf("%s: exit %d, stdout %q", arg, code, out.String())
		}
	}
}
