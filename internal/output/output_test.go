package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatJSONPreservesRawResponse(t *testing.T) {
	var out bytes.Buffer
	renderer := NewPlainRenderer(&out, &bytes.Buffer{})
	renderer.Format(map[string]any{"name": "Demo", "count": 2}, "json", true)
	if !strings.Contains(out.String(), `"name": "Demo"`) || !strings.Contains(out.String(), `"count": 2`) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestFormatTableShowsPaginationAndProjects(t *testing.T) {
	var out bytes.Buffer
	renderer := NewPlainRenderer(&out, &bytes.Buffer{})
	renderer.Format(map[string]any{
		"next_cursor":   "next",
		"total_results": 1,
		"total_pages":   1,
		"count":         1,
		"results":       []any{map[string]any{"identifier": "PROJ", "name": "Demo", "id": "uuid"}},
	}, "table", true)
	for _, expected := range []string{"Pagination: total: 1 | pages: 1 | showing: 1", "Next page: --cursor next", "PROJ", "Demo", "uuid"} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("output missing %q: %s", expected, out.String())
		}
	}
}

func TestFormatTableEmptyList(t *testing.T) {
	var out bytes.Buffer
	renderer := NewPlainRenderer(&out, &bytes.Buffer{})
	renderer.Format([]any{}, "table", true)
	if out.String() != "No results.\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestProjectTypeIsOptionalAcrossOutputFormats(t *testing.T) {
	for _, format := range []string{"json", "table"} {
		for _, value := range []any{nil, "", "   ", "feature", "bug", "other"} {
			data := map[string]any{"id": "uuid", "project_type": value, "parent": nil, "project": map[string]any{"id": "project", "project_type": value}}
			var out bytes.Buffer
			NewPlainRenderer(&out, &bytes.Buffer{}).Format(data, format, false)
			present := value == "feature" || value == "bug" || value == "other"
			if strings.Contains(out.String(), "project_type") != present {
				t.Fatalf("format %s, value %#v: %s", format, value, out.String())
			}
			if _, exists := data["project_type"]; !exists {
				t.Fatal("renderer mutated source response")
			}
			if !strings.Contains(out.String(), "parent") {
				t.Fatal("unrelated null field removed")
			}
		}
	}
}

func TestProjectTypeColumnScansMixedLists(t *testing.T) {
	for _, issue := range []bool{false, true} {
		rows := []any{
			map[string]any{"id": "one", "name": "Untyped", "identifier": "ONE", "project_type": nil},
			map[string]any{"id": "two", "name": "Typed", "identifier": "TWO", "project_type": "bug"},
		}
		if issue {
			for _, item := range rows {
				row := item.(map[string]any)
				delete(row, "identifier")
				row["priority"] = "high"
			}
		}
		var out bytes.Buffer
		renderer := NewPlainRenderer(&out, &bytes.Buffer{})
		renderer.Format(rows, "table", false)
		if !strings.Contains(out.String(), "PROJECT_TYPE") || !strings.Contains(out.String(), "bug") {
			t.Fatalf("mixed output = %s", out.String())
		}
		out.Reset()
		renderer.Format(rows[:1], "table", false)
		if strings.Contains(out.String(), "PROJECT_TYPE") {
			t.Fatalf("empty type column shown: %s", out.String())
		}
	}
}
