package markdown

import (
	"strings"
	"testing"
)

func TestToHTMLSupportsPlaneMarkdownSubset(t *testing.T) {
	value := ToHTML("# Summary\n\nAdd **bold** and `field_name`.\n\n- First\n- Second\n\n> Note\n\n```go\nfmt.Println(\"ok\")\n```")
	want := `<h1>Summary</h1><p>Add <strong>bold</strong> and <code>field_name</code>.</p><ul><li>First</li><li>Second</li></ul><blockquote><p>Note</p></blockquote><pre><code class="language-go">fmt.Println("ok")</code></pre>`
	if value != want {
		t.Fatalf("ToHTML() = %s\nwant %s", value, want)
	}
}

func TestInlineEscapesHTMLAndAllowsSafeLinks(t *testing.T) {
	value := Inline(`[read](https://example.com?a=1&b=2) <script>alert('x')</script>`)
	if !strings.Contains(value, `<a href="https://example.com?a=1&amp;b=2">read</a>`) {
		t.Fatalf("safe link missing: %s", value)
	}
	if strings.Contains(value, "<script>") || !strings.Contains(value, "&lt;script&gt;") {
		t.Fatalf("raw HTML was not escaped: %s", value)
	}
}

func TestWarningsReportUnsupportedMarkdown(t *testing.T) {
	warnings := Warnings("| a | b |\n|---|---|\n- [ ] task\n![image](x.png)\n<div>raw</div>")
	if len(warnings) != 4 {
		t.Fatalf("warnings = %#v", warnings)
	}
}
