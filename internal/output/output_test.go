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
