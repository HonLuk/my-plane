package commands

import (
	"fmt"

	"github.com/HonLuk/my-plane/internal/markdown"
)

// contentHTML converts plain text or the supported Markdown subset into the
// HTML accepted by Plane's editor. Explicit HTML bypasses conversion so the
// caller can preserve formatting that the simple Markdown converter cannot
// represent.
func (r *runner) contentHTML(plainText, explicitHTML, plainName, htmlName string) (string, bool, error) {
	if plainText != "" && explicitHTML != "" {
		return "", false, fmt.Errorf("%s and %s are mutually exclusive", plainName, htmlName)
	}
	if explicitHTML != "" {
		return explicitHTML, true, nil
	}
	if plainText == "" {
		return "", false, nil
	}
	for _, warning := range markdown.Warnings(plainText) {
		r.output.Errorln(r.output.Yellow("Warning: " + warning + "; use " + htmlName + " for full control."))
	}
	return markdown.ToHTML(plainText), true, nil
}
