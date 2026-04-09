#!/usr/bin/env python3
"""
Build script for plane CLI zipapp.

Creates a single executable .pyz file from the modular codebase.

Usage:
    python build.py

Output:
    dist/plane.pyz - Single executable file
"""

import os
import sys
import shutil
import subprocess
from pathlib import Path

# Directories
ROOT_DIR = Path(__file__).parent
SCRIPTS_DIR = ROOT_DIR / "scripts"
DIST_DIR = ROOT_DIR / "dist"
BUILD_DIR = ROOT_DIR / "build"

# Files to include (relative to scripts/)
INCLUDE_FILES = [
    "plane",
    "plane_api.py",
    "plane_output.py",
    "commands/__init__.py",
    "commands/user.py",
    "commands/workspace.py",
    "commands/projects.py",
    "commands/work_items.py",
    "commands/cycles.py",
    "commands/modules.py",
    "commands/states.py",
    "commands/labels.py",
    "commands/comments.py",
]


def clean():
    """Remove build artifacts."""
    if DIST_DIR.exists():
        shutil.rmtree(DIST_DIR)
    if BUILD_DIR.exists():
        shutil.rmtree(BUILD_DIR)
    print("✓ Cleaned build artifacts")


def build_zipapp():
    """Build the zipapp executable."""
    # Create build directory
    BUILD_DIR.mkdir(parents=True, exist_ok=True)

    # Copy files to build directory
    for rel_path in INCLUDE_FILES:
        src = SCRIPTS_DIR / rel_path
        dst = BUILD_DIR / rel_path

        # Create parent directories
        dst.parent.mkdir(parents=True, exist_ok=True)

        if rel_path == "plane":
            # Rename main script to __main__.py for zipapp
            dst = BUILD_DIR / "__main__.py"

        shutil.copy2(src, dst)

    print(f"✓ Copied {len(INCLUDE_FILES)} files to build/")

    # Create dist directory
    DIST_DIR.mkdir(parents=True, exist_ok=True)

    # Build zipapp
    output_path = DIST_DIR / "plane"

    # Use Python's zipapp module
    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "zipapp",
            str(BUILD_DIR),
            "-o", str(output_path),
            "-p", "/usr/bin/env python3",  # Add shebang for direct execution
            "-c",  # Compress
        ],
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"Error building zipapp: {result.stderr}")
        sys.exit(1)

    # Make executable
    output_path.chmod(0o755)

    size_kb = output_path.stat().st_size / 1024
    print(f"✓ Built {output_path} ({size_kb:.1f} KB)")

    return output_path


def main():
    print("Building plane CLI zipapp...\n")

    # Clean previous builds
    clean()
    print()

    # Build zipapp
    output_path = build_zipapp()

    print(f"\n✅ Success! Output: {output_path}")
    print("\nTo install:")
    print(f"  cp {output_path} ~/.local/bin/plane")
    print("  chmod +x ~/.local/bin/plane")
    print("\nOr download directly:")
    print("  curl -o ~/.local/bin/plane https://github.com/HonLuk/my-plane/releases/latest/download/plane")


if __name__ == "__main__":
    main()