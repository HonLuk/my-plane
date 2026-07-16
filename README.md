# ✈️ plane-skill 🦞

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Plane.so skill for [OpenClaw](https://github.com/openclaw/openclaw) agents** — manage projects, work items, cycles, modules, comments, and members via a zero-dependency Python CLI.

## Features

- **Projects** — list, get, create
- **Work Items** — list, get, create, update, assign, delete, search
- **Comments** — list activity, add comments to work items
- **Images** — download images embedded in work item descriptions
- **Cycles** — list, get, create sprints
- **Modules** — list, get, create feature modules
- **Members** — list workspace members (for finding assignee IDs)
- **States & Labels** — list for reference
- Zero dependencies — Python 3.8+ stdlib only
- Color output with graceful degradation
- Table and JSON output formats
- Modular architecture for easy extension

## Architecture

```
scripts/
├── plane                 # Main CLI entry point
├── plane_api.py          # API layer (auth, requests, errors)
├── plane_output.py       # Output formatting (table/JSON)
└── commands/             # Command handlers by endpoint
    ├── user.py           # me
    ├── workspace.py      # members
    ├── projects.py       # projects CRUD
    ├── work_items.py     # issues CRUD + search
    ├── cycles.py         # cycles CRUD
    ├── modules.py        # modules CRUD
    ├── states.py         # states list
    ├── labels.py         # labels list
    ├── comments.py       # comments list/add
    └── images.py         # image download (get-image, get-images)
```

## Installation

### Via ClawHub (recommended)

```bash
npx clawhub install my-plane
```

### Manual Download

Download the single executable file:

```bash
curl -L -o ~/.local/bin/plane https://github.com/HonLuk/my-plane/releases/latest/download/plane
chmod +x ~/.local/bin/plane
```

Make sure `~/.local/bin` is in your PATH.

### Build from Source

```bash
git clone https://github.com/HonLuk/my-plane.git
cd my-plane
python build.py
cp dist/plane ~/.local/bin/plane
```

## Setup

You need two environment variables:

| Variable | Description | Where to get it |
|---|---|---|
| `PLANE_API_KEY` | Personal access token | Plane → Profile → Personal Access Tokens |
| `PLANE_WORKSPACE` | Workspace slug | The URL path segment (e.g., `my-team` from `app.plane.so/my-team/`) |

```bash
export PLANE_API_KEY="your-api-key-here"
export PLANE_WORKSPACE="your-workspace-slug"
```

Optionally, set `PLANE_BASE_URL` if you're self-hosting Plane (default: `https://api.plane.so`).

## Usage

```bash
# Who am I?
plane me

# List workspace members
plane members

# List all projects
plane projects list

# Get project details
plane projects get PROJECT_ID

# Create a new project
plane projects create --name "My App" --identifier "APP"

# List work items in a project
plane issues list -p PROJECT_ID

# Filter work items
plane issues list -p PROJECT_ID --priority high --state STATE_ID

# Get work item by short ID (fastest way)
plane issues get-short PROJ-123

# Get work item by UUID
plane issues get -p PROJECT_ID ISSUE_UUID

# Create a work item
plane issues create -p PROJECT_ID --name "Fix login bug" --priority high

# Update a work item
plane issues update -p PROJECT_ID ISSUE_UUID --state STATE_ID --priority medium

# Assign to members
plane issues assign -p PROJECT_ID ISSUE_UUID USER_ID1 USER_ID2

# Delete a work item
plane issues delete -p PROJECT_ID ISSUE_UUID

# Search across workspace
plane issues search "login bug"

# Add a comment
plane comments add -p PROJECT_ID -i ISSUE_UUID "Working on this now"

# List comments
plane comments list -p PROJECT_ID -i ISSUE_UUID

# List all activity (including field changes)
plane comments list -p PROJECT_ID -i ISSUE_UUID --all

# List cycles
plane cycles list -p PROJECT_ID

# Create a cycle
plane cycles create -p PROJECT_ID --name "Sprint 1" --start 2026-01-27 --end 2026-02-10

# List modules
plane modules list -p PROJECT_ID

# Create a module
plane modules create -p PROJECT_ID --name "Auth Module" --description "Authentication features"

# List workflow states (for getting state IDs)
plane states -p PROJECT_ID

# List labels (for getting label IDs)
plane labels -p PROJECT_ID

# Download all images from a work item
plane get-images PROJ-123 ./images/

# Download a single image by asset UUID
plane get-image PROJ-123 20745b59-e398-460d-9532-e0d56fbe7919 ./screenshot.png

# JSON output (returns downloaded file paths and metadata)
plane get-images PROJ-123 ./images/ -f json

# JSON output
plane projects list -f json
plane issues list -p PROJECT_ID -f json
```

### Getting Help

Every command has detailed help available:

```bash
plane --help
plane issues --help
plane issues create --help
plane cycles --help
```

### All Commands

| Command | Description |
|---|---|
| `plane me` | Show current user info |
| `plane members` | List workspace members |
| `plane projects list` | List all projects |
| `plane projects get PROJECT_ID` | Get project details |
| `plane projects create --name N --identifier I` | Create project |
| `plane issues list -p PROJECT_ID` | List work items |
| `plane issues list -p PROJECT_ID --priority high` | Filter by priority |
| `plane issues list -p PROJECT_ID --state STATE_ID` | Filter by state |
| `plane issues get -p PROJECT_ID ISSUE_ID` | Get work item by UUID |
| `plane issues get-short PROJECT-SEQ` | Get work item by short ID (e.g., PROJ-SEQ) |
| `plane issues create -p PROJECT_ID --name N` | Create work item |
| `plane issues update -p PROJECT_ID ISSUE_ID [--fields]` | Update work item |
| `plane issues assign -p PROJECT_ID ISSUE_ID USER_ID...` | Assign work item |
| `plane issues delete -p PROJECT_ID ISSUE_ID` | Delete work item |
| `plane issues search QUERY` | Search work items |
| `plane comments list -p PROJECT_ID -i ISSUE_ID` | List comments/activity |
| `plane comments list -p PROJECT_ID -i ISSUE_ID --all` | Show all activity |
| `plane comments add -p PROJECT_ID -i ISSUE_ID "text"` | Add comment |
| `plane cycles list -p PROJECT_ID` | List cycles |
| `plane cycles get -p PROJECT_ID CYCLE_ID` | Get cycle details |
| `plane cycles create -p PROJECT_ID --name N` | Create cycle |
| `plane modules list -p PROJECT_ID` | List modules |
| `plane modules get -p PROJECT_ID MODULE_ID` | Get module details |
| `plane modules create -p PROJECT_ID --name N` | Create module |
| `plane states -p PROJECT_ID` | List workflow states |
| `plane labels -p PROJECT_ID` | List labels |
| `plane get-images PROJ-SEQ DIR` | Download all images from a work item |
| `plane get-image PROJ-SEQ UUID FILE` | Download a single image by asset UUID |

### Filters

Work item listing supports filters:

```bash
plane issues list -p PROJECT_ID --state STATE_ID
plane issues list -p PROJECT_ID --priority high
plane issues list -p PROJECT_ID --assignee USER_ID
```

### Pagination

All list commands support cursor-based pagination:

```bash
# Basic list (shows pagination info)
plane projects list

# Specify page size
plane issues list -p PROJECT_ID --per-page 20

# Navigate pages using cursor (displayed in output)
plane issues list -p PROJECT_ID --cursor "5:1:0"
```

Pagination info is displayed above the results:
```
Pagination: total: 50 | pages: 5 | showing: 10
Next page: --cursor 10:1:0
Prev page: --cursor 10:-1:1
```

### Field Selection & Expansion

Control which fields are returned and expand related objects:

```bash
# Return only specific fields
plane issues list -p PROJECT_ID --fields "id,name,state"

# Expand related objects (assignees, state, labels, etc.)
plane issues list -p PROJECT_ID --expand "assignees,state"

# Sort results
plane issues list -p PROJECT_ID --order-by "-created_at"
```

### Image Downloads

Work item descriptions often contain embedded images (screenshots, diagrams, etc.). These are stored as `<image-component src="UUID">` tags in the `description_html` field. Use `get-images` to download all images or `get-image` for a specific one:

```bash
# Download all images from a work item to a directory
plane get-images PROJ-123 ./images/

# Download a single image by asset UUID (from description_html)
plane get-image PROJ-123 20745b59-e398-460d-9532-e0d56fbe7919 ./output.png

# JSON output returns file paths and metadata
plane get-images PROJ-123 ./images/ -f json
```

Files are named by asset UUID with the correct extension (e.g., `20745b59-....png`). If no extension is provided in the output path for `get-image`, it is auto-appended based on the image's Content-Type.

## How It Works

The CLI is a single Python script (`scripts/plane`) that wraps the [Plane.so REST API v1](https://developers.plane.so/). It uses only Python standard library modules (`urllib`, `json`, `argparse`) — no pip installs needed.

## Acknowledgments

Forked from [JinkoLLC/plane-skill](https://github.com/JinkoLLC/plane-skill).

## License

[MIT](LICENSE) © 2026 [HonLuk](https://github.com/HonLuk)
