---
name: my-plane
description: "Manage Plane.so projects and work items using the `plane` CLI. List projects, create/update/search issues, manage cycles and modules, add comments, and assign members."
metadata: {"moltbot":{"requires":{"env":["PLANE_API_KEY","PLANE_WORKSPACE"]},"primaryEnv":"PLANE_API_KEY","emoji":"✈️","homepage":"https://github.com/HonLuk/my-plane"}}
---

# Plane Skill

Interact with [Plane.so](https://plane.so) project management via the `plane` CLI.

## Installation

Install this skill together with its bundled CLI:

```bash
npx skills add https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill.zip -g -y
```

Install from the release archive so the installer receives the built
single-file CLI instead of the repository source tree. The `-g` flag installs
the skill in the user's global skill directory and `-y` skips the confirmation
prompt.

If the direct download fails because of a network error, retry with the
[GitHub proxy URL](https://gh-proxy.com/https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill.zip):

```bash
npx skills add https://gh-proxy.com/https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill.zip -g -y
```

The CLI is bundled at `scripts/plane` relative to this `SKILL.md`. Resolve the
directory containing this file and set the path before running commands:

```bash
# Replace this with the actual directory containing this SKILL.md.
PLANE_CLI="/path/to/my-plane/scripts/plane"
"$PLANE_CLI" me
```

Always invoke `"$PLANE_CLI"`; do not assume that a bare `plane` command is on
the user's `PATH`. In a repository checkout, the generated package path is
`./skills/scripts/plane`; inside an installed skill directory it is
`./scripts/plane`.

If Node.js is unavailable, download the release package into the agent's
`skills/` directory instead:

```bash
mkdir -p skills
curl -L -o /tmp/my-plane-skill.zip https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill.zip
unzip -o /tmp/my-plane-skill.zip -d skills
chmod +x skills/scripts/plane
```

The bundled CLI still requires Python 3.8 or newer, but this installation path
does not require Node.js.

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

The examples below assume `PLANE_CLI` has been set as shown above.

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

# Search across workspace
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

### Images

Download images embedded in work item descriptions. Image asset UUIDs are found in the `<image-component src="UUID">` tags of a work item's `description_html` field (visible via `"$PLANE_CLI" issues get-short PROJ-SEQ -f json`).

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
