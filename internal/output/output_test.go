package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nnstt1/hcpt/internal/output"
)

func TestPrint(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"NAME", "ID"}
	rows := [][]string{
		{"workspace-1", "ws-abc123"},
		{"workspace-2", "ws-def456"},
	}

	output.Print(&buf, headers, rows)

	got := buf.String()
	if !strings.Contains(got, "NAME") {
		t.Errorf("expected header NAME, got:\n%s", got)
	}
	if !strings.Contains(got, "workspace-1") {
		t.Errorf("expected workspace-1, got:\n%s", got)
	}
	if !strings.Contains(got, "ws-def456") {
		t.Errorf("expected ws-def456, got:\n%s", got)
	}

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (1 header + 2 rows), got %d", len(lines))
	}
}

func TestPrintEmptyRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"NAME", "ID"}

	output.Print(&buf, headers, nil)

	got := buf.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (header only), got %d", len(lines))
	}
}

func TestPrintKeyValue(t *testing.T) {
	var buf bytes.Buffer
	pairs := []output.KeyValue{
		{Key: "Name", Value: "my-workspace"},
		{Key: "ID", Value: "ws-abc123"},
		{Key: "Description", Value: "A test workspace"},
	}

	output.PrintKeyValue(&buf, pairs)

	got := buf.String()
	if !strings.Contains(got, "Name:") {
		t.Errorf("expected 'Name:', got:\n%s", got)
	}
	if !strings.Contains(got, "my-workspace") {
		t.Errorf("expected 'my-workspace', got:\n%s", got)
	}

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestPrintKeyValueEmpty(t *testing.T) {
	var buf bytes.Buffer
	output.PrintKeyValue(&buf, nil)

	got := buf.String()
	if got != "" {
		t.Errorf("expected empty output, got:\n%s", got)
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{
		"name": "test",
		"id":   "123",
	}

	err := output.PrintJSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `"name": "test"`) {
		t.Errorf("expected JSON with name field, got:\n%s", got)
	}
	if !strings.Contains(got, `"id": "123"`) {
		t.Errorf("expected JSON with id field, got:\n%s", got)
	}
}

func TestPrintJSON_Array(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]string{
		{"name": "a"},
		{"name": "b"},
	}

	err := output.PrintJSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(got), "[") {
		t.Errorf("expected JSON array, got:\n%s", got)
	}
	if strings.Count(got, `"name"`) != 2 {
		t.Errorf("expected 2 name fields in JSON array, got:\n%s", got)
	}
}

func TestPrintJSON_EmptyArray(t *testing.T) {
	var buf bytes.Buffer
	data := []string{}

	err := output.PrintJSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "[]" {
		t.Errorf("expected '[]', got: %q", got)
	}
}

func TestPrint_ColumnAlignment(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"SHORT", "LONG HEADER"}
	rows := [][]string{
		{"a", "value1"},
		{"longer", "v2"},
	}

	output.Print(&buf, headers, rows)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Verify second column starts at the same position for all lines
	headerPos := strings.Index(lines[0], "LONG HEADER")
	row1Pos := strings.Index(lines[1], "value1")
	row2Pos := strings.Index(lines[2], "v2")

	if headerPos != row1Pos || headerPos != row2Pos {
		t.Errorf("columns not aligned: header=%d, row1=%d, row2=%d", headerPos, row1Pos, row2Pos)
	}
}
