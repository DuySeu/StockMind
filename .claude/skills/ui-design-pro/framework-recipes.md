# Framework Recipes

Where the theme lives and what to write, per UI framework. Design the option in shadcn token vocabulary (`shadcn-tokens.md`), then map it here.

> **Version gate.** Every API below changed across a recent major. Read the installed version from `package.json` and confirm the syntax against that version's docs before writing. If the installed major does not match what is documented here, trust `package.json` and the library's own docs — not this file, and not memory.
>
> On a **new project** there is no `package.json` to read until step 12 has run. Scaffold first (`references/scaffolding.md`), then read the majors the scaffolder actually installed — they are frequently newer than what you expected to get.

> **No comments in the CSS token file.** `index.css`, `app.css`, `globals.css`, `styles.css` — values only, zero `/* … */`, including the header on the ported option CSS. Explanatory comments in the examples below are documentation of this file, not something to copy into the project. The rule covers CSS token files only; the JS theme objects on this page (`createTheme`, `definePreset`, `ConfigProvider`) follow the project's normal conventions.

## Scaffolding a new app

Greenfield only, and only after the user picked a direction **and approved `docs/design/UI-PLAN.md`**. Everything about it — the per-framework commands, the Vite template table, the swap dance that preserves `docs/design/`, the Tailwind v4 and v3 installs, and the shadcn `@/*` alias requirement — is in **`references/scaffolding.md`**. This file picks up once the app exists.

## Where the theme lives

| UI framework | Theme home | Mechanism |
|---|---|---|
| shadcn/ui (React) | `src/index.css` | CSS custom properties + Tailwind |
| shadcn-vue | `src/assets/index.css` | same variables as shadcn/ui |
| shadcn-svelte | `src/app.css` | same variables as shadcn/ui |
| Ant Design | `ConfigProvider` at app root | JS token object + algorithm |
| MUI | `createTheme()` + `ThemeProvider` | JS theme object |
| Mantine | `createTheme()` + `MantineProvider` | JS theme object |
| Chakra UI | `createSystem()` + `ChakraProvider` | JS config with tokens/semanticTokens |
| Vuetify | `createVuetify({ theme })` | JS theme object → CSS vars |
| PrimeVue / PrimeNG | `definePreset()` on a base preset | JS preset object → CSS vars |
| Naive UI | `n-config-provider` `theme-overrides` | JS override object |
| Element Plus | SCSS vars at build, `--el-*` at runtime | CSS custom properties |
| Angular Material | `styles.scss` | SCSS theme mixin |
| DaisyUI | CSS (`@plugin`) or `tailwind.config` | named theme with CSS vars |
| Tailwind only | `src/index.css` | CSS custom properties |

---

## shadcn/ui, shadcn-vue, shadcn-svelte

All three consume the identical variable names. Write the token file from `shadcn-tokens.md` (v4 or v3 form to match the installed Tailwind), then:

- **Dark mode:** toggle the `dark` class on `<html>`. Persist the preference and honour `prefers-color-scheme` on first load.
- **Components:** already generated into `src/components/ui/` (React/Svelte) or `src/components/ui/` (Vue). Restyle by editing the token file, not the components. Only edit a component when the design changes its *structure* (e.g. a button size scale).
- **Do not re-run `npx shadcn init`** on a project that already has `components.json` — it will overwrite the token file.

Verify: `node scripts/contrast-check.mjs src/index.css`

---

## Ant Design (React)

```tsx
import { ConfigProvider, theme } from "antd";

const tokens = {
  colorPrimary: "#4f46e5",
  colorInfo: "#4f46e5",
  colorSuccess: "#16a34a",
  colorWarning: "#d97706",
  colorError: "#dc2626",
  colorBgBase: "#ffffff",
  colorTextBase: "#0f172a",
  borderRadius: 10,
  fontFamily: "'Inter', ui-sans-serif, system-ui, sans-serif",
  fontSize: 14,
  controlHeight: 36,          // density
};

export function AppTheme({ dark, children }) {
  return (
    <ConfigProvider
      theme={{
        algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: tokens,
        components: {
          Button: { primaryShadow: "none", fontWeight: 500 },
          Card: { paddingLG: 20 },
          Table: { headerBg: "transparent" },
        },
      }}
    >
      {children}
    </ConfigProvider>
  );
}
```

- `algorithm` derives the full palette from the seed tokens — set `colorPrimary` / `colorBgBase` / `colorTextBase` and let it generate the rest rather than hand-listing every derived colour.
- `theme.compactAlgorithm` can be combined in an array with the light/dark algorithm for a compact density.
- Reading tokens inside components: `const { token } = theme.useToken()`.
- Token mapping: `background`→`colorBgBase`, `foreground`→`colorTextBase`, `primary`→`colorPrimary`, `destructive`→`colorError`, `border`→`colorBorder`, `muted-foreground`→`colorTextSecondary`, `radius`→`borderRadius`.
- For Angular use **NG-ZORRO**, for Vue use **Ant Design Vue** — same token names, different provider setup.

---

## MUI (React)

```ts
import { createTheme } from "@mui/material/styles";

export const buildTheme = (mode: "light" | "dark") =>
  createTheme({
    palette: {
      mode,
      primary: { main: "#4f46e5", contrastText: "#ffffff" },
      error: { main: "#dc2626" },
      background: {
        default: mode === "light" ? "#ffffff" : "#0f1117",
        paper: mode === "light" ? "#ffffff" : "#161922",
      },
      text: {
        primary: mode === "light" ? "#0f172a" : "#f8fafc",
        secondary: mode === "light" ? "#64748b" : "#94a3b8",
      },
      divider: mode === "light" ? "#e2e8f0" : "rgba(255,255,255,0.1)",
    },
    shape: { borderRadius: 10 },
    typography: {
      fontFamily: "'Inter', ui-sans-serif, system-ui, sans-serif",
      h1: { fontFamily: "'Source Serif 4', serif", letterSpacing: "-0.02em" },
      button: { textTransform: "none", fontWeight: 500 },
    },
    components: {
      MuiButton: { defaultProps: { disableElevation: true } },
      MuiPaper: { defaultProps: { elevation: 0 }, styleOverrides: { root: { border: "1px solid var(--mui-palette-divider)" } } },
    },
  });
```

Wrap in `<ThemeProvider theme={...}><CssBaseline />…</ThemeProvider>`. For CSS-variable mode (v6+), use `createTheme({ cssVariables: true, colorSchemes: { light, dark } })` and skip the two-theme swap.

Mapping: `background`→`palette.background.default`, `card`→`palette.background.paper`, `foreground`→`palette.text.primary`, `muted-foreground`→`palette.text.secondary`, `border`→`palette.divider`, `radius`→`shape.borderRadius`.

---

## Mantine (React)

```ts
import { createTheme, MantineProvider } from "@mantine/core";
import "@mantine/core/styles.css";

const theme = createTheme({
  primaryColor: "brand",
  colors: {
    // exactly 10 shades, lightest → darkest
    brand: ["#eef2ff","#e0e7ff","#c7d2fe","#a5b4fc","#818cf8","#6366f1","#4f46e5","#4338ca","#3730a3","#312e81"],
  },
  primaryShade: { light: 6, dark: 5 },
  fontFamily: "'Inter', ui-sans-serif, system-ui, sans-serif",
  headings: { fontFamily: "'Source Serif 4', serif", fontWeight: "600" },
  defaultRadius: "md",
  radius: { md: "10px" },
});
```

`<MantineProvider theme={theme} defaultColorScheme="auto">`. Mantine requires exactly 10 shades per custom colour — generate the ladder by holding hue/chroma and walking lightness.

---

## Chakra UI v3 (React)

```ts
import { createSystem, defaultConfig, defineConfig } from "@chakra-ui/react";

const config = defineConfig({
  theme: {
    tokens: {
      colors: { brand: { 500: { value: "#4f46e5" }, 600: { value: "#4338ca" } } },
      fonts: { body: { value: "'Inter', sans-serif" }, heading: { value: "'Source Serif 4', serif" } },
      radii: { l2: { value: "10px" } },
    },
    semanticTokens: {
      colors: {
        bg: { DEFAULT: { value: { _light: "white", _dark: "#0f1117" } } },
        fg: { DEFAULT: { value: { _light: "#0f172a", _dark: "#f8fafc" } } },
      },
    },
  },
});

export const system = createSystem(defaultConfig, config);
```

v2 used `extendTheme` — a different API entirely. Check the installed major before writing.

---

## Vuetify (Vue 3)

```ts
import { createVuetify } from "vuetify";

export default createVuetify({
  theme: {
    defaultTheme: "light",
    themes: {
      light: {
        dark: false,
        colors: {
          background: "#ffffff",
          surface: "#ffffff",
          primary: "#4f46e5",
          secondary: "#64748b",
          error: "#dc2626",
          success: "#16a34a",
          warning: "#d97706",
          "on-background": "#0f172a",
          "on-surface": "#0f172a",
          "on-primary": "#ffffff",
        },
      },
      dark: { dark: true, colors: { background: "#0f1117", surface: "#161922", primary: "#818cf8", "on-surface": "#f8fafc" } },
    },
  },
  defaults: {
    VBtn: { rounded: "lg", variant: "flat", style: "text-transform: none;" },
    VCard: { rounded: "lg", flat: true, border: true },
    VTextField: { variant: "outlined", density: "comfortable" },
  },
});
```

Vuetify's `defaults` block is where density and radius live globally — use it instead of repeating props on every component. Colours become CSS vars (`--v-theme-primary`) usable in custom CSS.

---

## PrimeVue v4 / PrimeNG (preset system)

```ts
import Aura from "@primevue/themes/aura";
import { definePreset } from "@primevue/themes";

const AppPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: "#eef2ff", 100: "#e0e7ff", 200: "#c7d2fe", 300: "#a5b4fc", 400: "#818cf8",
      500: "#6366f1", 600: "#4f46e5", 700: "#4338ca", 800: "#3730a3", 900: "#312e81", 950: "#1e1b4b",
    },
    colorScheme: {
      light: { surface: { 0: "#ffffff", 50: "#f8fafc", 100: "#f1f5f9" } },
      dark:  { surface: { 0: "#0f1117", 50: "#161922", 100: "#1e2230" } },
    },
  },
});

app.use(PrimeVue, { theme: { preset: AppPreset, options: { darkModeSelector: ".dark" } } });
```

PrimeVue v3 used SASS themes / `.css` theme files — a completely different mechanism. Confirm the major version.

---

## Naive UI (Vue 3)

```ts
const themeOverrides = {
  common: {
    primaryColor: "#4f46e5",
    primaryColorHover: "#6366f1",
    primaryColorPressed: "#4338ca",
    errorColor: "#dc2626",
    borderRadius: "10px",
    fontFamily: "'Inter', ui-sans-serif, system-ui, sans-serif",
    bodyColor: "#ffffff",
    textColorBase: "#0f172a",
  },
  Button: { fontWeight: "500" },
};
```

```vue
<n-config-provider :theme="isDark ? darkTheme : null" :theme-overrides="themeOverrides">
```

---

## Element Plus (Vue 3)

Runtime (simplest, works with a plain CSS file):

```css
:root {
  --el-color-primary: #4f46e5;
  --el-color-danger: #dc2626;
  --el-border-radius-base: 10px;
  --el-font-family: "Inter", ui-sans-serif, system-ui, sans-serif;
}
html.dark { --el-bg-color: #0f1117; --el-text-color-primary: #f8fafc; }
```

Dark mode also needs `import "element-plus/theme-chalk/dark/css-vars.css"` and the `dark` class on `<html>`. Element Plus derives `--el-color-primary-light-3` … `-light-9` from the base — override them too if the generated tints clash with the design.

---

## Angular Material

SCSS in `styles.scss`. **The theming API changed three times recently** — verify against the installed `@angular/material` major:

- v17 and earlier: `mat.define-light-theme((color: (primary: …, accent: …)))`
- v18/v19: `mat.define-theme((color: (theme-type: light, primary: $palette), typography: (…), density: (scale: 0)))`
- v20+: `mat.theme((color: …, typography: …, density: …))`

```scss
@use "@angular/material" as mat;

$theme: mat.define-theme((
  color: (theme-type: light, primary: mat.$violet-palette),
  typography: (brand-family: "Source Serif 4", plain-family: "Inter"),
  density: (scale: 0),
));

html { @include mat.all-component-themes($theme); }
```

---

## DaisyUI

v5 (Tailwind v4), in CSS:

```css
@import "tailwindcss";
@plugin "daisyui";
@plugin "daisyui/theme" {
  name: "brand";
  default: true;
  --color-base-100: oklch(1 0 0);
  --color-base-content: oklch(0.145 0 0);
  --color-primary: oklch(0.52 0.19 275);
  --color-primary-content: oklch(0.99 0 0);
  --radius-box: 0.75rem;
  --radius-field: 0.5rem;
}
```

v4 configured themes in `tailwind.config.js` under a `daisyui.themes` array instead.

---

## Tailwind only / plain CSS / Astro

Use the shadcn token file verbatim — it is just CSS custom properties. Without a component library you also own the component classes; define them in `@layer components` so utilities can still override:

```css
@layer components {
  .btn-primary {
    background: var(--primary);
    color: var(--primary-foreground);
    border-radius: var(--radius);
    padding: 0.5rem 0.875rem;
    font-weight: 500;
  }
}
```

---

## Porting the surface kit

The colour tokens map onto whatever the framework uses; the seven `--surface-*` variables (`shadcn-tokens.md`) are **plain CSS custom properties in every one of these frameworks** and are ported verbatim into the token file. They are what makes the picked direction look like the option the user chose rather than like flat cards in the right colours.

| Target | Where the kit goes | How components consume it |
|---|---|---|
| Tailwind v4 (shadcn, Tailwind-only) | the same `:root` / `.dark` blocks as the colours | `shadow-(--surface-shadow)`, `border-(length:--surface-border-width)`, or a `.card` rule in `@layer components` |
| Tailwind v3 | `:root` / `.dark` in the CSS file — **not** `tailwind.config`, these are not colours | `boxShadow: { surface: "var(--surface-shadow)" }` in `theme.extend` if you want a utility |
| CSS Modules / plain CSS | `:root` / `.dark` | `var(--surface-shadow)` directly |
| JS theme objects (MUI, Mantine, Chakra, Vuetify, PrimeVue, Ant) | still the CSS token file; reference the vars from the theme object | `shadows: ["none", "var(--surface-shadow)", …]`, `components.MuiCard.styleOverrides` |

Three notes that decide whether it survives the port:

- **`--surface-blur` needs `backdrop-filter` on the element AND a translucent background** — a glass kit whose `--card` was flattened to an opaque hex renders as a plain card and the blur does nothing. Keep the alpha.
- **`--surface-wash` belongs on the page/app shell**, not on a card. It is what the blur samples.
- **A few things are per-kit, not per-variable** — ink borders on every control (`hard`), frosted sidebar and topbar (`glass`), debossed inputs (`soft`). Put the kit name on the app root (`class="kit-glass"`) and write those handful of rules against it, exactly as the preview harness does.

Component libraries with their own elevation scale (MUI's `shadows[25]`, Vuetify's `elevation`) will fight this. Override the first two or three levels with the kit's values rather than leaving both systems live.

## Fonts

| Setup | How |
|---|---|
| Next.js | `next/font/google` or `next/font/local` → assign `.variable` to `<html className>`, then `--font-sans: var(--font-inter)` |
| Vite (any framework) | `@fontsource/inter` + `@fontsource-variable/…`, imported in the entry file |
| CDN | `<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>` + the `css2` stylesheet link |
| System only | No load. `ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif` |

Rules that apply regardless of setup:

- Load **only the weights actually used**. Four weights of two families is already 8 files.
- Prefer variable fonts (`@fontsource-variable/*`) — one file covers the whole weight range.
- Always end the stack with a system fallback; never let a font failure fall through to Times New Roman.
- Map the family to a token (`--font-sans`) and reference the token everywhere. A component that names `"Inter"` directly cannot be re-themed.
- Self-host when the user mentioned privacy, GDPR, offline, or air-gapped anything.
