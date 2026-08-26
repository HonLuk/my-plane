package commands

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HonLuk/my-plane/internal/output"
)

func TestGetImageUsesWorkItemAttachmentEndpoint(t *testing.T) {
	assetID := "b9138ae0-dbca-4539-819c-0f432c59054e"
	projectID := "project-id"
	issueID := "issue-id"
	imageBytes := []byte("fake image bytes")
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
	if err := Run([]string{"get-image", "PROJ-123", assetID, outputPath}, renderer); err != nil {
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

func TestGetImagesUsesAttachmentEndpointDeduplicatesAndContinues(t *testing.T) {
	assetID := "b9138ae0-dbca-4539-819c-0f432c59054e"
	missingAssetID := "20745b59-e398-460d-9532-e0d56fbe7919"
	projectID := "project-id"
	issueID := "issue-id"
	description := `<image-component src="` + assetID + `"></image-component>` +
		`<image-component src="` + assetID + `"></image-component>` +
		`<image-component src="` + missingAssetID + `"></image-component>`
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/api/v1/workspaces/ws/work-items/PROJ-123/":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"project":"` + projectID + `","id":"` + issueID + `","description_html":` + strconv.Quote(description) + `}`))
		case "/api/v1/workspaces/ws/projects/" + projectID + "/work-items/" + issueID + "/attachments/" + assetID + "/":
			writer.Header().Set("Content-Type", "image/jpeg")
			_, _ = writer.Write([]byte("valid image"))
		case "/api/v1/workspaces/ws/projects/" + projectID + "/work-items/" + issueID + "/attachments/" + missingAssetID + "/":
			http.Error(writer, `{"error":"missing"}`, http.StatusNotFound)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	directory := t.TempDir()
	t.Setenv("PLANE_API_KEY", "test-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	var out bytes.Buffer
	renderer := output.NewPlainRenderer(&out, &bytes.Buffer{})
	if err := Run([]string{"get-images", "PROJ-123", directory}, renderer); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(directory, assetID+".jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, missingAssetID+".jpg")); !os.IsNotExist(err) {
		t.Fatalf("missing asset file error = %v", err)
	}
	if !strings.Contains(out.String(), "Downloaded 1/2") || !strings.Contains(out.String(), "Failed: "+missingAssetID) {
		t.Fatalf("output = %s", out.String())
	}
	wantCalls := []string{
		"GET /api/v1/workspaces/ws/work-items/PROJ-123/",
		"GET /api/v1/workspaces/ws/projects/" + projectID + "/work-items/" + issueID + "/attachments/" + assetID + "/",
		"GET /api/v1/workspaces/ws/projects/" + projectID + "/work-items/" + issueID + "/attachments/" + missingAssetID + "/",
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("API calls = %#v, want %#v", calls, wantCalls)
	}
}
