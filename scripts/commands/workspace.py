"""
Workspace Commands

Commands related to workspace members and workspace-level operations.
"""

import sys
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from urllib.parse import urlencode

from plane_api import api, get_workspace, _validate_id
from plane_output import format_output


def cmd_members_list(args):
    """List all members of the workspace with pagination support."""
    endpoint = f"/workspaces/{get_workspace()}/members/"
    params = {}

    if hasattr(args, 'cursor') and args.cursor:
        params["cursor"] = args.cursor
    if hasattr(args, 'per_page') and args.per_page:
        params["per_page"] = args.per_page

    if params:
        endpoint += "?" + urlencode(params)
    data = api("GET", endpoint)
    format_output(data, args.format)