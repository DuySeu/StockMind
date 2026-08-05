#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Seed 3-5 design directions from the local database into the preview harness.

    python seed-options.py "<query>" [--count 5] [--brand "#RRGGBB"]
                           [--variance 1-10] [--density 1-10] [--motion 1-10]
                           [--no-animation] [--archetype dashboard|landing|ecommerce|editorial]
                           [--inspiration "Label=url=why"] [--concept "..."]
                           [--format hex|oklch]
                           [--out docs/design/ui-options.html]
                           [--token-dir docs/design/option-tokens/]
                           [--project "Name"] [--json]

This is the bridge between the searchable database and the option-picking gate.
The database returns ONE best match per domain; the gate needs 3-5 genuinely
distinct directions. This script builds them:

  1. Pulls top-N palettes / styles / font pairings for the query.
  2. Anchors the set -- one safe, one bolder, one structurally different --
     and enforces a minimum hue separation so five slots don't all come back
     as the same blue at different lightness.
  3. Maps colors.csv columns onto the shadcn token vocabulary. All colour maths
     happens in OKLCH (perceptual lightness is what makes the contrast fixes
     predictable); values are WRITTEN as hex, which is what every framework,
     every design tool, and every human reading a diff understands. `--format
     oklch` keeps the OKLCH form for a Tailwind v4 project that prefers it.
  4. Derives a deliberate dark set. colors.csv has NO dark values -- 0 of 192
     rows -- so every dark theme here is computed, not looked up.
  5. Nudges foreground lightness until every WCAG text pair clears 4.5:1, in
     BOTH modes, so the contrast gate passes by construction. An option that
     can't be fixed is dropped rather than shipped failing.

`--brand` pins the user's own colour as `primary` in every option, so the set
becomes a choice of *treatment* rather than a choice of brand colour. Without
it, the options propose colours -- that is how a project without a brand colour
discovers one.

Design dials are inferred from the query when not passed, because the concept
of the product determines them: a dashboard is dense, a landing page is not.
The same applies to `--archetype`, which decides WHICH miniature product the
preview renders every option as -- a storefront judged as an admin console is
the user judging the wrong screen.

Each option also carries a MOTION personality (still / calm / crisp / springy /
cinematic), spread around the motion dial the way surface kits are spread across
the set. Colour and type are visible in a screenshot; how a screen arrives is
not, and leaving it out of the option set decides it silently. `--no-animation`
forces every option to `still` when the user said the product does not want it.

Outputs the filled-in preview HTML plus one sidecar CSS file per option, so
contrast-check.mjs runs against them unmodified.

Stdlib only.
"""

import argparse
import json as json_module
import math
import re
import sys
from pathlib import Path

from core import DATA_DIR, search

SCRIPT_DIR = Path(__file__).parent
TEMPLATE = SCRIPT_DIR / "mockup-template.html"

# WCAG thresholds. The target sits just above the requirement so that rounding
# in the browser can't drag a passing pair back under the line.
TEXT_MIN = 4.5
TEXT_TARGET = 4.6
UI_MIN = 3.0

# Minimum OKLCH hue separation between two options' primaries, in degrees.
# Below this the two directions read as the same idea twice.
HUE_SEPARATION = 40
HUE_SEPARATION_RELAXED = 20


# ======================================================================
# Colour conversion: sRGB <-> OKLCH
# ======================================================================

def _srgb_to_linear(c):
    return c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4


def _linear_to_srgb(c):
    return c * 12.92 if c <= 0.0031308 else 1.055 * (c ** (1 / 2.4)) - 0.055


def hex_to_oklch(hex_str):
    """'#EC4899' -> (L, C, H) with L in 0..1, C in 0..~0.4, H in degrees."""
    h = hex_str.strip().lstrip("#")
    if len(h) == 3:
        h = "".join(c * 2 for c in h)
    if len(h) != 6:
        raise ValueError(f"not a hex colour: {hex_str!r}")
    r, g, b = (int(h[i:i + 2], 16) / 255 for i in (0, 2, 4))
    return srgb_to_oklch(r, g, b)


def srgb_to_oklch(r, g, b):
    lr, lg, lb = _srgb_to_linear(r), _srgb_to_linear(g), _srgb_to_linear(b)

    l = 0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb
    m = 0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb
    s = 0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb

    l_, m_, s_ = (math.copysign(abs(v) ** (1 / 3), v) for v in (l, m, s))

    L = 0.2104542553 * l_ + 0.7936177850 * m_ - 0.0040720468 * s_
    a = 1.9779984951 * l_ - 2.4285922050 * m_ + 0.4505937099 * s_
    bb = 0.0259040371 * l_ + 0.7827717662 * m_ - 0.8086757660 * s_

    C = math.hypot(a, bb)
    H = math.degrees(math.atan2(bb, a)) % 360
    return (L, C, H)


def oklch_to_srgb(L, C, H):
    """Inverse of the above, clamped into gamut (as a browser would show it)."""
    a = C * math.cos(math.radians(H))
    b = C * math.sin(math.radians(H))

    l_ = L + 0.3963377774 * a + 0.2158037573 * b
    m_ = L - 0.1055613458 * a - 0.0638541728 * b
    s_ = L - 0.0894841775 * a - 1.2914855480 * b

    l, m, s = (v ** 3 for v in (l_, m_, s_))

    lr = +4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s
    lg = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s
    lb = -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s

    return tuple(min(1.0, max(0.0, _linear_to_srgb(c))) for c in (lr, lg, lb))


def fmt_oklch(lch):
    """Format for CSS. Chroma below 0.001 is written as 0 so greys stay grey."""
    L, C, H = lch
    L = round(min(1.0, max(0.0, L)), 4)
    C = round(max(0.0, C), 4)
    if C < 0.001:
        return f"oklch({L:g} 0 0)"
    return f"oklch({L:g} {C:g} {round(H, 1):g})"


def fmt_hex(lch):
    """Format for CSS as #RRGGBB -- the default output form.

    The maths above runs in OKLCH because perceptual lightness is what makes
    the contrast nudges predictable. What gets written is hex: it is the form
    every framework accepts without a wrapper, the form a designer can paste
    into any tool, and the form that survives a copy into a Figma swatch. The
    8-bit rounding this costs is ~1/255 per channel -- three orders of
    magnitude below the 0.1 margin the contrast targets are set with, and the
    browser quantises to 8-bit anyway on all but HDR displays.
    """
    r, g, b = oklch_to_srgb(*lch)
    return "#{:02X}{:02X}{:02X}".format(*(round(c * 255) for c in (r, g, b)))


# Set from --format. Every token value in the CSS and the preview goes through
# this one function, so the two can never disagree about the colour space.
FORMATTERS = {"hex": fmt_hex, "oklch": fmt_oklch}
_fmt = fmt_hex


def fmt(lch):
    return _fmt(lch)


# ======================================================================
# WCAG contrast
# ======================================================================

def relative_luminance(lch):
    r, g, b = oklch_to_srgb(*lch)
    lr, lg, lb = _srgb_to_linear(r), _srgb_to_linear(g), _srgb_to_linear(b)
    return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb


def contrast_ratio(fg, bg):
    l1, l2 = relative_luminance(fg), relative_luminance(bg)
    lo, hi = sorted((l1, l2))
    return (hi + 0.05) / (lo + 0.05)


def fix_contrast(fg, bg, target=TEXT_TARGET, max_steps=100):
    """Walk the foreground's lightness until it clears `target` against `bg`.

    Direction is chosen by which side of the background has more headroom, so
    dark text on a light surface gets darker and light text on a dark surface
    gets lighter -- the adjustment never inverts the design's intent. Hue is
    never touched; chroma only eases off at the extremes where the gamut runs
    out. Returns (fixed_fg, achieved_ratio).
    """
    L, C, H = fg
    if contrast_ratio(fg, bg) >= target:
        return fg, contrast_ratio(fg, bg)

    bg_lum = relative_luminance(bg)
    step = -0.01 if bg_lum > 0.18 else 0.01

    best = fg
    for _ in range(max_steps):
        L += step
        if not (0.0 <= L <= 1.0):
            break
        # Very light and very dark colours can't hold much chroma; easing it
        # off keeps the nudge inside the sRGB gamut instead of clipping.
        c = C * (1 - 0.5 * max(0.0, abs(L - 0.5) * 2 - 0.8) / 0.2) if C > 0 else 0
        cand = (L, max(0.0, c), H)
        best = cand
        if contrast_ratio(cand, bg) >= target:
            return cand, contrast_ratio(cand, bg)

    return best, contrast_ratio(best, bg)


# ======================================================================
# Token set construction
# ======================================================================

# colors.csv column -> shadcn token name.
COLOR_COLUMN_MAP = [
    ("Primary",           "primary"),
    ("On Primary",        "primary-foreground"),
    ("Secondary",         "secondary"),
    ("On Secondary",      "secondary-foreground"),
    ("Accent",            "accent"),
    ("On Accent",         "accent-foreground"),
    ("Background",        "background"),
    ("Foreground",        "foreground"),
    ("Card",              "card"),
    ("Card Foreground",   "card-foreground"),
    ("Muted",             "muted"),
    ("Muted Foreground",  "muted-foreground"),
    ("Border",            "border"),
    ("Destructive",       "destructive"),
    ("On Destructive",    "destructive-foreground"),
    ("Ring",              "ring"),
]

# Every text pair contrast-check.mjs verifies, in the same order.
TEXT_PAIRS = [
    ("foreground", "background"),
    ("muted-foreground", "background"),
    ("card-foreground", "card"),
    ("popover-foreground", "popover"),
    ("primary-foreground", "primary"),
    ("secondary-foreground", "secondary"),
    ("muted-foreground", "muted"),
    ("accent-foreground", "accent"),
    ("destructive-foreground", "destructive"),
    ("sidebar-foreground", "sidebar"),
]

TOKEN_ORDER = [
    "background", "foreground", "card", "card-foreground",
    "popover", "popover-foreground", "primary", "primary-foreground",
    "secondary", "secondary-foreground", "muted", "muted-foreground",
    "accent", "accent-foreground", "destructive", "destructive-foreground",
    "border", "input", "ring", "sidebar", "sidebar-foreground",
    "chart-1", "chart-2", "chart-3", "chart-4", "chart-5",
]

# Categorical series, hue-rotated around the theme's own primary so the charts
# read as part of the design rather than as a stock rainbow dropped on top.
CHART_OFFSETS = [0, 42, 84, -42, -84]


def chart_tokens(base_hue, mode):
    L = 0.62 if mode == "light" else 0.70
    return {f"chart-{i + 1}": (L, 0.15, (base_hue + d) % 360)
            for i, d in enumerate(CHART_OFFSETS)}


def light_tokens_from_row(row):
    """Build the light OKLCH token set from one colors.csv row."""
    tok = {}
    for col, name in COLOR_COLUMN_MAP:
        val = (row.get(col) or "").strip()
        if val.startswith("#"):
            tok[name] = hex_to_oklch(val)

    if "background" not in tok or "foreground" not in tok or "primary" not in tok:
        return None

    # Columns colors.csv doesn't carry. popover mirrors card (shadcn's own
    # default); input mirrors border; sidebar is background stepped one notch
    # away from the page so the two surfaces read as distinct.
    tok.setdefault("card", tok["background"])
    tok.setdefault("card-foreground", tok["foreground"])
    tok["popover"] = tok["card"]
    tok["popover-foreground"] = tok["card-foreground"]
    tok.setdefault("border", tok["foreground"])
    tok["input"] = tok["border"]
    tok.setdefault("ring", tok["primary"])
    tok.setdefault("muted", tok["background"])
    tok.setdefault("muted-foreground", tok["foreground"])
    tok.setdefault("secondary", tok["muted"])
    tok.setdefault("secondary-foreground", tok["foreground"])
    tok.setdefault("accent", tok["muted"])
    tok.setdefault("accent-foreground", tok["foreground"])
    tok.setdefault("destructive", hex_to_oklch("#DC2626"))
    tok.setdefault("destructive-foreground", hex_to_oklch("#FFFFFF"))

    bg_L = tok["background"][0]
    sb_L = bg_L - 0.02 if bg_L > 0.5 else bg_L + 0.02
    tok["sidebar"] = (sb_L, tok["background"][1], tok["background"][2])
    tok["sidebar-foreground"] = tok["foreground"]
    return tok


# ======================================================================
# Brand colour
# ======================================================================

# When the user supplies their own colour, `primary` is the same in every
# option and the choice becomes one of treatment, not of brand colour. Accent
# is what carries the variation: each option relates it to the brand a
# different way, which is a real design decision rather than a random hue.
ACCENT_STRATEGIES = [
    ("tonal", 0),               # accent is the brand, quietened
    ("analogous", 34),          # neighbouring hue: harmonious, low tension
    ("complementary", 180),     # opposite hue: highest contrast pairing
    ("counter-analogous", -34),
    ("triadic", 120),
]


def best_on_fill(fill, target=TEXT_TARGET):
    """Readable text colour for a fill we are not allowed to move.

    `primary` is the brand colour, so it is fixed -- the foreground has to do
    all the adapting. Tinted near-white and near-black are tried first because
    they read as part of the colour; pure white and pure black are the
    guaranteed fallbacks (max(white, black) against any colour is never worse
    than 4.58:1, so this cannot fail to clear 4.5).
    """
    L, C, H = fill
    candidates = [
        (0.985, min(0.012, C * 0.1), H),
        (1.0, 0.0, 0.0),
        (0.17, min(0.03, C * 0.2), H),
        (0.0, 0.0, 0.0),
    ]
    best, best_ratio = candidates[0], 0.0
    for cand in candidates:
        ratio = contrast_ratio(cand, fill)
        if ratio >= target:
            return cand, ratio
        if ratio > best_ratio:
            best, best_ratio = cand, ratio
    return best, best_ratio


def apply_brand(light, brand, slot, label, notes):
    """Pin the brand colour as `primary` and relate the accent to it.

    Everything else -- background, neutrals, muted, destructive -- stays as the
    palette row had it, because that is where the options still differ from
    each other once the primary is shared.
    """
    strategy, offset = ACCENT_STRATEGIES[slot % len(ACCENT_STRATEGIES)]
    bL, bC, bH = brand

    light["primary"] = brand
    light["primary-foreground"], ratio = best_on_fill(brand)
    light["ring"] = brand

    aL, aC, _ = light.get("accent", (0.94, 0.03, bH))
    accent_h = (bH + offset) % 360
    # A hover surface has to stay a surface: keep the palette's own lightness
    # and only give it enough chroma to be visible as a tint.
    light["accent"] = (aL, min(max(aC, 0.025), 0.06 if aL > 0.7 else 0.10), accent_h)

    notes.append(f"{label}: primary pinned to the brand colour {fmt_hex(brand)}; "
                 f"accent is {strategy} to it (hue {accent_h:.0f}); "
                 f"text on primary {ratio:.2f}:1")
    return strategy


def brand_dark_primary(brand, dark_bg, label, notes):
    """The brand colour on a dark page, lightened only as far as legibility needs.

    A mid-dark brand colour on a dark background is a real problem -- a button
    that cannot be distinguished from the surface it sits on. Hue and chroma are
    kept exactly, lightness moves the minimum required to clear 3:1, and the
    move is reported so nobody discovers later that dark mode is off-brand.
    """
    if contrast_ratio(brand, dark_bg) >= UI_MIN:
        return brand
    L, C, H = brand
    for _ in range(120):
        L += 0.01
        if L > 1.0:
            break
        cand = (L, C, H)
        if contrast_ratio(cand, dark_bg) >= UI_MIN + 0.1:
            notes.append(f"{label}: brand {fmt_hex(brand)} sits too close to the dark "
                         f"page ({contrast_ratio(brand, dark_bg):.2f}:1); lightened to "
                         f"{fmt_hex(cand)} for dark mode only -- hue and chroma unchanged")
            return cand
    notes.append(f"{label}: brand {fmt_hex(brand)} cannot be lifted clear of the dark page; "
                 f"dark mode needs a hand-picked variant")
    return brand


# How dark the page sits, per style family. A single constant here would make
# every option's dark mode identical -- so the light sets would differ while
# the dark toggle showed one design five times, and background strategy is one
# of the axes the options are supposed to vary on.
DARK_SURFACE = {
    "sharp": 0.145,   # instrument panel: deep, high-contrast, tight steps
    "mid":   0.180,
    "depth": 0.205,   # glass and elevation need room between layers
    "round": 0.215,   # soft styles read wrong on a near-black page
}


def derive_dark(light, surface="mid", brand=None, label="", notes=None):
    """Compute a dark set from a light one.

    colors.csv is light-only, so this is where every dark theme comes from.
    Hues carry over from light; lightness and chroma are set deliberately per
    the OKLCH conventions in shadcn-tokens.md. Background never goes to pure
    black -- 0 lightness kills the chroma that keeps a dark theme from looking
    muddy, and cards stop being distinguishable from the page.

    `surface` selects how dark the page sits, keyed to the style's own
    character, so the dark previews differ from each other as much as the
    light ones do.
    """
    def hue_of(name, fallback=0.0):
        return light[name][2] if name in light else fallback

    def chroma_of(name, scale=1.0, cap=None):
        c = light[name][1] * scale if name in light else 0.0
        return min(c, cap) if cap is not None else c

    neutral_h = hue_of("background", hue_of("primary"))
    # A near-grey light background carries a meaningless hue; borrow the
    # primary's so the dark surfaces stay tinted rather than dead grey.
    if light["background"][1] < 0.01:
        neutral_h = hue_of("primary", neutral_h)

    bg_c = min(0.012, max(0.004, chroma_of("background", 1.5)))
    base = DARK_SURFACE.get(surface, DARK_SURFACE["mid"])

    # Surface steps are proportional to the base, so a deep page keeps its
    # layers tight and a lifted page spreads them -- rather than every theme
    # using the same absolute offsets regardless of how dark it starts.
    step = 0.045 + (base - 0.18) * 0.25
    dark = {
        "background":             (base, bg_c, neutral_h),
        "foreground":             (0.965, min(0.01, bg_c), neutral_h),
        "card":                   (base + step, bg_c * 1.2, neutral_h),
        "popover":                (base + step, bg_c * 1.2, neutral_h),
        "secondary":              (base + step * 2.2, bg_c * 1.5, neutral_h),
        "muted":                  (base + step * 2.2, bg_c * 1.5, neutral_h),
        "muted-foreground":       (0.72, bg_c * 2.0, neutral_h),
        "border":                 (base + step * 3.1, bg_c * 1.5, neutral_h),
        "primary":                (0.68, chroma_of("primary", 0.9, 0.19), hue_of("primary")),
        "accent":                 (base + step * 3.5, chroma_of("accent", 0.6, 0.06), hue_of("accent", hue_of("primary"))),
        "destructive":            (0.62, chroma_of("destructive", 0.95, 0.2), hue_of("destructive", 27)),
    }
    dark["card-foreground"] = dark["foreground"]
    dark["popover-foreground"] = dark["foreground"]
    dark["secondary-foreground"] = dark["foreground"]
    dark["accent-foreground"] = dark["foreground"]
    dark["input"] = (base + step * 4.0, dark["border"][1], neutral_h)
    dark["ring"] = dark["primary"]
    dark["sidebar"] = (base + step * 0.55, bg_c * 1.1, neutral_h)
    dark["sidebar-foreground"] = dark["foreground"]

    # Foregrounds that sit on a saturated fill follow that fill's own hue, so
    # they read as part of it rather than as a grey patch dropped on top.
    dark["primary-foreground"] = (0.20, min(0.03, dark["primary"][1] * 0.2), dark["primary"][2])
    dark["destructive-foreground"] = (0.17, min(0.03, dark["destructive"][1] * 0.2), dark["destructive"][2])

    # A pinned brand colour is not re-derived for dark mode. Lightening it the
    # way an unconstrained primary gets lightened would ship a different colour
    # than the one the user gave us.
    if brand is not None:
        notes = notes if notes is not None else []
        dark["primary"] = brand_dark_primary(brand, dark["background"], label, notes)
        dark["ring"] = dark["primary"]
        dark["primary-foreground"], _ = best_on_fill(dark["primary"])
        dark["accent"] = (dark["accent"][0], dark["accent"][1], light["accent"][2])
    return dark


def enforce_contrast(tokens, label, notes):
    """Bring every text pair up to 4.5:1, recording what had to move."""
    for fg, bg in TEXT_PAIRS:
        if fg not in tokens or bg not in tokens:
            continue
        before = contrast_ratio(tokens[fg], tokens[bg])
        if before >= TEXT_MIN:
            continue
        fixed, after = fix_contrast(tokens[fg], tokens[bg])
        if after < TEXT_MIN:
            notes.append(f"{label}: {fg} on {bg} unfixable ({before:.2f} -> {after:.2f})")
            return False
        tokens[fg] = fixed
        notes.append(f"{label}: {fg} on {bg} {before:.2f} -> {after:.2f}")

    # border and input are advisory at 3:1 and deliberately left alone: a quiet
    # divider is a legitimate design choice, and forcing every palette's border
    # to 3:1 would darken it out of the direction the palette was chosen for.
    # `ring` is different -- an invisible focus ring is a real accessibility
    # failure, not a stylistic one -- so that one gets pushed up.
    if "ring" in tokens:
        ratio = contrast_ratio(tokens["ring"], tokens["background"])
        if ratio < UI_MIN:
            fixed, after = fix_contrast(tokens["ring"], tokens["background"], target=UI_MIN + 0.1)
            tokens["ring"] = fixed
            notes.append(f"{label}: ring on background {ratio:.2f} -> {after:.2f}")
    return True


# ======================================================================
# Style / typography interpretation
# ======================================================================

SERIF_HINT = re.compile(r"serif|playfair|merriweather|lora|georgia|garamond|bodoni|didot|spectral|crimson|libre baskerville|source serif|dm serif|eb garamond|cormorant|frank ruhl|newsreader|literata|petrona|bitter|zilla", re.I)
MONO_HINT = re.compile(r"mono|code|courier|consolas|menlo", re.I)


def font_stack(family, category=""):
    """Turn a Google Fonts family name into a full CSS font-family stack."""
    if not family:
        return "ui-sans-serif, system-ui, sans-serif"
    if MONO_HINT.search(family):
        return f"'{family}', ui-monospace, SFMono-Regular, Menlo, monospace"
    if SERIF_HINT.search(family) or SERIF_HINT.search(category or ""):
        return f"'{family}', ui-serif, Georgia, serif"
    return f"'{family}', ui-sans-serif, system-ui, sans-serif"


# Only 6 of the 84 style rows declare --border-radius, so a single constant
# fallback would collapse radius as an axis of difference and hand back five
# options that all share one corner treatment. These read the style's own
# character instead.
SHARP_HINT = re.compile(r"\bsharp\b|geometric|grid-based|brutal|swiss|editorial|technical|precision|angular|wireframe|terminal|monospac", re.I)
ROUND_HINT = re.compile(r"\bround|\bsoft\b|pill|organic|blob|friendly|playful|bubbl|clay|cute|candy|inflat|squish", re.I)
# Deliberately narrow. "shadow" and "elevation" appear in most styles' CSS
# keyword cells, so matching them classified almost everything as `depth` --
# which collapsed both radius and dark-surface variety back to one value.
DEPTH_HINT = re.compile(r"glassmorph|neumorph|frosted|backdrop-filter|\bglass\b", re.I)

# Used when nothing in the row settles it. Stepped so consecutive slots never
# land on the same value.
RADIUS_LADDER = ["0.375rem", "0.75rem", "0.125rem", "1rem", "0.5rem"]


def radius_for(style_row, slot, notes=None):
    """Corner radius for a direction, in rem.

    Explicit CSV value wins; otherwise inferred from the style's keywords;
    otherwise stepped off a ladder so options stay visually distinct. Anything
    other than an explicit read is reported, because it is this script's
    judgement rather than the database's.
    """
    blob = style_row.get("Design System Variables") or ""
    m = re.search(r"--border-radius:\s*([0-9.]+)\s*(px|rem)", blob, re.I)
    if m:
        value, unit = float(m.group(1)), m.group(2).lower()
        rem = value / 16 if unit == "px" else value
        return f"{round(rem, 4):g}rem"
    if re.search(r"--border-radius:\s*(0|none)\b", blob, re.I):
        return "0rem"

    haystack = " ".join(filter(None, (
        style_row.get("Style Category"), style_row.get("Keywords"),
        style_row.get("CSS/Technical Keywords"),
    )))
    label = style_row.get("Style Category") or "style"
    if SHARP_HINT.search(haystack):
        if notes is not None:
            notes.append(f"radius 0rem inferred from {label!r} (sharp/geometric keywords)")
        return "0rem"
    if ROUND_HINT.search(haystack):
        if notes is not None:
            notes.append(f"radius 1rem inferred from {label!r} (rounded/soft keywords)")
        return "1rem"
    if DEPTH_HINT.search(haystack):
        if notes is not None:
            notes.append(f"radius 0.75rem inferred from {label!r} (depth/glass keywords)")
        return "0.75rem"

    fallback = RADIUS_LADDER[slot % len(RADIUS_LADDER)]
    if notes is not None:
        notes.append(f"radius {fallback} is a stepped default -- {label!r} declares none "
                     f"and its keywords don't imply one")
    return fallback


# ======================================================================
# Surface treatment -- the part of a style that is not a colour
# ======================================================================
#
# A palette, a font pairing and a radius do not make a style visible.
# Glassmorphism, brutalism and neumorphism differ from each other -- and from
# plain flat design -- in *how a surface is drawn*: border weight, shadow
# geometry, translucency, sheen. Without this layer every option renders as the
# same flat card in a different hue, which is exactly the "the options differ
# only in hue" failure the skill warns about, one level below colour.
#
# So each option also carries a surface kit: seven CSS variables that the
# preview applies to cards, buttons, inputs and overlays, and that the
# implementation ports alongside the colour tokens.
#
#   --surface-border-width   how heavy the outline is
#   --surface-shadow         resting elevation of a card
#   --surface-shadow-raised  buttons, popovers, sticky bars
#   --surface-shadow-inset   debossed inputs (neumorphism's signature)
#   --surface-blur           backdrop-filter radius; 0 for everything but glass
#   --surface-gradient       background-image on a surface (sheen, soft convexity)
#   --surface-wash           background-image on the page, so glass has
#                            something worth blurring
#
# Matched in order -- the first hit wins, so the specific techniques are listed
# before the generic ones.
# Matched against the style's NAME first, before any technique text. A style
# whose whole identity is its surface ("Claymorphism") must not be re-classified
# by a stray "thick border" three columns later -- and several rows describe a
# technique they are contrasting themselves against.
SURFACE_NAME_HINTS = [
    ("glass", re.compile(r"glassmorph|frosted|acrylic|aero\b", re.I)),
    ("soft",  re.compile(r"neumorph|claymorph|soft ui|soft.?3d", re.I)),
    ("hard",  re.compile(r"brutal|memphis|y2k|neo.?geo", re.I)),
    ("elevated", re.compile(r"material design|bento", re.I)),
]

SURFACE_HINTS = [
    ("hard",  re.compile(r"brutal|memphis|y2k|neo.?geo|\bsticker\b|hard shadow|hard offset|"
                         r"offset shadow|\bbold border|thick border|border(?:width|-b)?:?\s*[34]", re.I)),
    # No bare "blur": half the style rows say "no blur" as a *contrast* with
    # their own hard shadows, and a negation is not a match.
    ("glass", re.compile(r"glassmorph|frosted|backdrop-filter|backdrop blur|\bglass\b|acrylic|"
                         r"blur\(", re.I)),
    ("soft",  re.compile(r"neumorph|soft ui|claymorph|\bclay\b|embossed|debossed|soft.?3d|"
                         r"inflat|squish|puffy|multiple shadow|shadow.?soft|soft box-shadow", re.I)),
    ("elevated", re.compile(r"material design|elevation|card.?based|bento|\bdepth\b|"
                            r"drop shadow|box-shadow:\s*0", re.I)),
    ("outlined", re.compile(r"wireframe|\boutline|editorial|swiss|typographic|"
                            r"grid-based|magazine|newspaper|technical draw", re.I)),
    ("flat",  re.compile(r"minimal|\bflat\b|no box-shadow|shadow:\s*none|\bclean\b", re.I)),
]

# Some rows name a heavy border AND rule shadows out in the same breath
# ("thick 4px borders, no shadows, strictly 2D"). The database is being precise
# there, so the shadow half of the kit is dropped rather than overridden.
NO_SHADOW_HINT = re.compile(r"no shadows?\b|shadows?:\s*none|box-shadow:\s*none|strictly 2d|"
                            r"zero shadow|flat.{0,12}no depth", re.I)
SHADOWLESS_SWAP = {"hard": "outlined", "elevated": "flat"}

# A page-level colour wash is orthogonal to the surface kit: an "aurora
# gradient" style can be flat-surfaced and still want a tinted page.
WASH_HINT = re.compile(r"gradient|aurora|mesh|\bglow\b|vibrant background|iridescen|"
                       r"holograph|duotone", re.I)

# Nothing in the row settles it -> step the ladder, so consecutive options don't
# all land on the same treatment.
SURFACE_LADDER = ["flat", "elevated", "outlined"]

SURFACE_BLURB = {
    "flat":     "hairline border, no shadow",
    "outlined": "2px rule, no shadow",
    "elevated": "hairline border, diffuse shadow",
    "soft":     "borderless, dual soft shadow, debossed inputs",
    "glass":    "translucent surfaces, backdrop blur, lit top edge",
    "hard":     "3px border, hard offset shadow",
}


def surface_for(style_row, slot, notes=None):
    """Which surface kit a style is drawn with.

    Read from the style row's own technique cells -- Effects & Animation and
    CSS/Technical Keywords are where the database actually records shadows,
    blur and border weight -- and reported when it had to be guessed.
    """
    haystack = " ".join(filter(None, (
        style_row.get("Style Category"), style_row.get("Keywords"),
        style_row.get("Effects & Animation"), style_row.get("CSS/Technical Keywords"),
        style_row.get("Design System Variables"),
    )))
    label = style_row.get("Style Category") or "style"

    for kit, pattern in SURFACE_NAME_HINTS:
        if pattern.search(label):
            return kit, bool(WASH_HINT.search(haystack))

    for kit, pattern in SURFACE_HINTS:
        if pattern.search(haystack):
            swap = SHADOWLESS_SWAP.get(kit)
            if swap and NO_SHADOW_HINT.search(haystack):
                if notes is not None:
                    notes.append(f"surface {swap!r} rather than {kit!r} for {label!r} -- the row "
                                 f"names the border weight but rules shadows out")
                kit = swap
            return kit, bool(WASH_HINT.search(haystack))

    kit = SURFACE_LADDER[slot % len(SURFACE_LADDER)]
    if notes is not None:
        notes.append(f"surface {kit!r} is a stepped default -- {label!r} names no "
                     f"shadow, border or blur technique")
    return kit, bool(WASH_HINT.search(haystack))


# ======================================================================
# Motion: the fourth axis of a direction
# ======================================================================

# Colour, type, and surface are all visible in a screenshot. How a screen
# arrives and how it answers a pointer is not -- and it is half of what makes a
# launch page feel like a launch page. So an option carries a motion
# personality the way it carries a surface kit, and the preview renders it.
#
# The five styles are complete feels, not speed settings: the entrance
# distance, the easing curve, and the hover lift move together. `still` is what
# every option becomes when the user said no to animation.
MOTION_STYLES = ["still", "calm", "crisp", "springy", "cinematic"]

MOTION_NOTES = {
    "still": "state changes only, no entrance animation -- nothing moves that was not asked to",
    "calm": "short cross-fades and a 1px hover lift -- motion that never interrupts a number being read",
    "crisp": "fast, near-linear, no bounce -- the page behaves like print that happens to respond",
    "springy": "slight overshoot on entrance and press -- the surfaces read as physical",
    "cinematic": "long staggered reveals and a deep hover lift -- surfaces float in rather than appear",
}


def motion_style_for(tier):
    """Tier 1-10 -> one of the five personalities."""
    if tier <= 2:
        return "still"
    if tier <= 4:
        return "calm"
    if tier <= 6:
        return "crisp"
    if tier <= 8:
        return "springy"
    return "cinematic"


# The same reason the surface kit varies per slot: five options that all move
# identically have made the motion decision for the user instead of showing it.
# The dial sets the centre, the ladder spreads the set around it.
MOTION_OFFSETS = [0, 2, -2, 1, -1]


def motion_for(tier_dial, index, animated, notes=None):
    if not animated:
        return {"tier": 1, "style": "still", "note": "animation was not requested for this product"}
    # Clamped, deliberately not slid into range. Near an extreme the spread
    # compresses and several options land on the same personality -- which is
    # the honest answer: a trading terminal at dial 2 should not be offered a
    # cinematic direction just so the tab strip looks varied. Sliding the window
    # up would produce a tier-5 option for a product whose concept asked for 1.
    tier = max(1, min(10, tier_dial + MOTION_OFFSETS[index % len(MOTION_OFFSETS)]))
    style = motion_style_for(tier)
    if notes is not None and tier != tier_dial:
        notes.append(f"motion tier {tier} ({style}) for slot {index} -- spread around the "
                     f"dial's {tier_dial} so the set offers a motion choice too")
    return {"tier": tier, "style": style, "note": MOTION_NOTES[style]}


# ======================================================================
# Archetype: which product the options are demonstrated on
# ======================================================================

# The preview harness renders one miniature product, and it has to be the
# user's. An option set for a storefront shown as an admin console is answering
# a question nobody asked: the sidebar, the stat cards and the data table say
# "dashboard" far louder than the palette says anything, so the user ends up
# judging the wrong screen.
ARCHETYPE_HINTS = [
    ("ecommerce", r"e-?commerce|storefront|online store|\bshop\b|shopping|retail|checkout|"
                  r"cart\b|catalog|product page|marketplace|boutique|fashion|apparel|"
                  r"grocery|\bmenu\b|restaurant order"),
    ("editorial", r"blog|magazine|editorial|publication|news\b|newsletter|article|essay|"
                  r"journal|documentation site|knowledge base|content site|zine"),
    ("landing", r"landing|marketing site|marketing page|homepage|home page|brochure|"
                r"portfolio|agency|promo|campaign|launch|waitlist|pricing page|"
                r"showcase|company site|website for|spa\b|salon|clinic site|hotel"),
    ("dashboard", r"dashboard|admin|analytics|console|crm\b|erp\b|back.?office|portal|"
                  r"internal tool|monitoring|trading|workspace|saas app|control panel|"
                  r"inbox|ticket|report"),
]

ARCHETYPES = ["dashboard", "landing", "ecommerce", "editorial"]


def infer_archetype(query, given, notes):
    if given:
        return given
    q = query.lower()
    # Ordered most-specific first: "e-commerce landing page" is a storefront
    # whose landing page is one screen of it, and the storefront is the harder
    # thing to get right.
    for name, pattern in ARCHETYPE_HINTS:
        m = re.search(pattern, q)
        if m:
            notes.append(f"--archetype {name} inferred from {m.group(0)!r} in the concept")
            return name
    notes.append("--archetype landing (the concept names no product type; a marketing page "
                 "is the one every project has)")
    return "landing"


def shade(lch, dl, dc=1.0):
    """Same hue, moved in perceptual lightness. Used for shadow and sheen
    colours that have to stay in the theme's own family."""
    L, C, H = lch
    return (min(1.0, max(0.0, L + dl)), max(0.0, C * dc), H)


def mix(lch, other, t):
    """Move a colour t of the way toward another, keeping its own hue.

    Hue is kept because these mixes exist to derive a *weight* -- a border
    somewhere between the ink and the page -- and interpolating hue as well
    would drift the neutral off the palette's temperature.
    """
    L, C, H = lch
    return (L + (other[0] - L) * t, C + (other[1] - C) * t, H)


def composite(lch, backdrop, alpha):
    """What a translucent colour actually looks like over its backdrop.

    Compositing is a source-over blend in gamma-encoded sRGB, which is what the
    browser does and what contrast-check.mjs re-does when it reads the file. The
    result goes back into OKLCH so the contrast pass can keep nudging lightness
    in the space it works in.
    """
    fore = oklch_to_srgb(*lch)
    back = oklch_to_srgb(*backdrop)
    return srgb_to_oklch(*(f * alpha + b * (1 - alpha) for f, b in zip(fore, back)))


def fmt_alpha(lch, alpha):
    """A token value with an alpha channel, in whichever format --format chose.

    Translucent surfaces are the whole point of glassmorphism, and both the
    contrast gate and the browser composite them over the page background, so
    the value has to carry its alpha rather than be pre-flattened.
    """
    a = round(min(1.0, max(0.0, alpha)), 3)
    if _fmt is fmt_oklch:
        L, C, H = lch
        L, C = round(min(1.0, max(0.0, L)), 4), round(max(0.0, C), 4)
        if C < 0.001:
            return f"oklch({L:g} 0 0 / {a:g})"
        return f"oklch({L:g} {C:g} {round(H, 1):g} / {a:g})"
    return fmt_hex(lch) + f"{round(a * 255):02X}"


def ink(alpha, mode):
    """Shadow colour. Real shadows are the absence of light, not a tinted
    surface, so these stay neutral -- and dark mode needs them heavier to read
    against a dark background at all."""
    a = alpha * (1.9 if mode == "dark" else 1.0)
    return f"rgb(0 0 0 / {round(min(0.85, a), 3):g})"


# How translucent a glass surface is. High enough that text on it still clears
# 4.5:1 once composited, low enough to read as glass rather than as a card.
GLASS_ALPHA = {"light": 0.78, "dark": 0.72}


def surface_tokens(kit, tokens, mode, wash):
    """The seven --surface-* values for one option in one colour mode.

    `tokens` is the mode's OKLCH map, so every derived colour comes out of the
    option's own palette rather than from a fixed grey.
    """
    card = tokens.get("card") or tokens["background"]
    fg = tokens["foreground"]
    primary = tokens["primary"]
    accent = tokens.get("accent") or primary
    dark = mode == "dark"

    out = {
        "surface-border-width": "1px",
        "surface-shadow": "none",
        "surface-shadow-raised": "none",
        "surface-shadow-inset": "none",
        "surface-blur": "0px",
        "surface-gradient": "none",
        "surface-wash": "none",
    }

    if kit == "outlined":
        out["surface-border-width"] = "2px"

    elif kit == "elevated":
        out["surface-shadow"] = f"0 1px 2px 0 {ink(0.05, mode)}, 0 10px 24px -10px {ink(0.16, mode)}"
        out["surface-shadow-raised"] = f"0 2px 4px -1px {ink(0.08, mode)}, 0 14px 32px -12px {ink(0.22, mode)}"

    elif kit == "soft":
        lift = fmt(shade(card, 0.075 if not dark else 0.055))
        press = fmt(shade(card, -0.075 if not dark else -0.045))
        out["surface-border-width"] = "0px"
        out["surface-shadow"] = f"-6px -6px 14px {lift}, 7px 7px 18px {press}"
        out["surface-shadow-raised"] = f"-3px -3px 8px {lift}, 4px 4px 10px {press}"
        out["surface-shadow-inset"] = f"inset 3px 3px 7px {press}, inset -3px -3px 7px {lift}"
        out["surface-gradient"] = f"linear-gradient(145deg, {fmt(shade(card, 0.02))}, {fmt(shade(card, -0.02))})"

    elif kit == "glass":
        sheen = 0.20 if not dark else 0.12
        out["surface-blur"] = "14px"
        out["surface-shadow"] = f"0 8px 32px -12px {ink(0.22, mode)}"
        out["surface-shadow-raised"] = f"0 14px 44px -14px {ink(0.30, mode)}"
        out["surface-shadow-inset"] = f"inset 0 1px 0 0 rgb(255 255 255 / {sheen:g})"
        out["surface-gradient"] = (f"linear-gradient(160deg, rgb(255 255 255 / {sheen:g}), "
                                  f"rgb(255 255 255 / {sheen / 4:g}))")
        wash = True

    elif kit == "hard":
        edge = fmt(fg) if not dark else fmt(shade(fg, -0.12))
        out["surface-border-width"] = "3px"
        out["surface-shadow"] = f"5px 5px 0 0 {edge}"
        out["surface-shadow-raised"] = f"3px 3px 0 0 {edge}"

    if wash:
        a1, a2 = (0.16, 0.10) if not dark else (0.24, 0.16)
        out["surface-wash"] = (
            f"radial-gradient(90% 70% at 12% 0%, {fmt_alpha(primary, a1)}, transparent 60%), "
            f"radial-gradient(80% 60% at 100% 8%, {fmt_alpha(accent, a2)}, transparent 62%)")

    return out


DENSITY_LABELS = [
    (3, "spacious"),
    (7, "comfortable"),
    (10, "compact"),
]


def density_label(dial):
    if dial is None:
        return "comfortable"
    for ceiling, label in DENSITY_LABELS:
        if dial <= ceiling:
            return label
    return "compact"


def rows(result):
    """Unwrap core.search()'s envelope. A missing file or a 0-result query both
    surface as an empty list, so callers report the miss instead of proceeding
    on fabricated data."""
    if not isinstance(result, dict) or result.get("error"):
        return []
    return result.get("results") or []


def load_all(filename):
    """Every row of a data CSV, in file order.

    BM25 only returns rows that match the query at all, so a narrow query
    ("beauty spa") yields one hue family and a handful of styles. That is the
    correct relevance answer and the wrong answer for building a *choice*: the
    structurally-different slot exists precisely to look outside the query's
    own neighbourhood. These rows are the widening pool, used only after the
    ranked results run out, and every use is reported.
    """
    import csv
    path = DATA_DIR / filename
    if not path.exists():
        return []
    with open(path, "r", encoding="utf-8") as f:
        return list(csv.DictReader(f))


WIDENED = "_widened"


def extend_pool(ranked, filename, key):
    """Ranked results first, then everything else from the CSV.

    Widened rows are tagged on the row itself rather than tracked by index, so
    the tag survives the de-duplication and complexity sorting that happen
    downstream.
    """
    seen = {_identity(r, key) for r in ranked}
    extra = []
    for r in load_all(filename):
        if _identity(r, key) in seen:
            continue
        r[WIDENED] = True
        extra.append(r)
    return ranked + extra


def slugify(text, fallback="option"):
    s = re.sub(r"[^a-z0-9]+", "-", (text or "").lower()).strip("-")
    return s or fallback


def build_thesis(style_row, color_row, typo_row):
    """One line describing what this direction actually commits to."""
    kw = [k.strip() for k in (style_row.get("Keywords") or "").split(",") if k.strip()]
    mood = [m.strip() for m in (typo_row.get("Mood/Style Keywords") or "").split(",") if m.strip()]
    notes = re.sub(r"\[.*?\]", "", color_row.get("Notes") or "").strip(" .")

    bits = []
    if kw:
        bits.append(", ".join(kw[:3]).lower())
    if notes:
        bits.append(notes[0].lower() + notes[1:])
    if mood:
        bits.append(f"{mood[0].lower()} typography")
    head = "; ".join(bits) if bits else "database-seeded direction"
    return head[0].upper() + head[1:] + "."


# ======================================================================
# Option selection
# ======================================================================

def hue_distance(a, b):
    d = abs(a - b) % 360
    return min(d, 360 - d)


def complexity_rank(style_row):
    return {"low": 0, "medium": 1, "high": 2}.get((style_row.get("Complexity") or "").strip().lower(), 1)


def shape_bucket(style_row):
    """Which corner-treatment family a style belongs to.

    Used to keep the structurally-different slot actually different: a query
    like "beauty spa wellness" ranks nothing but soft, rounded styles, so
    without this the third anchor comes back round like the other two and the
    set varies in hue alone.
    """
    blob = style_row.get("Design System Variables") or ""
    m = re.search(r"--border-radius:\s*([0-9.]+)\s*(px|rem)", blob, re.I)
    if m:
        rem = float(m.group(1)) / 16 if m.group(2).lower() == "px" else float(m.group(1))
        return "sharp" if rem <= 0.2 else ("round" if rem >= 0.6 else "mid")
    if re.search(r"--border-radius:\s*(0|none)\b", blob, re.I):
        return "sharp"

    haystack = " ".join(filter(None, (
        style_row.get("Style Category"), style_row.get("Keywords"),
        style_row.get("CSS/Technical Keywords"),
    )))
    if SHARP_HINT.search(haystack):
        return "sharp"
    if ROUND_HINT.search(haystack):
        return "round"
    if DEPTH_HINT.search(haystack):
        return "depth"
    return "mid"


def _dedupe(rows_, key):
    """First occurrence wins, ranking order preserved."""
    seen, out = set(), []
    for r in rows_:
        k = _identity(r, key)
        if k and k not in seen:
            seen.add(k)
            out.append(r)
    return out


def _identity(row, key):
    """What makes two rows the same for de-duplication purposes.

    Font pairings are identified by the families they actually name, not by
    their pairing name: typography.csv has distinct rows that resolve to the
    same two families, and two options with identical type read as one option
    however differently their source rows are labelled.
    """
    if key == "Font Pairing Name":
        heading = (row.get("Heading Font") or "").strip().lower()
        body = (row.get("Body Font") or "").strip().lower()
        if heading or body:
            return f"{heading}|{body}"
    return (row.get(key) or "").strip().lower()


def pick_options(colors, styles, typos, count, variance, brand=None, dark_ok=False):
    """Assemble `count` directions, anchored and forced apart.

    Anchoring follows the skill's rule: one safe (the database's own top
    match), one bolder, one structurally different. Without it, top-N ranking
    hands back five variations on a single idea.

    Distinctness is enforced on three axes at once, because any one of them
    alone is easy to satisfy while still shipping the same design twice: no
    style category repeats, no font pairing repeats, and no two primaries sit
    within HUE_SEPARATION degrees of each other. A slot that cannot be filled
    distinctly is reported and left empty -- a near-duplicate presented as a
    fourth choice is worse than three real ones.

    With `brand` set, the hue axis is spent: every option shares the user's
    colour as `primary`. Palettes are then chosen for their *neutrals* -- the
    page lightness and surface temperature -- and the accent strategy becomes
    the third axis in the hue's place.
    """
    if not colors or not styles or not typos:
        return [], ["no database match for one of color/style/typography"]

    notes = []
    styles = _dedupe(styles, "Style Category")
    typos = _dedupe(typos, "Font Pairing Name")

    by_complex_desc = sorted(styles, key=complexity_rank, reverse=True)
    by_complex_asc = sorted(styles, key=complexity_rank)
    serif_typos = [t for t in typos if SERIF_HINT.search(t.get("Heading Font") or "")
                   or "serif" in (t.get("Category") or "").lower()]
    mono_typos = [t for t in typos if MONO_HINT.search(t.get("Heading Font") or "")
                  or MONO_HINT.search(t.get("Body Font") or "")]

    used_styles, used_typos, used_hues, used_palettes = set(), set(), [], set()
    used_surfaces = set()

    def surface_of(row):
        """The surface kit a style row would render with, for diversity biasing.

        Distinct style *categories* are not automatically distinct *surfaces*:
        Minimalism, Flat Design and Swiss Style are three rows that all draw a
        hairline border and no shadow. Preferring an unused kit is what stops a
        set of five from being five flat cards in different hues.
        """
        return surface_for(row, len(options))[0]

    def fresh_surface(pool):
        return [s for s in pool if surface_of(s) not in used_surfaces]

    def claim(pool, used, key, *preferred_pools):
        """First unused row, checking the preferred pools before the fallback."""
        for candidate_pool in (*preferred_pools, pool):
            for row in candidate_pool:
                k = _identity(row, key)
                if k and k not in used:
                    used.add(k)
                    return row
        return None

    def primary_hue(row):
        primary = (row.get("Primary") or "").strip()
        if not primary.startswith("#"):
            return None
        try:
            return hex_to_oklch(primary)[2]
        except ValueError:
            return None

    def bg_lightness(row):
        bg = (row.get("Background") or "").strip()
        if not bg.startswith("#"):
            return None
        try:
            return hex_to_oklch(bg)[0]
        except ValueError:
            return None

    def page_signature(row):
        """What makes two pages feel like different pages.

        Not raw lightness: every light page surface in colors.csv sits between
        0.94 and 1.0, so a lightness threshold saturates after three picks and
        the rest of the set collapses onto one background. What actually reads
        as a different surface strategy is the pairing of tier (pure white /
        off-white / tinted) with tint temperature -- a cool blue-grey page and
        a warm paper page are different designs at the same lightness.
        """
        bg = (row.get("Background") or "").strip()
        if not bg.startswith("#"):
            return None
        try:
            L, C, H = hex_to_oklch(bg)
        except ValueError:
            return None
        tier = ("white" if L >= 0.99 else "off-white" if L >= 0.93
                else "tinted" if L > 0.35 else "dark")
        temp = "neutral" if C < 0.004 else int(H // 30)
        return (tier, temp)

    # A page surface is either light or dark. Mid-lightness "backgrounds" in
    # colors.csv are not pages -- the one row that has one (#888888, the spatial
    # computing palette) means a blurred passthrough behind glass. Used as a
    # page it produces a mud-grey screen that no amount of contrast fixing
    # rescues, because the fix only moves the text.
    def usable_page(row):
        L = bg_lightness(row)
        if L is None or L >= 0.85 or L <= 0.35:
            return True
        notes.append(f"skipped palette {row.get('Product Type', '?')!r}: its background "
                     f"{(row.get('Background') or '').strip()} is mid-lightness, which is a "
                     f"swatch, not a page surface")
        return False

    # A dark-first palette is a legitimate direction, and more than one of them
    # in a set of five is not. Every option already ships a derived dark mode,
    # so a second dark *page* buys nothing and costs the user a light option
    # they might have wanted. Say "dark" in the query to lift the cap.
    dark_budget = [count if dark_ok else 1]
    capped_once = [False]

    def dark_allowed(row):
        L = bg_lightness(row)
        if L is None or L >= 0.5:
            return True
        if dark_budget[0] > 0:
            return True
        if not capped_once[0]:
            capped_once[0] = True
            notes.append("dark-first pages capped at 1 in this set -- every option already has a "
                         "derived dark mode. Put 'dark' in the query to seed more of them")
        return False

    def spend_dark(row):
        L = bg_lightness(row)
        if L is not None and L < 0.5:
            dark_budget[0] -= 1

    def take_palette(min_sep):
        for idx, row in enumerate(colors):
            if idx in used_palettes:
                continue
            h = primary_hue(row)
            if h is None or not usable_page(row) or not dark_allowed(row):
                continue
            if all(hue_distance(h, u) >= min_sep for u in used_hues):
                spend_dark(row)
                return idx, row, h
        return None, None, None

    used_pages = set()

    def take_palette_pinned():
        """Palette selection when `primary` is already decided.

        Nearest-hue rows come first: a palette built around a colour in the
        brand's own family carries tints and muted surfaces that suit it. Among
        those, a page lightness the set hasn't used yet wins, so the options
        differ in surface strategy (paper white / off-white / tinted) rather
        than being five identical pages behind one shared button colour.
        """
        candidates = []
        for idx, row in enumerate(colors):
            if idx in used_palettes:
                continue
            h = primary_hue(row)
            if h is None or not usable_page(row) or not dark_allowed(row):
                continue
            candidates.append((hue_distance(h, brand[2]), idx, row))
        if not candidates:
            return None, None, None
        candidates.sort(key=lambda c: c[0])

        for _, idx, row in candidates:
            sig = page_signature(row)
            if sig is None or sig not in used_pages:
                if sig is not None:
                    used_pages.add(sig)
                spend_dark(row)
                return idx, row, brand[2]

        _, idx, row = candidates[0]
        spend_dark(row)
        notes.append("every remaining palette repeats a surface strategy already in the set -- "
                     "this option differs in style, type, and accent strategy only")
        return idx, row, brand[2]

    def note_widening(role, kind, row):
        """Say so when a slot was filled from outside the query's own matches."""
        if row is not None and row.get(WIDENED):
            notes.append(f"{role}: {kind} taken from the widened pool, not the query's "
                         f"own matches -- verify it still suits the brief")

    # variance biases which end of the complexity range fills the free slots.
    v = variance or 5
    fill_styles = by_complex_desc if v >= 7 else (by_complex_asc if v <= 3 else styles)

    roles = ["safe", "bolder", "structural"] + [f"middle-{i}" for i in range(1, count - 2)]
    options = []
    safe_bucket = None

    for role in roles[:count]:
        if role == "safe":
            style_row = claim(styles, used_styles, "Style Category")
            typo_row = claim(typos, used_typos, "Font Pairing Name")
            if style_row is not None:
                safe_bucket = shape_bucket(style_row)
        elif role == "bolder":
            style_row = claim(styles, used_styles, "Style Category",
                              fresh_surface(by_complex_desc), by_complex_desc)
            typo_row = claim(typos, used_typos, "Font Pairing Name")
        elif role == "structural":
            # Differs from `safe` in corner treatment, surface treatment AND
            # typographic character, not just in hue. When safe came back
            # rounded, sharp is preferred here explicitly -- the reverse of
            # "five soft options".
            opposite = "sharp" if safe_bucket in ("round", "depth") else "round"
            first_choice = [s for s in styles if shape_bucket(s) == opposite]
            other_bucket = [s for s in styles if shape_bucket(s) != safe_bucket]
            style_row = claim(styles, used_styles, "Style Category",
                              fresh_surface(first_choice), first_choice,
                              fresh_surface(other_bucket), other_bucket, by_complex_asc)
            typo_row = claim(typos, used_typos, "Font Pairing Name", serif_typos, mono_typos)
            if style_row is not None and shape_bucket(style_row) == safe_bucket:
                notes.append(f"structural: no style available outside the {safe_bucket!r} corner "
                             f"family -- this slot differs in type and palette only")
        else:
            style_row = claim(styles, used_styles, "Style Category",
                              fresh_surface(fill_styles), fill_styles)
            typo_row = claim(typos, used_typos, "Font Pairing Name")

        if style_row is None or typo_row is None:
            short = "styles" if style_row is None else "font pairings"
            notes.append(f"{role}: skipped -- ran out of distinct {short} "
                         f"({len(styles)} styles, {len(typos)} pairings available)")
            continue

        used_surfaces.add(surface_of(style_row))

        if brand is not None:
            idx, color_row, h = take_palette_pinned()
            if color_row is None:
                notes.append(f"{role}: skipped -- no palette rows left to draw neutrals from")
                continue
        else:
            idx, color_row, h = take_palette(HUE_SEPARATION)
            if color_row is None:
                idx, color_row, h = take_palette(HUE_SEPARATION_RELAXED)
                if color_row is not None:
                    notes.append(f"{role}: hue separation relaxed to {HUE_SEPARATION_RELAXED} degrees "
                                 f"(style and typography still differ)")
            if color_row is None:
                notes.append(f"{role}: skipped -- no palette left more than "
                             f"{HUE_SEPARATION_RELAXED} degrees from the ones already chosen")
                continue

        note_widening(role, "palette", color_row)
        note_widening(role, "style", style_row)
        note_widening(role, "font pairing", typo_row)

        used_palettes.add(idx)
        used_hues.append(h)
        options.append({"role": role, "style": style_row, "color": color_row, "typo": typo_row})

    return options, notes


def build_option(raw, index, density_dial, brand=None, motion_dial=5, animated=True):
    """Turn a (style, palette, pairing) triple into a preview-ready option."""
    style_row, color_row, typo_row = raw["style"], raw["color"], raw["typo"]
    notes = []

    light = light_tokens_from_row(color_row)
    if light is None:
        return None, [f"option {index}: palette row missing background/foreground/primary"]

    name = (style_row.get("Style Category") or f"Direction {index + 1}").split("&")[0].strip()
    ident = slugify(f"{name}-{raw['role']}", f"option-{index}")

    strategy = None
    if brand is not None:
        strategy = apply_brand(light, brand, index, ident, notes)

    dark = derive_dark(light, shape_bucket(style_row), brand=brand,
                       label=f"{ident}/dark", notes=notes)

    # The surface kit is decided before the contrast pass because a glass kit
    # makes `card` and `popover` translucent, and a translucent surface is a
    # different background to check text against. The alpha-carrying value is
    # what gets written; the composited value is what gets checked -- which is
    # exactly what the browser and contrast-check.mjs both do with it.
    kit, wash = surface_for(style_row, index, notes)

    # Two kits need the palette itself adjusted, because the treatment is not
    # something you can paint on top of any set of colours.
    if kit == "soft":
        # Soft UI extrudes a surface out of the page: the card is the SAME
        # colour as the background and the two shadows do all the work. A pure
        # white card has no headroom for a highlight above it, so the pair is
        # dropped off the ceiling -- which is why real neumorphic themes sit on
        # #E8E8E8 rather than on white.
        for mode_name, tok in (("light", light), ("dark", dark)):
            page = tok["background"]
            if mode_name == "light" and page[0] > 0.96:
                page = (0.95, page[1], page[2])
                tok["background"] = page
                notes.append(f"{ident}/{mode_name}: page dropped to 0.95 lightness so the soft "
                             f"surface has room for a highlight above it")
            for slot in ("card", "popover"):
                if slot in tok:
                    tok[slot] = page
    elif kit in ("hard", "outlined"):
        # A 3px border in a hairline colour is not a heavy border, it is a thick
        # smudge. The weight and the colour have to move together.
        weight = 0.0 if kit == "hard" else 0.45
        for mode_name, tok in (("light", light), ("dark", dark)):
            edge = mix(tok["foreground"], tok["background"], weight)
            for slot in ("border", "input"):
                if slot in tok:
                    tok[slot] = edge
        notes.append(f"{ident}: border darkened toward the ink for the {kit!r} surface "
                     f"-- a heavy border needs a heavy colour")

    translucent = {}
    if kit == "glass":
        for mode_name, tok in (("light", light), ("dark", dark)):
            alpha = GLASS_ALPHA[mode_name]
            for slot in ("card", "popover"):
                if slot in tok:
                    translucent[(mode_name, slot)] = fmt_alpha(tok[slot], alpha)
                    tok[slot] = composite(tok[slot], tok["background"], alpha)

    if not enforce_contrast(light, f"{ident}/light", notes):
        return None, notes
    if not enforce_contrast(dark, f"{ident}/dark", notes):
        return None, notes

    # Charts follow the primary, so they are derived after the contrast pass
    # rather than before it -- otherwise a nudged primary leaves the series
    # pointing at the hue it used to have.
    light.update(chart_tokens(light["primary"][2], "light"))
    dark.update(chart_tokens(dark["primary"][2], "dark"))

    radius = radius_for(style_row, index, notes)
    heading = (typo_row.get("Heading Font") or "").strip()
    body = (typo_row.get("Body Font") or "").strip()
    category = typo_row.get("Category") or ""

    light_out = {"radius": radius, **{k: fmt(light[k]) for k in TOKEN_ORDER if k in light},
                 **surface_tokens(kit, light, "light", wash)}
    dark_out = {"radius": radius, **{k: fmt(dark[k]) for k in TOKEN_ORDER if k in dark},
                **surface_tokens(kit, dark, "dark", wash)}
    for (mode_name, slot), value in translucent.items():
        (light_out if mode_name == "light" else dark_out)[slot] = value

    surface_label = f"{kit} — {SURFACE_BLURB[kit]}" + (" , tinted page" if wash and kit != "glass" else "")
    notes.append(f"surface {kit!r} ({SURFACE_BLURB[kit]}) for {name!r}"
                 + (" with a tinted page wash" if wash else ""))

    option = {
        "id": ident,
        "name": name,
        "thesis": build_thesis(style_row, color_row, typo_row),
        "hue": round(light["primary"][2]),
        "density": density_label(density_dial),
        "surface": kit,
        "surfaceNote": SURFACE_BLURB[kit] + (", tinted page" if wash and kit != "glass" else ""),
        "motion": motion_for(motion_dial, index, animated, notes),
        "fonts": {
            "display": font_stack(heading, category),
            "body": font_stack(body, category),
            "mono": "ui-monospace, SFMono-Regular, Menlo, monospace",
        },
        "light": light_out,
        "dark": dark_out,
        "_families": [f for f in (heading, body) if f],
        "_surface_label": surface_label,
        "_accent_strategy": strategy,
        "_role": raw["role"],
        "_palette": color_row.get("Product Type", ""),
        "_style": style_row.get("Style Category", ""),
        "_pairing": typo_row.get("Font Pairing Name", ""),
    }
    return option, notes


# ======================================================================
# Emitting
# ======================================================================

def js_options_literal(options):
    """Render the OPTIONS array. Hand-editable output, so it stays formatted."""
    out = ["const OPTIONS = ["]
    for o in options:
        out.append("  {")
        out.append(f'    id: {json_module.dumps(o["id"])},')
        out.append(f'    name: {json_module.dumps(o["name"])},')
        out.append(f'    thesis: {json_module.dumps(o["thesis"])},')
        out.append(f'    hue: {o["hue"]},')
        out.append(f'    density: {json_module.dumps(o["density"])},')
        out.append(f'    surface: {json_module.dumps(o["surface"])},')
        out.append(f'    surfaceNote: {json_module.dumps(o["surfaceNote"])},')
        m = o["motion"]
        out.append(f'    motion: {{ tier: {m["tier"]}, style: {json_module.dumps(m["style"])}, '
                   f'note: {json_module.dumps(m["note"])} }},')
        out.append(f'    role: {json_module.dumps(o["_role"])},')
        if o["_accent_strategy"]:
            out.append(f'    accent: {json_module.dumps(o["_accent_strategy"])},')
        out.append("    fonts: {")
        for k in ("display", "body", "mono"):
            out.append(f'      {k}: {json_module.dumps(o["fonts"][k])},')
        out.append("    },")
        for mode in ("light", "dark"):
            out.append(f"    {mode}: {{")
            for k, v in o[mode].items():
                key = k if re.fullmatch(r"[A-Za-z_$][\w$]*", k) else json_module.dumps(k)
                out.append(f"      {key}: {json_module.dumps(v)},")
            out.append("    },")
        out.append("  },")
    out.append("];")
    return "\n".join(out)


def google_fonts_link(options):
    families = sorted({f for o in options for f in o["_families"]})
    if not families:
        return ("<!-- FONTS: system stacks only, nothing to load -->", families)
    parts = "&".join(f"family={f.replace(' ', '+')}:wght@400;500;600;700" for f in families)
    url = f"https://fonts.googleapis.com/css2?{parts}&display=swap"
    link = (
        '<link rel="preconnect" href="https://fonts.googleapis.com">\n'
        '<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>\n'
        f'<link href="{url}" rel="stylesheet">'
    )
    return (link, families)


def js_inspiration_literal(refs):
    """The reference sites, as the harness's INSPIRATION array."""
    if not refs:
        return "const INSPIRATION = [];"
    out = ["const INSPIRATION = ["]
    for r in refs:
        parts = [f'label: {json_module.dumps(r["label"])}', f'url: {json_module.dumps(r["url"])}']
        if r.get("note"):
            parts.append(f'note: {json_module.dumps(r["note"])}')
        out.append("  { " + ", ".join(parts) + " },")
    out.append("];")
    return "\n".join(out)


def render_html(options, project, concept, archetype, animated, inspiration):
    template = TEMPLATE.read_text(encoding="utf-8")

    start = template.index("const OPTIONS = [")
    end = template.index("\n];", start) + len("\n];")
    html = template[:start] + js_options_literal(options) + template[end:]

    link, _ = google_fonts_link(options)
    html = re.sub(
        r'<link rel="preconnect" href="https://fonts\.googleapis\.com">.*?rel="stylesheet">',
        lambda _m: link,
        html,
        count=1,
        flags=re.S,
    )

    if project:
        html = re.sub(r'const PROJECT\s*=\s*"[^"]*"',
                      lambda _m: f'const PROJECT = {json_module.dumps(project)}',
                      html, count=1)

    # The brief and the answers that shape the preview, written in as literals
    # so the file stays self-contained and hand-editable.
    html = re.sub(r'const CONCEPT\s*=\s*"[^"]*";',
                  lambda _m: f'const CONCEPT = {json_module.dumps(concept)};', html, count=1)
    html = re.sub(r'const ARCHETYPE\s*=\s*"[^"]*";',
                  lambda _m: f'const ARCHETYPE = {json_module.dumps(archetype)};', html, count=1)
    html = re.sub(r'const ANIMATION\s*=\s*(?:true|false);',
                  f'const ANIMATION = {"true" if animated else "false"};', html, count=1)
    html = re.sub(r'const INSPIRATION = \[.*?\n\];',
                  lambda _m: js_inspiration_literal(inspiration), html, count=1, flags=re.S)
    return html


def render_option_css(option, letter):
    """Sidecar token file, in the exact shape contrast-check.mjs parses."""
    lines = [
        f"/* Option {letter} - {option['name']}",
        f"   {option['thesis']}",
        f"   Seeded from: style={option['_style']!r}, palette={option['_palette']!r},"
        f" pairing={option['_pairing']!r}",
    ]
    if option["_accent_strategy"]:
        lines.append(f"   Primary is the supplied brand colour; accent is "
                     f"{option['_accent_strategy']} to it.")
    lines += [
        f"   Fonts: display {option['fonts']['display'].split(',')[0]},"
        f" body {option['fonts']['body'].split(',')[0]}",
        f"   Surface: {option['_surface_label']} -- the --surface-* vars below carry it;"
        f" port them with the colours or the style is gone",
        f"   Motion: {option['motion']['style']} (tier {option['motion']['tier']}) --"
        f" {option['motion']['note']}. Not a token: it is the GSAP/transition pass */",
        "",
        ":root {",
    ]
    for k, v in option["light"].items():
        lines.append(f"  --{k}: {v};")
    lines += ["}", "", ".dark {"]
    for k, v in option["dark"].items():
        lines.append(f"  --{k}: {v};")
    lines += ["}", ""]
    return "\n".join(lines)


# ======================================================================
# Dials inferred from the concept
# ======================================================================

# The dials are a property of the product, not of the user's mood: an admin
# console is dense because it shows a lot of rows, a landing page is airy
# because it has one message. Asking a user to pick "compact / comfortable /
# spacious" moves that judgement onto someone who is paying for it to be made
# for them -- so it is read off the concept instead, and reported so it can be
# overridden with an explicit flag.
DENSITY_HINTS = [
    (9, r"dashboard|admin|analytics|console|trading|monitor|crm\b|erp\b|back.?office|"
        r"data.?(dense|table|grid)|spreadsheet|terminal|ticket|inbox"),
    (3, r"landing|marketing|portfolio|agency|brochure|editorial|magazine|blog|restaurant|"
        r"spa\b|wellness|luxur|hotel|gallery|showcase|storefront"),
]
VARIANCE_HINTS = [
    (8, r"creative|agency|portfolio|fashion|nightlife|gaming|entertainment|music|art\b|"
        r"experimental|bold|startup launch|nft|web3"),
    (2, r"bank|fintech|insurance|healthcare|medical|clinical|government|enterprise|legal|"
        r"compliance|payroll|accounting|utility|internal"),
]
MOTION_HINTS = [
    (8, r"landing|marketing|launch|portfolio|agency|showcase|storytell|campaign|promo"),
    (2, r"dashboard|admin|analytics|console|erp\b|crm\b|medical|healthcare|clinical|"
        r"trading|monitor|accessib|government"),
]


def infer_dials(query, given, notes):
    """Fill in whichever dials weren't passed explicitly, from the concept."""
    q = query.lower()
    resolved = {}
    for name, hints, default in (("density", DENSITY_HINTS, 5),
                                 ("variance", VARIANCE_HINTS, 5),
                                 ("motion", MOTION_HINTS, 5)):
        if given.get(name) is not None:
            resolved[name] = given[name]
            continue
        for value, pattern in hints:
            m = re.search(pattern, q)
            if m:
                resolved[name] = value
                notes.append(f"--{name} {value} inferred from {m.group(0)!r} in the concept")
                break
        else:
            resolved[name] = default
            notes.append(f"--{name} {default} (the concept gives no strong signal either way)")
    return resolved


def parse_inspiration(values):
    """'url' | 'Label=url' | 'Label=url=why they like it' -> the harness's shape."""
    refs = []
    for raw in values:
        parts = [p.strip() for p in raw.split("=", 2)]
        if len(parts) == 1 or parts[0].lower().startswith(("http", "www.")):
            url = raw.strip()
            label = re.sub(r"^https?://(www\.)?|/$", "", url)
            refs.append({"label": label, "url": url, "note": ""})
            continue
        label, url = parts[0], parts[1]
        refs.append({"label": label, "url": url, "note": parts[2] if len(parts) > 2 else ""})
    return refs


def parse_brand(raw):
    """Accept '#4F46E5', '4f46e5', or the 3-digit short form."""
    if raw is None:
        return None
    s = raw.strip().lstrip("#")
    if not re.fullmatch(r"[0-9A-Fa-f]{3}|[0-9A-Fa-f]{6}", s):
        raise ValueError(f"{raw!r} is not a hex colour")
    return hex_to_oklch("#" + s)


# ======================================================================
# CLI
# ======================================================================

def main():
    ap = argparse.ArgumentParser(
        description="Seed 3-5 design directions from the database into the preview harness.")
    ap.add_argument("query", help="what the product IS: type + industry + audience, "
                                  "e.g. 'beauty spa booking site for walk-in clients'")
    ap.add_argument("--count", "-n", type=int, default=5, help="how many directions (3-5; default 5)")
    ap.add_argument("--brand", "-b", metavar="HEX", default=None,
                    help="the user's own main colour, e.g. '#4F46E5'. Pins `primary` in every "
                         "option so the choice is about treatment, not about the colour. Omit "
                         "when the project has no brand colour -- then the options propose one")
    ap.add_argument("--format", dest="fmt", choices=sorted(FORMATTERS), default="hex",
                    help="colour form written to the CSS and preview (default: hex)")
    ap.add_argument("--variance", type=int, choices=range(1, 11), metavar="1-10",
                    help="1=centered/minimal, 10=bold/asymmetric; biases the free slots. "
                         "Inferred from the concept when omitted")
    ap.add_argument("--motion", type=int, choices=range(1, 11), metavar="1-10",
                    help="how much the product should move. Sets the centre of the motion "
                         "personalities the options are spread across, and the later GSAP "
                         "tier. Inferred from the concept when omitted")
    ap.add_argument("--no-animation", dest="animated", action="store_false",
                    help="the user said this product does not want animation. Every option "
                         "previews at the `still` tier and the preview's motion controls are "
                         "disabled -- nothing is sold that was not asked for")
    ap.add_argument("--archetype", choices=ARCHETYPES, default=None,
                    help="which miniature product the options are demonstrated on. Inferred "
                         "from the concept when omitted")
    ap.add_argument("--inspiration", action="append", default=[], metavar="URL",
                    help="a reference site the user pointed at, repeatable. "
                         "'https://linear.app' or 'Linear=https://linear.app' or "
                         "'Linear=https://linear.app=the density'. Shown above the options, "
                         "because a direction is only judgeable against the brief it answers")
    ap.add_argument("--density", type=int, choices=range(1, 11), metavar="1-10",
                    help="1=spacious, 10=dense/dashboard. Inferred from the concept when omitted")
    ap.add_argument("--out", "-o", default="docs/design/ui-options.html",
                    help="preview HTML path (default: docs/design/ui-options.html)")
    ap.add_argument("--token-dir", "-t", default="docs/design/option-tokens",
                    help="where per-option CSS goes, for the contrast gate")
    ap.add_argument("--project", "-p", default=None, help="project name shown in the preview")
    ap.add_argument("--concept", default=None,
                    help="the brief shown above the options, in the user's words. "
                         "Defaults to the query")
    ap.add_argument("--json", action="store_true", help="machine-readable summary on stdout")
    args = ap.parse_args()

    if not 3 <= args.count <= 5:
        ap.error("--count must be between 3 and 5: fewer than 3 is not a choice, "
                 "more than 5 is not a decision")

    global _fmt
    _fmt = FORMATTERS[args.fmt]

    try:
        brand = parse_brand(args.brand)
    except ValueError as e:
        ap.error(f"--brand: {e}. Give a hex colour like '#4F46E5'")

    dial_notes = []
    dials = infer_dials(args.query,
                        {"density": args.density, "variance": args.variance,
                         "motion": args.motion},
                        dial_notes)
    archetype = infer_archetype(args.query, args.archetype, dial_notes)
    inspiration = parse_inspiration(args.inspiration)
    if not args.animated:
        dial_notes.append("animation was declined: every option previews at the `still` tier "
                          "and the motion pass is skipped in the plan")

    ranked_colors = rows(search(args.query, "color", max(40, args.count * 8)))
    ranked_styles = rows(search(args.query, "style", max(20, args.count * 4)))
    ranked_typos = rows(search(args.query, "typography", max(20, args.count * 4)))

    if not ranked_colors or not ranked_styles or not ranked_typos:
        missing = [n for n, v in (("color", ranked_colors), ("style", ranked_styles),
                                  ("typography", ranked_typos)) if not v]
        print(f"error: no database match for: {', '.join(missing)}.\n"
              f"Retry with broader keywords (try product and style separately), "
              f"and do not present an empty search as if it returned data.", file=sys.stderr)
        return 1

    # Ranked matches first, then the rest of each CSV behind them. A narrow
    # query legitimately matches only a handful of rows, and three near-
    # identical options are not a choice -- so the pool extends past the
    # ranking and every entry drawn from beyond it is reported.
    colors = extend_pool(ranked_colors, "colors.csv", "Product Type")
    styles = extend_pool(ranked_styles, "styles.csv", "Style Category")
    typos = extend_pool(ranked_typos, "typography.csv", "Font Pairing Name")

    dark_ok = bool(re.search(r"\bdark\b|midnight|night mode|noir", args.query, re.I))
    raws, notes = pick_options(colors, styles, typos, args.count, dials["variance"],
                               brand, dark_ok)
    notes = dial_notes + notes

    options, dropped = [], []
    for i, raw in enumerate(raws):
        opt, opt_notes = build_option(raw, i, dials["density"], brand,
                                      dials["motion"], args.animated)
        notes.extend(opt_notes)
        if opt is None:
            dropped.append(raw["role"])
            continue
        options.append(opt)

    if len(options) < 3:
        print(f"error: only {len(options)} option(s) survived the contrast gate "
              f"(dropped: {', '.join(dropped) or 'none'}). The gate needs at least 3.\n"
              f"Retry with a different query, or widen the palette pool.", file=sys.stderr)
        for n in notes:
            print(f"  {n}", file=sys.stderr)
        return 1

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(
        render_html(options, args.project, args.concept or args.query,
                    archetype, args.animated, inspiration),
        encoding="utf-8")

    token_dir = Path(args.token_dir)
    token_dir.mkdir(parents=True, exist_ok=True)
    letters = "ABCDE"
    written = []
    for i, opt in enumerate(options):
        css_path = token_dir / f"{letters[i]}-{opt['id']}.css"
        css_path.write_text(render_option_css(opt, letters[i]), encoding="utf-8")
        written.append(str(css_path))

    _, families = google_fonts_link(options)
    summary = {
        "query": args.query,
        "concept": args.concept or args.query,
        "archetype": archetype,
        "animation": args.animated,
        "inspiration": inspiration,
        "preview": str(out_path),
        "token_files": written,
        "google_fonts": families,
        "format": args.fmt,
        "brand": fmt_hex(brand) if brand else None,
        "dials": dials,
        "options": [
            {"letter": letters[i], "id": o["id"], "name": o["name"], "role": o["_role"],
             "thesis": o["thesis"], "primary": o["light"]["primary"], "hue": o["hue"],
             "density": o["density"], "radius": o["light"]["radius"], "fonts": o["_families"],
             "accent_strategy": o["_accent_strategy"], "surface": o["surface"],
             "motion": o["motion"],
             "seeded_from": {"style": o["_style"], "palette": o["_palette"], "pairing": o["_pairing"]}}
            for i, o in enumerate(options)
        ],
        "adjustments": notes,
        "dropped": dropped,
    }

    if args.json:
        print(json_module.dumps(summary, indent=2, ensure_ascii=False))
        return 0

    print(f"Seeded {len(options)} directions -> {out_path}")
    if brand:
        print(f"Brand colour {fmt_hex(brand)} is pinned as `primary` in every option; "
              f"they differ in surface, accent strategy, type, and shape.")
    else:
        print("No brand colour given -- each option proposes its own. That is the point of "
              "the set: the user picks a colour by picking a direction.")
    print(f"Dials: density {dials['density']}, variance {dials['variance']}, "
          f"motion {dials['motion']}  ({args.fmt} output)")
    print(f"Archetype: {archetype} -- every option renders as that product, not as a "
          f"generic screen.")
    if args.animated:
        print("Animation: on -- each option carries its own motion personality; the preview "
              "plays it and can replay it.")
    else:
        print("Animation: off -- every option previews static, as asked.")
    if inspiration:
        print("Reference sites shown above the options: "
              + ", ".join(r["label"] for r in inspiration))
    for i, o in enumerate(options):
        accent = f"accent {o['_accent_strategy']:<17}" if o["_accent_strategy"] else f"hue {o['hue']:>3}   "
        print(f"  {letters[i]}  {o['name']:<26} {o['light']['primary']:<9} {accent} "
              f"radius {o['light']['radius']:<7} {' + '.join(o['_families']) or 'system fonts'}")
        print(f"     {o['thesis']}")
        print(f"     surface {o['surface']} · motion {o['motion']['style']} "
              f"(tier {o['motion']['tier']})")
    print(f"\nToken files for the contrast gate ({len(written)}):")
    for w in written:
        print(f"  {w}")
    if families:
        print(f"\nGoogle Fonts to load: {', '.join(families)}")
    if dropped:
        print(f"\nDropped (failed the contrast gate): {', '.join(dropped)}")
    if notes:
        print(f"\nAdjustments made ({len(notes)}):")
        for n in notes:
            print(f"  {n}")
    print("\nNext: run the contrast gate on every file above, then open the preview.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
