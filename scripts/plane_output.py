"""
Plane Output Formatting

Handles color output, table rendering, and JSON formatting.
"""
import os  # needed for NO_COLOR check
import sys
import json
from typing import Optional, List, Tuple, Dict, Any

# ---------------------------------------------------------------------------
# Color helpers — degrade gracefully when not a TTY
# ---------------------------------------------------------------------------

_COLOR_ENABLED = (
    sys.stdout.isatty()
    and os.environ.get("NO_COLOR") is None
    and os.environ.get("TERM") != "dumb"
)




def _c(code: str, text: str) -> str:
    """Wrap *text* in ANSI escape *code* if color is enabled."""
    if _COLOR_ENABLED:
        return f"\033[{code}m{text}\033[0m"
    return text


def dim(text: str) -> str:
    """Dim/grey text."""
    return _c("2", text)


def bold(text: str) -> str:
    """Bold text."""
    return _c("1", text)


def red(text: str) -> str:
    """Red text (for errors)."""
    return _c("31", text)


def green(text: str) -> str:
    """Green text (for success)."""
    return _c("32", text)


def yellow(text: str) -> str:
    """Yellow text (for warnings)."""
    return _c("33", text)


def cyan(text: str) -> str:
    """Cyan text."""
    return _c("36", text)


# ---------------------------------------------------------------------------
# Table formatting
# ---------------------------------------------------------------------------

def _truncate(text: str, width: int) -> str:
    """Truncate *text* to *width* chars, adding ellipsis if needed."""
    if len(text) <= width:
        return text
    return text[: width - 1] + "…"


def _resolve(val):
    """Pull a human-readable value out of nested dicts."""
    if isinstance(val, dict):
        return val.get("name", val.get("display_name", val.get("id", str(val))))
    if val is None:
        return "—"
    return str(val)


# Priority number → label mapping
_PRIORITY_LABELS = {0: "none", 1: "urgent", 2: "high", 3: "medium", 4: "low"}
_PRIORITY_COLORS = {
    "urgent": lambda t: _c("1;31", t),  # bold red
    "high": lambda t: _c("31", t),       # red
    "medium": lambda t: _c("33", t),     # yellow
    "low": lambda t: _c("34", t),        # blue
    "none": lambda t: dim(t),
}


def _priority_str(val) -> str:
    """Convert a priority value (int or str) to a colored label."""
    if isinstance(val, int):
        label = _PRIORITY_LABELS.get(val, str(val))
    else:
        label = str(val) if val else "none"
    color_fn = _PRIORITY_COLORS.get(label, str)
    return color_fn(label)


def _print_table(rows: List[Dict[str, Any]], columns: List[Tuple[str, str, int]]):
    """
    Print a list of dicts as an aligned table.

    *columns* is a list of (key, header, max_width) tuples.
    """
    if not rows:
        print(dim("No results."))
        return

    # Build string matrix
    header = [h for _, h, _ in columns]
    lines: List[List[str]] = []
    raw_lines: List[List[str]] = []  # without color for width calc

    for row in rows:
        cells = []
        raw_cells = []
        for key, _, maxw in columns:
            if key == "priority":
                raw_val = _resolve(row.get(key))
                raw_label = _PRIORITY_LABELS.get(row.get(key), raw_val) if isinstance(row.get(key), int) else raw_val
                cells.append(_priority_str(row.get(key)))
                raw_cells.append(raw_label)
            else:
                val = _truncate(_resolve(row.get(key)), maxw)
                cells.append(val)
                raw_cells.append(val)
        lines.append(cells)
        raw_lines.append(raw_cells)

    # Calculate column widths (max of header and content)
    widths = [len(h) for h in header]
    for raw in raw_lines:
        for i, cell in enumerate(raw):
            widths[i] = max(widths[i], len(cell))
    # Clamp to declared max
    for i, (_, _, maxw) in enumerate(columns):
        widths[i] = min(widths[i], maxw)

    # Print header
    hdr = "  ".join(bold(h.ljust(w)) for h, w in zip(header, widths))
    print(hdr)
    print(dim("─" * (sum(widths) + 2 * (len(widths) - 1))))

    # Print rows
    for cells, raw in zip(lines, raw_lines):
        parts = []
        for i, (cell, rc) in enumerate(zip(cells, raw)):
            # Pad using raw (uncolored) length
            pad = widths[i] - len(rc)
            parts.append(cell + " " * max(pad, 0))
        print("  ".join(parts))


# ---------------------------------------------------------------------------
# Output router
# ---------------------------------------------------------------------------

def format_output(data, format_type: str = "table", show_pagination: bool = True):
    """
    Route data to the appropriate formatter.

    For JSON mode, dumps the raw API response.
    For table mode, auto-detects the shape and prints a readable table.

    Args:
        data: The API response data
        format_type: 'table' or 'json'
        show_pagination: Whether to show pagination info
    """
    if format_type == "json":
        print(json.dumps(data, indent=2))
        return

    # Check for pagination
    pagination_info = None
    if isinstance(data, dict):
        next_cursor = data.get("next_cursor")
        prev_cursor = data.get("prev_cursor")
        total_results = data.get("total_results")
        total_pages = data.get("total_pages")
        count = data.get("count")

        if next_cursor or prev_cursor:
            pagination_info = {
                "next_cursor": next_cursor,
                "prev_cursor": prev_cursor,
                "total_results": total_results,
                "total_pages": total_pages,
                "count": count,
            }

    # Unwrap paginated responses
    if isinstance(data, dict) and "results" in data:
        results = data["results"]

        # Show pagination info before results
        if show_pagination and pagination_info:
            _print_pagination_info(pagination_info)

        data = results

    if isinstance(data, list):
        if not data:
            print(dim("No results."))
            return
        # Heuristic: pick table columns based on keys present
        sample = data[0] if data else {}
        if "name" in sample and "identifier" in sample:
            # Projects
            _print_table(data, [
                ("identifier", "ID", 10),
                ("name", "NAME", 40),
                ("id", "UUID", 36),
            ])
        elif "name" in sample and "priority" in sample:
            # Work items
            _print_table(data, [
                ("sequence_id", "SEQ", 8),
                ("name", "NAME", 50),
                ("priority", "PRI", 8),
                ("state", "STATE", 20),
                ("id", "UUID", 36),
            ])
        elif "name" in sample and "start_date" in sample:
            # Cycles or modules
            _print_table(data, [
                ("name", "NAME", 40),
                ("start_date", "START", 12),
                ("end_date", "END", 12),
                ("id", "UUID", 36),
            ])
        elif "display_name" in sample:
            # Members
            _print_table(data, [
                ("display_name", "NAME", 30),
                ("email", "EMAIL", 40),
                ("role", "ROLE", 10),
                ("id", "UUID", 36),
            ])
        else:
            # Generic fallback
            cols = []
            for key in ["id", "identifier", "name", "title", "state", "priority", "sequence_id"]:
                if key in sample:
                    cols.append((key, key.upper(), 40))
            if cols:
                _print_table(data, cols)
            else:
                for item in data:
                    print(json.dumps(item, indent=2))
    elif isinstance(data, dict):
        # Single-object detail view
        max_key = max((len(k) for k in data if not k.startswith("_")), default=0)
        for k, v in data.items():
            if k.startswith("_"):
                continue
            label = bold(k.ljust(max_key))
            val = _resolve(v)
            print(f"  {label}  {val}")


def _print_pagination_info(info: dict):
    """Print pagination information."""
    parts = []

    if info.get("total_results"):
        parts.append(f"total: {info['total_results']}")
    if info.get("total_pages"):
        parts.append(f"pages: {info['total_pages']}")
    if info.get("count"):
        parts.append(f"showing: {info['count']}")

    if parts:
        print(dim("Pagination: " + " | ".join(parts)))

    # Show cursor hints
    if info.get("next_cursor"):
        print(dim(f"Next page: --cursor {info['next_cursor']}"))
    if info.get("prev_cursor"):
        print(dim(f"Prev page: --cursor {info['prev_cursor']}"))

    print()  # Blank line before table


# ---------------------------------------------------------------------------
# Export priority constants for use in other modules
# ---------------------------------------------------------------------------

PRIORITY_LABELS = _PRIORITY_LABELS