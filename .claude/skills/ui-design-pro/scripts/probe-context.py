#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Read a project's frontend state before anything is asked of the user.

    python probe-context.py [PATH] [--json]

Step 1 of the planning phase. It answers the questions that are *facts about
files* -- which framework, which major, which UI library, is there already a
token file, did a previous run leave a plan -- so the interview only spends the
user's attention on what genuinely isn't written down anywhere.

WHAT IT DELIBERATELY DOES NOT DO
--------------------------------
Two rows of the Context Probe are judgement, and this script reports the inputs
rather than the conclusion:

  * Existing screens -- density and layout conventions worth preserving. The
    script lists the files; reading them is yours.
  * Refresh vs replacement. When a theme already exists, that is a question for
    the user, and the script's job is to make sure it gets ASKED. It prints the
    requirement; it never picks an answer.

Version numbers come from `node_modules` when the project has been installed,
because that is the major that actually ships -- a declared "^3.4.0" next to an
installed 4.1.2 is exactly the drift that makes a theme get written against the
wrong API. When node_modules is absent the declared range is reported AS a
range, labelled, so it is never mistaken for a resolved version.

Exit code is 0 whatever it finds; "no project here" is a result, not an error.

Stdlib only.
"""

import argparse
import json
import re
import sys
from pathlib import Path

# Directories that never hold the project's own source, and are large enough
# that walking them turns a fast probe into a slow one.
SKIP_DIRS = {
    "node_modules", ".git", ".svn", "dist", "build", "out", ".next", ".nuxt",
    ".svelte-kit", ".output", ".turbo", ".cache", "vendor", "coverage",
    "__pycache__", ".venv", "venv", ".astro", "target",
}

# Meta-frameworks first: a Next.js project also depends on react, and reporting
# "React" there sends the whole implementation down the wrong recipe.
FE_FRAMEWORKS = [
    ("Next.js", "next"),
    ("Nuxt 3", "nuxt"),
    ("SvelteKit", "@sveltejs/kit"),
    ("Astro", "astro"),
    ("Angular", "@angular/core"),
    ("React", "react"),
    ("Vue 3", "vue"),
    ("Svelte", "svelte"),
    ("SolidJS", "solid-js"),
]

# The base framework under a meta-framework, so the compatibility matrix can
# still be applied ("Nuxt 3" is a Vue 3 project for the purpose of picking a
# UI library port).
FE_BASE = {
    "Next.js": "react",
    "Nuxt 3": "vue",
    "SvelteKit": "svelte",
}

UI_FRAMEWORKS = [
    ("Ant Design", "antd"),
    ("Ant Design Vue", "ant-design-vue"),
    ("MUI", "@mui/material"),
    ("Mantine", "@mantine/core"),
    ("Chakra UI", "@chakra-ui/react"),
    ("HeroUI", "@heroui/react"),
    ("NextUI", "@nextui-org/react"),
    ("Vuetify", "vuetify"),
    ("PrimeVue", "primevue"),
    ("PrimeNG", "primeng"),
    ("Naive UI", "naive-ui"),
    ("Element Plus", "element-plus"),
    ("Nuxt UI", "@nuxt/ui"),
    ("Skeleton", "@skeletonlabs/skeleton"),
    ("Flowbite", "flowbite"),
    ("DaisyUI", "daisyui"),
    ("Angular Material", "@angular/material"),
    ("NG-ZORRO", "ng-zorro-antd"),
    ("Radix UI (headless)", "@radix-ui/*"),
    ("Reka UI (headless)", "reka-ui"),
    ("Bits UI (headless)", "bits-ui"),
    ("Melt UI (headless)", "@melt-ui/svelte"),
    ("Kobalte (headless)", "@kobalte/core"),
    ("Ark UI (headless)", "@ark-ui/*"),
]

STYLING_ENGINES = [
    ("Tailwind", "tailwindcss"),
    ("styled-components", "styled-components"),
    ("Emotion", "@emotion/react"),
    ("vanilla-extract", "@vanilla-extract/css"),
    ("Sass", "sass"),
    ("Stitches", "@stitches/react"),
]

# Where a token file actually lives, per framework convention. Checked in this
# order; anything found beyond the list is picked up by the scan below.
TOKEN_FILE_HINTS = [
    "src/index.css", "src/app.css", "src/styles.css", "src/global.css",
    "src/globals.css", "src/main.css", "src/style.css",
    "app/globals.css", "src/app/globals.css", "styles/globals.css",
    "src/assets/css/main.css", "assets/css/main.css", "assets/css/tailwind.css",
    "resources/css/app.css", "src/styles/global.css", "src/app.postcss",
]

COMPONENT_DIR_HINTS = [
    "src/components/ui", "components/ui", "app/components/ui",
    "src/lib/components/ui", "lib/components/ui", "resources/js/components/ui",
]

SCREEN_DIR_HINTS = [
    "src/pages", "src/routes", "src/views", "src/app", "app", "pages",
    "routes", "src/screens", "resources/views",
]

# The tokens that mean "somebody already made design decisions here", as
# opposed to a stray CSS variable.
TOKEN_MARKERS = ("--background", "--foreground", "--primary", "--card", "--muted")


# ======================================================================
# package.json
# ======================================================================

def read_json(path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


def all_deps(pkg):
    deps = {}
    for key in ("dependencies", "devDependencies", "peerDependencies"):
        deps.update(pkg.get(key) or {})
    return deps


def find_dep(deps, spec):
    """Exact name, or a scope prefix when the spec ends in '*'."""
    if spec.endswith("*"):
        prefix = spec[:-1]
        for name in sorted(deps):
            if name.startswith(prefix):
                return name
        return None
    return spec if spec in deps else None


def installed_version(root, name):
    """The version on disk. This is the one that ships."""
    meta = read_json(root / "node_modules" / Path(name) / "package.json")
    return (meta or {}).get("version")


def resolve(root, deps, name):
    """(version, source) -- 'installed' is authoritative, 'declared' is a range."""
    v = installed_version(root, name)
    if v:
        return v, "installed"
    return deps.get(name, "?"), "declared"


def major_of(version):
    m = re.search(r"(\d+)", str(version or ""))
    return m.group(1) if m else "?"


def detect(root, deps, table):
    found = []
    for label, spec in table:
        name = find_dep(deps, spec)
        if not name:
            continue
        version, source = resolve(root, deps, name)
        found.append({
            "label": label, "package": name, "version": version,
            "source": source, "major": major_of(version),
        })
    return found


# ======================================================================
# CSS
# ======================================================================

def scan_css(root, limit=60):
    """Every stylesheet shallow enough to be the project's own.

    docs/design/ is excluded on purpose: the option CSS this skill generates is
    full of shadcn tokens and would otherwise read as "this project already has
    a theme". Those files are a record of REJECTED directions, and they are
    reported under prior design work instead.
    """
    out = []
    for path in sorted(root.rglob("*.css")):
        rel = path.relative_to(root)
        if any(part in SKIP_DIRS for part in rel.parts):
            continue
        if rel.parts[:2] == ("docs", "design"):
            continue
        if len(rel.parts) > 5:
            continue
        out.append(rel)
        if len(out) >= limit:
            break
    return out


def colour_form(text):
    """Which colour vocabulary the token file is written in.

    Worth reporting because mixing two in one file is a defect, and because a
    wrapped value in a Tailwind v3 setup makes the colour disappear entirely --
    v3 wraps the token in hsl() itself, so it expects a bare triplet.
    """
    m = re.search(r"--(?:background|primary|foreground)\s*:\s*([^;]+);", text)
    if not m:
        return None
    value = m.group(1).strip()
    if value.startswith("#"):
        return "hex"
    if value.startswith("oklch("):
        return "oklch"
    if value.startswith("hsl("):
        return "hsl()"
    if re.fullmatch(r"[\d.]+\s+[\d.]+%\s+[\d.]+%", value):
        return "bare HSL triplet (Tailwind v3 form)"
    return "other"


def inspect_css(root, rel):
    text = (root / rel).read_text(encoding="utf-8", errors="replace")
    markers = [t for t in TOKEN_MARKERS if t + ":" in text or t + " :" in text]
    return {
        "path": str(rel),
        "has_tokens": len(markers) >= 2,
        "tokens_found": markers,
        "has_surface_kit": "--surface-shadow" in text,
        "has_dark_block": bool(re.search(r"\.dark\s*\{|@media\s*\(prefers-color-scheme:\s*dark", text)),
        "tailwind_v4_directive": '@import "tailwindcss"' in text or "@import 'tailwindcss'" in text,
        "tailwind_v3_directive": "@tailwind base" in text,
        "comment_count": text.count("/*"),
        "colour_form": colour_form(text),
    }


# ======================================================================
# The probe
# ======================================================================

def probe(root):
    root = root.resolve()
    r = {
        "root": str(root),
        "package_json": None,
        "fe_framework": None,
        "fe_base": None,
        "ui_frameworks": [],
        "styling_engines": [],
        "font_packages": [],
        "tailwind": None,
        "shadcn": None,
        "token_files": [],
        "component_dirs": [],
        "screen_dirs": [],
        "design_docs": {},
        "conflicts": [],
        "verdict": None,
    }

    pkg_path = root / "package.json"
    pkg = read_json(pkg_path) if pkg_path.exists() else None
    composer = (root / "composer.json").exists()

    if pkg is None and not composer:
        r["verdict"] = "GREENFIELD"
        r["design_docs"] = find_design_docs(root)
        return r

    deps = all_deps(pkg) if pkg else {}
    if pkg:
        r["package_json"] = {"name": pkg.get("name"), "dep_count": len(deps)}

    fes = detect(root, deps, FE_FRAMEWORKS)
    if fes:
        r["fe_framework"] = fes[0]
        base = FE_BASE.get(fes[0]["label"])
        if base:
            for f in fes[1:]:
                if f["package"] == base:
                    r["fe_base"] = f
    elif composer:
        r["fe_framework"] = {"label": "Laravel / Blade", "package": "composer.json",
                             "version": "?", "source": "declared", "major": "?"}

    r["ui_frameworks"] = detect(root, deps, UI_FRAMEWORKS)
    r["styling_engines"] = detect(root, deps, STYLING_ENGINES)
    r["font_packages"] = sorted(n for n in deps if n.startswith(("@fontsource", "@next/font")))

    tw = next((e for e in r["styling_engines"] if e["label"] == "Tailwind"), None)
    if tw:
        r["tailwind"] = {"version": tw["version"], "major": tw["major"], "source": tw["source"],
                         "config": next((str(p.relative_to(root))
                                         for p in sorted(root.glob("tailwind.config.*"))), None)}

    cj = root / "components.json"
    if cj.exists():
        data = read_json(cj) or {}
        r["shadcn"] = {
            "path": "components.json",
            "style": data.get("style"),
            "base_color": (data.get("tailwind") or {}).get("baseColor"),
            "css_variables": (data.get("tailwind") or {}).get("cssVariables"),
            "css": (data.get("tailwind") or {}).get("css"),
        }

    # Token files: the conventional locations first, then anything else that
    # actually carries tokens, so an unconventional layout is still found.
    seen, candidates = set(), []
    for hint in TOKEN_FILE_HINTS:
        p = root / hint
        if p.exists():
            candidates.append(Path(hint))
            seen.add(hint)
    for rel in scan_css(root):
        if str(rel) not in seen:
            candidates.append(rel)
    for rel in candidates:
        try:
            info = inspect_css(root, rel)
        except OSError:
            continue
        if info["has_tokens"] or info["tailwind_v3_directive"] or info["tailwind_v4_directive"]:
            r["token_files"].append(info)

    r["component_dirs"] = [
        {"path": h, "count": len([f for f in (root / h).iterdir() if f.is_file()])}
        for h in COMPONENT_DIR_HINTS if (root / h).is_dir()
    ]
    r["screen_dirs"] = [
        {"path": h, "files": sorted(
            str(f.relative_to(root)) for f in (root / h).rglob("*")
            if f.is_file() and f.suffix in {".tsx", ".jsx", ".vue", ".svelte", ".astro", ".html", ".php"}
            and not any(part in SKIP_DIRS for part in f.relative_to(root).parts)
        )[:20]}
        for h in SCREEN_DIR_HINTS if (root / h).is_dir()
    ]
    r["design_docs"] = find_design_docs(root)

    # The declared Tailwind major and the directive in the CSS disagreeing is a
    # real and silent failure: the theme gets written against one and compiled
    # by the other.
    if r["tailwind"]:
        tw_at = (f"installed at {r['tailwind']['version']}" if r["tailwind"]["source"] == "installed"
                 else f"declared as {r['tailwind']['version']}")
        for tf in r["token_files"]:
            if tf["tailwind_v4_directive"] and r["tailwind"]["major"] == "3":
                r["conflicts"].append(
                    f"{tf['path']} uses the v4 `@import \"tailwindcss\"` but tailwindcss is {tw_at}")
            if tf["tailwind_v3_directive"] and r["tailwind"]["major"] == "4":
                r["conflicts"].append(
                    f"{tf['path']} uses the v3 `@tailwind base` but tailwindcss is {tw_at}")
    if len([u for u in r["ui_frameworks"] if "headless" not in u["label"]]) > 1:
        r["conflicts"].append(
            "more than one component library is installed -- find out which one is live "
            "before theming either")

    themed = any(t["has_tokens"] for t in r["token_files"])
    if r["design_docs"].get("ui_plan"):
        r["verdict"] = "RESUME"
    elif themed:
        r["verdict"] = "THEMED"
    else:
        r["verdict"] = "FRESH"
    return r


def find_design_docs(root):
    d = root / "docs" / "design"
    legacy = sorted(str(p.relative_to(root)) for p in root.glob("design-system/*/MASTER.md"))
    return {
        "ui_plan": "docs/design/UI-PLAN.md" if (d / "UI-PLAN.md").exists() else None,
        "decisions": "docs/design/DECISIONS.md" if (d / "DECISIONS.md").exists() else None,
        "previews": sorted(str(p.relative_to(root)) for p in d.glob("ui-options*.html")),
        "option_tokens": sorted(str(p.relative_to(root)) for p in (d / "option-tokens").glob("*.css")),
        "legacy_master": legacy,
    }


# ======================================================================
# Reporting
# ======================================================================

def ver(entry):
    tag = "" if entry["source"] == "installed" else "  (declared range, not installed)"
    return f"{entry['version']}{tag}"


NEXT_STEPS = {
    "GREENFIELD": """\
  1. Say in one line that there is no project here, and go to Greenfield:
     create ./<project-slug>/docs/design/ and NOTHING else.
  2. The stack questions are load-bearing, not confirmatory -- there is no
     package.json to read, so nothing may be inferred. Ask all three.
  3. Then the four concept questions.""",
    "FRESH": """\
  1. Report what you found above in one short paragraph.
  2. Ask the stack questions -- state the versions above BACK to the user and
     have them confirm. Detected is not confirmed.
  3. Then the four concept questions. There is no existing theme, so there is
     no refresh-vs-replacement question to ask.""",
    "THEMED": """\
  1. Report what you found above in one short paragraph, naming the token file
     and the component count.
  2. ASK THE USER: refresh or replacement?
       refresh     -- keep the token NAMES and the component structure, change
                      the values. The option set is seeded to fit what exists.
       replacement -- a new direction. The option gate runs normally and the
                      existing values are discarded.
     Do not decide this yourself and do not proceed on silence. It changes what
     gets seeded, so it has to be answered before step 5.
  3. Read the screen files listed above for density and layout conventions
     worth preserving. This script cannot judge them -- it only found them.
  4. Confirm the stack by stating the versions above back to the user.""",
    "RESUME": """\
  1. A previous planning phase already ran and left UI-PLAN.md. READ IT.
  2. Restate its direction and scope in ONE line and ask whether it still
     stands.
       still stands -- resume at step 11 (scaffold / implement). Do not reseed.
       changed      -- revise the plan in place, re-ask at Gate 2. Never write
                      a UI-PLAN-v2.md.
  3. If a theme is also already implemented, the refresh-vs-replacement
     question applies as well.""",
}


def report(r):
    out = []
    a = out.append
    a(f"Project: {r['root']}")
    a("")

    if r["verdict"] == "GREENFIELD":
        a("No package.json and no composer.json -- there is no frontend project here yet.")
        docs = r["design_docs"]
        if docs.get("previews") or docs.get("ui_plan"):
            a("")
            a("But design documents DO exist -- a previous run got partway:")
            for k, label in (("ui_plan", "plan"), ("decisions", "decisions")):
                if docs.get(k):
                    a(f"  {label:<10} {docs[k]}")
            for p in docs.get("previews", []):
                a(f"  {'preview':<10} {p}")
            a(f"  {'options':<10} {len(docs.get('option_tokens', []))} token file(s)")
    else:
        a("STACK")
        fe = r["fe_framework"]
        a(f"  FE framework      {fe['label']} {ver(fe)}" if fe
          else "  FE framework      NOT DETECTED -- ask, do not guess")
        if r["fe_base"]:
            a(f"                    (on {r['fe_base']['package']} {ver(r['fe_base'])})")
        if r["ui_frameworks"]:
            for u in r["ui_frameworks"]:
                a(f"  UI framework      {u['label']} {ver(u)}")
        elif r["shadcn"]:
            a("  UI framework      shadcn/ui (copy-in source -- components.json present)")
        else:
            a("  UI framework      none installed -- Tailwind-only or custom")
        for s in r["styling_engines"]:
            a(f"  Styling           {s['label']} {ver(s)}")
        if not r["styling_engines"]:
            a("  Styling           plain CSS / CSS Modules -- no engine package found")
        if r["tailwind"] and r["tailwind"]["config"]:
            a(f"  Tailwind config   {r['tailwind']['config']}")
        if r["font_packages"]:
            a(f"  Font packages     {', '.join(r['font_packages'])}")
        a("")

        if r["shadcn"]:
            s = r["shadcn"]
            a("SHADCN")
            a(f"  components.json   style={s['style']}  baseColor={s['base_color']}  "
              f"cssVariables={s['css_variables']}")
            a(f"  token file        {s['css']}")
            a("  -> keep the token NAMES, replace values only")
            a("")

        a("TOKENS")
        if not r["token_files"]:
            a("  none found -- no design decisions have been made in CSS yet")
        for t in r["token_files"]:
            flags = []
            if t["has_tokens"]:
                flags.append(f"{len(t['tokens_found'])} shadcn tokens")
            if t["has_surface_kit"]:
                flags.append("--surface-* kit (this skill ran before)")
            if t["has_dark_block"]:
                flags.append("dark block")
            else:
                flags.append("NO dark block")
            if t["colour_form"]:
                flags.append(f"form: {t['colour_form']}")
            if t["tailwind_v4_directive"]:
                flags.append("Tailwind v4 directive")
            if t["tailwind_v3_directive"]:
                flags.append("Tailwind v3 directive")
            if t["comment_count"]:
                flags.append(f"{t['comment_count']} comment(s)")
            a(f"  {t['path']}")
            a(f"      {' · '.join(flags)}")
        a("")

        a("COMPONENTS")
        if r["component_dirs"]:
            for c in r["component_dirs"]:
                a(f"  {c['path']}  ({c['count']} files) -- implement INTO this, don't duplicate")
        else:
            a("  no ui/ component directory found")
        a("")

        a("SCREENS  (the script cannot judge these -- read them for density and layout)")
        if r["screen_dirs"]:
            for s in r["screen_dirs"]:
                a(f"  {s['path']}/")
                for f in s["files"][:8]:
                    a(f"      {f}")
                if len(s["files"]) > 8:
                    a(f"      … {len(s['files']) - 8} more")
        else:
            a("  none found")
        a("")

        docs = r["design_docs"]
        a("PRIOR DESIGN WORK")
        found = False
        for key, label in (("ui_plan", "UI-PLAN.md"), ("decisions", "DECISIONS.md")):
            if docs.get(key):
                a(f"  {label:<16} {docs[key]}")
                found = True
        for p in docs.get("previews", []):
            a(f"  {'preview':<16} {p}")
            found = True
        if docs.get("option_tokens"):
            a(f"  {'option tokens':<16} {len(docs['option_tokens'])} file(s)")
            found = True
        for p in docs.get("legacy_master", []):
            a(f"  {'legacy':<16} {p}")
            found = True
        if not found:
            a("  none -- this is the first run")
        a("")

    if r["conflicts"]:
        a("CONFLICTS  (resolve before theming anything)")
        for c in r["conflicts"]:
            a(f"  ! {c}")
        a("")

    a(f"VERDICT: {r['verdict']}")
    a("")
    a("REQUIRED NEXT STEPS")
    a(NEXT_STEPS[r["verdict"]])
    a("")
    a("None of the above is a confirmation. Detected versions get stated back to the")
    a("user and confirmed; nothing here replaces the stack questions.")
    return "\n".join(out)


def main():
    ap = argparse.ArgumentParser(
        description="Report a project's frontend state before the interview starts.")
    ap.add_argument("path", nargs="?", default=".", help="project root (default: cwd)")
    ap.add_argument("--json", action="store_true", help="machine-readable output")
    args = ap.parse_args()

    root = Path(args.path)
    if not root.exists():
        print(f"error: {root} does not exist", file=sys.stderr)
        return 1

    result = probe(root)
    print(json.dumps(result, indent=2, ensure_ascii=False) if args.json else report(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
