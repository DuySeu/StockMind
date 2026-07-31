# shadcn Token Reference

The shadcn variable vocabulary is the lingua franca for this skill: design every option in these names, then map them onto whatever the target framework actually uses (see `framework-recipes.md`).

> **The shipped token file carries no comments.** `index.css` (or `app.css` / `globals.css` / `styles.css`) is declarations only — no `/* … */`, no section banners, and not the `/* Option B - … */` header that `seed-options.py` writes onto the option CSS you port from. Comments in the examples below annotate *this reference*; they do not travel into the project. Reasoning about values lives in `docs/design/DECISIONS.md`.

## The Token Set

| Token | Meaning | Contrast partner |
|---|---|---|
| `background` / `foreground` | Page surface and its default text | 4.5:1 |
| `card` / `card-foreground` | Raised surface (cards, panels) | 4.5:1 |
| `popover` / `popover-foreground` | Floating surface (menus, tooltips) | 4.5:1 |
| `primary` / `primary-foreground` | Main action colour and text on it | 4.5:1 |
| `secondary` / `secondary-foreground` | Low-emphasis action | 4.5:1 |
| `muted` / `muted-foreground` | Subdued surface and secondary text | 4.5:1 — **the pair that usually fails** |
| `accent` / `accent-foreground` | Hover/active surface tint | 4.5:1 |
| `destructive` / `destructive-foreground` | Danger action | 4.5:1 |
| `border` | Dividers and component outlines | 3:1 vs `background` |
| `input` | Form control border | 3:1 vs `background` |
| `ring` | Focus ring | 3:1 vs `background` |
| `radius` | Base corner radius; others derive from it | — |
| `chart-1` … `chart-5` | Categorical data-viz series | distinguishable from each other |
| `sidebar`, `sidebar-foreground`, `sidebar-primary`, `sidebar-primary-foreground`, `sidebar-accent`, `sidebar-accent-foreground`, `sidebar-border`, `sidebar-ring` | Sidebar-scoped mirror of the main set | same as their main counterparts |

Sidebar tokens are only needed when the layout has a persistent sidebar with its own surface colour. Chart tokens only when the product has data viz.

## Which Colour Form to Write

Three forms are in play, and mixing them is the most common shadcn theming bug.

| Form | Where it belongs |
|---|---|
| **Hex** — `#4F46E5` | **The default.** What `seed-options.py` writes, what the preview renders, what a designer can paste into any tool. Valid in Tailwind v4, plain CSS, CSS-in-JS, and every component library's theme object. |
| **OKLCH** — `oklch(0.51 0.19 275)` | Stock shadcn's own form, and the right choice when you will be *deriving* values by hand — hold `C` and `H`, walk `L`. Reach it with `--format oklch`. |
| **Bare HSL triplet** — `222.2 47.4% 11.2%` | Tailwind **v3** only, because `tailwind.config` wraps it in `hsl()`. A hex or `oklch()` value here renders as `hsl(#4F46E5)` and the colour vanishes. |

Whichever you pick, use it for every token in the file. Half a theme in hex and half in OKLCH is legal CSS and an unreadable diff.

`contrast-check.mjs` parses all three, plus `rgb()` and 8-digit hex with alpha, so the gate works regardless.

## Tailwind v4 Form (current shadcn default)

Shown here in OKLCH because that is what `npx shadcn init` generates — hex is equally valid in v4 and is what this skill's seeder emits. `@theme inline` is what turns the variables into Tailwind utilities (`bg-background`, `text-muted-foreground`).

```css
@import "tailwindcss";
@import "tw-animate-css";

@custom-variant dark (&:is(.dark *));

:root {
  --radius: 0.625rem;

  --background: oklch(1 0 0);
  --foreground: oklch(0.145 0 0);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.145 0 0);
  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.145 0 0);
  --primary: oklch(0.205 0 0);
  --primary-foreground: oklch(0.985 0 0);
  --secondary: oklch(0.97 0 0);
  --secondary-foreground: oklch(0.205 0 0);
  --muted: oklch(0.97 0 0);
  --muted-foreground: oklch(0.556 0 0);
  --accent: oklch(0.97 0 0);
  --accent-foreground: oklch(0.205 0 0);
  --destructive: oklch(0.577 0.245 27.325);
  --destructive-foreground: oklch(0.985 0 0);
  --border: oklch(0.922 0 0);
  --input: oklch(0.922 0 0);
  --ring: oklch(0.708 0 0);

  --chart-1: oklch(0.646 0.222 41.116);
  --chart-2: oklch(0.6 0.118 184.704);
  --chart-3: oklch(0.398 0.07 227.392);
  --chart-4: oklch(0.828 0.189 84.429);
  --chart-5: oklch(0.769 0.188 70.08);

  --sidebar: oklch(0.985 0 0);
  --sidebar-foreground: oklch(0.145 0 0);
  --sidebar-primary: oklch(0.205 0 0);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.97 0 0);
  --sidebar-accent-foreground: oklch(0.205 0 0);
  --sidebar-border: oklch(0.922 0 0);
  --sidebar-ring: oklch(0.708 0 0);
}

.dark {
  --background: oklch(0.145 0 0);
  --foreground: oklch(0.985 0 0);
  --card: oklch(0.205 0 0);
  --card-foreground: oklch(0.985 0 0);
  --popover: oklch(0.205 0 0);
  --popover-foreground: oklch(0.985 0 0);
  --primary: oklch(0.922 0 0);
  --primary-foreground: oklch(0.205 0 0);
  --secondary: oklch(0.269 0 0);
  --secondary-foreground: oklch(0.985 0 0);
  --muted: oklch(0.269 0 0);
  --muted-foreground: oklch(0.708 0 0);
  --accent: oklch(0.269 0 0);
  --accent-foreground: oklch(0.985 0 0);
  --destructive: oklch(0.704 0.191 22.216);
  --destructive-foreground: oklch(0.985 0 0);
  --border: oklch(1 0 0 / 10%);
  --input: oklch(1 0 0 / 15%);
  --ring: oklch(0.556 0 0);

  --chart-1: oklch(0.488 0.243 264.376);
  --chart-2: oklch(0.696 0.17 162.48);
  --chart-3: oklch(0.769 0.188 70.08);
  --chart-4: oklch(0.627 0.265 303.9);
  --chart-5: oklch(0.645 0.246 16.439);

  --sidebar: oklch(0.205 0 0);
  --sidebar-foreground: oklch(0.985 0 0);
  --sidebar-primary: oklch(0.488 0.243 264.376);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.269 0 0);
  --sidebar-accent-foreground: oklch(0.985 0 0);
  --sidebar-border: oklch(1 0 0 / 10%);
  --sidebar-ring: oklch(0.556 0 0);
}

@theme inline {
  --font-sans: var(--font-sans);
  --font-serif: var(--font-serif);
  --font-mono: var(--font-mono);

  --radius-sm: calc(var(--radius) - 4px);
  --radius-md: calc(var(--radius) - 2px);
  --radius-lg: var(--radius);
  --radius-xl: calc(var(--radius) + 4px);

  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-destructive-foreground: var(--destructive-foreground);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  --color-chart-1: var(--chart-1);
  --color-chart-2: var(--chart-2);
  --color-chart-3: var(--chart-3);
  --color-chart-4: var(--chart-4);
  --color-chart-5: var(--chart-5);
  --color-sidebar: var(--sidebar);
  --color-sidebar-foreground: var(--sidebar-foreground);
  --color-sidebar-primary: var(--sidebar-primary);
  --color-sidebar-primary-foreground: var(--sidebar-primary-foreground);
  --color-sidebar-accent: var(--sidebar-accent);
  --color-sidebar-accent-foreground: var(--sidebar-accent-foreground);
  --color-sidebar-border: var(--sidebar-border);
  --color-sidebar-ring: var(--sidebar-ring);
}

@layer base {
  * {
    @apply border-border outline-ring/50;
  }
  body {
    @apply bg-background text-foreground;
  }
}
```

**Do not add a `--font-sans: var(--font-sans)` line without also defining `--font-sans` in `:root`** — it resolves to nothing and silently falls back. Define the family in `:root`, alias it in `@theme inline`.

## Tailwind v3 Form

v3 stores colours as **bare HSL channel triplets** (no `hsl()` wrapper) because `tailwind.config` wraps them. Mixing the two forms is the single most common shadcn theming bug: a v4-style `oklch(...)` value inside a v3 setup renders as `hsl(oklch(...))` and the colour disappears.

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --radius: 0.5rem;
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --card: 0 0% 100%;
    --card-foreground: 222.2 84% 4.9%;
    --popover: 0 0% 100%;
    --popover-foreground: 222.2 84% 4.9%;
    --primary: 222.2 47.4% 11.2%;
    --primary-foreground: 210 40% 98%;
    --secondary: 210 40% 96.1%;
    --secondary-foreground: 222.2 47.4% 11.2%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;
    --accent: 210 40% 96.1%;
    --accent-foreground: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 210 40% 98%;
    --border: 214.3 31.8% 91.4%;
    --input: 214.3 31.8% 91.4%;
    --ring: 222.2 84% 4.9%;
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    /* …same keys, dark values… */
  }
}

@layer base {
  * { @apply border-border; }
  body { @apply bg-background text-foreground; }
}
```

With the matching `tailwind.config.js`:

```js
module.exports = {
  darkMode: ["class"],
  theme: {
    extend: {
      colors: {
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        // …one entry per token…
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
    },
  },
}
```

## Which Version Am I In?

| Signal in the CSS/config | Version |
|---|---|
| `@import "tailwindcss";` | v4 |
| `@tailwind base;` | v3 |
| `@theme inline { … }` | v4 |
| `oklch(…)` token values | v4 convention |
| Bare `222.2 47.4% 11.2%` values | v3 convention |
| `@custom-variant dark` | v4 |
| `darkMode: ["class"]` in config | v3 |

Confirm against `package.json` too — `tailwindcss@^4` vs `^3`.

## Reasoning in OKLCH, Writing Hex

Even when the file ships hex, *think* in OKLCH — it is the only one of the three where lightness means what it looks like, so a ladder built once holds across every hue. Derive the value, then write it out in the file's chosen form.

`oklch(L C H)` — lightness `0`–`1`, chroma `0`–~`0.37`, hue `0`–`360`.

- **Lightness is perceptual.** Two colours at the same `L` read as equally bright regardless of hue, which is why OKLCH is worth using: pick your lightness ladder once and reuse it across hues.
- **A usable light-mode ladder:** surfaces `0.98`–`1.0`, subtle surfaces `0.96`–`0.97`, borders `0.90`–`0.93`, muted text `0.52`–`0.58`, body text `0.14`–`0.25`.
- **Dark mode is not an inversion.** Lift the background off pure black (`0.14`–`0.21`, not `0`), raise cards *above* the background rather than below, and nudge chroma **up** slightly — the same chroma reads flatter on dark surfaces.
- **Keep chroma low on surfaces.** `0`–`0.02` for backgrounds and borders; a tiny amount of the brand hue in the neutrals (`0.005`–`0.015`) is what makes a theme feel designed rather than grey.
- **Accent chroma:** `0.15`–`0.25` for a confident brand colour. Above `0.28` you leave sRGB gamut on many hues and browsers clip unpredictably.
- **Alpha:** `oklch(1 0 0 / 10%)` is valid and is what shadcn uses for dark-mode borders.

Derive a full palette from one seed by holding `C` and `H` and walking `L`; adjust `C` down at the extremes (very light and very dark colours cannot hold high chroma).

## The Surface Kit — `--surface-*`

**Seven variables that carry the visual style, and the half of a direction that colour tokens cannot express.** Not part of stock shadcn; `seed-options.py` writes them into every option, the preview harness renders them, and the implementation ports them alongside the colours. Drop them and a glassmorphism pick ships as flat cards in glass-ish colours — the user picked a *treatment*, and the treatment lives here.

| Variable | What it controls | Flat | Glass | Soft (neumorphic) | Hard (brutalist) |
|---|---|---|---|---|---|
| `--surface-border-width` | outline weight on cards, chrome, controls | `1px` | `1px` | `0px` | `3px` |
| `--surface-shadow` | resting elevation of a card | `none` | diffuse | dual light/dark | hard offset |
| `--surface-shadow-raised` | buttons, popovers, sticky bars | `none` | deeper diffuse | tighter dual | shorter offset |
| `--surface-shadow-inset` | debossed inputs / lit top edge | `none` | top highlight | pressed-in | `none` |
| `--surface-blur` | `backdrop-filter` radius | `0px` | `14px` | `0px` | `0px` |
| `--surface-gradient` | `background-image` on a surface | `none` | sheen | soft convexity | `none` |
| `--surface-wash` | `background-image` on the page | `none` | tinted, so the blur has something to sample | `none` | `none` |

Six kits: `flat`, `outlined`, `elevated`, `soft`, `glass`, `hard`. The kit name is recorded in `DECISIONS.md`; the values are in the token file.

Two of them change the *colour* tokens as well, which is why they cannot be bolted on afterwards:

- **`glass`** makes `--card` and `--popover` translucent (`#RRGGBBAA` / `oklch(… / a)`). `contrast-check.mjs` composites them over `--background` before checking text on them, which is exactly what the browser does.
- **`soft`** sets `card` = `popover` = `background` (soft UI extrudes a surface out of the page rather than layering one on top) and keeps the page off pure white so a highlight can exist above it. `hard` and `outlined` darken `--border`/`--input` toward the ink, because a 3px hairline is a smudge, not a heavy border.

Usage, in the components:

```css
.card {
  background-color: var(--card);
  background-image: var(--surface-gradient);
  border: var(--surface-border-width) solid var(--border);
  box-shadow: var(--surface-shadow);
  backdrop-filter: blur(var(--surface-blur));
}
```

A handful of things a variable cannot express — ink borders on every control for `hard`, frosted chrome for `glass`, debossed inputs for `soft` — key off the kit name instead. The harness does it with a `kit-<name>` class on the app root; do the same in the real app.

## Non-Colour Tokens Worth Defining

Not part of stock shadcn, but worth adding when the design calls for it — the mockup harness already renders them:

```css
:root {
  --font-sans: "Inter", ui-sans-serif, system-ui, sans-serif;
  --font-serif: "Source Serif 4", ui-serif, Georgia, serif;
  --font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, monospace;

  --tracking-tight: -0.02em;   /* display headings */
  --leading-body: 1.6;

  --shadow-sm: 0 1px 2px 0 oklch(0 0 0 / 0.05);
  --shadow-md: 0 4px 12px -2px oklch(0 0 0 / 0.08);

  --space-unit: 0.25rem;       /* density: 0.2 compact / 0.25 comfortable / 0.3 spacious */
}
```
