"""
Cycle Commands

Commands for managing cycles (sprints) in projects.
"""

import sys
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from urllib.parse import urlencode

from plane_api import api, get_workspace, _validate_id
from plane_output import format_output, green


def cmd_cycles_list(args):
    """List cycles (sprints) in a project with pagination support."""
    _validate_id(args.project, "project ID")
    endpoint = f"/workspaces/{get_workspace()}/projects/{args.project}/cycles/"
    params = {}

    if hasattr(args, 'cursor') and args.cursor:
        params["cursor"] = args.cursor
    if hasattr(args, 'per_page') and args.per_page:
        params["per_page"] = args.per_page

    if params:
        endpoint += "?" + urlencode(params)
    data = api("GET", endpoint)
    format_output(data, args.format)


def cmd_cycles_get(args):
    """Get details for a specific cycle."""
    _validate_id(args.project, "project ID")
    _validate_id(args.cycle, "cycle ID")
    data = api("GET", f"/workspaces/{get_workspace()}/projects/{args.project}/cycles/{args.cycle}/")
    format_output(data, args.format)


def cmd_cycles_create(args):
    """Create a new cycle in a project."""
    _validate_id(args.project, "project ID")
    payload = {"name": args.name}
    if args.start:
        payload["start_date"] = args.start
    if args.end:
        payload["end_date"] = args.end
    data = api("POST", f"/workspaces/{get_workspace()}/projects/{args.project}/cycles/", payload)
    print(green("✓ Cycle created"))
    format_output(data, args.format)