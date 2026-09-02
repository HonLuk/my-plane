# ✈️ plane-skill 🦞

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Plane.so skill for [OpenClaw](https://github.com/openclaw/openclaw) agents** — manage projects, work items, cycles, modules, comments, and members via a zero-dependency Go CLI.

## Features

- **Projects** — list, get, create
- **Work Items** — list, get, create, update, assign, delete, search
- **Comments** — list activity, add, edit, and delete comments on work items
- **Attachments & Images** — upload, embed, list, delete, and download work item files
- **Cycles** — list, get, create sprints
- **Modules** — list, get, create feature modules
- **Members** — list workspace members (for finding assignee IDs)
- **States & Labels** — list for reference
- Native Go binary with no runtime dependencies
- Color output with graceful degradation
- Table and JSON output formats
- Modular architecture for easy extension

## Architecture

```
cmd/plane/main.go          # CLI entry point
internal/api/              # auth, requests, errors, uploads, downloads
internal/commands/         # command tree and Plane resources
internal/output/           # table/JSON output and terminal colors
internal/markdown/         # simple Markdown-to-HTML conversion
```

The installable Agent Skill is published as one package for each supported
platform and architecture:

```
my-plane-skill-linux-amd64.zip
├── SKILL.md
├── references/
│   └── work-item-description.md  # Detailed body formatting rules
└── scripts/
    └── plane                     # Matching native CLI binary
```

The other packages use the same layout. The Windows package contains
`scripts/plane.exe`. Each package already includes its matching native binary,
so no second binary download is required after installation.

## Requirements

- **Go 1.22 or newer** — only required when building from source.
- **Supported release targets** — Linux amd64/arm64, macOS amd64/arm64, and Windows amd64.
- **Node.js with `npx`** — optional; only required when installing through `npx skills add`.

## Installation

### Via `npx skills add` (recommended)

Choose the package matching the host platform and architecture. The direct
download link and complete `npx` command are listed below:

| Target | Direct skill package | npx command |
|---|---|---|
| Linux x86-64 | [my-plane-skill-linux-amd64.zip](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-amd64.zip) | `npx skills add https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-amd64.zip` |
| Linux ARM64 | [my-plane-skill-linux-arm64.zip](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-arm64.zip) | `npx skills add https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-arm64.zip` |
| macOS Intel | [my-plane-skill-darwin-amd64.zip](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-darwin-amd64.zip) | `npx skills add https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-darwin-amd64.zip` |
| macOS Apple Silicon | [my-plane-skill-darwin-arm64.zip](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-darwin-arm64.zip) | `npx skills add https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-darwin-arm64.zip` |
| Windows x86-64 | [my-plane-skill-windows-amd64.zip](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-windows-amd64.zip) | `npx skills add https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-windows-amd64.zip` |

Each selected skill package includes the CLI, so there is no follow-up
installation step and no global `plane` command or `PATH` modification is
needed.

If the download fails because of a network error, retry with the [GitHub proxy
URL](https://gh-proxy.com/https://github%2Ecom/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-amd64.zip),
replacing the target suffix when necessary:

```bash
npx skills add https://gh-proxy.com/https://github%2Ecom/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-amd64.zip
```

### Via Release Download (no Node.js required)

If Node.js is unavailable, download and extract the same platform-specific
skill package into the agent's skills directory at `~/.agents/skills`:

```bash
mkdir -p ~/.agents/skills/my-plane
curl -fL -o /tmp/my-plane-skill-linux-amd64.zip https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-amd64.zip
unzip -o /tmp/my-plane-skill-linux-amd64.zip -d ~/.agents/skills/my-plane
```

Change `linux-amd64` in both URLs for the target platform. The extracted skill
already contains the matching CLI binary.

### Build from Source

```bash
git clone https://github.com/HonLuk/my-plane.git
cd my-plane
go build -trimpath -o dist/plane ./cmd/plane
./dist/plane --help
```

To build every Release asset locally, including all platform-specific skill
packages, run:

```bash
make release
```

The output is written to `dist/`. It contains five standalone binaries, five
skill packages named `my-plane-skill-<platform>-<architecture>.zip`, and
`SHA256SUMS`.

### Release Assets

Each tagged GitHub Release contains these directly executable binaries:

| Asset | Target |
|---|---|
| `plane-linux-amd64` | Linux x86-64 |
| `plane-linux-arm64` | Linux ARM64 |
| `plane-darwin-amd64` | macOS Intel |
| `plane-darwin-arm64` | macOS Apple Silicon |
| `plane-windows-amd64.exe` | Windows x86-64 |

It also contains a platform-specific skill package for each target:

| Package | Target |
|---|---|
| [`my-plane-skill-linux-amd64.zip`](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-amd64.zip) | Linux x86-64 |
| [`my-plane-skill-linux-arm64.zip`](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-linux-arm64.zip) | Linux ARM64 |
| [`my-plane-skill-darwin-amd64.zip`](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-darwin-amd64.zip) | macOS Intel |
| [`my-plane-skill-darwin-arm64.zip`](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-darwin-arm64.zip) | macOS Apple Silicon |
| [`my-plane-skill-windows-amd64.zip`](https://github.com/HonLuk/my-plane/releases/latest/download/my-plane-skill-windows-amd64.zip) | Windows x86-64 |

Each skill package contains `SKILL.md`, `references/`, and the matching CLI at
`scripts/plane` (or `scripts/plane.exe` on Windows). `SHA256SUMS` covers all
standalone binaries and skill packages.

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

### Agent-Assisted Configuration

If `PLANE_API_KEY` or `PLANE_WORKSPACE` is missing, the Agent should pause
before making API calls and ask the user for the missing values:

1. Ask for the Plane personal access token. Treat it as a secret and never echo or log it.
2. Ask for the workspace slug, taken from the Plane URL (for example, `my-team`).
3. If the user uses a self-hosted Plane instance, ask for `PLANE_BASE_URL` too.
4. Export the values for the current session, then retry the requested command.

Do not write credentials to the repository or modify a shell profile without
the user's explicit permission.

### Work Item Description Format

Detailed body formatting rules are maintained in
[`references/work-item-description.md`](references/work-item-description.md)
and included in the installable skill. In short, use `--description` only for
plain text or simple Markdown and `--description-html` for exact Plane editor
HTML. Complex Markdown may convert incompletely and produces a warning; use
`--description-html` when exact formatting is required. The web editor also
supports pasting common Markdown/GFM content.

## Usage

The examples below use `PLANE_CLI` so they do not depend on a global `PATH`
entry. Set it once from the directory that contains the installed skill:

```bash
# From an installed Unix-like skill directory:
PLANE_CLI="/path/to/installed/my-plane/scripts/plane"

# Windows PowerShell uses:
# $PlaneCli = "C:\path\to\installed\my-plane\scripts\plane.exe"
```

Invoke commands as `"$PLANE_CLI" ...`; the skill does not assume that a bare
`plane` command is available.

```bash
# Who am I?
"$PLANE_CLI" me

# List workspace members
"$PLANE_CLI" members

# List all projects
"$PLANE_CLI" projects list

# Get project details
"$PLANE_CLI" projects get PROJECT_ID

# Create a new project
"$PLANE_CLI" projects create --name "My App" --identifier "APP"

# List work items in a project
"$PLANE_CLI" issues list -p PROJECT_ID

# Filter work items
"$PLANE_CLI" issues list -p PROJECT_ID --priority high --state STATE_ID

# Get work item by short ID (fastest way)
"$PLANE_CLI" issues get-short PROJ-123

# Get work item by UUID
"$PLANE_CLI" issues get -p PROJECT_ID ISSUE_UUID

# Create a work item
"$PLANE_CLI" issues create -p PROJECT_ID --name "Fix login bug" --priority high
"$PLANE_CLI" issues create -p PROJECT_ID --name "Rich feature" \
  --description-html '<p><strong>Acceptance criteria</strong></p><ul><li>List renders correctly</li></ul>'

# Update a work item
"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_UUID --state STATE_ID --priority medium
"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_UUID \
  --description-html '<p><span data-text-color="pink">Updated note</span></p>'

# Assign to members
"$PLANE_CLI" issues assign -p PROJECT_ID ISSUE_UUID USER_ID1 USER_ID2

# Delete a work item
"$PLANE_CLI" issues delete -p PROJECT_ID ISSUE_UUID

# Search across workspace by title or supported body-search fields
"$PLANE_CLI" issues search "login bug"

# Add a comment
"$PLANE_CLI" comments add -p PROJECT_ID -i ISSUE_UUID "Working on this now"
"$PLANE_CLI" comments add -p PROJECT_ID -i ISSUE_UUID \
  --body-html '<p><strong>Rich comment</strong></p>'

# Update a comment (the comment UUID comes from comments list -f json)
"$PLANE_CLI" comments update -p PROJECT_ID -i ISSUE_UUID COMMENT_UUID "Updated comment"
"$PLANE_CLI" comments update -p PROJECT_ID -i ISSUE_UUID COMMENT_UUID \
  --body-html '<p><em>Updated rich comment</em></p>'

# Delete a comment (prompts for confirmation; use --yes for automation)
"$PLANE_CLI" comments delete -p PROJECT_ID -i ISSUE_UUID COMMENT_UUID
"$PLANE_CLI" comments delete -p PROJECT_ID -i ISSUE_UUID COMMENT_UUID --yes

# List comments
"$PLANE_CLI" comments list -p PROJECT_ID -i ISSUE_UUID

# List all activity (including field changes)
"$PLANE_CLI" comments list -p PROJECT_ID -i ISSUE_UUID --all

# List cycles
"$PLANE_CLI" cycles list -p PROJECT_ID

# Create a cycle
"$PLANE_CLI" cycles create -p PROJECT_ID --name "Sprint 1" --start 2026-01-27 --end 2026-02-10

# List modules
"$PLANE_CLI" modules list -p PROJECT_ID

# Create a module
"$PLANE_CLI" modules create -p PROJECT_ID --name "Auth Module" --description "Authentication features"

# List workflow states (for getting state IDs)
"$PLANE_CLI" states -p PROJECT_ID

# List labels (for getting label IDs)
"$PLANE_CLI" labels -p PROJECT_ID

# Download all images from a work item
"$PLANE_CLI" get-images PROJ-123 ./images/

# Download a single image by asset UUID
"$PLANE_CLI" get-image PROJ-123 20745b59-e398-460d-9532-e0d56fbe7919 ./screenshot.png

# Upload an attachment; images are also inserted into the description
"$PLANE_CLI" attachments upload PROJ-123 ./screenshot.png -f json

# List, download, complete, or delete attachments
"$PLANE_CLI" attachments list PROJ-123 -f json
"$PLANE_CLI" attachments get PROJ-123 RESOURCE_UUID ./downloaded-file
"$PLANE_CLI" attachments complete PROJ-123 RESOURCE_UUID
"$PLANE_CLI" attachments delete PROJ-123 RESOURCE_UUID

# JSON output (returns downloaded file paths and metadata)
"$PLANE_CLI" get-images PROJ-123 ./images/ -f json

# JSON output
"$PLANE_CLI" projects list -f json
"$PLANE_CLI" issues list -p PROJECT_ID -f json
```

### Getting Help

Every command has detailed help available:

```bash
"$PLANE_CLI" --help
"$PLANE_CLI" issues --help
"$PLANE_CLI" issues create --help
"$PLANE_CLI" cycles --help
```

### All Commands

| Command | Description |
|---|---|
| `"$PLANE_CLI" me` | Show current user info |
| `"$PLANE_CLI" members` | List workspace members |
| `"$PLANE_CLI" projects list` | List all projects |
| `"$PLANE_CLI" projects get PROJECT_ID` | Get project details |
| `"$PLANE_CLI" projects create --name N --identifier I` | Create project |
| `"$PLANE_CLI" issues list -p PROJECT_ID` | List work items |
| `"$PLANE_CLI" issues list -p PROJECT_ID --priority high` | Filter by priority |
| `"$PLANE_CLI" issues list -p PROJECT_ID --state STATE_ID` | Filter by state |
| `"$PLANE_CLI" issues get -p PROJECT_ID ISSUE_ID` | Get work item by UUID |
| `"$PLANE_CLI" issues get-short PROJECT-SEQ` | Get work item by short ID (e.g., PROJ-SEQ) |
| `"$PLANE_CLI" issues create -p PROJECT_ID --name N` | Create work item |
| `"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_ID [--fields]` | Update work item |
| `"$PLANE_CLI" issues assign -p PROJECT_ID ISSUE_ID USER_ID...` | Assign work item |
| `"$PLANE_CLI" issues delete -p PROJECT_ID ISSUE_ID` | Delete work item |
| `"$PLANE_CLI" issues search QUERY [QUERY...]` | Search work items by title; show body snippets when the server supports body search |
| `"$PLANE_CLI" comments list -p PROJECT_ID -i ISSUE_ID` | List comments/activity |
| `"$PLANE_CLI" comments list -p PROJECT_ID -i ISSUE_ID --all` | Show all activity |
| `"$PLANE_CLI" comments add -p PROJECT_ID -i ISSUE_ID "text"` | Add plain text or Markdown comment |
| `"$PLANE_CLI" comments add -p PROJECT_ID -i ISSUE_ID --body-html HTML` | Add HTML comment |
| `"$PLANE_CLI" comments update -p PROJECT_ID -i ISSUE_ID COMMENT_ID "text"` | Update a plain text or Markdown comment |
| `"$PLANE_CLI" comments update -p PROJECT_ID -i ISSUE_ID COMMENT_ID --body-html HTML` | Update an HTML comment |
| `"$PLANE_CLI" comments delete -p PROJECT_ID -i ISSUE_ID COMMENT_ID [--yes]` | Delete a comment |
| `"$PLANE_CLI" cycles list -p PROJECT_ID` | List cycles |
| `"$PLANE_CLI" cycles get -p PROJECT_ID CYCLE_ID` | Get cycle details |
| `"$PLANE_CLI" cycles create -p PROJECT_ID --name N` | Create cycle |
| `"$PLANE_CLI" modules list -p PROJECT_ID` | List modules |
| `"$PLANE_CLI" modules get -p PROJECT_ID MODULE_ID` | Get module details |
| `"$PLANE_CLI" modules create -p PROJECT_ID --name N` | Create module |
| `"$PLANE_CLI" states -p PROJECT_ID` | List workflow states |
| `"$PLANE_CLI" labels -p PROJECT_ID` | List labels |
| `"$PLANE_CLI" attachments list PROJ-SEQ` | List all file attachments on a work item |
| `"$PLANE_CLI" attachments get PROJ-SEQ UUID FILE` | Download an attachment by resource UUID |
| `"$PLANE_CLI" attachments upload PROJ-SEQ FILE` | Upload a file; insert an image into the description when applicable |
| `"$PLANE_CLI" attachments complete PROJ-SEQ UUID` | Mark an attachment as uploaded |
| `"$PLANE_CLI" attachments delete PROJ-SEQ UUID` | Delete an attachment |
| `"$PLANE_CLI" get-images PROJ-SEQ DIR` | Download all images from a work item |
| `"$PLANE_CLI" get-image PROJ-SEQ UUID FILE` | Download a single image by asset UUID |

### Filters

Work item listing supports filters:

```bash
"$PLANE_CLI" issues list -p PROJECT_ID --state STATE_ID
"$PLANE_CLI" issues list -p PROJECT_ID --priority high
"$PLANE_CLI" issues list -p PROJECT_ID --assignee USER_ID
```

### Pagination

All list commands support cursor-based pagination:

```bash
# Basic list (shows pagination info)
"$PLANE_CLI" projects list

# Specify page size
"$PLANE_CLI" issues list -p PROJECT_ID --per-page 20

# Navigate pages using cursor (displayed in output)
"$PLANE_CLI" issues list -p PROJECT_ID --cursor "5:1:0"
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
"$PLANE_CLI" issues list -p PROJECT_ID --fields "id,name,state"

# Expand related objects (assignees, state, labels, etc.)
"$PLANE_CLI" issues list -p PROJECT_ID --expand "assignees,state"

# Sort results
"$PLANE_CLI" issues list -p PROJECT_ID --order-by "-created_at"
```

### Image Downloads

Work item descriptions often contain embedded images (screenshots, diagrams, etc.). These are stored as `<image-component src="UUID">` tags in the `description_html` field. Use `get-images` to download all images or `get-image` for a specific one:

```bash
# Download all images from a work item to a directory
"$PLANE_CLI" get-images PROJ-123 ./images/

# Download a single image by asset UUID (from description_html)
"$PLANE_CLI" get-image PROJ-123 20745b59-e398-460d-9532-e0d56fbe7919 ./output.png

# JSON output returns file paths and metadata
"$PLANE_CLI" get-images PROJ-123 ./images/ -f json
```

Files are named by asset UUID with the correct extension (e.g., `20745b59-....png`). If no extension is provided in the output path for `get-image`, it is auto-appended based on the image's Content-Type.

### Attachments and Image Embedding

The `attachments` command group mirrors the Plane issue-attachments API:

```bash
# List all attachments on a work item.
"$PLANE_CLI" attachments list PROJ-123 -f json

# Upload a file. Images are confirmed and inserted into description_html.
"$PLANE_CLI" attachments upload PROJ-123 ./screenshot.png -f json

# Download, complete, and delete an attachment.
"$PLANE_CLI" attachments get PROJ-123 RESOURCE_UUID ./downloaded-file
"$PLANE_CLI" attachments complete PROJ-123 RESOURCE_UUID
"$PLANE_CLI" attachments delete PROJ-123 RESOURCE_UUID
```

`attachments upload` performs the credentials, object-storage upload, and
completion steps as one operation. It accepts any file type whose MIME type
can be determined from the extension; use `--type MIME` for an extensionless or
uncommon file. JPEG, PNG, WebP, and GIF files also append this element to the
existing description:

```html
<image-component src="ASSET_UUID"></image-component>
```

The JSON result contains a safe `asset_id`, `asset_url`, file metadata, and
`inserted` status. Signed form fields and signed URLs are never printed. Use
`asset_id`, not `asset_url`, in the stored description HTML. If the description
update fails after the attachment is confirmed, the attachment is retained and
the command reports its asset ID for manual repair:

```bash
"$PLANE_CLI" issues get-short PROJ-123 -f json
"$PLANE_CLI" issues update -p PROJECT_ID ISSUE_UUID \
  --description-html '<p>Existing content.</p><image-component src="ASSET_UUID"></image-component>'
```

When creating an issue and inserting a screenshot, create the issue first and
then run `attachments upload PROJECT-SEQ FILE -f json`. The database stores
only the short `<image-component>` element. The editor creates the browser-only
React wrapper, `/api/assets/v2/...` image URL, download button, and fullscreen
button at runtime; do not paste that generated DOM into the description.

## How It Works

The source CLI is `cmd/plane`, compiled as a pure Go binary with
`CGO_ENABLED=0`. GitHub Actions cross-compiles the supported targets and
publishes the standalone binaries together with target-specific skill packages
that embed the matching binary. The binary wraps the [Plane.so REST API
v1](https://developers.plane.so/) without runtime dependencies.

## Acknowledgments

Forked from [JinkoLLC/plane-skill](https://github.com/JinkoLLC/plane-skill).

## License

[MIT](LICENSE) © 2026 [HonLuk](https://github.com/HonLuk)
