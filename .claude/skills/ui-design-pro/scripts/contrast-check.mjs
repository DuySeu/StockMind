#!/usr/bin/env node
// WCAG contrast gate for a design-token CSS file.
//
//   node contrast-check.mjs src/index.css
//   node contrast-check.mjs src/index.css --pair sidebar-foreground:sidebar
//
// Parses the :root (light) and .dark / [data-theme="dark"] (dark) blocks, resolves
// var() references, composites translucent values over their backdrop, and checks
// each foreground/background pair.
//
//   text pairs -> 4.5:1 required. A failure exits 1.
//   UI pairs (border, input, ring vs their surface) -> 3:1 advisory warning.
//
// Understands hex, rgb(), hsl(), oklch(), and the bare "H S% L%" triplets that
// Tailwind v3 shadcn setups use.

import { readFileSync } from "node:fs";

// ---------------------------------------------------------------- colour parsing

const NAMED = { white: [1, 1, 1], black: [0, 0, 0], transparent: [0, 0, 0] };

const clamp01 = (n) => Math.min(1, Math.max(0, n));

/**
 * Parse a CSS colour to { srgb: [r,g,b] gamma-encoded 0..1, a: 0..1 }, or null.
 * Gamma-encoded rather than linear because alpha compositing happens in that
 * space in browsers — linearising early would shift blended results.
 */
function parseColor(raw) {
  const s = String(raw).trim().replace(/\s*!important$/, "");
  if (!s) return null;

  const named = NAMED[s.toLowerCase()];
  if (named) return { srgb: named, a: s.toLowerCase() === "transparent" ? 0 : 1 };

  if (s.startsWith("#")) {
    const h = s.slice(1);
    const short = h.length === 3 || h.length === 4;
    if (!short && h.length !== 6 && h.length !== 8) return null;
    const size = short ? 1 : 2;
    const chan = (i) => {
      const c = h.substr(i * size, size);
      const v = parseInt(short ? c + c : c, 16);
      return Number.isNaN(v) ? null : v / 255;
    };
    const srgb = [chan(0), chan(1), chan(2)];
    if (srgb.some((c) => c === null)) return null;
    const a = h.length === 4 || h.length === 8 ? chan(3) : 1;
    return { srgb, a };
  }

  const fn = s.match(/^(oklch|hsl|hsla|rgb|rgba)\(([^)]*)\)$/i);
  if (fn) {
    const name = fn[1].toLowerCase();
    const [main, alphaPart] = fn[2].split("/");
    const parts = main.trim().split(/[\s,]+/).filter(Boolean);
    // Comma syntax puts alpha in the 4th slot; slash syntax puts it after the "/".
    const a = alphaPart !== undefined ? pct(alphaPart.trim()) : parts[3] !== undefined ? pct(parts[3]) : 1;

    if (name === "oklch") return { srgb: oklchToSrgb(num(parts[0], true), num(parts[1]), num(parts[2])), a };
    if (name.startsWith("hsl")) return { srgb: hslToSrgb(num(parts[0]), pct(parts[1]), pct(parts[2])), a };
    if (name.startsWith("rgb")) {
      return { srgb: parts.slice(0, 3).map((p) => clamp01(p.endsWith("%") ? pct(p) : num(p) / 255)), a };
    }
  }

  // Bare Tailwind-v3 triplet: "222.2 47.4% 11.2%"
  const bare = s.split(/[\s,]+/).filter(Boolean);
  if (bare.length === 3 && bare[1].endsWith("%") && bare[2].endsWith("%")) {
    return { srgb: hslToSrgb(num(bare[0]), pct(bare[1]), pct(bare[2])), a: 1 };
  }

  return null;
}

/** Numeric value; percentages become 0..1 when `unit` is true (OKLCH lightness). */
function num(v, unit = false) {
  if (v === undefined) return 0;
  const n = parseFloat(v);
  if (Number.isNaN(n)) return 0;
  return unit && v.endsWith("%") ? n / 100 : n;
}

function pct(v) {
  if (v === undefined) return 1;
  const n = parseFloat(v);
  if (Number.isNaN(n)) return 1;
  return v.endsWith("%") ? n / 100 : n;
}

function hslToSrgb(h, s, l) {
  const hue = ((h % 360) + 360) % 360;
  const sat = clamp01(s);
  const lig = clamp01(l);
  const c = (1 - Math.abs(2 * lig - 1)) * sat;
  const x = c * (1 - Math.abs(((hue / 60) % 2) - 1));
  const m = lig - c / 2;
  const seg = [
    [c, x, 0], [x, c, 0], [0, c, x],
    [0, x, c], [x, 0, c], [c, 0, x],
  ][Math.floor(hue / 60) % 6];
  return seg.map((v) => clamp01(v + m));
}

/** OKLCH -> gamma-encoded sRGB (Björn Ottosson's matrices), gamut-clipped. */
function oklchToSrgb(L, C, H) {
  const hRad = (H * Math.PI) / 180;
  const a = C * Math.cos(hRad);
  const b = C * Math.sin(hRad);

  const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3;

  return [
    4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
  ].map((c) => clamp01(gammaEncode(c)));
}

function gammaEncode(c) {
  const v = clamp01(c);
  return v <= 0.0031308 ? v * 12.92 : 1.055 * v ** (1 / 2.4) - 0.055;
}

function gammaDecode(c) {
  const v = clamp01(c);
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

/** Source-over composite of a translucent colour onto an opaque backdrop. */
function flatten(color, backdrop) {
  if (color.a >= 1) return color.srgb;
  return color.srgb.map((c, i) => c * color.a + backdrop[i] * (1 - color.a));
}

function luminance(srgb) {
  const [r, g, b] = srgb.map(gammaDecode);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(fgSrgb, bgSrgb) {
  const a = luminance(fgSrgb);
  const b = luminance(bgSrgb);
  const [hi, lo] = a > b ? [a, b] : [b, a];
  return (hi + 0.05) / (lo + 0.05);
}

// ------------------------------------------------------------------ css parsing

/** Extract the body of every block whose selector matches `test`. Handles nesting. */
function blocksFor(css, test) {
  const bodies = [];
  const re = /([^{}]*)\{/g;
  let m;
  while ((m = re.exec(css)) !== null) {
    const selector = m[1].trim().split("\n").pop().trim();
    let depth = 1;
    let i = re.lastIndex;
    while (i < css.length && depth > 0) {
      if (css[i] === "{") depth++;
      else if (css[i] === "}") depth--;
      i++;
    }
    // Do not advance lastIndex past the body — nested blocks must be visited too.
    if (test(selector)) bodies.push(css.slice(re.lastIndex, i - 1));
  }
  return bodies;
}

function declarations(bodies) {
  const out = {};
  for (const body of bodies) {
    const re = /--([\w-]+)\s*:\s*([^;{}]+)/g;
    let m;
    while ((m = re.exec(body)) !== null) out[m[1]] = m[2].trim();
  }
  return out;
}

/** Resolve var(--x, fallback) chains, bounded so a cycle cannot hang. */
function resolve(value, vars, depth = 0) {
  if (depth > 10 || typeof value !== "string") return value;
  const m = value.match(/^var\(\s*--([\w-]+)\s*(?:,\s*([^)]+))?\)$/);
  if (!m) return value;
  const next = vars[m[1]] ?? m[2];
  return next === undefined ? value : resolve(next.trim(), vars, depth + 1);
}

// -------------------------------------------------------------------- the pairs

const TEXT = { min: 4.5, kind: "text" };
const UI = { min: 3.0, kind: "ui" };

const PAIRS = [
  ["foreground", "background", TEXT],
  ["muted-foreground", "background", TEXT],
  ["card-foreground", "card", TEXT],
  ["popover-foreground", "popover", TEXT],
  ["primary-foreground", "primary", TEXT],
  ["secondary-foreground", "secondary", TEXT],
  ["muted-foreground", "muted", TEXT],
  ["accent-foreground", "accent", TEXT],
  ["destructive-foreground", "destructive", TEXT],
  ["sidebar-foreground", "sidebar", TEXT],
  ["sidebar-accent-foreground", "sidebar-accent", TEXT],
  ["sidebar-primary-foreground", "sidebar-primary", TEXT],
  ["border", "background", UI],
  ["input", "background", UI],
  ["ring", "background", UI],
];

// ------------------------------------------------------------------------- main

const argv = process.argv.slice(2);
let file;
for (let i = 0; i < argv.length; i++) {
  if (argv[i] === "--pair") {
    const [fg, bg] = (argv[++i] ?? "").split(":");
    if (fg && bg) PAIRS.push([fg, bg, TEXT]);
  } else if (!argv[i].startsWith("--") && !file) {
    file = argv[i];
  }
}

if (!file) {
  console.error("usage: contrast-check.mjs <css-file> [--pair fg:bg ...]");
  process.exit(2);
}

let css;
try {
  css = readFileSync(file, "utf8");
} catch (err) {
  console.error(`cannot read ${file}: ${err.message}`);
  process.exit(2);
}

const themes = [
  { name: "light", vars: declarations(blocksFor(css, (s) => /(^|[\s,])(:root|html)(?![\w-])/.test(s))) },
  { name: "dark", vars: declarations(blocksFor(css, (s) => /\.dark(?![\w-])|\[data-theme=["']?dark/.test(s))) },
];

let failures = 0;
let warnings = 0;
let checked = 0;
const unparsed = new Set();

for (const theme of themes) {
  const varCount = Object.keys(theme.vars).length;
  if (varCount === 0) {
    console.log(`\n${theme.name}: no tokens found — skipped`);
    continue;
  }

  console.log(`\n${theme.name} (${varCount} tokens)`);
  console.log("  " + "-".repeat(60));

  // Translucent tokens are composited over the page background.
  const page = parseColor(resolve(theme.vars.background ?? "", theme.vars));

  for (const [fgName, bgName, level] of PAIRS) {
    const fgRaw = theme.vars[fgName];
    const bgRaw = theme.vars[bgName];
    if (fgRaw === undefined || bgRaw === undefined) continue;

    const fgColor = parseColor(resolve(fgRaw, theme.vars));
    const bgColor = parseColor(resolve(bgRaw, theme.vars));
    if (!fgColor || !bgColor) {
      if (!fgColor) unparsed.add(`--${fgName}: ${fgRaw}`);
      if (!bgColor) unparsed.add(`--${bgName}: ${bgRaw}`);
      continue;
    }

    const bg = flatten(bgColor, page ? page.srgb : [1, 1, 1]);
    const fg = flatten(fgColor, bg);

    checked++;
    const ratio = contrast(fg, bg);
    const ok = ratio >= level.min;
    let mark;
    if (ok) mark = "PASS";
    else if (level.kind === "text") { mark = "FAIL"; failures++; }
    else { mark = "WARN"; warnings++; }

    const label = `${fgName} on ${bgName}`.padEnd(46);
    console.log(`  ${mark}  ${label}${ratio.toFixed(2)}:1 (need ${level.min})`);
  }
}

if (unparsed.size > 0) {
  console.log("\nunparsed values (not checked):");
  for (const u of unparsed) console.log(`  ${u}`);
}

console.log(
  `\n${checked} pair${checked === 1 ? "" : "s"} checked — ` +
    `${failures} failing, ${warnings} advisory.`
);
if (failures) console.log("Fix by adjusting lightness, not by lowering the threshold.");
if (warnings) console.log("WARN = non-text contrast below 3:1. Acceptable for decorative dividers, not for focus rings.");

process.exit(failures > 0 ? 1 : 0);
