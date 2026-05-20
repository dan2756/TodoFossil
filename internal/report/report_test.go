package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

var sample = []Row{
	{
		File: "src/a.go", Line: 12, Tag: "TODO", Text: `say "hi", then leave`,
		Owner: "ana", Author: "Ana, Jr.", Email: "ana@example.com",
		Commit: "1a2b3c4", Date: "2025-03-01", AgeDays: 400,
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
