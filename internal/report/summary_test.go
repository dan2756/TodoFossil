package report

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/dan2756/todo-age/internal/age"
)

func summaryRows() []Row {
	ages := map[string][]time.Duration{
		"TODO":  {0, 6 * age.Day, 45 * age.Day, 400 * age.Day, 3 * age.Year},
		"FIXME": {age.Week, 200 * age.Day},
		"HACK":  {2 * age.Year},
	}
	var rows []Row
	for tag, list := range ages {
		for _, a := range list {
			rows = append(rows, Row{Tag: tag, Age: a})
		}
	}
	return rows
}

func TestSummarize(t *testing.T) {
	s := Summarize(summaryRows())
	if s.Total != 8 {
		t.Errorf("total = %d, want 8", s.Total)
	}
	if s.ByTag["TODO"] != 5 || s.ByTag["FIXME"] != 2 || s.ByTag["HACK"] != 1 {
		t.Errorf("by tag = %v", s.ByTag)
	}
	want := []BucketCount{
		{"<1w", 2}, {"1w-1mo", 1}, {"1mo-3mo", 1}, {"3mo-6mo", 0},
		{"6mo-1y", 1}, {"1y-2y", 1}, {">2y", 2},
	}
	for i, b := range want {
		if s.ByAge[i] != b {
			t.Errorf("bucket %d = %+v, want %+v", i, s.ByAge[i], b)
		}
	}
}

func TestSummaryTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSummary(&buf, "table", Summarize(summaryRows())); err != nil {
		t.Fatal(err)
	}
	want := `TAG      COUNT
TODO         5
FIXME        2
HACK         1

AGE      COUNT
<1w          2
1w-1mo       1
1mo-3mo      1
3mo-6mo      0
6mo-1y       1
1y-2y        1
>2y          2

TOTAL        8
`
	if got := buf.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSummaryCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSummary(&buf, "csv", Summarize([]Row{{Tag: "TODO", Age: age.Year}})); err != nil {
		t.Fatal(err)
	}
	want := `group,key,count
tag,TODO,1
age,<1w,0
age,1w-1mo,0
age,1mo-3mo,0
age,3mo-6mo,0
age,6mo-1y,0
age,1y-2y,1
age,>2y,0
total,,1
`
	if got := buf.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSummaryJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSummary(&buf, "json", Summarize(nil)); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Total int            `json:"total"`
		ByTag map[string]int `json:"by_tag"`
		ByAge []BucketCount  `json:"by_age"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Total != 0 || got.ByTag == nil || len(got.ByAge) != 7 {
		t.Errorf("empty summary = %s", buf.String())
	}
}
