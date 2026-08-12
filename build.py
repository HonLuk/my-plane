#!/usr/bin/env python3
"""
Build script for plane CLI zipapp.

Creates a single executable .pyz file from the modular codebase.

Usage:
    python build.py

Output:
    skills/SKILL.md                         - Installable skill instructions
    skills/references/work-item-description.md - Detailed body format reference
    skills/scripts/plane                    - Bundled executable CLI
"""

import sys
import shutil
import subprocess
from pathlib import Path

# Directories
ROOT_DIR = Path(__file__).parent
SCRIPTS_DIR = ROOT_DIR / "scripts"
BUILD_DIR = ROOT_DIR / "build"
LEGACY_DIST_DIR = ROOT_DIR / "dist"
SKILLS_DIR = ROOT_DIR / "skills"
SKILL_SOURCE = ROOT_DIR / "SKILL.md"
SKILL_OUTPUT = SKILLS_DIR / "SKILL.md"
REFERENCES_SOURCE = ROOT_DIR / "references"
REFERENCES_OUTPUT = SKILLS_DIR / "references"
CLI_OUTPUT = SKILLS_DIR / "scripts" / "plane"

REFERENCE_FILES = [
    "work-item-description.md",
]

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
    "commands/images.py",
]


def clean():
    """Remove build artifacts."""
    # Remove the old dist/ location so stale binaries cannot be mistaken for
    # the skill package output.
    if LEGACY_DIST_DIR.exists():
        shutil.rmtree(LEGACY_DIST_DIR)
    if BUILD_DIR.exists():
        shutil.rmtree(BUILD_DIR)
    if SKILL_OUTPUT.exists():
        SKILL_OUTPUT.unlink()
    for rel_path in REFERENCE_FILES:
        reference_output = REFERENCES_OUTPUT / rel_path
        if reference_output.exists():
            reference_output.unlink()
    if CLI_OUTPUT.exists():
        CLI_OUTPUT.unlink()
    print("✓ Cleaned build artifacts")


def prepare_skill_package():
    """Create the installable skill directory and copy its instructions."""
    if not SKILL_SOURCE.exists():
        print(f"Error: missing skill source: {SKILL_SOURCE}")
        sys.exit(1)

    SKILL_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(SKILL_SOURCE, SKILL_OUTPUT)
    print(f"✓ Copied skill instructions to {SKILL_OUTPUT}")

    for rel_path in REFERENCE_FILES:
        reference_source = REFERENCES_SOURCE / rel_path
        reference_output = REFERENCES_OUTPUT / rel_path
        if not reference_source.is_file():
            print(f"Error: missing skill reference: {reference_source}")
            sys.exit(1)

        reference_output.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(reference_source, reference_output)

    print(f"✓ Copied {len(REFERENCE_FILES)} skill reference(s) to {REFERENCES_OUTPUT}")


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

    # Place the executable beside SKILL.md so the release archive preserves
    # the relative scripts/plane path when installing the skill.
    CLI_OUTPUT.parent.mkdir(parents=True, exist_ok=True)

    # Build zipapp
    output_path = CLI_OUTPUT

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

    # Prepare the directory consumed by add-skill.
    prepare_skill_package()
    print()

    # Build zipapp
    output_path = build_zipapp()

    print(f"\n✅ Success! Output: {output_path}")
    print(f"Skill package: {SKILLS_DIR}")
    print("\nRun locally:")
    print(f"  {output_path} --help")


if __name__ == "__main__":
    main()
