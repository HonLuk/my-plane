"""
Plane API Core Layer

Handles authentication, HTTP requests, and error handling for Plane.so REST API.
"""

import os
import sys
import json
from typing import Optional, Dict, Any
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

# ---------------------------------------------------------------------------
# Configuration — all from environment variables
# ---------------------------------------------------------------------------

API_KEY = os.environ.get("PLANE_API_KEY")
WORKSPACE = os.environ.get("PLANE_WORKSPACE")
BASE_URL = os.environ.get("PLANE_BASE_URL", "https://api.plane.so")


# ---------------------------------------------------------------------------
# Color helper (minimal, avoids circular import)
# ---------------------------------------------------------------------------

def _red(text: str) -> str:
    """Red text for error messages."""
    if sys.stdout.isatty() and os.environ.get("NO_COLOR") is None:
        return f"\033[31m{text}\033[0m"
    return text


# ---------------------------------------------------------------------------
# Environment validation
# ---------------------------------------------------------------------------

def _check_env():
    """
    Validate required environment variables and exit with helpful messages.

    Raises:
        SystemExit if PLANE_API_KEY or PLANE_WORKSPACE is not set.
    """
    if not API_KEY:
        print(_red("Error: PLANE_API_KEY is not set."), file=sys.stderr)
        print(
            "\nTo get an API key:\n"
            "  1. Open Plane → Profile Settings → Personal Access Tokens\n"
            "  2. Create a new token and copy it\n"
            "  3. Export it:  export PLANE_API_KEY=\"your-token\"\n",
            file=sys.stderr,
        )
        sys.exit(1)
    if not WORKSPACE:
        print(_red("Error: PLANE_WORKSPACE is not set."), file=sys.stderr)
        print(
            "\nSet it to your workspace slug (the part after plane.so/):\n"
            "  export PLANE_WORKSPACE=\"my-workspace\"\n",
            file=sys.stderr,
        )
        sys.exit(1)


def _validate_id(value: str, name: str = "ID") -> str:
    """
    Validate that an ID doesn't contain path-injection characters.

    Args:
        value: The ID string to validate.
        name:  Human-readable name for error messages.

    Returns:
        The validated ID.

    Raises:
        SystemExit if validation fails.
    """
    if "/" in value or "\\" in value or ".." in value:
        print(_red(f"Invalid {name}: contains illegal characters"), file=sys.stderr)
        sys.exit(1)
    return value


# ---------------------------------------------------------------------------
# API request handler
# ---------------------------------------------------------------------------

def api(method: str, endpoint: str, data: Optional[Dict[str, Any]] = None) -> Optional[Any]:
    """
    Make an authenticated request to the Plane API.

    Args:
        method:   HTTP method (GET, POST, PATCH, DELETE).
        endpoint: API path starting with / (e.g., /users/me/).
        data:     Optional JSON body for POST/PATCH.

    Returns:
        Parsed JSON response, or None for 204 No Content.

    Raises:
        SystemExit on HTTP error, connection error, or timeout.
    """
    _check_env()

    url = f"{BASE_URL}/api/v1{endpoint}"

    # Defense-in-depth: ensure we only make HTTP(S) requests
    if not url.startswith(("http://", "https://")):
        print(_red(f"Invalid URL scheme: {url}"), file=sys.stderr)
        sys.exit(1)

    headers = {
        "X-API-Key": API_KEY,
        "Content-Type": "application/json",
    }

    body = json.dumps(data).encode() if data else None
    req = Request(url, data=body, headers=headers, method=method)

    try:
        with urlopen(req, timeout=30) as resp:
            if resp.status == 204:
                return None
            return json.loads(resp.read().decode())
    except HTTPError as e:
        error_body = e.read().decode() if e.fp else str(e)
        try:
            error_json = json.loads(error_body)
            error_body = json.dumps(error_json, indent=2)
        except (json.JSONDecodeError, ValueError):
            pass
        print(_red(f"API Error {e.code}:") + f" {error_body}", file=sys.stderr)
        sys.exit(1)
    except URLError as e:
        print(_red(f"Connection error: {e.reason}"), file=sys.stderr)
        sys.exit(1)
    except TimeoutError:
        print(_red("Request timed out (30s). The Plane API may be unreachable."), file=sys.stderr)
        sys.exit(1)


# ---------------------------------------------------------------------------
# Binary asset download
# ---------------------------------------------------------------------------

def download_binary(endpoint: str, output_path: str) -> Dict[str, Any]:
    """
    Download a binary asset from the Plane API and save to a file.

    Uses the same authentication as api() but reads the response as raw
    binary data instead of JSON.  The attachment endpoint returns a 302
    redirect to a presigned URL; urlopen follows redirects automatically.

    Args:
        endpoint:     API path starting with / (e.g., /workspaces/.../attachments/.../).
        output_path:  File path where the binary data will be saved.

    Returns:
        Dict with keys: content_type, size, path.

    Raises:
        SystemExit on HTTP error, connection error, or timeout.
    """
    _check_env()

    url = f"{BASE_URL}/api/v1{endpoint}"

    # Defense-in-depth: ensure we only make HTTP(S) requests
    if not url.startswith(("http://", "https://")):
        print(_red(f"Invalid URL scheme: {url}"), file=sys.stderr)
        sys.exit(1)

    headers = {
        "X-API-Key": API_KEY,
    }

    req = Request(url, headers=headers, method="GET")

    try:
        with urlopen(req, timeout=60) as resp:
            content_type = resp.headers.get("Content-Type", "application/octet-stream")
            data = resp.read()
            with open(output_path, "wb") as f:
                f.write(data)
            return {
                "content_type": content_type,
                "size": len(data),
                "path": output_path,
            }
    except HTTPError as e:
        error_body = e.read().decode() if e.fp else str(e)
        try:
            error_json = json.loads(error_body)
            error_body = json.dumps(error_json, indent=2)
        except (json.JSONDecodeError, ValueError):
            pass
        print(_red(f"API Error {e.code}:") + f" {error_body}", file=sys.stderr)
        sys.exit(1)
    except URLError as e:
        print(_red(f"Connection error: {e.reason}"), file=sys.stderr)
        sys.exit(1)
    except TimeoutError:
        print(_red("Request timed out (60s). The Plane API may be unreachable."), file=sys.stderr)
        sys.exit(1)


# ---------------------------------------------------------------------------
# Convenience getters for configuration
# ---------------------------------------------------------------------------

def get_workspace() -> str:
    """Return the configured workspace slug."""
    return WORKSPACE

def get_base_url() -> str:
    """Return the configured API base URL."""
    return BASE_URL