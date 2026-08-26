package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HonLuk/my-plane/internal/output"
)

func TestAttachmentsUploadUsesAttachmentObjectStoreConfirmationAndInsertion(t *testing.T) {
	assetID := "20745b59-e398-460d-9532-e0d56fbe7919"
	var objectStoreBody []byte
	var objectStoreAPIKey string
	objectStore := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		objectStoreBody, _ = io.ReadAll(request.Body)
		objectStoreAPIKey = request.Header.Get("X-API-Key")
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer objectStore.Close()

	var calls []string
	var attachmentPayload map[string]any
	var descriptionPayload map[string]any
	plane := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/work-items/PROJ-123/"):
			_, _ = writer.Write([]byte(`{"project":"project-id","id":"issue-id","description_html":"<p>Existing</p>"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/":
			if err := json.NewDecoder(request.Body).Decode(&attachmentPayload); err != nil {
				t.Errorf("decode attachment payload: %v", err)
			}
			_, _ = writer.Write([]byte(`{"asset_id":"` + assetID + `","upload_data":{"url":"` + objectStore.URL + `","fields":{"key":"private/key","policy":"private-policy"}}}`))
		case request.Method == http.MethodPatch && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/"+assetID+"/":
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPatch && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/":
			if err := json.NewDecoder(request.Body).Decode(&descriptionPayload); err != nil {
				t.Errorf("decode description payload: %v", err)
			}
			_, _ = writer.Write([]byte(`{"description_html":"<p>Existing</p><image-component src=\"` + assetID + `\"></image-component>"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer plane.Close()

	directory := t.TempDir()
	imagePath := filepath.Join(directory, "screenshot.png")
	if err := os.WriteFile(imagePath, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", plane.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"attachments", "upload", "PROJ-123", imagePath, "-f", "json"}, renderer); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("output = %s: %v", out.String(), err)
	}
	if result["asset_id"] != assetID || !strings.Contains(result["image_component"].(string), assetID) {
		t.Fatalf("result = %#v", result)
	}
	if result["inserted"] != true || result["asset_url"] != "/api/assets/v2/workspaces/ws/projects/project-id/issues/issue-id/attachments/"+assetID+"/" {
		t.Fatalf("result insertion metadata = %#v", result)
	}
	if attachmentPayload["name"] != "screenshot.png" || attachmentPayload["type"] != "image/png" {
		t.Fatalf("attachment payload = %#v", attachmentPayload)
	}
	if _, ok := attachmentPayload["project_id"]; ok {
		t.Fatalf("attachment payload should not use generic asset fields: %#v", attachmentPayload)
	}
	if descriptionPayload["description_html"] != `<p>Existing</p><image-component src="`+assetID+`"></image-component>` {
		t.Fatalf("description payload = %#v", descriptionPayload)
	}
	if strings.Contains(out.String(), "upload_data") || strings.Contains(out.String(), "private-policy") {
		t.Fatalf("signed upload data leaked: %s", out.String())
	}
	if objectStoreAPIKey != "" || !strings.Contains(string(objectStoreBody), "fake image bytes") {
		t.Fatalf("object-store request was unsafe or incomplete")
	}
	wantCalls := []string{
		"GET /api/v1/workspaces/ws/work-items/PROJ-123/",
		"POST /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/",
		"PATCH /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/" + assetID + "/",
		"PATCH /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/",
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("API calls = %#v", calls)
	}
}

func TestAttachmentsUploadRetainsConfirmedAttachmentWhenInsertionFails(t *testing.T) {
	assetID := "20745b59-e398-460d-9532-e0d56fbe7919"
	objectStore := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.Copy(io.Discard, request.Body)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer objectStore.Close()

	var calls []string
	plane := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/work-items/PROJ-123/"):
			_, _ = writer.Write([]byte(`{"project":"project-id","id":"issue-id","description_html":"<p>Existing</p>"}`))
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/work-items/issue-id/attachments/"):
			_, _ = writer.Write([]byte(`{"asset_id":"` + assetID + `","upload_data":{"url":"` + objectStore.URL + `","fields":{"key":"private/key"}}}`))
		case request.Method == http.MethodPatch && strings.HasSuffix(request.URL.Path, "/attachments/"+assetID+"/"):
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPatch && strings.HasSuffix(request.URL.Path, "/work-items/issue-id/"):
			http.Error(writer, `{"error":"cannot update description"}`, http.StatusInternalServerError)
		case request.Method == http.MethodDelete:
			t.Errorf("confirmed attachment must not be deleted: %s", request.URL.Path)
			http.Error(writer, `{"error":"unexpected delete"}`, http.StatusInternalServerError)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer plane.Close()

	imagePath := filepath.Join(t.TempDir(), "screenshot.png")
	if err := os.WriteFile(imagePath, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", plane.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	err := Run([]string{"attachments", "upload", "PROJ-123", imagePath, "-f", "json"}, renderer)
	if err == nil || !strings.Contains(err.Error(), "attachment retained with asset ID "+assetID) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(strings.Join(calls, "\n"), "DELETE ") {
		t.Fatalf("API calls = %#v", calls)
	}
}

func TestAttachmentFileMetadataRequiresMIMETypeForUnknownExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.unknown")
	if err := os.WriteFile(path, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := attachmentFileMetadata(path, ""); err == nil || !strings.Contains(err.Error(), "Could not determine MIME type") {
		t.Fatalf("error = %v", err)
	}
	metadata, err := attachmentFileMetadata(path, "application/octet-stream")
	if err != nil || metadata.ContentType != "application/octet-stream" || metadata.IsImage {
		t.Fatalf("metadata = %#v, error = %v", metadata, err)
	}
}

func TestAppendImageComponentKeepsPlaneEditorWrapper(t *testing.T) {
	component := `<image-component src="asset-id"></image-component>`
	wrapped := `<div><p>Existing</p></div>`
	if got := appendImageComponent(wrapped, component); got != `<div><p>Existing</p>`+component+`</div>` {
		t.Fatalf("wrapped description = %s", got)
	}
	plain := `<p>Existing</p>`
	if got := appendImageComponent(plain, component); got != plain+component {
		t.Fatalf("plain description = %s", got)
	}
}

func TestAttachmentsGetUsesWorkItemAttachmentEndpoint(t *testing.T) {
	assetID := "b9138ae0-dbca-4539-819c-0f432c59054e"
	projectID := "project-id"
	issueID := "issue-id"
	imageBytes := []byte("fake png bytes")
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/api/v1/workspaces/ws/work-items/PROJ-123/":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"project":"` + projectID + `","id":"` + issueID + `","description_html":""}`))
		case "/api/v1/workspaces/ws/projects/" + projectID + "/work-items/" + issueID + "/attachments/" + assetID + "/":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(imageBytes)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "downloaded")
	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"attachments", "get", "PROJ-123", assetID, outputPath}, renderer); err != nil {
		t.Fatal(err)
	}

	actual, err := os.ReadFile(outputPath + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(imageBytes) {
		t.Fatalf("downloaded bytes = %q", actual)
	}
	wantCalls := []string{
		"GET /api/v1/workspaces/ws/work-items/PROJ-123/",
		"GET /api/v1/workspaces/ws/projects/" + projectID + "/work-items/" + issueID + "/attachments/" + assetID + "/",
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("API calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestAttachmentsListCompleteAndDelete(t *testing.T) {
	assetID := "20745b59-e398-460d-9532-e0d56be7919"
	var calls []string
	var patchPayloads []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/work-items/PROJ-123/"):
			_, _ = writer.Write([]byte(`{"project":"project-id","id":"issue-id","description_html":""}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/":
			_, _ = writer.Write([]byte(`[{"id":"` + assetID + `","attributes":{"name":"file.pdf"}}]`))
		case request.Method == http.MethodPatch && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/"+assetID+"/":
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode patch payload: %v", err)
			}
			patchPayloads = append(patchPayloads, payload)
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodDelete && request.URL.Path == "/api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/"+assetID+"/":
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	commands := [][]string{
		{"attachments", "list", "PROJ-123", "-f", "json"},
		{"attachments", "complete", "PROJ-123", assetID, "-f", "json"},
		{"attachments", "delete", "PROJ-123", assetID, "-f", "json"},
	}
	for _, command := range commands {
		if err := Run(command, renderer); err != nil {
			t.Fatalf("%v: %v", command, err)
		}
	}
	wantCalls := []string{
		"GET /api/v1/workspaces/ws/work-items/PROJ-123/",
		"GET /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/",
		"GET /api/v1/workspaces/ws/work-items/PROJ-123/",
		"PATCH /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/" + assetID + "/",
		"GET /api/v1/workspaces/ws/work-items/PROJ-123/",
		"DELETE /api/v1/workspaces/ws/projects/project-id/work-items/issue-id/attachments/" + assetID + "/",
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("API calls = %#v, want %#v", calls, wantCalls)
	}
	if len(patchPayloads) != 1 || patchPayloads[0]["is_uploaded"] != true {
		t.Fatalf("patch payloads = %#v", patchPayloads)
	}
}
