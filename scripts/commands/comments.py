"""
Comment Commands

Commands for managing comments and activity on work items.
"""

import sys
import os
import json
import re

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from html import escape as html_escape

from plane_api import api, get_workspace, _validate_id
from plane_output import format_output, green, dim, bold, _resolve


def cmd_comments_list(args):
    """List comments and activity on a work item."""
    _validate_id(args.project, "project ID")
    _validate_id(args.issue, "issue ID")

    if args.all:
        # Use /activities/ endpoint for full activity history (comments + field changes)
        endpoint = f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/activities/"
        data = api("GET", endpoint)

        if isinstance(data, dict) and "results" in data:
            items = data["results"]
        elif isinstance(data, list):
            items = data
        else:
            items = []

        if args.format == "json":
            print(json.dumps(items, indent=2))
            return

        if not items:
            print(dim("No activity found."))
            return

        for item in items:
            actor = _resolve(item.get("actor_detail", item.get("actor", "?")))
            comment = item.get("comment", item.get("new_value", ""))
            field = item.get("field", "")
            created = str(item.get("created_at", ""))[:19]

            if field == "comment":
                print(f"  {bold(actor)}  {dim(created)}")
                print(f"    {comment}")
                print()
            else:
                verb = item.get("verb", "changed")
                print(f"  {dim(created)}  {actor} {verb} {field}: {item.get('old_value', '—')} → {item.get('new_value', '—')}")
    else:
        # Use /comments/ endpoint for comments only
        endpoint = f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/comments/"
        data = api("GET", endpoint)

        if isinstance(data, dict) and "results" in data:
            items = data["results"]
        elif isinstance(data, list):
            items = data
        else:
            items = []

        if args.format == "json":
            print(json.dumps(items, indent=2))
            return

        if not items:
            print(dim("No comments found."))
            return

        for item in items:
            actor = _resolve(item.get("actor_detail", item.get("actor", "?")))
            comment_html = item.get("comment_html", "")
            # Strip HTML tags for terminal display
            comment = re.sub(r'<[^>]+>', '', comment_html).strip()
            created = str(item.get("created_at", ""))[:19]
            print(f"  {bold(actor)}  {dim(created)}")
            print(f"    {comment}")
            print()


def cmd_comments_add(args):
    """Add a comment to a work item."""
    _validate_id(args.project, "project ID")
    _validate_id(args.issue, "issue ID")
    payload = {
        "comment_html": f"<p>{html_escape(args.body)}</p>",
    }
    data = api("POST", f"/workspaces/{get_workspace()}/projects/{args.project}/work-items/{args.issue}/comments/", payload)
    print(green("✓ Comment added"))
    if data and args.format == "json":
        print(json.dumps(data, indent=2))