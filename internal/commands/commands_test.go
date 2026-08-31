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

func TestRunHelpDoesNotRequireAPIConfiguration(t *testing.T) {
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"--help"}, renderer); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Available commands:") || !strings.Contains(out.String(), "attachments") || strings.Contains(out.String(), "upload"+"-image") {
		t.Fatalf("help output = %s", out.String())
	}
}

func TestAttachmentsHelpOmitsUnsupportedUpdateCommand(t *testing.T) {
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"attachments", "--help"}, renderer); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "list|get|upload|complete|delete") || strings.Contains(out.String(), "update") {
		t.Fatalf("attachments help output = %s", out.String())
	}
	if err := Run([]string{"attachments", "update"}, renderer); err == nil {
		t.Fatal("removed attachment subcommand should be unavailable")
	}
}

func TestRunProjectsListAcceptsFormatAfterCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/workspaces/ws/projects/" {
			t.Errorf("path = %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	t.Setenv("PLANE_API_KEY", "test")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"projects", "list", "-f", "json"}, renderer); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "{\n  \"results\": []\n}" {
		t.Fatalf("output = %s", out.String())
	}
}

func TestIssuesSearchUsesDocumentedRouteAndHandlesOptionalDescriptionSnippet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/workspaces/ws/work-items/search/" {
			t.Errorf("path = %s, want documented search path", request.URL.Path)
		}
		if request.URL.Query().Get("workspace_search") != "true" {
			t.Errorf("workspace_search = %q, want true", request.URL.Query().Get("workspace_search"))
		}
		if request.URL.Query().Get("limit") != "100" {
			t.Errorf("limit = %q, want 100", request.URL.Query().Get("limit"))
		}
		if request.URL.Query().Get("search") != "alpha beta" {
			t.Errorf("search = %q, want joined query", request.URL.Query().Get("search"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"issues":[
			{"id":"issue-1","name":"Body match","sequence_id":1,"project__identifier":"PRJ","description_snippet":"...alpha beta..."},
			{"id":"issue-2","name":"Title match","sequence_id":2,"project__identifier":"PRJ","description_snippet":null},
			{"id":"issue-3","name":"Older title-only match","sequence_id":3,"project__identifier":"PRJ"}
		]}`))
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"issues", "search", "alpha", "beta"}, renderer); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"PROJECT", "DESCRIPTION", "Body match", "Title match", "...alpha beta..."} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("output missing %q: %s", expected, out.String())
		}
	}

	var jsonOut bytes.Buffer
	jsonRenderer := output.NewPlainRenderer(&jsonOut, &bytes.Buffer{})
	if err := Run([]string{"issues", "search", "alpha", "beta", "-f", "json"}, jsonRenderer); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(jsonOut.Bytes(), &response); err != nil {
		t.Fatalf("JSON output = %s: %v", jsonOut.String(), err)
	}
	issues, ok := response["issues"].([]any)
	if !ok {
		t.Fatalf("issues = %#v", response["issues"])
	}
	for _, rawIssue := range issues {
		issue, ok := rawIssue.(map[string]any)
		if ok && issue["id"] == "issue-3" {
			if snippet, exists := issue["description_snippet"]; !exists || snippet != nil {
				t.Fatalf("issue-3 description_snippet = %#v, want null", snippet)
			}
		}
	}
}

func TestIssueWritesSendStringPriorityValues(t *testing.T) {
	var priorities []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
			return
		}
		priority, ok := payload["priority"].(string)
		if !ok {
			t.Errorf("priority type = %T, want string", payload["priority"])
		} else {
			priorities = append(priorities, priority)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"issue-id","priority":"` + priority + `"}`))
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"issues", "create", "-p", "project-id", "--name", "Smoke", "--priority", "low"}, renderer); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"issues", "update", "-p", "project-id", "issue-id", "--priority", "high"}, renderer); err != nil {
		t.Fatal(err)
	}
	if strings.Join(priorities, ",") != "low,high" {
		t.Fatalf("priorities = %v, want [low high]", priorities)
	}
}
