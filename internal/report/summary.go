package report

import (
	"bytes"
	"cmp"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/dan2756/todo-age/internal/age"
)

// Summary counts markers per tag and per age bucket.
type Summary struct {
	Total int            `json:"total"`
	ByTag map[string]int `json:"by_tag"`
	ByAge []BucketCount  `json:"by_age"`
}

// BucketCount is the number of markers in one age bucket.
type BucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// buckets are the age ranges of a summary, youngest first. Each holds the
// ages below its limit that the previous one does not.
var buckets = []struct {
	label string
	limit time.Duration // zero: no upper limit
}{
	{"<1w", age.Week},
	{"1w-1mo", age.Month},
	{"1mo-3mo", 3 * age.Month},
	{"3mo-6mo", 6 * age.Month},
	{"6mo-1y", age.Year},
	{"1y-2y", 2 * age.Year},
	{">2y", 0},
}

// Summarize counts rows per tag and age bucket.
func Summarize(rows []Row) Summary {
	s := Summary{Total: len(rows), ByTag: make(map[string]int)}
	for _, b := range buckets {
		s.ByAge = append(s.ByAge, BucketCount{Bucket: b.label})
	}
	for _, r := range rows {
		s.ByTag[r.Tag]++
		for i, b := range buckets {
			if b.limit == 0 || r.Age < b.limit {
				s.ByAge[i].Count++
				break
			}
		}
	}
	return s
}

// entry is one count of a summary: group "tag", "age" or "total".
type entry struct {
	Group string `json:"group"`
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// entries flattens s: tags most frequent first, then age buckets youngest
// first, then the total.
func (s Summary) entries() []entry {
	var tags []entry
	for tag, n := range s.ByTag {
		tags = append(tags, entry{"tag", tag, n})
	}
	slices.SortFunc(tags, func(a, b entry) int {
		if c := cmp.Compare(b.Count, a.Count); c != 0 {
			return c
		}
		return cmp.Compare(a.Key, b.Key)
	})
	out := tags
	for _, b := range s.ByAge {
		out = append(out, entry{"age", b.Bucket, b.Count})
	}
	return append(out, entry{"total", "", s.Total})
}

// WriteSummary writes s to w in format. The table format prints a block of
// tag counts and a block of age buckets; csv and ndjson print one
// group,key,count entry per line.
func WriteSummary(w io.Writer, format string, s Summary) error {
	switch format {
	case "table":
		return writeSummaryTable(w, s)
	case "json":
		enc := json.NewEncoder(w)
		// Bucket names are "<1w" and ">2y". Leave the brackets as characters,
		// matching the table and the marker JSON.
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	case "ndjson":
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		for _, e := range s.entries() {
			if err := enc.Encode(e); err != nil {
				return err
			}
		}
		return nil
	case "csv":
		cw := csv.NewWriter(w)
		cw.Write([]string{"group", "key", "count"})
		for _, e := range s.entries() {
			cw.Write([]string{e.Group, e.Key, strconv.Itoa(e.Count)})
		}
		cw.Flush()
		return cw.Error()
	}
	return fmt.Errorf("unknown format %q", format)
}

func writeSummaryTable(w io.Writer, s Summary) error {
	entries := s.entries()
	width := len("TOTAL")
	for _, e := range entries {
		width = max(width, len(e.Key))
	}
	var buf bytes.Buffer
	group := ""
	for _, e := range entries {
		if e.Group != group {
			if group != "" {
				buf.WriteByte('\n')
			}
			if e.Group == "total" {
				fmt.Fprintf(&buf, "%-*s  %5d\n", width, "TOTAL", e.Count)
				break
			}
			heading := map[string]string{"tag": "TAG", "age": "AGE"}[e.Group]
			fmt.Fprintf(&buf, "%-*s  %5s\n", width, heading, "COUNT")
			group = e.Group
		}
		fmt.Fprintf(&buf, "%-*s  %5d\n", width, e.Key, e.Count)
	}
	_, err := w.Write(buf.Bytes())
	return err
}
