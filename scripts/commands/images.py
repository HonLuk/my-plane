"""
Image Commands

Download images embedded in work item descriptions.

Plane stores description images as FileAsset records.  The ``<image-component
src="UUID">`` tags in ``description_html`` reference asset UUIDs that can be
downloaded via the work-item attachment detail endpoint.
"""

import sys
import os
import re
import json

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(SCRIPT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from plane_api import api, download_binary, get_workspace, _validate_id
from plane_output import green, yellow, dim

# Regex to extract image asset UUIDs from <image-component src="...">
_IMAGE_COMPONENT_RE = re.compile(
    r'<image-component[^>]+src="([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"',
    re.IGNORECASE,
)

# Content-Type → file extension
_EXT_MAP = {
    "image/png": ".png",
    "image/jpeg": ".jpg",
    "image/jpg": ".jpg",
    "image/gif": ".gif",
    "image/webp": ".webp",
}


def _ext_from_content_type(content_type: str) -> str:
    """Return a file extension for a given Content-Type string."""
    ct = content_type.split(";")[0].strip().lower()
    return _EXT_MAP.get(ct, ".bin")


def _resolve_work_item(issue_short: str):
    """
    Fetch a work item by short ID and return (project_id, issue_id, description_html).

    Raises SystemExit on error.
    """
    _validate_id(issue_short, "issue short ID")
    data = api("GET", f"/workspaces/{get_workspace()}/work-items/{issue_short}/")
    if not data:
        print(yellow("Work item not found."), file=sys.stderr)
        sys.exit(1)
    project_id = data.get("project")
    issue_id = data.get("id")
    description_html = data.get("description_html", "") or ""
    if not project_id or not issue_id:
        print(yellow("Could not determine project/issue ID from work item."), file=sys.stderr)
        sys.exit(1)
    return project_id, issue_id, description_html


def _download_one(workspace, project_id, issue_id, asset_id, output_path):
    """
    Download a single image asset to *output_path*.

    Returns the final file path (may differ from *output_path* if an
    extension was appended).  Raises SystemExit on error.
    """
    _validate_id(asset_id, "asset ID")
    endpoint = (
        f"/workspaces/{workspace}/projects/{project_id}"
        f"/work-items/{issue_id}/attachments/{asset_id}/"
    )
    result = download_binary(endpoint, output_path)

    # If the user-provided path has no extension, append the correct one
    final_path = output_path
    if not os.path.splitext(output_path)[1]:
        ext = _ext_from_content_type(result["content_type"])
        final_path = output_path + ext
        os.rename(output_path, final_path)
        result["path"] = final_path

    return result, final_path


def cmd_get_image(args):
    """Download a single image from a work item by asset UUID."""
    project_id, issue_id, _ = _resolve_work_item(args.issue_short)

    print(dim(f"Downloading image {args.asset_id}..."))
    result, final_path = _download_one(
        get_workspace(), project_id, issue_id, args.asset_id, args.file
    )

    print(green(f"✓ Image saved to {final_path}"))
    print(dim(f"  Size: {result['size']:,} bytes | Type: {result['content_type']}"))

    if args.format == "json":
        print(json.dumps({
            "asset_id": args.asset_id,
            "path": final_path,
            "size": result["size"],
            "content_type": result["content_type"],
        }, indent=2))


def cmd_get_images(args):
    """Download all images embedded in a work item's description."""
    project_id, issue_id, description_html = _resolve_work_item(args.issue_short)

    # Extract all image asset UUIDs from the description HTML
    asset_ids = _IMAGE_COMPONENT_RE.findall(description_html)

    if not asset_ids:
        print(yellow("No images found in this work item's description."))
        return

    # Deduplicate while preserving order
    seen = set()
    unique_ids = []
    for aid in asset_ids:
        if aid not in seen:
            seen.add(aid)
            unique_ids.append(aid)

    # Ensure the output directory exists
    os.makedirs(args.dir, exist_ok=True)

    print(dim(f"Found {len(unique_ids)} image(s) in {args.issue_short}"))
    print()

    ws = get_workspace()
    results = []
    for idx, aid in enumerate(unique_ids, 1):
        tmp_path = os.path.join(args.dir, f"{aid}.tmp")
        try:
            result = download_binary(
                f"/workspaces/{ws}/projects/{project_id}"
                f"/work-items/{issue_id}/attachments/{aid}/",
                tmp_path,
            )
            ext = _ext_from_content_type(result["content_type"])
            final_path = os.path.join(args.dir, f"{aid}{ext}")
            os.rename(tmp_path, final_path)
            print(green(f"  [{idx}/{len(unique_ids)}] {aid}{ext}")
                  + dim(f"  ({result['size']:,} bytes)"))
            results.append({
                "asset_id": aid,
                "path": final_path,
                "size": result["size"],
                "content_type": result["content_type"],
            })
        except SystemExit:
            # download_binary already printed an error — clean up & continue
            if os.path.exists(tmp_path):
                os.remove(tmp_path)
            print(yellow(f"  [{idx}/{len(unique_ids)}] Failed: {aid}"))
            continue

    print()
    print(green(f"✓ Downloaded {len(results)}/{len(unique_ids)} image(s) to {args.dir}"))

    if args.format == "json":
        print(json.dumps(results, indent=2))
