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

func TestContentHTMLSupportsMarkdownAndExplicitHTML(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	r := &runner{output: output.NewPlainRenderer(&out, &errOut)}

	converted, hasContent, err := r.contentHTML("# Title\n\n- item", "", "--description", "--description-html")
	if err != nil {
		t.Fatal(err)
	}
	if !hasContent || converted != "<h1>Title</h1><ul><li>item</li></ul>" {
		t.Fatalf("converted content = %q, hasContent = %v", converted, hasContent)
	}

	explicit := `<div><strong>Raw HTML</strong></div>`
	converted, hasContent, err = r.contentHTML("", explicit, "BODY", "--body-html")
	if err != nil {
		t.Fatal(err)
	}
	if !hasContent || converted != explicit {
		t.Fatalf("explicit content = %q, hasContent = %v", converted, hasContent)
	}

	if _, _, err := r.contentHTML("plain", explicit, "BODY", "--body-html"); err == nil || !strings.Contains(err.Error(), "BODY and --body-html") {
		t.Fatalf("mutually exclusive content error = %v", err)
	}
}

func TestCommentsAddAndUpdateUseCommentHTML(t *testing.T) {
	var calls []string
	var payloads []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/":
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode add payload: %v", err)
			}
			payloads = append(payloads, payload)
			_ = json.NewEncoder(writer).Encode(map[string]any{"id": "added-comment", "comment_html": payload["comment_html"]})
		case request.Method == http.MethodPatch && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/comment-id/":
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode update payload: %v", err)
			}
			payloads = append(payloads, payload)
			_ = json.NewEncoder(writer).Encode(map[string]any{"id": "comment-id", "comment_html": payload["comment_html"]})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &errOut)
	if err := Run([]string{"comments", "add", "-p", "project-id", "-i", "issue-id", "# Title\n\n- item"}, renderer); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "✓ Comment added") {
		t.Fatalf("table add output = %s", out.String())
	}
	if payloads[0]["comment_html"] != "<h1>Title</h1><ul><li>item</li></ul>" {
		t.Fatalf("Markdown add payload = %#v", payloads[0])
	}

	out.Reset()
	errOut.Reset()
	rawHTML := `<p>Raw <strong>HTML</strong></p>`
	if err := Run([]string{"comments", "add", "-p", "project-id", "-i", "issue-id", "--body-html", rawHTML, "-f", "json"}, renderer); err != nil {
		t.Fatal(err)
	}
	var addResponse map[string]any
	if err := json.Unmarshal(out.Bytes(), &addResponse); err != nil {
		t.Fatalf("JSON add output = %s: %v", out.String(), err)
	}
	if strings.Contains(out.String(), "Comment added") || payloads[1]["comment_html"] != rawHTML {
		t.Fatalf("HTML add output/payload = %s / %#v", out.String(), payloads[1])
	}

	out.Reset()
	errOut.Reset()
	if err := Run([]string{"comments", "update", "-p", "project-id", "-i", "issue-id", "comment-id", "**Updated**"}, renderer); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "✓ Comment updated") {
		t.Fatalf("table update output = %s", out.String())
	}
	if payloads[2]["comment_html"] != "<p><strong>Updated</strong></p>" {
		t.Fatalf("Markdown update payload = %#v", payloads[2])
	}

	out.Reset()
	errOut.Reset()
	if err := Run([]string{"comments", "update", "-p", "project-id", "-i", "issue-id", "comment-id", "--body-html", rawHTML, "-f", "json"}, renderer); err != nil {
		t.Fatal(err)
	}
	var updateResponse map[string]any
	if err := json.Unmarshal(out.Bytes(), &updateResponse); err != nil {
		t.Fatalf("JSON update output = %s: %v", out.String(), err)
	}
	if strings.Contains(out.String(), "Comment updated") || payloads[3]["comment_html"] != rawHTML {
		t.Fatalf("HTML update output/payload = %s / %#v", out.String(), payloads[3])
	}

	wantCalls := []string{
		"POST /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/",
		"POST /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/",
		"PATCH /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/comment-id/",
		"PATCH /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/comment-id/",
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("API calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestCommentsRejectMixedContentSources(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &errOut)
	err := Run([]string{"comments", "add", "-p", "project-id", "-i", "issue-id", "BODY", "--body-html", "<p>HTML</p>"}, renderer)
	if err == nil || !strings.Contains(err.Error(), "BODY and --body-html are mutually exclusive") {
		t.Fatalf("mixed content error = %v", err)
	}
}

func TestCommentsDeleteConfirmsAndReturnsPureJSON(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		if request.Method != http.MethodDelete || request.URL.Path != "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/comments/comment-id/" {
			http.NotFound(writer, request)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)

	var out bytes.Buffer
	var errOut bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &errOut)
	if err := runWithInput([]string{"comments", "delete", "-p", "project-id", "-i", "issue-id", "comment-id"}, renderer, strings.NewReader("yes\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "Delete comment comment-id? [y/N]") || !strings.Contains(out.String(), "✓ Comment deleted") {
		t.Fatalf("confirmation/output = %q / %q", errOut.String(), out.String())
	}

	out.Reset()
	errOut.Reset()
	if err := runWithInput([]string{"comments", "delete", "-p", "project-id", "-i", "issue-id", "comment-id", "--yes", "-f", "json"}, renderer, strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("JSON delete output = %s: %v", out.String(), err)
	}
	if result["comment_id"] != "comment-id" || result["deleted"] != true || errOut.Len() != 0 {
		t.Fatalf("delete result/stderr = %#v / %q", result, errOut.String())
	}

	if len(calls) != 2 {
		t.Fatalf("delete calls = %#v, want 2", calls)
	}
}

func TestCommentsDeleteDoesNotCallAPIWhenConfirmationFails(t *testing.T) {
	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete {
			deleteCalls++
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)

	for _, input := range []string{"n\n", ""} {
		var out bytes.Buffer
		var errOut bytes.Buffer
		renderer := output.NewPlainRenderer(&out, &errOut)
		err := runWithInput([]string{"comments", "delete", "-p", "project-id", "-i", "issue-id", "comment-id"}, renderer, strings.NewReader(input))
		if err == nil || !strings.Contains(err.Error(), "comment deletion cancelled") {
			t.Fatalf("confirmation input %q error = %v", input, err)
		}
		if out.Len() != 0 || !strings.Contains(errOut.String(), "Delete comment comment-id? [y/N]") {
			t.Fatalf("confirmation input %q output/stderr = %q / %q", input, out.String(), errOut.String())
		}
	}
	if deleteCalls != 0 {
		t.Fatalf("delete calls = %d, want 0", deleteCalls)
	}
}
