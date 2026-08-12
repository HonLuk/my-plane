"""
Work Item (Issue) Commands

Commands for managing work items (issues) in projects.
"""

import os
import re
import sys

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from html import escape as html_escape
from urllib.parse import urlencode

from plane_api import api, get_workspace, _validate_id
from plane_output import format_output, green, yellow, PRIORITY_LABELS


def _markdown_inline(text):
    """Convert common inline Markdown to safe HTML.

    This deliberately handles a small, predictable Markdown subset. The
    explicit HTML option remains available for content that needs complete
    control over Plane editor markup.
    """
    escaped = html_escape(text, quote=True)
    protected = []

    def protect(value):
        token = f"\x00{len(protected)}\x00"
        protected.append(value)
        return token

    escaped = re.sub(
        r"`([^`\n]+)`",
        lambda match: protect(f"<code>{match.group(1)}</code>"),
        escaped,
    )

    def replace_link(match):
        label = match.group(1)
        href = match.group(2)
        if not href.startswith(("http://", "https://", "mailto:", "tel:")):
            return match.group(0)
        return protect(f'<a href="{href}">{label}</a>')

    escaped = re.sub(
        r"\[([^\]]+)\]\(((?:https?://|mailto:|tel:)[^\s)]+)\)",
        replace_link,
        escaped,
    )
    escaped = re.sub(
        r"\*\*(.+?)\*\*|__(.+?)__",
        lambda match: f"<strong>{match.group(1) or match.group(2)}</strong>",
        escaped,
    )
    escaped = re.sub(r"~~(.+?)~~", r"<s>\1</s>", escaped)
    escaped = re.sub(r"(?<!\*)\*([^*\n]+)\*(?!\*)", r"<em>\1</em>", escaped)
    escaped = re.sub(r"(?<!\w)_([^_\n]+)_(?!\w)", r"<em>\1</em>", escaped)

    for index, value in enumerate(protected):
        escaped = escaped.replace(f"\x00{index}\x00", value)
    return escaped


def _markdown_complexity_warnings(markdown):
    """Return warnings for Markdown constructs outside the simple converter."""
    warnings = []
    checks = [
        (r"(?m)^\s*\|.*\|\s*$", "tables may not convert correctly"),
        (r"(?m)^\s{2,}[-+*]\s+", "nested lists may not convert correctly"),
        (r"(?m)^\s*[-+*]\s+\[[ xX]\]\s+", "task lists may not convert correctly"),
        (r"!\[[^\]]*\]\(", "images should be added through the Plane editor"),
        (r"\[\^[^\]]+\]", "footnotes are not supported by the simple converter"),
        (r"</?[a-zA-Z][^>]*>", "raw HTML is escaped; use --description-html for HTML"),
    ]
    for pattern, message in checks:
        if re.search(pattern, markdown):
            warnings.append(message)
    return warnings


def _markdown_to_html(markdown):
    """Convert common block-level Markdown to Plane-compatible HTML."""
    lines = markdown.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    blocks = []
    paragraph_lines = []
    list_tag = None
    list_items = []
    quote_lines = []
    code_lines = None
    code_language = ""

    def flush_paragraph():
        if paragraph_lines:
            content = " ".join(line.strip() for line in paragraph_lines)
            blocks.append(f"<p>{_markdown_inline(content)}</p>")
            paragraph_lines.clear()

    def flush_list():
        nonlocal list_tag
        if list_tag:
            blocks.append(f"<{list_tag}>" + "".join(list_items) + f"</{list_tag}>")
            list_tag = None
            list_items.clear()

    def flush_quote():
        if quote_lines:
            content = " ".join(line.strip() for line in quote_lines)
            blocks.append(f"<blockquote><p>{_markdown_inline(content)}</p></blockquote>")
            quote_lines.clear()

    def flush_open_blocks():
        flush_paragraph()
        flush_list()
        flush_quote()

    for line in lines:
        fence = re.match(r"^\s{0,3}(`{3,}|~{3,})([\w+-]*)\s*$", line)
        if code_lines is not None:
            if fence:
                language_attr = (
                    f' class="language-{html_escape(code_language, quote=True)}"' if code_language else ""
                )
                blocks.append(
                    f"<pre><code{language_attr}>{html_escape(chr(10).join(code_lines), quote=False)}</code></pre>"
                )
                code_lines = None
                code_language = ""
            else:
                code_lines.append(line)
            continue

        if fence:
            flush_open_blocks()
            code_lines = []
            code_language = fence.group(2)
            continue

        if not line.strip():
            flush_open_blocks()
            continue

        heading = re.match(r"^\s{0,3}(#{1,6})\s+(.+?)\s*#*\s*$", line)
        if heading:
            flush_open_blocks()
            level = len(heading.group(1))
            blocks.append(f"<h{level}>{_markdown_inline(heading.group(2))}</h{level}>")
            continue

        if re.match(r"^\s{0,3}((\*\s*){3,}|(-\s*){3,}|(_\s*){3,})$", line):
            flush_open_blocks()
            blocks.append("<hr>")
            continue

        quote = re.match(r"^\s{0,3}>\s?(.*)$", line)
        if quote:
            flush_paragraph()
            flush_list()
            quote_lines.append(quote.group(1))
            continue
        flush_quote()

        unordered = re.match(r"^\s*[-+*]\s+(.*)$", line)
        ordered = re.match(r"^\s*\d+[.)]\s+(.*)$", line)
        if unordered or ordered:
            flush_paragraph()
            new_list_tag = "ul" if unordered else "ol"
            if list_tag and list_tag != new_list_tag:
                flush_list()
            list_tag = list_tag or new_list_tag
            item = unordered.group(1) if unordered else ordered.group(1)
            list_items.append(f"<li>{_markdown_inline(item)}</li>")
            continue
        flush_list()

        paragraph_lines.append(line)

    if code_lines is not None:
        language_attr = f' class="language-{html_escape(code_language, quote=True)}"' if code_language else ""
        blocks.append(
            f"<pre><code{language_attr}>{html_escape(chr(10).join(code_lines), quote=False)}</code></pre>"
        )
    else:
        flush_open_blocks()

    return "".join(blocks)


def _get_description_html(args):
    """Return the description payload while preserving explicit Plane HTML.

    ``--description`` accepts plain text and common Markdown. Complex Markdown
    may not convert perfectly, so ``--description-html`` remains the explicit
    path for Plane editor markup such as ``data-text-color`` attributes.
    """
    description_html = getattr(args, "description_html", None)
    description = getattr(args, "description", None)
    if description_html is not None:
        return description_html
    if description is not None:
        for warning in _markdown_complexity_warnings(description):
            print(yellow(f"Warning: {warning}; use --description-html for full control."), file=sys.stderr)
        return _markdown_to_html(description)
    return None


def cmd_issues_list(args):
    """List work items in a project, with optional filters and pagination."""
    _validate_id(args.project, "project ID")
    endpoint = f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/"
    params = {}

    # Filters
    if args.state:
        params["state"] = args.state
    if args.priority:
        params["priority"] = args.priority
    if args.assignee:
        params["assignees"] = args.assignee

    # Pagination
    if hasattr(args, 'cursor') and args.cursor:
        params["cursor"] = args.cursor
    if hasattr(args, 'per_page') and args.per_page:
        params["per_page"] = args.per_page

    # Field selection and expansion
    if hasattr(args, 'fields') and args.fields:
        params["fields"] = args.fields
    if hasattr(args, 'expand') and args.expand:
        params["expand"] = args.expand

    # Ordering
    if hasattr(args, 'order_by') and args.order_by:
        params["order_by"] = args.order_by

    if params:
        endpoint += "?" + urlencode(params)
    data = api("GET", endpoint)
    format_output(data, args.format)


def cmd_issues_get(args):
    """Get full details of a single work item."""
    _validate_id(args.project, "project ID")
    _validate_id(args.issue, "issue ID")
    data = api("GET", f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/")
    format_output(data, args.format)


def cmd_issues_create(args):
    """Create a new work item in a project."""
    _validate_id(args.project, "project ID")
    payload = {"name": args.name}
    description_html = _get_description_html(args)
    if description_html is not None:
        payload["description_html"] = description_html
    if args.priority:
        payload["priority"] = PRIORITY_LABELS.get(args.priority.lower(), 3)
    if args.state:
        payload["state"] = args.state
    if args.assignee:
        payload["assignees"] = [args.assignee]
    if args.label:
        payload["labels"] = [args.label]

    data = api("POST", f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/", payload)
    print(green("✓ Work item created"))
    format_output(data, args.format)


def cmd_issues_update(args):
    """Update fields on an existing work item."""
    _validate_id(args.project, "project ID")
    _validate_id(args.issue, "issue ID")
    payload = {}
    if args.name:
        payload["name"] = args.name
    description_html = _get_description_html(args)
    if description_html is not None:
        payload["description_html"] = description_html
    if args.priority:
        payload["priority"] = PRIORITY_LABELS.get(args.priority.lower(), 3)
    if args.state:
        payload["state"] = args.state

    if not payload:
        print(yellow("Nothing to update — pass at least one field."), file=sys.stderr)
        sys.exit(1)

    data = api("PATCH", f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/", payload)
    print(green("✓ Work item updated"))
    format_output(data, args.format)


def cmd_issues_assign(args):
    """Assign a work item to one or more members."""
    _validate_id(args.project, "project ID")
    _validate_id(args.issue, "issue ID")
    for assignee in args.assignees:
        _validate_id(assignee, "assignee ID")
    payload = {"assignees": args.assignees}
    data = api("PATCH", f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/", payload)
    print(green(f"✓ Assigned to {len(args.assignees)} member(s)"))
    format_output(data, args.format)


def cmd_issues_delete(args):
    """Delete a work item (irreversible)."""
    _validate_id(args.project, "project ID")
    _validate_id(args.issue, "issue ID")
    api("DELETE", f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/")
    print(green(f"✓ Deleted work item {args.issue}"))


def cmd_issues_search(args):
    """Search work items across the entire workspace."""
    params = urlencode({"search": args.query, "type": "work_item"})
    endpoint = f"/workspaces/{get_workspace()}/search/?{params}"
    data = api("GET", endpoint)
    format_output(data, args.format)


def cmd_issues_get_short(args):
    """Get work item details by short ID (e.g., PROJ-123)."""
    _validate_id(args.issue_short, "issue short ID")
    data = api("GET", f"/workspaces/{get_workspace()}/work-items/{args.issue_short}/")
    format_output(data, args.format)
