---
name: my-plane
description: "Manage Plane.so projects and work items using the `plane` CLI. List projects, create/update/search issues, manage file attachments and embedded images, manage cycles and modules, add comments, and assign members."
metadata: {"moltbot":{"requires":{"env":["PLANE_API_KEY","PLANE_WORKSPACE"]},"primaryEnv":"PLANE_API_KEY","emoji":"✈️","homepage":"https://github.com/HonLuk/my-plane"}}
---

# Plane Skill

Interact with [Plane.so](https://plane.so) project management via the `plane` CLI.

## CLI

Each platform-specific release archive already contains the matching native Go
binary. It is located at `scripts/plane` on Unix-like systems and
`scripts/plane.exe` on Windows.

Resolve the directory containing this `SKILL.md`, set `PLANE_CLI` to the bundled
binary, and invoke that path directly. Do not assume that a bare `plane`
command is available on the user's `PATH`:

```bash
# Replace this with the directory containing this SKILL.md.
PLANE_CLI="/path/to/installed/my-plane/scripts/plane"
"$PLANE_CLI" me
```

No additional binary download is required after installing the matching
package.

## Setup

```bash
export PLANE_API_KEY="your-api-key"
export PLANE_WORKSPACE="your-workspace-slug"
# Optional: for self-hosted Plane (default: https://api.plane.so)
export PLANE_BASE_URL="https://api.plane.so"
```

Get your API key from: **Plane → Profile Settings → Personal Access Tokens**

The workspace slug is the URL path segment (e.g., for `https://app.plane.so/my-team/` the slug is `my-team`).

## Agent-Assisted Configuration

Before running a command that calls the API, check `PLANE_API_KEY` and
`PLANE_WORKSPACE`. If either is missing, ask the user for it instead of
guessing. Ask for `PLANE_BASE_URL` only when the user is self-hosting Plane.

- Treat the API key as a secret: do not echo, log, commit, or place it in skill files.
- After the user confirms the values, export them for the current session and retry the command.
- Do not modify shell startup files or persist credentials unless the user explicitly requests it.

## Work Item Descriptions

Before creating or updating a formatted work item, read
`references/work-item-description.md`. It defines the HTML structure, text
marks, font/display limitations, Plane color and background attributes, and
the Markdown-to-HTML boundary.

- Use `--description` for plain text or simple Markdown; complex Markdown may
  be converted incompletely and will produce a warning.
- Use `--description-html` for exact Plane editor HTML, colors, backgrounds, or
  complex Markdown converted to HTML.
- Verify the target workspace and project before writing.

## Commands

The examples below assume `PLANE_CLI` points to the bundled binary.

### Current User

```bash
"$PLANE_CLI" me                      # Show authenticated user info
```

### Workspace Members

```bash
"$PLANE_CLI" members                 # List all workspace members (name, email, role, ID)
```

### Projects

```bash
"$PLANE_CLI" projects list                                      # List all projects
"$PLANE_CLI" projects get PROJECT_ID                            # Get project details
"$PLANE_CLI" projects create --name "My Project" --identifier "PROJ"  # Create project
```

### Work Items (Issues)

```bash
# List work items
"$PLANE_CLI" issues list -p PROJECT_ID
"$PLANE_CLI" issues list -p PROJECT_ID --priority high --assignee USER_ID
"$PLANE_CLI" issues list -p PROJECT_ID --state STATE_ID

# Get details
"$PLANE_CLI" issues get -p PROJECT_ID ISSUE_ID
"$PLANE_CLI" issues get-short PROJ-SEQ  # e.g., PROJ-123 (fastest way)

# Create
"$PLANE_CLI" issues create -p PROJECT_ID --name "Fix login bug" --priority high
"$PLANE_CLI" issues create -p PROJECT_ID --name "Feature" --assignee USER_ID --label LABEL_ID
"$PLANE_CLI" issues create -p PROJECT_ID --name "Rich feature" \
  --description-html '<p><strong>Acceptance criteria</strong></p><ul><li>List renders correctly</li></ul>'

# Update
"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_ID --state STATE_ID --priority medium
"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_ID \
  --description-html '<p><span data-text-color="pink">Updated note</span></p>'

# Assign to members
"$PLANE_CLI" issues assign -p PROJECT_ID ISSUE_ID USER_ID_1 USER_ID_2

# Delete
"$PLANE_CLI" issues delete -p PROJECT_ID ISSUE_ID

# Search across workspace by title or supported body-search fields
"$PLANE_CLI" issues search "login bug"
```

### Comments

```bash
"$PLANE_CLI" comments list -p PROJECT_ID -i ISSUE_ID            # List comments on a work item
"$PLANE_CLI" comments list -p PROJECT_ID -i ISSUE_ID --all      # Show all activity (not just comments)
"$PLANE_CLI" comments add -p PROJECT_ID -i ISSUE_ID "Looks good, merging now"  # Add a comment
```

### Cycles (Sprints)

```bash
"$PLANE_CLI" cycles list -p PROJECT_ID
"$PLANE_CLI" cycles get -p PROJECT_ID CYCLE_ID
"$PLANE_CLI" cycles create -p PROJECT_ID --name "Sprint 1" --start 2026-01-27 --end 2026-02-10
```

### Modules

```bash
"$PLANE_CLI" modules list -p PROJECT_ID
"$PLANE_CLI" modules get -p PROJECT_ID MODULE_ID
"$PLANE_CLI" modules create -p PROJECT_ID --name "Auth Module" --description "Authentication features"
```

### States & Labels

```bash
"$PLANE_CLI" states -p PROJECT_ID    # List workflow states (useful for getting state IDs)
"$PLANE_CLI" labels -p PROJECT_ID    # List labels (useful for getting label IDs)
```

### Attachments

Use the `attachments` command group for every file attachment operation. Read
`references/work-item-description.md` when you need the HTML rules or need to
recover from a failed image description update.

```bash
# List all file attachments on a work item.
"$PLANE_CLI" attachments list PROJ-123 -f json

# Upload a file. For images, also append an image-component to description_html.
"$PLANE_CLI" attachments upload PROJ-123 ./screenshot.png -f json

# Download any attachment by its resource UUID.
"$PLANE_CLI" attachments get PROJ-123 RESOURCE_UUID ./downloaded-file

# Complete or delete an attachment.
"$PLANE_CLI" attachments complete PROJ-123 RESOURCE_UUID
"$PLANE_CLI" attachments delete PROJ-123 RESOURCE_UUID
```

`attachments upload` performs the complete Plane flow: request upload
credentials, send the file to object storage, then mark the attachment as
uploaded. It accepts any MIME type known from the file extension; pass
`--type MIME` for an extensionless or uncommon file. JPEG, PNG, WebP, and GIF
files are inserted into the existing description with:

```html
<image-component src="ASSET_UUID"></image-component>
```

The JSON output includes the safe `asset_id`, `asset_url`, file metadata, and
`inserted` status. Signed upload fields and signed URLs are never printed. If
the description update fails after confirmation, the attachment is retained
and the error includes the asset ID so the existing HTML can be repaired.

When creating an issue and then adding a screenshot, create the issue first,
then use `attachments upload PROJECT-SEQ FILE -f json`; do not put an object
storage URL in `description_html`. The editor's `react-renderer` wrapper,
`/api/assets/v2/...` URL, and download or fullscreen controls are browser-
generated DOM, not content to store in the description.

### Image Downloads

Image asset UUIDs are found in the `<image-component src="UUID">` tags of a
work item's `description_html` field (visible via
`"$PLANE_CLI" issues get-short PROJ-SEQ -f json`).

```bash
# Download all images from a work item to a directory
"$PLANE_CLI" get-images PROJ-123 ./images/

# Download a single image by asset UUID
"$PLANE_CLI" get-image PROJ-123 20745b59-e398-460d-9532-e0d56fbe7919 ./output.png

# JSON output (returns downloaded file paths and metadata)
"$PLANE_CLI" get-images PROJ-123 ./images/ -f json
"$PLANE_CLI" get-image PROJ-123 20745b59-... ./output.png -f json
```

Files are named by asset UUID with the correct extension (e.g., `20745b59-....png`). If no extension is provided in the output path for `get-image`, it is auto-appended based on the image's Content-Type.

## Output Formats

Default output is a formatted table. Add `-f json` for raw JSON:

```bash
"$PLANE_CLI" projects list -f json
"$PLANE_CLI" issues list -p PROJECT_ID -f json
```

## Pagination

All list commands support cursor-based pagination. Pagination info is displayed above results:

```bash
"$PLANE_CLI" projects list
# Shows:
# Pagination: total: 50 | pages: 5 | showing: 10
# Next page: --cursor 10:1:0

# Use cursor to navigate
"$PLANE_CLI" issues list -p PROJECT_ID --cursor "10:1:0"

# Control page size
"$PLANE_CLI" issues list -p PROJECT_ID --per-page 20
```

## Field Selection

```bash
# Return only specific fields
"$PLANE_CLI" issues list -p PROJECT_ID --fields "id,name,state"

# Expand related objects
"$PLANE_CLI" issues list -p PROJECT_ID --expand "assignees,state"

# Sort results
"$PLANE_CLI" issues list -p PROJECT_ID --order-by "-created_at"
```

## Getting Help

Every command has detailed help:

```bash
"$PLANE_CLI" --help
"$PLANE_CLI" issues --help
"$PLANE_CLI" issues create --help
```

## Typical Workflow

1. `"$PLANE_CLI" projects list` — find your project ID
2. `"$PLANE_CLI" states -p PROJECT_ID` — see available states
3. `"$PLANE_CLI" members` — find member IDs for assignment
4. `"$PLANE_CLI" issues create -p PROJECT_ID --name "Task" --priority high --assignee USER_ID`
5. `"$PLANE_CLI" comments add -p PROJECT_ID -i ISSUE_ID "Started working on this"`
6. `"$PLANE_CLI" get-images PROJ-123 ./images/` — download all images from a work item
