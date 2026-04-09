# ✈️ plane-skill 🦞

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Plane.so skill for [OpenClaw](https://github.com/openclaw/openclaw) agents** — manage projects, work items, cycles, modules, comments, and members via a zero-dependency Python CLI.

## Features

- **Projects** — list, get, create
- **Work Items** — list, get, create, update, assign, delete, search
- **Comments** — list activity, add comments to work items
- **Cycles** — list, get, create sprints
- **Modules** — list, get, create feature modules
- **Members** — list workspace members (for finding assignee IDs)
- **States & Labels** — list for reference
- Zero dependencies — Python 3.8+ stdlib only
- Color output with graceful degradation
- Table and JSON output formats

## Installation

### Via ClawHub (recommended)

```bash
npx clawhub install my-plane
```

### Manual

Copy `SKILL.md` and `scripts/plane` into your agent's skill directory:

```bash
mkdir -p ~/.openclaw/skills/my-plane/scripts
cp SKILL.md ~/.openclaw/skills/my-plane/
cp scripts/plane ~/.openclaw/skills/my-plane/scripts/
chmod +x ~/.openclaw/skills/my-plane/scripts/plane
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

# Create a work item
plane issues create -p PROJECT_ID --name "Fix login bug" --priority high

# Assign it to someone
plane issues assign -p PROJECT_ID ISSUE_ID USER_ID

# Add a comment
plane comments add -p PROJECT_ID -i ISSUE_ID "Working on this now"

# Search across workspace
plane issues search "login bug"

# List cycles in a project
plane cycles list -p PROJECT_ID

# JSON output
plane projects list -f json
```

### All Commands

| Command | Description |
|---|---|
| `plane me` | Show current user |
| `plane members` | List workspace members |
| `plane projects list` | List all projects |
| `plane projects get PROJECT_ID` | Get project details |
| `plane projects create --name N --identifier I` | Create project |
| `plane issues list -p PROJECT_ID` | List work items |
| `plane issues get -p PROJECT_ID ISSUE_ID` | Get work item details |
| `plane issues get-short PROJECT-SEQ` | Get work item by short ID (e.g., PROJ-SEQ) |
| `plane issues create -p PROJECT_ID --name N` | Create work item |
| `plane issues update -p PROJECT_ID ISSUE_ID [--fields]` | Update work item |
| `plane issues assign -p PROJECT_ID ISSUE_ID USER_ID...` | Assign work item |
| `plane issues delete -p PROJECT_ID ISSUE_ID` | Delete work item |
| `plane issues search QUERY` | Search work items |
| `plane comments list -p PROJECT_ID -i ISSUE_ID` | List comments/activity |
| `plane comments add -p PROJECT_ID -i ISSUE_ID "text"` | Add comment |
| `plane cycles list -p PROJECT_ID` | List cycles |
| `plane cycles get -p PROJECT_ID CYCLE_ID` | Get cycle details |
| `plane cycles create -p PROJECT_ID --name N` | Create cycle |
| `plane modules list -p PROJECT_ID` | List modules |
| `plane modules get -p PROJECT_ID MODULE_ID` | Get module details |
| `plane modules create -p PROJECT_ID --name N` | Create module |
| `plane states -p PROJECT_ID` | List workflow states |
| `plane labels -p PROJECT_ID` | List labels |

### Filters

Work item listing supports filters:

```bash
plane issues list -p PROJECT_ID --state STATE_ID
plane issues list -p PROJECT_ID --priority high
plane issues list -p PROJECT_ID --assignee USER_ID
```

## How It Works

The CLI is a single Python script (`scripts/plane`) that wraps the [Plane.so REST API v1](https://developers.plane.so/). It uses only Python standard library modules (`urllib`, `json`, `argparse`) — no pip installs needed.

## Acknowledgments

Forked from [JinkoLLC/plane-skill](https://github.com/JinkoLLC/plane-skill).

## License

[MIT](LICENSE) © 2026 [HonLuk](https://github.com/HonLuk)
