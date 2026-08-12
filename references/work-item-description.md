# Plane Work Item Description Format

Read this reference before creating or updating a work item whose description
contains formatting, colors, links, code, or Markdown.

## Contents

- [1. Choose the input mode](#1-choose-the-input-mode)
- [2. Recommended structure](#2-recommended-structure)
- [3. Text marks and HTML blocks](#3-text-marks-and-html-blocks)
- [4. Display settings, headings, and font size](#4-display-settings-headings-and-font-size)
- [5. Text colors and backgrounds](#5-text-colors-and-backgrounds)
- [6. Markdown support](#6-markdown-support)
- [7. Complete examples](#7-complete-examples)
- [8. Safety and validation](#8-safety-and-validation)

## 1. Choose the input mode

The CLI provides two description options:

| Option | Use it for | Behavior |
| --- | --- | --- |
| `--description` | Plain text or simple Markdown | Converts a supported Markdown subset to safe HTML; complex syntax may be incomplete and produces a warning |
| `--description-html` | Full Plane editor HTML | Sends the supplied `description_html` directly; the Plane API sanitizes it |

Use `--description` for ordinary text and simple Markdown. Use
`--description-html` when exact structure, colors, background marks, custom
HTML, or complex Markdown conversion is important. Raw HTML passed through
`--description` is escaped rather than treated as markup.

## 2. Recommended structure

Pass a short work item title through `--name`, then organize the body by
meaning. A general-purpose structure is:

```html
<h2>Background</h2>
<p>Explain why the change is needed and what problem exists today.</p>

<h2>Requirements</h2>
<ol>
  <li><p>Describe the first rule or expected change.</p></li>
  <li><p>Describe the second rule or expected change.</p></li>
</ol>

<h2>Acceptance criteria</h2>
<ul>
  <li><p>State the expected visible or functional result.</p></li>
  <li><p>Cover relevant edge cases and error handling.</p></li>
</ul>

<h2>Additional information</h2>
<p>Record dependencies, links, data definitions, or open questions.</p>
```

Omit sections that do not apply. Use one paragraph element per independent
idea; do not rely on repeated spaces or blank lines to create layout.

## 3. Text marks and HTML blocks

Plane's editor supports these inline text marks. Marks can be combined and
should wrap only the text that needs the effect:

| Effect | HTML | Markdown |
| --- | --- | --- |
| Bold | `<strong>Important</strong>` | `**Important**` |
| Italic | `<em>Emphasis</em>` | `*Emphasis*` |
| Underline | `<u>Underlined</u>` | No standard Markdown syntax; use HTML |
| Strikethrough | `<s>Deprecated</s>` | `~~Deprecated~~` (GFM) |
| Inline code | `<code>field_name</code>` | `` `field_name` `` |

Use these block elements for structured content:

```html
<p>Normal paragraph text.</p>
<h1>Heading 1</h1>
<h2>Heading 2</h2>
<h3>Heading 3</h3>
<ul><li><p>Unordered item</p></li></ul>
<ol><li><p>Ordered item</p></li></ol>
<blockquote><p>Quoted content.</p></blockquote>
<pre><code class="language-python">print("hello")</code></pre>
<a href="https://example.com">Reference link</a>
```

Use `<pre><code>` for a code block. Add a language class such as
`language-python` or `language-javascript` when syntax highlighting matters.
Use a real `href` for links instead of leaving bare URLs in the body.

## 4. Display settings, headings, and font size

Do not treat editor display settings as body HTML. Plane defines `fontSize`,
`fontStyle`, and `lineSpacing` as `displayConfig` values and applies them as
classes on the outer editor container. They are not saved in
`description_html`, and the CLI cannot set them for one work item or span.

The current issue editor wrapper renders descriptions with
`displayConfig={{ fontSize: "large-font" }}`. The values `small-font`,
`large-font`, and `mobile-font` are renderer configuration values, not tags or
attributes to put in `--description-html`.

Use semantic headings for block-level size and hierarchy:

```html
<h1>Page title</h1>
<h2>Section title</h2>
<p>Regular paragraph text.</p>
```

The editor stylesheet controls the actual sizes of heading blocks and regular
paragraphs. Use the heading levels exposed by the target page (for example,
H1 or H2). Do not invent `style="font-size: ..."`, `data-font-size`, or a
Markdown extension for arbitrary per-character font size.

When using `--description`, Markdown `#` and `##` become `<h1>` and `<h2>`;
they request heading blocks, not a custom font size.

The same limitation applies to font family (`sans-serif`, `serif`,
`monospace`) and line spacing (`regular`, `small`, `mobile-regular`). For local
monospace content, use `<code>` or `<pre><code>`.

## 5. Text colors and backgrounds

Use Plane's data attributes for the built-in palette. Do not rely on CSS
`style` for these marks:

```html
<p>
  <span data-text-color="pink">Pink text</span>
  <span data-text-color="light-blue">Blue text</span>
  <span data-text-color="green">Green text</span>
</p>
<p><span data-background-color="orange">Orange background</span></p>
<p><span data-text-color="peach" data-background-color="orange">Highlighted note</span></p>
```

Supported palette keys are `gray`, `peach`, `pink`, `orange`, `green`,
`light-blue`, `dark-blue`, and `purple`.

Rules:

- Set text color with `data-text-color`.
- Set inline background color with `data-background-color`.
- Put both attributes on the same `<span>` when both effects are needed.
- Do not replace palette keys with `style="color: ..."`,
  `style="background-color: ..."`, or arbitrary CSS color names.
- Keep a span as small as possible so unrelated text is not colored.
- `data-background` belongs to Plane callout components; it is not a
  replacement for `data-background-color` on an inline span.

Markdown has no standard syntax for the Plane palette, underline, or arbitrary
background styling. Use `--description-html` for those features.

## 6. Markdown support

The web editor supports common Markdown/GFM content when Markdown text is
pasted into the editor. The CLI also accepts simple Markdown through
`--description` and converts it to HTML before calling the API.

The simple converter supports the common forms below:

````markdown
# Summary

Add **a new metric** and verify the `field_name` value.

- First acceptance criterion
- Second acceptance criterion

1. First ordered step
2. Second ordered step

> A note for reviewers

```python
print("example")
```
````

Complex Markdown can be only partially converted. Tables, nested lists, task
lists, images, footnotes, raw HTML, and unusual nesting produce a warning and
may not render as intended. For those cases, write the final HTML explicitly
and pass it with `--description-html`.

Do not pass Markdown to `--description-html`: that option expects HTML. Do not
expect raw HTML inside `--description` to remain active; it is escaped for
safety. If Markdown needs Plane colors or backgrounds, convert the surrounding
structure to HTML and use `--description-html`.

## 7. Complete examples

The commands below assume the skill's `PLANE_CLI` variable has been set as
described in `SKILL.md`.

Simple Markdown through the CLI:

```bash
"$PLANE_CLI" issues create -p PROJECT_ID --name "Improve reporting" \
  --description '# Summary

Add **a new metric** and verify the `field_name` value.

- Display the value in the expected position
- Cover the empty-data case'
```

Explicit Plane HTML for exact formatting:

```bash
"$PLANE_CLI" issues create -p PROJECT_ID --name "Improve reporting" \
  --description-html '<h2>Requirements</h2><p><strong>New metric</strong> uses the agreed data definition.</p><h2>Acceptance criteria</h2><ul><li><p><span data-text-color="peach" data-background-color="orange">The value is highlighted for review.</span></p></li><li><p>Empty data has a defined result.</p></li></ul>'
```

Use the same `--description-html` option when updating an existing work item:

```bash
"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_ID \
  --description-html '<p><span data-text-color="pink">Updated note</span></p>'
```

## 8. Safety and validation

- Treat user-provided text as text. Escape `<`, `>`, `&`, and quotes before interpolating it into hand-written HTML.
- Do not send scripts, event-handler attributes such as `onclick`, or unnecessary inline styles. The API sanitizes content, but sanitization is not a formatting strategy.
- After a write, inspect `"$PLANE_CLI" issues get -p PROJECT_ID ISSUE_ID -f json` and verify the returned `description_html`.
- If literal `<h2>` or `<span>` appears, check whether the wrong option was used.
- If a color does not render, verify the attribute name and palette key.
- Confirm the workspace, project ID, and work item target before writing.
