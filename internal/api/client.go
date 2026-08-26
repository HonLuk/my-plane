// Package api contains the authenticated Plane REST API client and the
// binary transport helpers used by the attachment and image commands.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.plane.so"
	apiPrefix      = "/api/v1"
	jsonTimeout    = 30 * time.Second
	binaryTimeout  = 60 * time.Second
)

// Client holds runtime configuration for one Plane workspace.
type Client struct {
	APIKey    string
	Workspace string
	BaseURL   string

	jsonHTTP   *http.Client
	binaryHTTP *http.Client
}

// APIError is an error returned by Plane or by a transport operation. The
// status code is kept for multi-step flows that need to report a safe,
// context-specific cleanup message.
type APIError struct {
	Message    string
	StatusCode int
}

func (e *APIError) Error() string { return e.Message }

// BinaryResult describes a downloaded asset after it has been written to disk.
type BinaryResult struct {
	ContentType string
	Size        int
	Path        string
}

// NewClientFromEnv loads configuration only when a command is about to call
// the API. Help output therefore remains usable without credentials.
func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("PLANE_API_KEY")
	workspace := os.Getenv("PLANE_WORKSPACE")
	if apiKey == "" {
		return nil, errors.New("Error: PLANE_API_KEY is not set.\n\nTo get an API key:\n  1. Open Plane → Profile Settings → Personal Access Tokens\n  2. Create a new token and copy it\n  3. Export it:  export PLANE_API_KEY=\"your-token\"\n")
	}
	if workspace == "" {
		return nil, errors.New("Error: PLANE_WORKSPACE is not set.\n\nSet it to your workspace slug (the part after plane.so/):\n  export PLANE_WORKSPACE=\"my-workspace\"\n")
	}

	baseURL := strings.TrimRight(os.Getenv("PLANE_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return nil, errors.New("Invalid PLANE_BASE_URL: expected an http:// or https:// URL.")
	}

	return &Client{
		APIKey:    apiKey,
		Workspace: workspace,
		BaseURL:   baseURL,
		jsonHTTP:  &http.Client{Timeout: jsonTimeout},
		binaryHTTP: &http.Client{
			Timeout: binaryTimeout,
		},
	}, nil
}

// ValidateID rejects path traversal and path separator characters before a
// user-provided identifier is inserted into an API endpoint.
func ValidateID(value, name string) error {
	if strings.ContainsAny(value, `/\\`) || strings.Contains(value, "..") {
		return fmt.Errorf("Invalid %s: contains illegal characters", name)
	}
	return nil
}

// DoJSON performs an authenticated JSON API request. A nil payload means that
// the request has no body; a 204 response returns a nil value.
func (c *Client) DoJSON(ctx context.Context, method, endpoint string, payload any) (any, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("could not encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := c.newRequest(ctx, method, endpoint, body, true)
	if err != nil {
		return nil, err
	}
	response, err := c.jsonHTTP.Do(request)
	if err != nil {
		return nil, transportError(err, jsonTimeout)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read API response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, apiHTTPError(response.StatusCode, responseBody)
	}
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var value any
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("API returned an invalid JSON response.")
	}
	return value, nil
}

// DoBinary downloads an authenticated asset and writes it to outputPath.
func (c *Client) DoBinary(ctx context.Context, endpoint, outputPath string) (BinaryResult, error) {
	request, err := c.newRequest(ctx, http.MethodGet, endpoint, nil, false)
	if err != nil {
		return BinaryResult{}, err
	}
	response, err := c.binaryHTTP.Do(request)
	if err != nil {
		return BinaryResult{}, transportError(err, binaryTimeout)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return BinaryResult{}, fmt.Errorf("could not read downloaded asset: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return BinaryResult{}, apiHTTPError(response.StatusCode, responseBody)
	}
	if err := os.WriteFile(outputPath, responseBody, 0o644); err != nil {
		return BinaryResult{}, fmt.Errorf("could not save downloaded asset: %w", err)
	}

	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return BinaryResult{
		ContentType: contentType,
		Size:        len(responseBody),
		Path:        outputPath,
	}, nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body io.Reader, jsonRequest bool) (*http.Request, error) {
	requestURL := c.BaseURL + apiPrefix + endpoint
	parsed, err := url.Parse(requestURL)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return nil, errors.New("Invalid URL scheme for API request.")
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("could not create API request: %w", err)
	}
	request.Header.Set("X-API-Key", c.APIKey)
	if jsonRequest {
		request.Header.Set("Content-Type", "application/json")
	}
	return request, nil
}

func transportError(err error, timeout time.Duration) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("Request timed out (%s). The Plane API may be unreachable.", timeout)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return fmt.Errorf("Request timed out (%s). The Plane API may be unreachable.", timeout)
		}
		return fmt.Errorf("Connection error: %v", urlErr.Err)
	}
	return fmt.Errorf("Connection error: %v", err)
}

func apiHTTPError(statusCode int, body []byte) error {
	errorBody := "Request failed."
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err == nil {
		if sanitized, err := json.MarshalIndent(SanitizeErrorValue(value), "", "  "); err == nil {
			errorBody = string(sanitized)
		}
	}
	return &APIError{
		Message:    fmt.Sprintf("API Error %d: %s", statusCode, errorBody),
		StatusCode: statusCode,
	}
}

var sensitiveErrorKeys = map[string]struct{}{
	"authorization": {},
	"credential":    {},
	"fields":        {},
	"policy":        {},
	"secret":        {},
	"signature":     {},
	"token":         {},
	"upload_data":   {},
	"url":           {},
}

// SanitizeErrorValue recursively removes signed-upload fields and URLs before
// an API response reaches terminal output.
func SanitizeErrorValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, item := range typed {
			lowerKey := strings.ToLower(key)
			if _, sensitive := sensitiveErrorKeys[lowerKey]; sensitive || strings.HasPrefix(lowerKey, "x-amz-") {
				sanitized[key] = "[redacted]"
			} else {
				sanitized[key] = SanitizeErrorValue(item)
			}
		}
		return sanitized
	case []any:
		items := make([]any, len(typed))
		for index, item := range typed {
			items[index] = SanitizeErrorValue(item)
		}
		return items
	case string:
		if strings.Contains(typed, "http://") || strings.Contains(typed, "https://") {
			return "[redacted URL]"
		}
		return typed
	default:
		return value
	}
}

// UploadSignedFile sends Plane's server-provided form fields and the file to
// object storage. It intentionally does not send the Plane API key.
func UploadSignedFile(uploadData any, filePath, contentType string) error {
	data, ok := uploadData.(map[string]any)
	if !ok {
		return errors.New("The API returned invalid signed upload data.")
	}
	uploadURL, ok := data["url"].(string)
	if !ok || !isHTTPURL(uploadURL) {
		return errors.New("The API returned invalid signed upload data.")
	}
	fields, ok := data["fields"].(map[string]any)
	if !ok {
		return errors.New("The API returned invalid signed upload data.")
	}
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("Could not read attachment file: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for fieldName, fieldValue := range fields {
		if err := writer.WriteField(safeMultipartValue(fieldName), safeMultipartValue(fmt.Sprint(fieldValue))); err != nil {
			return errors.New("Could not build signed attachment upload request.")
		}
	}
	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, safeMultipartValue(filepath.Base(filePath))))
	fileHeader.Set("Content-Type", safeMultipartValue(contentType))
	part, err := writer.CreatePart(fileHeader)
	if err != nil {
		return errors.New("Could not build signed attachment upload request.")
	}
	if _, err := part.Write(fileBytes); err != nil {
		return errors.New("Could not build signed attachment upload request.")
	}
	if err := writer.Close(); err != nil {
		return errors.New("Could not build signed attachment upload request.")
	}

	request, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(body.Bytes()))
	if err != nil {
		return errors.New("The API returned invalid signed upload data.")
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Content-Length", fmt.Sprintf("%d", body.Len()))
	response, err := (&http.Client{Timeout: binaryTimeout}).Do(request)
	if err != nil {
		return errors.New("Signed attachment upload could not reach object storage.")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &APIError{Message: fmt.Sprintf("Signed attachment upload failed (HTTP %d).", response.StatusCode), StatusCode: response.StatusCode}
	}
	return nil
}

func safeMultipartValue(value string) string {
	return strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(value)
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// ContentTypeExtension returns the extension used by downloaded files.
func ContentTypeExtension(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	switch strings.ToLower(mediaType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
