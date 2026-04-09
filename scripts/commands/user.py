"""
User Commands

Commands related to current user info.
"""

import sys
import os

# Add parent directory to path for imports
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from plane_api import api
from plane_output import format_output


def cmd_me(args):
    """Display the currently authenticated user."""
    data = api("GET", "/users/me/")
    format_output(data, args.format)