package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HonLuk/my-plane/internal/output"
)

func TestIssueReadsIncludeCrossProjectRelations(t *testing.T) {
	for _, command := range [][]string{
		{"issues", "get", "-p", "project-id", "issue-id"},
		{"issues", "get-short", "PROJ-1"},
		{"issues", "relations", "-p", "project-id", "issue-id"},
	} {
		for _, format := range []string{"json", "table"} {
			t.Run(strings.Join(command, " ")+"/"+format, func(t *testing.T) {
				relationRequests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch req.URL.Path {
					case "/api/v1/workspaces/ws/work-items/PROJ-1/", "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/":
						_, _ = w.Write([]byte(`{"id":"issue-id","project":"project-id","parent":null,"project_type":"bug"}`))
					case "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/relations/":
						relationRequests++
						_, _ = w.Write([]byte(`{"relates_to":[{"project_id":"feature-project","issue_id":"requirement-id"}],"blocked_by":[]}`))
					default:
						t.Errorf("unexpected path: %s", req.URL.Path)
						http.NotFound(w, req)
					}
				}))
				defer server.Close()
				t.Setenv("PLANE_API_KEY", "test")
				t.Setenv("PLANE_WORKSPACE", "ws")
				t.Setenv("PLANE_BASE_URL", server.URL)
				var out bytes.Buffer
				args := append(append([]string{}, command...), "-f", format)
				if err := Run(args, output.NewPlainRenderer(&out, &bytes.Buffer{})); err != nil {
					t.Fatal(err)
				}
				if relationRequests != 1 {
					t.Fatalf("relation requests = %d", relationRequests)
				}
				for _, expected := range []string{"relates_to", "feature-project", "requirement-id", "blocked_by"} {
					if !strings.Contains(out.String(), expected) {
						t.Errorf("missing %s: %s", expected, out.String())
					}
				}
				if command[1] != "relations" && format == "json" {
					var item map[string]any
					if err := json.Unmarshal(out.Bytes(), &item); err != nil {
						t.Fatal(err)
					}
					if parent, exists := item["parent"]; !exists || parent != nil {
						t.Fatalf("parent changed: %#v", item)
					}
					if item["project_type"] != "bug" {
						t.Fatalf("project_type missing: %#v", item)
					}
				}
			})
		}
	}
}

func TestIssueRelationsHandleEmptyEmbeddedAndFailedResponses(t *testing.T) {
	for _, test := range []struct {
		name, detail, relations string
		status, requests        int
		wantError               string
	}{
		{name: "empty", detail: `{"id":"issue-id","project":{"id":"project-id"}}`, relations: `{"relates_to":[],"blocked_by":[]}`, status: 200, requests: 1},
		{name: "embedded", detail: `{"relations":{"relates_to":[]}}`, requests: 0},
		{name: "missing IDs", detail: `{"parent":null}`, wantError: "missing project or work item ID"},
		{name: "invalid ID", detail: `{"id":"../issue","project":"project-id"}`, wantError: "illegal characters"},
		{name: "unsupported endpoint", detail: `{"id":"issue-id","project":"project-id"}`, relations: `{"detail":"Not found"}`, status: 404, requests: 1, wantError: "could not read work item relations"},
		{name: "forbidden", detail: `{"id":"issue-id","project":"project-id"}`, relations: `{"detail":"Forbidden"}`, status: 403, requests: 1, wantError: "could not read work item relations"},
		{name: "server error", detail: `{"id":"issue-id","project":"project-id"}`, relations: `{}`, status: 500, requests: 1, wantError: "could not read work item relations"},
		{name: "invalid relations", detail: `{"id":"issue-id","project":"project-id"}`, relations: `null`, status: 200, requests: 1, wantError: "expected relations grouped by type"},
	} {
		t.Run(test.name, func(t *testing.T) {
			count := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(req.URL.Path, "/relations/") {
					count++
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(test.relations))
				} else {
					_, _ = w.Write([]byte(test.detail))
				}
			}))
			defer server.Close()
			t.Setenv("PLANE_API_KEY", "test")
			t.Setenv("PLANE_WORKSPACE", "ws")
			t.Setenv("PLANE_BASE_URL", server.URL)
			var out bytes.Buffer
			err := Run([]string{"issues", "get-short", "PROJ-1", "-f", "json"}, output.NewPlainRenderer(&out, &bytes.Buffer{}))
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error = %v", err)
				}
				if out.Len() != 0 {
					t.Fatalf("failed read printed incomplete details: %s", out.String())
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), `"relates_to": []`) {
					t.Fatalf("empty relations missing: %s", out.String())
				}
			}
			if count != test.requests {
				t.Fatalf("requests = %d, want %d", count, test.requests)
			}
		})
	}
}
