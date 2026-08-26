package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HonLuk/my-plane/internal/api"
)

const assetUUIDPattern = `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`

var (
	assetUUIDRE = regexp.MustCompile(`(?i)^` + assetUUIDPattern + `$`)
)

var imageMIMEByExtension = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func (r *runner) runAttachments(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		r.output.Println(`Usage: plane attachments <list|get|upload|complete|delete> [options]

Manage work-item file attachments. The upload command also inserts image attachments into the description.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.runAttachmentsList(args[1:])
	case "get":
		return r.runAttachmentsGet(args[1:])
	case "upload":
		return r.runAttachmentsUpload(args[1:])
	case "complete":
		return r.runAttachmentsComplete(args[1:])
	case "delete":
		return r.runAttachmentsDelete(args[1:])
	default:
		return fmt.Errorf("unknown attachments subcommand %q", args[0])
	}
}

func (r *runner) runAttachmentsList(args []string) error {
	set := r.newFlagSet("attachments list", "Usage: plane attachments list PROJECT-SEQ [options]\n\nList all file attachments for a work item.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 1, "plane attachments list PROJECT-SEQ"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{set.Args()[0], "issue short ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, _, err := r.resolveWorkItem(client, set.Args()[0])
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", attachmentCollectionEndpoint(client, projectID, issueID), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) runAttachmentsGet(args []string) error {
	set := r.newFlagSet("attachments get", "Usage: plane attachments get PROJECT-SEQ RESOURCE_UUID OUTPUT_FILE [options]\n\nDownload an attachment from a work item.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 3, "plane attachments get PROJECT-SEQ RESOURCE_UUID OUTPUT_FILE"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, err := r.resolveAttachmentTarget(client, set.Args()[0], set.Args()[1])
	if err != nil {
		return err
	}
	r.output.Println(r.output.Dim("Downloading attachment " + set.Args()[1] + "..."))
	result, finalPath, err := downloadAttachment(client, projectID, issueID, set.Args()[1], set.Args()[2])
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Attachment saved to " + finalPath))
	r.output.Println(r.output.Dim(fmt.Sprintf("  Size: %s bytes | Type: %s", formatNumber(result.Size), result.ContentType)))
	if options.format == "json" {
		r.printJSON(map[string]any{"asset_id": set.Args()[1], "path": finalPath, "size": result.Size, "content_type": result.ContentType})
	}
	return nil
}

func (r *runner) runAttachmentsUpload(args []string) error {
	set := r.newFlagSet("attachments upload", "Usage: plane attachments upload PROJECT-SEQ FILE [options]\n\nUpload a file as a work-item attachment. Images are also inserted into the description.")
	options := addCommon(set, false)
	contentType := set.String("type", "", "MIME type (detected from the file extension when omitted)")
	set.StringVar(contentType, "content-type", "", "MIME type (alias for --type)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 2, "plane attachments upload PROJECT-SEQ FILE"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	issueShort, filePath := set.Args()[0], set.Args()[1]
	metadata, err := attachmentFileMetadata(filePath, *contentType)
	if err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, description, err := r.resolveWorkItem(client, issueShort)
	if err != nil {
		return err
	}
	attachmentCollection := attachmentCollectionEndpoint(client, projectID, issueID)
	var assetID string
	metadataResponse, err := r.request(client, "POST", attachmentCollection, map[string]any{
		"name": metadata.Name,
		"type": metadata.ContentType,
		"size": metadata.Size,
	})
	confirmed := false
	if err == nil {
		object, ok := metadataResponse.(map[string]any)
		if !ok {
			err = errors.New("The API returned an invalid attachment response.")
		} else {
			assetID, _ = object["asset_id"].(string)
			if !assetUUIDRE.MatchString(assetID) {
				err = errors.New("The API returned an invalid attachment ID.")
			} else if _, ok := object["upload_data"].(map[string]any); !ok {
				err = errors.New("The API did not return valid attachment upload data.")
			} else if uploadErr := api.UploadSignedFile(object["upload_data"], filePath, metadata.ContentType); uploadErr != nil {
				err = uploadErr
			} else {
				_, err = r.request(client, "PATCH", attachmentEndpoint(client, projectID, issueID, assetID), map[string]any{"is_uploaded": true})
				confirmed = err == nil
			}
		}
	}
	if err != nil {
		if assetID != "" && !confirmed {
			_, _ = r.request(client, "DELETE", attachmentEndpoint(client, projectID, issueID, assetID), nil)
		}
		return fmt.Errorf("Attachment upload failed: %s", err)
	}

	inserted := false
	imageComponent := ""
	if metadata.IsImage {
		imageComponent = `<image-component src="` + escapeHTMLText(assetID) + `"></image-component>`
		updatedDescription := appendImageComponent(description, imageComponent)
		issueEndpoint := "/workspaces/" + client.Workspace + "/projects/" + projectID + "/work-items/" + issueID + "/"
		if _, err := r.request(client, "PATCH", issueEndpoint, map[string]any{"description_html": updatedDescription}); err != nil {
			return fmt.Errorf("Attachment uploaded, but image insertion failed; attachment retained with asset ID %s: %s", assetID, err)
		}
		inserted = true
	}

	result := map[string]any{
		"asset_id":     assetID,
		"asset_url":    attachmentAssetURL(client, projectID, issueID, assetID),
		"name":         metadata.Name,
		"content_type": metadata.ContentType,
		"size":         metadata.Size,
		"inserted":     inserted,
	}
	if imageComponent != "" {
		result["image_component"] = imageComponent
	}
	if options.format == "json" {
		r.printJSON(result)
		return nil
	}
	r.output.Println(r.output.Green("✓ Attachment uploaded"))
	if inserted {
		r.output.Println(r.output.Green("✓ Image inserted into description"))
	}
	r.output.Println("  Asset ID: " + assetID)
	r.output.Println("  Asset URL: " + result["asset_url"].(string))
	r.output.Println("  Name: " + metadata.Name)
	r.output.Println("  Type: " + metadata.ContentType)
	r.output.Println(fmt.Sprintf("  Size: %s bytes", formatNumber(metadata.Size)))
	if imageComponent != "" {
		r.output.Println("  Image component: " + imageComponent)
	}
	return nil
}

type attachmentMetadata struct {
	Name        string
	ContentType string
	Size        int64
	IsImage     bool
}

func attachmentFileMetadata(path, explicitContentType string) (attachmentMetadata, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return attachmentMetadata{}, fmt.Errorf("Attachment file not found: %s", path)
	}
	extension := strings.ToLower(filepath.Ext(path))
	contentType := strings.TrimSpace(explicitContentType)
	if contentType == "" {
		contentType = imageMIMEByExtension[extension]
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(extension)
	}
	if contentType == "" {
		return attachmentMetadata{}, fmt.Errorf("Could not determine MIME type for '%s'. Pass --type.", displayExtension(extension))
	}
	contentType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	return attachmentMetadata{
		Name:        filepath.Base(path),
		ContentType: contentType,
		Size:        info.Size(),
		IsImage:     strings.HasPrefix(contentType, "image/"),
	}, nil
}

func displayExtension(value string) string {
	if value == "" {
		return "[no extension]"
	}
	return value
}

func (r *runner) resolveWorkItem(client *api.Client, issueShort string) (string, string, string, error) {
	if err := validateIDs(struct{ value, name string }{issueShort, "issue short ID"}); err != nil {
		return "", "", "", err
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/work-items/"+issueShort+"/", nil)
	if err != nil {
		return "", "", "", err
	}
	object, ok := data.(map[string]any)
	if !ok || len(object) == 0 {
		return "", "", "", errors.New("Work item not found.")
	}
	projectID, _ := object["project"].(string)
	issueID, _ := object["id"].(string)
	if projectID == "" || issueID == "" {
		return "", "", "", errors.New("Could not determine project/issue ID from work item.")
	}
	description, _ := object["description_html"].(string)
	return projectID, issueID, description, nil
}

func (r *runner) runAttachmentsComplete(args []string) error {
	set := r.newFlagSet("attachments complete", "Usage: plane attachments complete PROJECT-SEQ RESOURCE_UUID [options]\n\nMark an attachment as uploaded after the file transfer succeeds.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 2, "plane attachments complete PROJECT-SEQ RESOURCE_UUID"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, err := r.resolveAttachmentTarget(client, set.Args()[0], set.Args()[1])
	if err != nil {
		return err
	}
	if _, err := r.request(client, "PATCH", attachmentEndpoint(client, projectID, issueID, set.Args()[1]), map[string]any{"is_uploaded": true}); err != nil {
		return err
	}
	result := map[string]any{"asset_id": set.Args()[1], "is_uploaded": true}
	if options.format == "json" {
		r.printJSON(result)
		return nil
	}
	r.output.Println(r.output.Green("✓ Attachment marked as uploaded"))
	r.output.Println("  Asset ID: " + set.Args()[1])
	return nil
}

func (r *runner) runAttachmentsDelete(args []string) error {
	set := r.newFlagSet("attachments delete", "Usage: plane attachments delete PROJECT-SEQ RESOURCE_UUID [options]\n\nDelete an attachment from a work item.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 2, "plane attachments delete PROJECT-SEQ RESOURCE_UUID"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, err := r.resolveAttachmentTarget(client, set.Args()[0], set.Args()[1])
	if err != nil {
		return err
	}
	if _, err := r.request(client, "DELETE", attachmentEndpoint(client, projectID, issueID, set.Args()[1]), nil); err != nil {
		return err
	}
	result := map[string]any{"asset_id": set.Args()[1], "deleted": true}
	if options.format == "json" {
		r.printJSON(result)
		return nil
	}
	r.output.Println(r.output.Green("✓ Attachment deleted"))
	r.output.Println("  Asset ID: " + set.Args()[1])
	return nil
}

func (r *runner) resolveAttachmentTarget(client *api.Client, issueShort, assetID string) (string, string, error) {
	if err := validateIDs(
		struct{ value, name string }{issueShort, "issue short ID"},
		struct{ value, name string }{assetID, "resource ID"},
	); err != nil {
		return "", "", err
	}
	projectID, issueID, _, err := r.resolveWorkItem(client, issueShort)
	if err != nil {
		return "", "", err
	}
	return projectID, issueID, nil
}

func attachmentCollectionEndpoint(client *api.Client, projectID, issueID string) string {
	return "/workspaces/" + client.Workspace + "/projects/" + projectID + "/work-items/" + issueID + "/attachments/"
}

func attachmentEndpoint(client *api.Client, projectID, issueID, assetID string) string {
	return attachmentCollectionEndpoint(client, projectID, issueID) + assetID + "/"
}

func attachmentAssetURL(client *api.Client, projectID, issueID, assetID string) string {
	return "/api/assets/v2/workspaces/" + client.Workspace + "/projects/" + projectID + "/issues/" + issueID + "/attachments/" + assetID + "/"
}

func appendImageComponent(description, imageComponent string) string {
	// Plane's editor commonly wraps the full body in one <div>. Insert inside
	// that wrapper so the saved HTML remains a single editor document; plain
	// descriptions without a wrapper still get the component appended.
	closingWrapper := strings.LastIndex(strings.ToLower(description), "</div>")
	if closingWrapper >= 0 && strings.TrimSpace(description[closingWrapper+len("</div>"):]) == "" {
		return description[:closingWrapper] + imageComponent + description[closingWrapper:]
	}
	return description + imageComponent
}

func downloadAttachment(client *api.Client, projectID, issueID, assetID, outputPath string) (api.BinaryResult, string, error) {
	result, err := client.DoBinary(context.Background(), attachmentEndpoint(client, projectID, issueID, assetID), outputPath)
	if err != nil {
		return api.BinaryResult{}, "", err
	}
	finalPath := outputPath
	if filepath.Ext(outputPath) == "" {
		finalPath += api.ContentTypeExtension(result.ContentType)
		if err := os.Rename(outputPath, finalPath); err != nil {
			return api.BinaryResult{}, "", fmt.Errorf("could not rename downloaded attachment: %w", err)
		}
		result.Path = finalPath
	}
	return result, finalPath, nil
}

func formatNumber(value any) string {
	switch typed := value.(type) {
	case int:
		return formatInt(int64(typed))
	case int64:
		return formatInt(typed)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func formatInt(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	digits := fmt.Sprint(value)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}
