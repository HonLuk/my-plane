// Package markdown converts the deliberately small Markdown subset accepted
// by --description into safe Plane editor HTML.
package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	inlineCodeRE    = regexp.MustCompile("`([^`\\n]+)`")
	linkRE          = regexp.MustCompile(`\[([^\]]+)\]\(((https?://|mailto:|tel:)[^\s)]+)\)`)
	strongRE        = regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	strikeRE        = regexp.MustCompile(`~~(.+?)~~`)
	emphasisStarRE  = regexp.MustCompile(`\*([^*\n]+)\*`)
	emphasisUnderRE = regexp.MustCompile(`(^|[^\w_])_([^_\n]+)_([^\w_]|$)`)

	tableRE      = regexp.MustCompile(`(?m)^\s*\|.*\|\s*$`)
	nestedListRE = regexp.MustCompile(`(?m)^\s{2,}[-+*]\s+`)
	taskListRE   = regexp.MustCompile(`(?m)^\s*[-+*]\s+\[[ xX]\]\s+`)
	imageRE      = regexp.MustCompile(`!\[[^\]]*\]\(`)
	footnoteRE   = regexp.MustCompile(`\[\^[^\]]+\]`)
	rawHTMLRE    = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	fenceRE      = regexp.MustCompile("^\\s{0,3}([`~]{3,})([A-Za-z0-9_+-]*)\\s*$")
	headingRE    = regexp.MustCompile(`^\s{0,3}(#{1,6})\s+(.+?)\s*#*\s*$`)
	horizontalRE = regexp.MustCompile(`^\s{0,3}((\*\s*){3,}|(-\s*){3,}|(_\s*){3,})$`)
	quoteRE      = regexp.MustCompile(`^\s{0,3}>\s?(.*)$`)
	unorderedRE  = regexp.MustCompile(`^\s*[-+*]\s+(.*)$`)
	orderedRE    = regexp.MustCompile(`^\s*\d+[.)]\s+(.*)$`)
)

// Warnings reports constructs that the simple converter intentionally does
// not try to reproduce exactly.
func Warnings(value string) []string {
	checks := []struct {
		pattern *regexp.Regexp
		message string
	}{
		{tableRE, "tables may not convert correctly"},
		{nestedListRE, "nested lists may not convert correctly"},
		{taskListRE, "task lists may not convert correctly"},
		{imageRE, "images should be added through the Plane editor"},
		{footnoteRE, "footnotes are not supported by the simple converter"},
		{rawHTMLRE, "raw HTML is escaped; use --description-html for HTML"},
	}
	warnings := make([]string, 0, len(checks))
	for _, check := range checks {
		if check.pattern.MatchString(value) {
			warnings = append(warnings, check.message)
		}
	}
	return warnings
}

// Inline converts inline Markdown after escaping user text. It protects
// generated code and links while applying the remaining simple marks.
func Inline(value string) string {
	escaped := escapeHTML(value, true)
	protected := make([]string, 0, 4)
	protect := func(content string) string {
		token := fmt.Sprintf("\x00%d\x00", len(protected))
		protected = append(protected, content)
		return token
	}

	escaped = inlineCodeRE.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := inlineCodeRE.FindStringSubmatch(match)
		return protect("<code>" + parts[1] + "</code>")
	})
	escaped = linkRE.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := linkRE.FindStringSubmatch(match)
		return protect(`<a href="` + parts[2] + `">` + parts[1] + `</a>`)
	})
	escaped = strongRE.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := strongRE.FindStringSubmatch(match)
		content := parts[1]
		if content == "" {
			content = parts[2]
		}
		return "<strong>" + content + "</strong>"
	})
	escaped = strikeRE.ReplaceAllString(escaped, "<s>$1</s>")
	escaped = emphasisStarRE.ReplaceAllString(escaped, "<em>$1</em>")
	escaped = emphasisUnderRE.ReplaceAllString(escaped, "$1<em>$2</em>$3")
	for index, content := range protected {
		escaped = strings.ReplaceAll(escaped, fmt.Sprintf("\x00%d\x00", index), content)
	}
	return escaped
}

// ToHTML converts block-level Markdown into the Plane editor HTML subset.
func ToHTML(value string) string {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n"), "\n")
	blocks := make([]string, 0)
	paragraph := make([]string, 0)
	listTag := ""
	listItems := make([]string, 0)
	quoteLines := make([]string, 0)
	codeLines := []string(nil)
	codeLanguage := ""

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		parts := make([]string, len(paragraph))
		for index, line := range paragraph {
			parts[index] = strings.TrimSpace(line)
		}
		blocks = append(blocks, "<p>"+Inline(strings.Join(parts, " "))+"</p>")
		paragraph = paragraph[:0]
	}
	flushList := func() {
		if listTag == "" {
			return
		}
		blocks = append(blocks, "<"+listTag+">"+strings.Join(listItems, "")+"</"+listTag+">")
		listTag = ""
		listItems = listItems[:0]
	}
	flushQuote := func() {
		if len(quoteLines) == 0 {
			return
		}
		parts := make([]string, len(quoteLines))
		for index, line := range quoteLines {
			parts[index] = strings.TrimSpace(line)
		}
		blocks = append(blocks, "<blockquote><p>"+Inline(strings.Join(parts, " "))+"</p></blockquote>")
		quoteLines = quoteLines[:0]
	}
	flushOpen := func() {
		flushParagraph()
		flushList()
		flushQuote()
	}

	for _, line := range lines {
		if codeLines != nil {
			if matches := fenceRE.FindStringSubmatch(line); matches != nil {
				class := ""
				if codeLanguage != "" {
					class = ` class="language-` + escapeHTML(codeLanguage, true) + `"`
				}
				blocks = append(blocks, "<pre><code"+class+">"+escapeHTML(strings.Join(codeLines, "\n"), false)+"</code></pre>")
				codeLines = nil
				codeLanguage = ""
			} else {
				codeLines = append(codeLines, line)
			}
			continue
		}

		if matches := fenceRE.FindStringSubmatch(line); matches != nil {
			flushOpen()
			codeLines = make([]string, 0)
			codeLanguage = strings.TrimSpace(matches[2])
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushOpen()
			continue
		}
		if matches := headingRE.FindStringSubmatch(line); matches != nil {
			flushOpen()
			blocks = append(blocks, fmt.Sprintf("<h%d>%s</h%d>", len(matches[1]), Inline(matches[2]), len(matches[1])))
			continue
		}
		if horizontalRE.MatchString(line) {
			flushOpen()
			blocks = append(blocks, "<hr>")
			continue
		}
		if matches := quoteRE.FindStringSubmatch(line); matches != nil {
			flushParagraph()
			flushList()
			quoteLines = append(quoteLines, matches[1])
			continue
		}
		flushQuote()

		if matches := unorderedRE.FindStringSubmatch(line); matches != nil {
			flushParagraph()
			if listTag != "" && listTag != "ul" {
				flushList()
			}
			if listTag == "" {
				listTag = "ul"
			}
			listItems = append(listItems, "<li>"+Inline(matches[1])+"</li>")
			continue
		}
		if matches := orderedRE.FindStringSubmatch(line); matches != nil {
			flushParagraph()
			if listTag != "" && listTag != "ol" {
				flushList()
			}
			if listTag == "" {
				listTag = "ol"
			}
			listItems = append(listItems, "<li>"+Inline(matches[1])+"</li>")
			continue
		}
		flushList()
		paragraph = append(paragraph, line)
	}

	if codeLines != nil {
		class := ""
		if codeLanguage != "" {
			class = ` class="language-` + escapeHTML(codeLanguage, true) + `"`
		}
		blocks = append(blocks, "<pre><code"+class+">"+escapeHTML(strings.Join(codeLines, "\n"), false)+"</code></pre>")
	} else {
		flushOpen()
	}
	return strings.Join(blocks, "")
}

func escapeHTML(value string, quote bool) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	if quote {
		value = strings.ReplaceAll(value, `"`, "&quot;")
		value = strings.ReplaceAll(value, "'", "&#x27;")
	}
	return value
}
