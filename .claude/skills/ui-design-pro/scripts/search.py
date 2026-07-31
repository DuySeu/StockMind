#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
UI/UX Pro Max Search - BM25 search engine for UI/UX style guides
Usage: python search.py "<query>" [--domain <domain>] [--stack <stack>] [--max-results 3]
       python search.py "<query>" --design-system [-p "Project Name"]
       python search.py "<query>" --design-system --variance 8 --motion 9 --density 7

Domains: style, color, chart, landing, product, ux, typography, google-fonts, icons, gsap, react
Stacks (web only): react, nextjs, vue, nuxtjs, nuxt-ui, svelte, astro, angular, laravel,
        html-tailwind, shadcn, threejs

Design dials (1-10, only with --design-system):
  --variance   DESIGN_VARIANCE: 1=centered/minimal, 10=bold/asymmetric
  --motion     MOTION_INTENSITY: 1=subtle, 10=complex; attaches a GSAP snippet from motion.csv
  --density    VISUAL_DENSITY: 1=spacious, 10=dense/dashboard; overrides the spacing scale

Nothing here writes to disk. --design-system returns one seed recommendation for
the query; the design decisions themselves live in the project's token file.
"""

import argparse
import json as json_module
import sys
import io
from core import CSV_CONFIG, AVAILABLE_STACKS, MAX_RESULTS, UNTRUNCATED_COLS, search, search_stack
from design_system import generate_design_system

# Force UTF-8 for stdout/stderr to handle emojis on Windows (cp1252 default)
if sys.stdout.encoding and sys.stdout.encoding.lower() != 'utf-8':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
if sys.stderr.encoding and sys.stderr.encoding.lower() != 'utf-8':
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8')

TRUNCATE_AT = 300


def format_output(result, full=False):
    """Format results for Claude consumption (token-optimized)"""
    if "error" in result:
        return f"Error: {result['error']}"

    output = []
    if result.get("stack"):
        output.append("## UI Pro Max Stack Guidelines")
        output.append(f"**Stack:** {result['stack']} | **Query:** {result['query']}")
    else:
        output.append("## UI Pro Max Search Results")
        domain_note = result['domain']
        if result.get("auto_detected"):
            domain_note += " (auto-detected"
            if result.get("runner_up_domain"):
                domain_note += f", runner-up: {result['runner_up_domain']}"
            domain_note += ")"
        output.append(f"**Domain:** {domain_note} | **Query:** {result['query']}")
    output.append(f"**Source:** {result['file']} | **Found:** {result['count']} results\n")

    if result['count'] == 0:
        output.append(
            "No matches. This is not a match with an empty value -- the query "
            "did not hit the database. Retry with broader/different keywords "
            "before falling back to general defaults, and say explicitly that "
            "no database match was found if you do fall back."
        )
        suggestions = result.get("suggestions") or []
        if suggestions:
            output.append(f"**Closest known terms:** {', '.join(suggestions)}")
        return "\n".join(output)

    for i, row in enumerate(result['results'], 1):
        output.append(f"### Result {i}")
        for key, value in row.items():
            value_str = str(value)
            if not full and key not in UNTRUNCATED_COLS and len(value_str) > TRUNCATE_AT:
                value_str = value_str[:TRUNCATE_AT] + "..."
            output.append(f"- **{key}:** {value_str}")
        output.append("")

    return "\n".join(output)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="UI Pro Max Search")
    parser.add_argument("query", help="Search query")
    parser.add_argument("--domain", "-d", choices=list(CSV_CONFIG.keys()), help="Search domain")
    parser.add_argument("--stack", "-s", choices=AVAILABLE_STACKS, help=f"Stack-specific search. Available: {', '.join(AVAILABLE_STACKS)}")
    parser.add_argument("--max-results", "-n", type=int, default=MAX_RESULTS, help="Max results (default: 3)")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    parser.add_argument("--full", action="store_true", help="Do not truncate long field values in text output")
    # Design system generation
    parser.add_argument("--design-system", "-ds", action="store_true", help="Generate complete design system recommendation")
    parser.add_argument("--project-name", "-p", type=str, default=None, help="Project name for design system output")
    parser.add_argument("--format", "-f", choices=["ascii", "markdown"], default="markdown",
                        help="Output format for design system, default markdown (ignored if --json). "
                             "'ascii' draws a box with ANSI colour -- for a human at a terminal, "
                             "it costs ~3x the tokens for the same content")
    # Design dials (1-10), only applied with --design-system
    parser.add_argument("--variance", type=int, choices=range(1, 11), metavar="1-10", help="DESIGN_VARIANCE dial: 1=centered/minimal, 10=bold/asymmetric (only with --design-system)")
    parser.add_argument("--motion", type=int, choices=range(1, 11), metavar="1-10", help="MOTION_INTENSITY dial: 1=subtle, 10=complex; pulls a matching GSAP snippet from motion.csv (only with --design-system)")
    parser.add_argument("--density", type=int, choices=range(1, 11), metavar="1-10", help="VISUAL_DENSITY dial: 1=spacious, 10=dense/dashboard; overrides the spacing scale (only with --design-system)")

    args = parser.parse_args()

    # Design system takes priority
    if args.design_system:
        result = generate_design_system(
            args.query,
            args.project_name,
            args.format,
            variance=args.variance,
            motion=args.motion,
            density=args.density,
        )

        if args.json:
            print(json_module.dumps(
                {"design_system": result["design_system"]},
                indent=2, ensure_ascii=False,
            ))
        else:
            print(result["text"])
    # Stack search
    elif args.stack:
        result = search_stack(args.query, args.stack, args.max_results)
        if args.json:
            print(json_module.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(format_output(result, full=args.full))
    # Domain search
    else:
        result = search(args.query, args.domain, args.max_results)
        if args.json:
            print(json_module.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(format_output(result, full=args.full))
