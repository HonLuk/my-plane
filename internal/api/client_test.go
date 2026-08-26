package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoJSONUsesWorkspaceAPIKeyAndQueryParameters(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/v1/workspaces/ws/projects/" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("cursor") != "next page" {
			t.Errorf("cursor = %q", request.URL.Query().Get("cursor"))
		}
		if request.Header.Get("X-API-Key") != "secret-key" {
			t.Errorf("missing API key header")
		}
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true,"count":2}`))
	}))
	defer server.Close()

	t.Setenv("PLANE_API_KEY", "secret-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.DoJSON(context.Background(), http.MethodPost, "/workspaces/ws/projects/?cursor=next+page", map[string]any{"name": "Demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(receivedBody, `"name":"Demo"`) {
		t.Fatalf("request body = %s", receivedBody)
	}
	object, ok := value.(map[string]any)
	if !ok || object["ok"] != true {
		t.Fatalf("response = %#v", value)
	}
}

func TestDoJSONSanitizesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"url":"https://signed.example/?token=secret","fields":{"policy":"private","x-amz-signature":"secret"},"message":"bad request"}`))
	}))
	defer server.Close()
	t.Setenv("PLANE_API_KEY", "secret-key")
	t.Setenv("PLANE_WORKSPACE", "ws")
	t.Setenv("PLANE_BASE_URL", server.URL)
	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.DoJSON(context.Background(), http.MethodGet, "/users/me/", nil)
	if err == nil {
		t.Fatal("expected API error")
	}
	message := err.Error()
	for _, secret := range []string{"signed.example", "private", "secret", "x-amz-signature"} {
		if strings.Contains(message, secret) && secret != "x-amz-signature" {
			t.Errorf("error contains secret %q: %s", secret, message)
		}
	}
	if !strings.Contains(message, "API Error 400") || !strings.Contains(message, "[redacted]") {
		t.Errorf("unexpected sanitized error: %s", message)
	}
}

func TestUploadSignedFileSendsFieldsAndFileWithoutAPIKey(t *testing.T) {
	var body []byte
	var contentType string
	var apiKey string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ = io.ReadAll(request.Body)
		contentType = request.Header.Get("Content-Type")
		apiKey = request.Header.Get("X-API-Key")
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	directory := t.TempDir()
	imagePath := filepath.Join(directory, "screen.png")
	if err := os.WriteFile(imagePath, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := UploadSignedFile(map[string]any{
		"url":    server.URL,
		"fields": map[string]any{"key": "private/key", "policy": "private-policy"},
	}, imagePath, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if apiKey != "" {
		t.Fatalf("signed upload received API key %q", apiKey)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Fatalf("content type = %q", contentType)
	}
	for _, expected := range []string{`name="key"`, "private/key", `name="file"; filename="screen.png"`, "fake image bytes"} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("multipart body missing %q", expected)
		}
	}
}

func TestNewClientRequiresCredentials(t *testing.T) {
	t.Setenv("PLANE_API_KEY", "")
	t.Setenv("PLANE_WORKSPACE", "")
	if _, err := NewClientFromEnv(); err == nil || !strings.Contains(err.Error(), "PLANE_API_KEY") {
		t.Fatalf("error = %v", err)
	}
}
