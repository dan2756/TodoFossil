package report

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dan2756/todo-age/internal/age"
)

var sample = []Row{
	{
		File: "src/a.go", Line: 12, Tag: "TODO", Text: `say "hi", then leave`,
		Owner: "ana", Author: "Ana, Jr.", Email: "ana@example.com",
		Commit: "1a2b3c4", Date: "2025-03-01", AgeDays: 400, Age: 400 * age.Day,
	},
	{
		File: "b c.txt", Line: 1, Tag: "FIXME", Text: " leading space",
		Author: "Not Committed Yet", Date: "2026-04-05", Uncommitted: true,
	},
}

func TestCSVQuoting(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, "csv", sample); err != nil {
		t.Fatal(err)
	}
	want := `file,line,tag,text,owner,author,email,commit,date,age_days,uncommitted
src/a.go,12,TODO,"say ""hi"", then leave",ana,"Ana, Jr.",ana@example.com,1a2b3c4,2025-03-01,400,false
b c.txt,1,FIXME," leading space",,Not Committed Yet,,,2026-04-05,0,true
`
	if got := buf.String(); got != want {
		t.Errorf("csv output:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, "json", sample); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	if got[0]["text"] != `say "hi", then leave` || got[0]["age_days"] != 400.0 || got[0]["owner"] != "ana" {
		t.Errorf("row 0 = %v", got[0])
	}
	if got[1]["uncommitted"] != true || got[1]["commit"] != "" {
		t.Errorf("row 1 = %v", got[1])
	}
	if _, ok := got[0]["Time"]; ok {
		t.Error("internal fields leaked into JSON")
	}
}

func TestJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, "json", nil); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "[]\n" {
		t.Errorf("got %q, want an empty array", got)
	}
}

func TestNDJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, "ndjson", sample); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), buf.String())
	}
	for _, l := range lines {
		var r Row
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Errorf("line %q: %v", l, err)
		}
	}
}

func TestUnknownFormat(t *testing.T) {
	if err := Write(&bytes.Buffer{}, "yaml", sample); err == nil {
		t.Error("expected an error for an unknown format")
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, "table", sample); err != nil {
		t.Fatal(err)
	}
	want := `AGE     DATE        AUTHOR             TAG        LOCATION     TEXT
1y 1mo  2025-03-01  Ana, Jr.           TODO(ana)  src/a.go:12  say "hi", then leave
today   2026-04-05  Not Committed Yet  FIXME      b c.txt:1     leading space
`
	if got := buf.String(); got != want {
		t.Errorf("table output:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableStripsControls(t *testing.T) {
	text := "bell\x07 and\x1b[0m esc\u2028line"
	rows := []Row{{
		File: "a.go", Line: 3, Tag: "TODO", Text: text,
		Author: "Ada\tLovelace", Date: "2026-01-01", Age: 2 * age.Day,
	}}
	var table bytes.Buffer
	if err := Write(&table, "table", rows); err != nil {
		t.Fatal(err)
	}
	got := table.String()
	if strings.ContainsAny(got, "\a\x1b\t") || strings.ContainsRune(got, '\u2028') {
		t.Fatalf("table leaked controls:\n%q", got)
	}
	if !strings.Contains(got, "Ada Lovelace") || !strings.Contains(got, "bell  and [0m esc line") {
		t.Fatalf("table:\n%s", got)
	}
	var js bytes.Buffer
	if err := Write(&js, "json", rows); err != nil {
		t.Fatal(err)
	}
	var decoded []Row
	if err := json.Unmarshal(js.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0].Text != text || decoded[0].Author != "Ada\tLovelace" {
		t.Fatalf("json changed the fields: %+v", decoded[0])
	}
}

func TestSort(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2025, 1, n, 0, 0, 0, 0, time.UTC) }
	rows := func() []Row {
		return []Row{
			{File: "b.go", Line: 9, Author: "zed", Time: day(3)},
			{File: "a.go", Line: 5, Author: "Amy", Time: day(2)},
			{File: "b.go", Line: 2, Author: "amy", Time: day(2)},
			{File: "a.go", Line: 1, Author: "Zed", Time: day(1)},
		}
	}
	tests := map[string][]string{
		"age":    {"a.go:1", "a.go:5", "b.go:2", "b.go:9"},
		"file":   {"a.go:1", "a.go:5", "b.go:2", "b.go:9"},
		"author": {"a.go:5", "b.go:2", "a.go:1", "b.go:9"},
	}
	for key, want := range tests {
		rs := rows()
		if err := Sort(rs, key); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, r := range rs {
			got = append(got, r.File+":"+strconv.Itoa(r.Line))
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("sort by %s: got %v, want %v", key, got, want)
		}
	}
	if err := Sort(rows(), "size"); err == nil {
		t.Error("expected an error for an unknown sort key")
	}
}
