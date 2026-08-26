package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/HonLuk/my-plane/internal/api"
)

var imageComponentRE = regexp.MustCompile(`(?i)<image-component[^>]+src="(` + assetUUIDPattern + `)"`)

// runGetImage is the original image download command kept for existing CLI
// users. New attachment workflows should use the attachments command group.
func (r *runner) runGetImage(args []string) error {
	set := r.newFlagSet("get-image", "Usage: plane get-image PROJECT-SEQ ASSET_UUID OUTPUT_FILE [options]\n\nDownload a specific image embedded in a work item description.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 3, "plane get-image PROJECT-SEQ ASSET_UUID OUTPUT_FILE"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	issueShort, assetID, outputPath := set.Args()[0], set.Args()[1], set.Args()[2]
	if err := validateIDs(struct{ value, name string }{assetID, "asset ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, _, err := r.resolveWorkItem(client, issueShort)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Dim("Downloading image " + assetID + "..."))
	result, finalPath, err := downloadAttachment(client, projectID, issueID, assetID, outputPath)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Attachment saved to " + finalPath))
	r.output.Println(r.output.Dim(fmt.Sprintf("  Size: %s bytes | Type: %s", formatNumber(result.Size), result.ContentType)))
	if options.format == "json" {
		r.printJSON(map[string]any{"asset_id": assetID, "path": finalPath, "size": result.Size, "content_type": result.ContentType})
	}
	return nil
}

// runGetImages downloads every image-component referenced by the work item.
// It intentionally remains separate from attachments list, which returns all
// file attachments including non-image files.
func (r *runner) runGetImages(args []string) error {
	set := r.newFlagSet("get-images", "Usage: plane get-images PROJECT-SEQ OUTPUT_DIR [options]\n\nDownload every image embedded in a work item's description.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 2, "plane get-images PROJECT-SEQ OUTPUT_DIR"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	issueShort, directory := set.Args()[0], set.Args()[1]
	client, err := r.client()
	if err != nil {
		return err
	}
	projectID, issueID, description, err := r.resolveWorkItem(client, issueShort)
	if err != nil {
		return err
	}
	assetIDs := uniqueAssetIDs(imageComponentRE.FindAllStringSubmatch(description, -1))
	if len(assetIDs) == 0 {
		r.output.Println(r.output.Yellow("No images found in this work item's description."))
		return nil
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}
	r.output.Println(r.output.Dim(fmt.Sprintf("Found %d image(s) in %s", len(assetIDs), issueShort)))
	r.output.Println()
	results := make([]map[string]any, 0, len(assetIDs))
	for index, assetID := range assetIDs {
		tmpPath := filepath.Join(directory, assetID+".tmp")
		result, err := client.DoBinary(context.Background(), attachmentEndpoint(client, projectID, issueID, assetID), tmpPath)
		if err != nil {
			_ = os.Remove(tmpPath)
			r.output.Println(r.output.Yellow(fmt.Sprintf("  [%d/%d] Failed: %s", index+1, len(assetIDs), assetID)))
			continue
		}
		extension := api.ContentTypeExtension(result.ContentType)
		finalPath := filepath.Join(directory, assetID+extension)
		if err := os.Rename(tmpPath, finalPath); err != nil {
			_ = os.Remove(tmpPath)
			r.output.Println(r.output.Yellow(fmt.Sprintf("  [%d/%d] Failed: %s", index+1, len(assetIDs), assetID)))
			continue
		}
		r.output.Println(r.output.Green(fmt.Sprintf("  [%d/%d] %s%s", index+1, len(assetIDs), assetID, extension)) + r.output.Dim(fmt.Sprintf("  (%s bytes)", formatNumber(result.Size))))
		results = append(results, map[string]any{"asset_id": assetID, "path": finalPath, "size": result.Size, "content_type": result.ContentType})
	}
	r.output.Println()
	r.output.Println(r.output.Green(fmt.Sprintf("✓ Downloaded %d/%d image(s) to %s", len(results), len(assetIDs), directory)))
	if options.format == "json" {
		r.printJSON(results)
	}
	return nil
}

func uniqueAssetIDs(matches [][]string) []string {
	seen := make(map[string]bool, len(matches))
	unique := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		unique = append(unique, match[1])
	}
	return unique
}
