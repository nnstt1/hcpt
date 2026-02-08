package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// Print writes tabular data to the writer.
// headers defines the column names, and rows contains the data.
func Print(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Print headers
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, h)
	}
	_, _ = fmt.Fprintln(tw)

	// Print rows
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			_, _ = fmt.Fprint(tw, col)
		}
		_, _ = fmt.Fprintln(tw)
	}

	_ = tw.Flush()
}

// PrintKeyValue writes key-value pairs in a vertical format.
func PrintKeyValue(w io.Writer, pairs []KeyValue) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, kv := range pairs {
		_, _ = fmt.Fprintf(tw, "%s:\t%s\n", kv.Key, kv.Value)
	}
	_ = tw.Flush()
}

// KeyValue represents a key-value pair for vertical display.
type KeyValue struct {
	Key   string
	Value string
}

// PrintJSON writes data as formatted JSON to the writer.
func PrintJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
