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
