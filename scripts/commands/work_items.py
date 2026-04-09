"""
Work Item (Issue) Commands

Commands for managing work items (issues) in projects.
"""

import sys
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from html import escape as html_escape
from urllib.parse import urlencode

from plane_api import api, get_workspace, _validate_id
from plane_output import format_output, green, yellow, PRIORITY_LABELS


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
    if args.description:
        payload["description_html"] = f"<p>{html_escape(args.description)}</p>"
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
    if args.description:
        payload["description_html"] = f"<p>{html_escape(args.description)}</p>"
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