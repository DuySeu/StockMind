# Scaffolding the App — full recipe

Read this at **step 11** — the first action of the implementation phase, so only after the
user approved `docs/design/UI-PLAN.md` — and only on a greenfield project. On a project
that already has a `package.json` there is nothing here for you: skip to step 12.

Skim it once earlier, at step 10, for one reason only: the plan's Stack section names the
exact scaffold command and template, and the user approves that command by approving the
plan. Naming it is not running it.

The three hard rules are in SKILL.md and repeated here because they are what actually
goes wrong: always the non-interactive flag, **never** `--overwrite`, never point the
scaffolder at the folder that already holds `docs/design/`.

## The Vite swap dance

Vite is the default scaffolder. It creates its own target directory and **refuses to
write into a non-empty one**, so the design docs move in after the scaffold rather
than the scaffold working around them:

```bash
# from the directory that contains <project-slug>/
npm create vite@latest <project-slug>-app -- --template react-ts --no-interactive
mv <project-slug>/docs <project-slug>-app/docs
rmdir <project-slug>
mv <project-slug>-app <project-slug>
cd <project-slug> && npm install
```

The app lands at the project root — `src/`, `index.html`, `package.json` — with
`docs/design/` preserved beside it (the preview, the option token files, and the approved
`UI-PLAN.md`) and the dotfiles (`.gitignore`) intact.

- **`--no-interactive` is required.** The CLI starts an interactive prompt when it
  detects a TTY, and a prompt in an agent session hangs the turn.
- **Never pass `--overwrite`.** It deletes the existing contents of the target
  directory — which is exactly `docs/design/`, the one thing that has to survive.
  There is no situation in this workflow where it is the right flag.
- **Never point the scaffolder at the folder that already holds the design docs.** It
  prints `Operation cancelled`, creates nothing, and exits 0 — a silent no-op that
  looks like success.
Templates (`create-vite` 9.x) — use the `-ts` variant unless the user asked for plain JS:

| Vite template | For |
|---|---|
| `react-ts` / `react-compiler-ts` | React |
| `vue-ts` | Vue 3 |
| `svelte-ts` | Svelte 5 |
| `solid-ts` | SolidJS |
| `preact-ts`, `lit-ts`, `qwik-ts` | Preact / Lit / Qwik |
| `vanilla-ts` | Plain HTML/CSS/TS |

## Frameworks Vite does not scaffold

Use their own CLI — the swap dance above is identical, only the first command changes:

| Confirmed FE framework | Scaffold with |
|---|---|
| React, Vue 3, Svelte 5, Solid, Preact, Lit, Qwik, plain HTML/CSS | `npm create vite@latest <dir> -- --template <t> --no-interactive` |
| Next.js | `npx create-next-app@latest <dir> --ts --tailwind --eslint --app --no-src-dir --import-alias "@/*"` |
| Nuxt 3 | `npx nuxi@latest init <dir> --packageManager npm --no-gitInit` |
| Astro | `npm create astro@latest <dir> -- --template minimal --typescript strict --no-install --no-git` |
| Angular | `npx @angular/cli@latest new <name> --style=scss --ssr=false --defaults` |
| Laravel / Blade | `composer create-project laravel/laravel <dir>` |

Every one of these needs its non-interactive flags. If a scaffolder you reach for has
none, say so and ask the user to run it themselves rather than firing a command that
will hang.

## Installing the confirmed stack

Styling engine first, UI framework second. Tailwind v4 on Vite:

```bash
npm install tailwindcss @tailwindcss/vite
```

```ts
// vite.config.ts
import tailwindcss from "@tailwindcss/vite";
export default defineConfig({ plugins: [react(), tailwindcss()] });
```

Then `src/index.css`, replacing the template's contents entirely — one line, and no
comment above it: the token file carries no `/* … */` at all.

```css
@import "tailwindcss";
```

Tailwind **v3** is a different install:
`npm install -D tailwindcss@3 postcss autoprefixer && npx tailwindcss init -p`, plus a
`content` glob in `tailwind.config.js` and
`@tailwind base; @tailwind components; @tailwind utilities;` in the CSS — and the
bare-HSL-triplet token form, not hex. Install the major the user confirmed, not the one
you remember.

Then the UI framework, if they chose one — `npx shadcn@latest init` is allowed *here*,
at step 11, because the framework was confirmed at step 2, the direction was picked at
step 9, and the plan naming this install was approved at step 10. It is a Red Flag at
any earlier point. On a fresh Vite app it also needs the
`@/*` path alias in **both** `tsconfig.json` and `tsconfig.app.json`, and a matching
`resolve.alias` in `vite.config.ts`, or it fails on the alias check.

Confirm the app runs before touching the theme:

```bash
npm run build
```

Once it builds, `src/index.css` is the token file and step 12 continues normally.
