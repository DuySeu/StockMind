---
name: ui-design-pro
description: "Use when starting a new frontend project from scratch, or designing, building, reviewing, or improving UI — pages, components, colour theme, brand colour, index.css or theme config, design tokens, typography, font pairing, layout, spacing, radius, density, accessibility, animation, dark mode, data visualization — for a website, landing page, dashboard, admin panel, e-commerce, SaaS, portfolio, or blog, in any style (glassmorphism, brutalism, neumorphism, minimalism, bento grid)."
---

# UI Design Pro: Direction Into Themed Components

Turn a vague "make it look good" into a confirmed design direction and a working theme: token file, fonts, and the key components — in the user's actual stack.

**Two phases, and they never happen in the same turn.**

| Phase | You do | Ends with | Artifacts |
|---|---|---|---|
| **1 — Planning** | Probe, confirm the stack, ask two concept questions, seed 3-5 real options, gate on contrast, get a pick, scope, write the plan | The user approving the plan | `docs/design/ui-options.html`, `option-tokens/*.css`, `UI-PLAN.md` |
| **2 — Implementation** | Scaffold if new, then build exactly what the plan says | A verified, reviewed theme | the app + `docs/design/DECISIONS.md` |

Phase 1 writes no code. Phase 2 makes no new decisions.

<HARD-GATE>
**Gate 1 — the pick.** Do NOT write a plan file until BOTH are true:
1. The user confirmed the FE framework AND the UI framework (never inferred, never defaulted).
2. The user picked one of the presented options (or a named hybrid).

Presenting options is not approval. Enthusiasm is not approval. "Looks nice" is not approval — ask which option, by letter.

**Gate 2 — the plan.** Do NOT write `index.css`, theme config, tokens, any component, **or scaffold an app** until the user has read `docs/design/UI-PLAN.md` and said to implement it.

A pick approves a *direction*. It does not approve a scope, a file list, or a stack install — those are the plan, and the plan gets its own yes. After writing the plan you **end your turn** and wait.

On a new project Gate 2 holds back `npm create vite@latest`: until the plan is approved the only files anywhere are `<project>/docs/design/`.
</HARD-GATE>

## Anti-Pattern: "I Can Tell What They Want"

You cannot. Taste is the deliverable, and it is not inferable from a repo — nor extractable by quizzing the user about fonts and radii. Each of these is a violation:

- Installing shadcn/ui because the project uses React and Tailwind.
- Shipping default zinc/neutral because the user said "clean and modern".
- Picking the font you always pick.
- Writing `index.css` first and asking "does this work?" after.
- Treating `--design-system`'s single best match as the answer. It is one seed.
- **Interviewing your way out of it** — asking for a font category, a radius, a density, or three mood adjectives. That is delegating the design.
- Asking about the brand colour twice.
- **Treating the pick as the go-ahead** — it approves the direction; the plan approves the work.

A design that arrives without a choice is one the user has to argue with instead of pick.

## Checklist

Create a task per item and complete them in order. Steps 1-11 are Phase 1; 12-19 are Phase 2 and do not start until 11 came back approved.

### Phase 1 — Planning

1. **Read the frontend context** — `package.json`, existing CSS/theme files, components, `tailwind.config.*`, fonts. See Context Probe. **No project yet → Greenfield instead**: create `./<project-slug>/docs/design/` and nothing else.
2. **Ask the stack questions** — FE framework, UI framework, styling engine + version. REQUIRED. Never assume.
3. **Check compatibility** — validate the FE × UI pair against the matrix; resolve conflicts first.
4. **Ask the two concept questions** — what the product is, and whether they have a main colour. Nothing else.
5. **Query the database** — `--domain product`, then `color` / `style` / `typography`.
6. **Seed 3-5 options** — `scripts/seed-options.py`, `--brand <hex>` if they gave a colour.
7. **Contrast-gate every seed** — `contrast-check.mjs` per option CSS. Fix or drop before the user sees anything.
8. **Sharpen the seeds** — rewrite generated names and theses; check fonts, radius, and surface kits against the brief.
9. **GATE 1 — user picks an option.** Iterate on the preview for a hybrid or a tweak.
10. **Scope the work** — which colour mode is default, and which 3-6 components/pages come first. The only questions after the pick, and they are about scope, not taste.
11. **Write `docs/design/UI-PLAN.md`, then GATE 2** — tell the user where it is, ask them to review it, ask whether to implement. **End the turn.**

### Phase 2 — Implementation

Entry condition: the plan was approved. Build what it says; if something in it turns out wrong, say so and amend the plan rather than improvising past it.

12. **Scaffold the app** — greenfield only, with the command the plan names. See Scaffolding. Skip on an existing project.
13. **Implement the token file** — `index.css` or the framework's theme config, light + dark. Port the winning option's CSS — colours **and** its `--surface-*` kit. No comments.
14. **Wire the fonts** — as the plan says, mapped to the token variables. `CSS Import` and `Tailwind Config` come from `typography.csv`.
15. **Implement the scoped components and pages** — the plan's list, in its order, tokens only.
16. **Motion pass** — `--domain gsap` at the resolved tier if the motion dial is ≥3. Skip at 1-2.
17. **Verify** — `contrast-check.mjs` on the real token file, `npm run build`, both modes render.
18. **Review loop** — dispatch the reviewer subagent (`reviewer-prompt.md`); fix and re-dispatch until approved (max 5, then surface to the user).
19. **Document** — `docs/design/DECISIONS.md` by hand, including anything that differed from the plan.

Four steps loop rather than advance: the contrast gate reseeds (7 → 6), Gate 1 iterates the preview (9 → 8) until a letter is named, Gate 2 revises the plan (11 → 10) until the scope is approved, and the review loop re-dispatches (18) until approved or five iterations are spent.

## Review Priority

When reviewing existing UI rather than building new, work in this order — ranked by what each costs a user when wrong, not by visibility:

1. **Accessibility** (CRITICAL, `ux`) · 2. **Touch & interaction** (CRITICAL, `ux`) · 3. **Performance** (HIGH, `ux`) · 4. **Style selection** (HIGH, `style`+`product`) · 5. **Layout & responsive** (HIGH, `ux`) · 6. **Typography & colour** (MEDIUM, `typography`+`color`) · 7. **Animation** (MEDIUM, `ux`+`gsap`) · 8. **Forms & feedback** (MEDIUM, `ux`) · 9. **Navigation** (HIGH, `ux`) · 10. **Charts & data** (LOW, `chart`)

All 98 rules, with must-haves and anti-patterns per category, are in `references/quick-reference.md` — read the section you need, not the file.

## Context Probe

Find out what is already true before asking anything.

| Look at | To learn |
|---|---|
| `package.json` | FE framework + major, UI library, Tailwind v3 vs v4, font packages |
| `index.css` / `app.css` / `globals.css` | Existing tokens, `@import "tailwindcss"` (v4) vs `@tailwind base` (v3) |
| `tailwind.config.*` | v3 setup, colour mapping, custom fonts |
| `components.json` | shadcn already installed — note `style`, `baseColor`, `cssVariables` |
| `src/components/ui/` | What exists — implement into it, don't duplicate |
| `docs/design/DECISIONS.md` (or legacy `design-system/*/MASTER.md`) | A previous run picked a direction. Read it first; ask refresh vs replacement |
| `docs/design/UI-PLAN.md` | Phase 1 already ran. Restate its direction and scope in one line, ask whether it still stands, then resume at step 12 instead of reseeding |
| Existing screens | Density and layout conventions worth preserving |

Report what you found in one short paragraph, then ask only what is genuinely undecided. If a theme exists, ask **refresh** (keep structure, change values) or **replacement**.

No `package.json` at all — or the user asked for a new app? Say so in one line and go to Greenfield.

## Greenfield: No Project Yet

The checklist still applies; only step 1 changes and step 12 switches on.

```bash
mkdir -p ./<project-slug>/docs/design
```

`<project-slug>` is kebab-case from what they are building — `clinic-portal`, `nail-salon-booking`. Ask for the name if the concept doesn't hand you one.

**For all of Phase 1 the project folder holds exactly this:**

```
<project-slug>/docs/design/
├── ui-options.html
├── option-tokens/{A,B,…}-<name>.css
└── UI-PLAN.md          # step 11, after the pick
```

No `package.json`, no `src/`, no framework, no `index.css` — the app is scaffolded at step 12, after the plan is approved. Nothing loose in the current directory and nothing in `/tmp`: the preview is a deliverable the user opens and keeps as the record of the rejected directions.

"Current working directory" is where this session runs. If that is already a worktree, its root **is** the current path — create the project there, at top level, **never under `.claude/`** (the user cannot find it, cannot commit it, and loses it when the worktree is cleaned).

**The stack questions are load-bearing here**, not confirmatory: there is no `package.json` to read, so nothing may be inferred. Ask, then put the resolved scaffold template and versions in the plan — that is where the user confirms them, before anything is installed.

## Stack Questions (REQUIRED)

Interdependent, so this group may go in one message; the concept questions after it go one at a time.

1. **FE framework** — React, Vue 3, Nuxt 3, Svelte 5, Angular, SolidJS, Astro, Laravel/Blade, or plain HTML/CSS?
2. **UI framework** — offer only the valid options for their answer, from the matrix. Include "Tailwind only" and "headless (Radix/Ark/Kobalte) + custom" as real choices.
3. **Styling engine** — Tailwind v4, Tailwind v3, CSS-in-JS, CSS Modules, or plain CSS? Confirm the major from `package.json` and state it back.

No preference? Recommend one pairing with a one-line reason and get explicit confirmation. **There is no default stack** — do not proceed on silence; a silent default misroutes every recommendation downstream.

## Compatibility Matrix

Getting this wrong wastes an entire implementation pass. shadcn/ui and Ant Design are React libraries; other frameworks get ports with different names and APIs.

| FE framework | Valid UI frameworks | Watch out for |
|---|---|---|
| React | shadcn/ui, Ant Design, MUI, Mantine, Chakra, HeroUI, Radix + Tailwind | shadcn/ui is copy-in source, not a dependency |
| Vue 3 | shadcn-vue, Ant Design Vue, Vuetify, PrimeVue, Naive UI, Element Plus | "shadcn" for Vue = shadcn-vue (Reka UI) |
| Nuxt 3 | Nuxt UI, shadcn-vue, PrimeVue, Vuetify (via modules) | Nuxt UI v2 → v3 moved the theming API |
| Svelte 5 | shadcn-svelte, Skeleton, Flowbite Svelte, Melt UI + Tailwind | Svelte 4 vs 5 changes component APIs |
| Angular | Angular Material, PrimeNG, NG-ZORRO, Spartan | NG-ZORRO is the Angular Ant Design, not `antd` |
| SolidJS | solid-ui, Kobalte + Tailwind, Ark UI | Small ecosystem — confirm it is maintained |
| Laravel / Blade | Tailwind + Flux, Livewire + DaisyUI, Filament | Filament ships its own theme layer |
| Astro / plain HTML | Tailwind + DaisyUI, Preline, Flowbite, plain CSS vars | No component runtime — theme is pure CSS |

Named an incompatible pair ("Vue with shadcn/ui")? Say so, name the correct port, let them confirm.

**Three.js is not a UI framework.** A WebGL canvas has no tokens; the surrounding DOM does. Theme the overlay normally; `--stack threejs` is for WebGL-specific guidance.

## Concept Questions — and Nothing Else

1. **What is this product, and who uses it?** One or two sentences in their words. "A patient portal where clinic staff and patients both log in" is better than "a healthcare app". This answer *is* the database query, and it decides density, motion, accent loudness, and which styles are candidates.
2. **Do you have a main colour?** If yes, take the hex (or read it off the logo) and pass `--brand`. Every option then pins *their* colour and the choice becomes which treatment of it they want. If no, say "I'll propose one per direction — you'll pick a colour by picking a direction" and omit the flag.

### Do not ask about

**Fonts. Corner radius. Density. Spacing. Mood adjectives. Style names. Light vs dark.**

Asking "sharp or rounded corners?" hands the work back to the person who came here to avoid it. Most users have no defensible answer, so they guess — and then you have built to a guess and called it taste. These are consequences of what the product is, and the option set makes them visible: five screens answer "do you want rounded corners" better than the question does.

Volunteered constraints ("our brand font is Söhne", "must be dark") are different — take them and thread them through.

Two things do need asking, but **after** the pick: default colour mode and which components come first. Step 10 — scope, not taste.

### Dials come from the concept

`--density`, `--variance`, `--motion` are properties of the product, not the user's mood. `seed-options.py` infers all three and prints what it inferred — read those lines; override only when you can say why.

| The concept | Inferred |
|---|---|
| Dashboard, admin, analytics, console, CRM, ERP, trading, monitoring | `--density 9`, `--motion 2` |
| Landing, marketing, portfolio, agency, editorial, spa, hotel, luxury | `--density 3`, `--motion 8` |
| Bank, fintech, insurance, healthcare, government, legal, enterprise | `--variance 2` |
| Creative, agency, fashion, gaming, entertainment, art, experimental | `--variance 8` |
| Anything the words don't settle | `5` — "no strong signal either way" |

`--density` rewrites the `--space-*` scale, `--variance` biases which styles fill open slots, `--motion` picks the GSAP tier.

## Querying the Database

Invoke by full path — never assume the working directory. Python 3, stdlib only (`python3` if `python` is missing).

```bash
python "<skill-dir>/scripts/search.py" "<query>" --domain <domain> [-n 3]
```

**Query with multi-dimensional keywords.** `"healthcare SaaS dashboard data-dense"` beats `"app"` — product + industry + tone + density. `-n 1` when one match is enough: the `style` domain ships untruncated checklists, so three results cost ~4× one.

| Need | Domain |
|---|---|
| Product-type pattern, landing structure, dashboard style | `product` |
| Style guides — colours, effects, framework fit, complexity | `style` |
| Palettes by product type (shadcn token columns) | `color` |
| Font pairings + Google Fonts URL, CSS import, Tailwind config | `typography` |
| Individual families, variable axes, popularity | `google-fonts` |
| Page structure and CTA strategy | `landing` |
| Chart type, library, accessibility grade | `chart` |
| UX rules, do/don't, severity | `ux` |
| Icon names with import code | `icons` |
| GSAP presets by intensity tier | `gsap` |
| React/Next render and bundle issues that cause visible jank | `react` |
| Per-stack guidelines | `--stack <name>` |

Stacks: `react`, `nextjs`, `vue`, `nuxtjs`, `nuxt-ui`, `svelte`, `astro`, `angular`, `laravel`, `html-tailwind`, `shadcn`, `threejs`.

Domain is auto-detected when omitted, but overlapping terms misroute ("font" hits both `typography` and `google-fonts`) — pass it when results look off.

**0 results:** retry once with broader keywords (product and style separately). Still empty? Say so out loud — "no palette match for X, falling back to general SaaS defaults" — and use the Review Priority order. Never present an empty search as data.

## Seeding the Options

**3-5 named directions**, each a complete light + dark token set with identical variable names, so the winner ports by copy.

```bash
python "<skill-dir>/scripts/seed-options.py" "<what the product is, in the user's words>" \
  --count 5 --brand "#4F46E5" --project "<Project Name>" \
  --out <project-slug>/docs/design/ui-options.html \
  --token-dir <project-slug>/docs/design/option-tokens
```

Paths are relative to the current directory; the script creates missing parents. On an existing project drop the `<project-slug>/` prefix and write to `docs/design/` at the repo root. Drop `--brand` when there is no brand colour. Dials are inferred — pass `--density`/`--variance`/`--motion` only to override.

Each option is a distinct direction, not a hue variant: no repeated style, no repeated font pairing, ≥40° OKLCH hue separation, a different **surface kit** per slot where the query allows, then a **derived** dark theme (`colors.csv` has no dark values) with every WCAG text pair nudged clear of 4.5:1 in both modes. It reports every fallback and adjustment — read those lines. With `--brand` the hue axis is spent, so distinctness moves to surface, type, shape, and accent strategy.

Three surface kits also adjust the palette and say so: `glass` (translucent `card`/`popover`, contrast enforced against the composited surface), `soft` (card = background, page off pure white), `hard`/`outlined` (border darkened toward the ink). Mechanics: `references/seeding-internals.md` — read it when a seed looks wrong.

Then gate every seed before the user sees it:

```bash
for f in <project-slug>/docs/design/option-tokens/*.css; do node "<skill-dir>/scripts/contrast-check.mjs" "$f" || echo "FAILED: $f"; done
```

Any FAIL means reseed or hand-fix. Zero failures is the entry condition for showing the preview.

### Sharpening the seeds

The script produces mechanical names and theses. Rewrite before showing anything:

- **Name** — the style category ("Neumorphism") names a technique, not a direction. Give it character: "Editorial Warm", "Signal", "Slate Precision".
- **Thesis** — one line on what it commits to and who it is for. Not a comma-joined keyword dump.
- **Fonts** — the ranked pairing is sometimes a poor fit (a CJK family for an English spa site). Re-pick with `--domain typography`.
- **Radius** — usually inferred, and the script says so per option. A data-dense console with 1rem corners is wrong even if the ladder produced it.
- **Surface kit** — the script says which kit and whether it was read or stepped off a ladder (32 of 84 style rows declare no technique). Check it suits the brief (`soft` on a trading terminal is wrong) *and* that the set varies — five options on `flat` is the hue-only failure one level down.

Edit `ui-options.html` directly — the `OPTIONS` array is formatted for hand-editing. Change names, theses, fonts, and kits; if you change a colour or a `--surface-*` value, change the matching `option-tokens/*.css` too, so the file the gate reads stays the file the preview shows.

## Rendering the Preview

`seed-options.py` writes `ui-options.html` filled in — one self-contained file, no server, no build, no app needed to view it.

**One option on screen at a time, switched by a tab bar.** Each tab carries its letter, name, surface kit, radius, and a five-swatch strip; the active option's header adds role, surface description, density, accent strategy, and primary hex. Arrow keys and `1`–`5` switch tabs, `d` toggles dark, and the active letter is in the URL hash so an option can be linked or reopened. A stacked page answers "how do these differ" — the question that decides a pick is "does *this* screen work", and that needs the screen undivided.

Every option renders against the *same* miniature app (sidebar, topbar, stat cards, chart, table, form row, buttons, badges, an open popover) so the comparison is fair, with light/dark and focus-state toggles because both modes and keyboard states ship.

**The options differ in styling, not just palette.** Each carries a surface kit — `flat`, `outlined`, `elevated`, `soft`, `glass`, `hard` — deciding border weight, shadow geometry, translucency, blur and sheen, applied to every surface: glass renders frosted over a tinted page, brutalist draws 3px ink borders with hard offset shadows, neumorphic extrudes borderless surfaces out of the page. It is seven `--surface-*` variables in the same token file as the colours (`shadcn-tokens.md`), so what the user picks is what gets ported. The popover is open on purpose — elevation and blur only show over content.

Tell the user the path and end your turn:

> "Five directions are in `<project-slug>/docs/design/ui-options.html` — click through the tabs and toggle dark on each. Tell me which letter, or which parts to combine. Nothing else is created yet; once you've picked I'll write the implementation plan for you to review."

Offline or a system-font stack? Remove the Google Fonts `<link>` from the preview too.

**Iterate before advancing.** Hybrids are the common outcome ("B's colours, D's typography"). Build the hybrid as a new option in `ui-options-v2.html` and get a clean pick on it. Never plan around a hybrid you have not shown.

## The Plan File

Step 11 — the last thing Phase 1 produces and the thing Phase 2 executes. Written **by hand** to `docs/design/UI-PLAN.md`. It makes the whole implementation reviewable in one read *before* any of it exists: the user should catch a wrong stack version, a missing page, or an over-broad scope from a page of markdown instead of from a diff.

**Copy the skeleton in `references/plan-template.md`** — Direction, Stack, Tokens & fonts, Scope table, Out of scope, Steps, Verification, Risks — rather than inventing a layout. It also carries what must stay *out* of each section.

### Gate 2 — asking for the plan review

Say where it is, what it commits to in two or three lines, and ask both questions. Then **end the turn.**

> "The plan is in `docs/design/UI-PLAN.md` — direction **B "Signal"**, React + Vite + Tailwind v4 + shadcn/ui, dark by default, five components: app shell, stat card, data table, form row, button set.
>
> Have a read and tell me if the scope and stack are right. Nothing is generated yet — say the word and I'll scaffold the Vite app and build from the plan."

- **Revisions loop back to step 10.** Edit `UI-PLAN.md` in place, restate what changed, ask again. One copy — never a `UI-PLAN-v2.md`.
- **A comment is not an approval.** "Looks good, though maybe fewer components" is a revision request.
- **A direction change loops further back** — "actually, C's typography" reopens Gate 1. Rebuild the option, get a clean pick, rewrite the plan.
- **Nothing in Phase 2 starts in the plan's turn.** Not the scaffold, not `npm install`, not the token file.

## Scaffolding the App

Step 12, **greenfield only** — skip entirely when a `package.json` exists. First action of Phase 2, so only after the plan is approved, using the command the plan names.

**Read `references/scaffolding.md` first** — per-framework commands, the Vite swap dance that preserves `docs/design/`, the styling-engine install. Three rules that do not wait for that file:

- **Always the non-interactive flag** (`--no-interactive` or its equivalent). A TTY prompt hangs the turn.
- **Never `--overwrite`.** It deletes `docs/design/` — the preview and token files the pick was made from.
- **Never scaffold into the folder that holds the design docs.** Vite prints `Operation cancelled`, creates nothing, and exits 0 — a silent no-op that reads as success.

Confirm `npm run build` passes before touching the theme. Then `src/index.css` is the token file.

## Implementing

Phase 2, after the plan was approved and (on a new project) after the app exists. The plan's Steps section is the running order; this is how each step is done. File locations and syntax per framework: `framework-recipes.md`. Token reference: `shadcn-tokens.md`.

1. **Tokens first, in one file** — complete light and dark blocks before any component. The winning option's CSS exists at `docs/design/option-tokens/<letter>-*.css`; port it, don't retype it. On a fresh Vite app, replace `src/index.css` outright — its demo styles are not a starting point.
2. **Port the whole option, `--surface-*` included.** The kit carries the visual style; colours alone ship a flat app in the right palette, which is not what the user picked. Components read `var(--surface-shadow)` / `var(--surface-border-width)` instead of inventing their own, the kit name goes on the app root for the few per-kit rules (`framework-recipes.md` → *Porting the surface kit*), and a translucent `card` stays translucent — flattening it deletes a glass direction.
3. **No comments in the token file.** `index.css` (and `app.css` / `globals.css` / `styles.css`) ships zero `/* … */` — not the seeder's `/* Option B … */` header, not section banners, not a note on a nudged lightness. Comments there go stale on the first value change, and the file is machine-read. Reasoning goes in `DECISIONS.md`. CSS token files only: JS theme objects and components keep the project's normal conventions.
4. **Fonts second** — mapped to `--font-sans` / `--font-serif` / `--font-mono` (or the framework's equivalent) so no component names a family directly.
5. **Components and pages last, tokens only** — every colour, radius, and spacing value references a token. A hex in a component ignores the mode toggle and drifts.
6. **Both modes, every component**, before starting the next.
7. **Only what the plan scopes.** An extra page you thought would help is unapproved scope. If the build genuinely needs something unlisted, add it to the plan and say so.
8. **Verify version-sensitive APIs** — Tailwind 3→4, Chakra 2→3, PrimeVue 3→4, Nuxt UI 2→3, Angular Material 17→18→20 all moved theming. Read `package.json`, never memory.
9. **Icons from one family** — `--domain icons` gives names with import code. One library, one stroke width, one size scale. Never emoji.
10. **Delete the scaffold's demo content** — `App.css`, the logo assets in `src/assets/` and `public/`, the counter markup. Set the real `<title>` in `index.html`.

Then the gate on the real token file:

```bash
node "<skill-dir>/scripts/contrast-check.mjs" src/index.css
```

- **FAIL** — a text pair below 4.5:1. Blocking, exits 1. Fix by adjusting lightness, not by lowering the bar.
- **WARN** — a non-text pair (`border`, `input`, `ring`) below 3:1. Advisory: fine for a divider, not for a focus `ring`.

Stock shadcn trips one FAIL out of the box (`muted-foreground on muted`, 4.34:1). Your options should not — the seeder already fixed that pair.

Before declaring done, grep: components for `#` and `rgb(` (hardcoded colours), and the token file for `/*` (comments). Both should return nothing.

## Documenting

Write `docs/design/DECISIONS.md` **by hand** — no generator, deliberately: the token file is the source of truth for values, and a generated copy only drifts. Keep it to what the code cannot say:

- The chosen option's **letter, name, thesis, and surface kit**.
- The **brand colour** if supplied, and the option's **accent relationship** — the next person needs to know whether `primary` is negotiable.
- **Component inventory** — what was built and where.
- **Usage rules** — which token for which purpose, plus no-hardcoded-colours and no-comments-in-the-token-file.
- **Why the non-obvious values are what they are** — a lightness nudged for contrast, a token deliberately off the seeded value. This is the home for the reasoning banned from `index.css`.
- Pointers to `ui-options.html` (rejected directions) and `UI-PLAN.md` (what was agreed).
- **Deviations from the plan**, and why. None? Say so in one line.
- On a new project: the **scaffold command, template, and installed majors**.

Do **not** restate token values, the spacing scale, component specs, or the plan's step list. The plan is the work order; this is the record of what shipped.

An existing `DECISIONS.md` — or a legacy `design-system/*/MASTER.md` — gets read and updated. Prior decisions are not yours to discard.

## Situations → What To Do

Rules stated once above are not repeated here. These are the judgement calls that live nowhere else.

| Situation | Do this |
|---|---|
| Only one option feels worth showing | Show three anyway. The choice *is* the deliverable. |
| The options differ only in hue, or all share one surface kit | Not a choice. Vary structure, type, density, and surface — check the tab strip. If the query only ranked one style family, broaden it or hand-swap a kit. |
| Only 2 options survive the seeder | Broaden the query and rerun. Two is not a choice. |
| Seeder reports "taken from the widened pool" | Check that direction against the brief by hand before showing it. |
| User has a brand colour | `--brand "#hex"`. Never hand-edit an option's `primary` afterwards — it breaks the "one colour, five ways" premise. |
| User picks an option and says "go ahead" in the same message | Still Phase 1. Write the plan, show it, ask. "Go ahead" came before the scope existed to agree to. |
| User skips the preview: "just implement something reasonable" | The options are the deliverable, not a formality. Offer the fast path: seed 3, pick in one turn — then plan, then build. |
| Project already has shadcn installed | Read `components.json`, keep the token names, replace values only. |
| Target is a component library with its own tokens | Design in shadcn vocabulary, map at implementation time via `framework-recipes.md`. |
| Contrast check FAILs on `muted-foreground` | Darken in light / lighten in dark. Never "it looks fine" your way past the gate. |
| Dark mode looks muddy | You inverted lightness instead of deriving it. `references/quick-reference.md` §6; check the background is lifted off pure black. |
| Animation feels wrong | `references/quick-reference.md` §7, then `--domain gsap` at the right tier. |
| Font is loaded but text looks unchanged | The `<link>` is not the wiring — map it to `--font-sans` / `--font-serif` / `--font-mono`. |
| User is offline or privacy-constrained | System font stack, no CDN link — in the preview too. |

## Red Flags — STOP

- About to run `npx shadcn init` before the UI framework was confirmed
- About to run any scaffolder or `npm install` before the plan was approved
- About to create a project folder, or any file, under `.claude/`
- About to pass `--overwrite` to a scaffolder because the target is not empty
- About to run a scaffolder without its non-interactive flag
- About to write `index.css` before the plan was approved
- About to write a `/* comment */` into a CSS token file
- About to port an option's colours and leave its `--surface-*` variables behind
- About to present options that all render as the same flat card in different hues
- About to write `UI-PLAN.md` before the user named an option by letter
- Ending the plan turn **without** asking whether to implement — the ask is the gate
- Starting Phase 2 in the plan's turn, on the strength of an earlier "go ahead"
- Building a component that is not in the plan's scope table
- Thinking "the default theme is fine for now" or "dark mode can come later"
- About to ask "rounded or sharp corners?" — that is what the options are for
- About to ask for a brand colour a second time, or design around one you never got
- Leaving a Vite logo or counter demo next to a themed app
- About to present a 0-result search as data instead of naming the fallback

**All of these mean: stop, back up to the checklist, get the missing confirmation.**

## Supporting Files

Read on demand, at the step that needs them — never up front:

| File | For | Step |
|---|---|---|
| `framework-recipes.md` | where the theme lives, what to write, porting the surface kit | 11, 13 |
| `shadcn-tokens.md` | token reference, `--surface-*` family, v4/v3 forms, OKLCH | 13 |
| `references/plan-template.md` | the `UI-PLAN.md` skeleton | 11 |
| `references/scaffolding.md` | scaffold commands, the Vite swap dance, stack install | 12 (skim at 11) |
| `references/seeding-internals.md` | how an option is built and derived | 8, when a seed looks wrong |
| `references/quick-reference.md` | all 98 UX rules by category | review passes |
| `reviewer-prompt.md` | reviewer subagent prompt | 18 |

Run, don't read — these never enter context:

- `scripts/seed-options.py` — database → preview-ready options with derived dark mode and surface kits
- `scripts/search.py` — database search, `--design-system`, dials (`design_system.py` is its generator)
- `scripts/contrast-check.mjs` — WCAG gate for a token file
- `scripts/mockup-template.html` — the tabbed preview harness the seeder fills in
- `scripts/validate_data.py` — data integrity; run after editing any CSV
- `data/` — 12 domain CSVs + 12 web stack CSVs
