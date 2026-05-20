// Package report formats todo-age results as a table, JSON, CSV or NDJSON.
package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// Row is one marker together with the blame of its line.
type Row struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Tag         string `json:"tag"`
	Text        string `json:"text"`
	Owner       string `json:"owner"`
	Author      string `json:"author"`
	Email       string `json:"email"`
	Commit      string `json:"commit"`
	Date        string `json:"date"`
	AgeDays     int    `json:"age_days"`
	Uncommitted bool   `json:"uncommitted"`

	Time time.Time     `json:"-"` // author time, or the current time if uncommitted
	Age  time.Duration `json:"-"` // exact age, for filtering and sorting
}

// Formats are the supported output formats; the first is the default.
var Formats = []string{"table", "json", "csv", "ndjson"}

// ValidFormat reports whether format is one of Formats.
func ValidFormat(format string) bool {
	return slices.Contains(Formats, format)
}

// Write writes rows to w in format.
func Write(w io.Writer, format string, rows []Row) error {
	switch format {
	case "table":
		return writeTable(w, rows)
	case "json":
		if rows == nil {
			rows = []Row{}
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case "ndjson":
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	case "csv":
		return writeCSV(w, rows)
	}
	return fmt.Errorf("unknown format %q (want %s)", format, strings.Join(Formats, ", "))
}

var csvHeader = []string{
	"file", "line", "tag", "text", "owner", "author", "email",
	"commit", "date", "age_days", "uncommitted",
}

func writeCSV(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, r := range rows {
		rec := []string{
			r.File, strconv.Itoa(r.Line), r.Tag, r.Text, r.Owner, r.Author, r.Email,
			r.Commit, r.Date, strconv.Itoa(r.AgeDays), strconv.FormatBool(r.Uncommitted),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeTable(w io.Writer, rows []Row) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "AGE\tDATE\tAUTHOR\tTAG\tLOCATION\tTEXT")
	for _, r := range rows {
		tag := r.Tag
		if r.Owner != "" {
			tag += "(" + r.Owner + ")"
		}
		fmt.Fprintf(tw, "%dd\t%s\t%s\t%s\t%s:%d\t%s\n",
			r.AgeDays, r.Date, cell(r.Author), cell(tag), cell(r.File), r.Line, cell(r.Text))
	}
	return tw.Flush()
}

// cell keeps tabs and newlines in a value from breaking table columns.
func cell(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", "").Replace(s)
}
