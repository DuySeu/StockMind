#!/usr/bin/env python3
"""Verify that every `path` and `path:line` cited in an onboarding draft resolves.

Reads one or more markdown files (or "-" for stdin), extracts every backticked reference
that looks like a repo path, and checks it against the filesystem. Exits non-zero when any
citation is missing or points past the end of its file.

    python3 check_citations.py draft.md --root /path/to/repo
    python3 check_citations.py draft.md --verbose --json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path

INLINE_CODE = re.compile(r"`([^`\n]+)`")
CITATION = re.compile(r"^(?P<path>[^\s:]+?)(?::(?P<line>\d+)(?:-(?P<end>\d+))?)?$")
CODE_FENCE = re.compile(r"^\s*```")

# Extensions and bare filenames that make a token a path even without a directory separator.
SOURCE_SUFFIXES = frozenset(
    """
    .ts .tsx .js .jsx .mjs .cjs .vue .svelte .astro
    .py .pyi .rb .php .go .rs .java .kt .kts .scala .swift .dart .cs .fs .ex .exs .clj .lua
    .c .h .cc .cpp .hpp .m .mm .sql .graphql .gql .proto .sh .bash .zsh .ps1
    .json .yaml .yml .toml .ini .cfg .conf .env .xml .gradle .lock .md .mdx .css .scss .html
    """.split()
)
BARE_FILENAMES = frozenset(
    """
    Makefile Dockerfile Procfile Gemfile Rakefile Jenkinsfile Taskfile justfile
    go.mod go.sum go.work config.ru requirements.txt CMakeLists.txt
    """.split()
)
# Tokens that look path-shaped but never denote a file in the repo.
SKIP_PREFIXES = ("http://", "https://", "@", "-", "--", "$", "~", "/etc/", "/usr/", "/var/")
SKIP_SUBSTRINGS = ("://", "*", "(", ")", "[", "]", "{", "}", "|", "<", ">", "=")


@dataclass(frozen=True)
class Citation:
    """A single backticked reference and what checking it against the filesystem found."""

    raw: str
    path: str
    line: int | None
    status: str
    detail: str


# Extract every path-like citation from markdown text and check it against root.
def check_text(text: str, root: Path) -> list[Citation]:
    results: dict[str, Citation] = {}
    in_fence = False

    for source_line in text.splitlines():
        # Skip fenced code blocks — they hold templates and shell snippets, not claims
        if CODE_FENCE.match(source_line):
            in_fence = not in_fence
            continue
        if in_fence:
            continue

        for token in INLINE_CODE.findall(source_line):
            candidate = token.strip().rstrip(".,;:)")
            if candidate in results:
                continue

            match = CITATION.match(candidate)
            if not match:
                continue

            # Reject tokens that are symbols, flags, or URLs rather than repo paths
            path_part = match.group("path")
            if not path_part or path_part.startswith(SKIP_PREFIXES):
                continue
            if any(bad in path_part for bad in SKIP_SUBSTRINGS):
                continue
            is_path = (
                path_part in BARE_FILENAMES
                or Path(path_part).suffix.lower() in SOURCE_SUFFIXES
                or (path_part.endswith("/") and "/" in path_part)
            )
            if not is_path:
                continue

            line_part = int(match.group("line")) if match.group("line") else None

            # Resolve against the repo root and classify what we found
            target = root / path_part.lstrip("./")
            if not target.exists():
                status, detail = "missing", "no such file or directory"
            elif path_part.endswith("/") and not target.is_dir():
                status, detail = "missing", "cited as a directory but is a file"
            elif line_part is None or target.is_dir():
                status, detail = "ok", "exists"
            else:
                total = len(target.read_text(errors="replace").splitlines())
                if line_part > total:
                    status, detail = "out-of-range", f"file has {total} lines"
                else:
                    status, detail = "ok", f"line {line_part} of {total}"

            results[candidate] = Citation(candidate, path_part, line_part, status, detail)

    return sorted(results.values(), key=lambda c: (c.status != "ok", c.raw))


# Check every cited path in the given documents and report the unresolved ones
def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("files", nargs="+", help="markdown files to check, or - for stdin")
    parser.add_argument("--root", default=".", help="repo root the paths are relative to")
    parser.add_argument("--verbose", action="store_true", help="also list citations that pass")
    parser.add_argument("--json", action="store_true", help="emit machine-readable results")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    if not root.is_dir():
        print(f"error: --root {root} is not a directory", file=sys.stderr)
        return 2

    # Gather citations across every input document
    citations: list[Citation] = []
    for name in args.files:
        text = sys.stdin.read() if name == "-" else Path(name).read_text(errors="replace")
        citations.extend(check_text(text, root))

    problems = [c for c in citations if c.status != "ok"]

    if args.json:
        print(
            json.dumps(
                {
                    "root": str(root),
                    "checked": len(citations),
                    "problems": len(problems),
                    "citations": [c.__dict__ for c in citations],
                },
                indent=2,
            )
        )
        return 1 if problems else 0

    # Report: problems always, passing citations only when asked
    for citation in citations:
        if citation.status == "ok" and not args.verbose:
            continue
        marker = "ok  " if citation.status == "ok" else "FAIL"
        print(f"{marker} `{citation.raw}` — {citation.detail}")

    print(f"\n{len(citations)} citations checked against {root}, {len(problems)} unresolved")
    if problems:
        print("Fix or downgrade each one before delivering: open the real path, or state the")
        print("claim as not-inspected instead of citing a path you did not read.")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
