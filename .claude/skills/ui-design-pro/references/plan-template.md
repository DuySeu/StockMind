# UI-PLAN.md skeleton

The shape of the step 11 plan file. Copy it, fill every placeholder, delete the rows and
sections that do not apply — an unfilled `<…>` reaching the user is a defect.

Keep the whole thing to roughly one screen. It is a document the user reads in a minute
to catch a wrong stack or a wrong scope, not a specification. Its job is to make the
implementation reviewable *before* it exists.

**What stays out**, per section — a plan that restates the code is a second copy to keep
in sync:

| Section | Keep out |
|---|---|
| Direction | The token values — they are in the option CSS |
| Stack | Alternatives already ruled out |
| Tokens & fonts | A restated token table |
| Scope | Anything the user did not ask for |
| Steps | Prose restating the skill's checklist |
| Risks | Questions you should have asked at step 10 |

One page, no token dumps, no invented components, every file path concrete. If a scoped
item needs a decision you do not have, it goes in Risks — not silently resolved.

---

````markdown
# UI Implementation Plan — <Project Name>

Status: awaiting review · written <YYYY-MM-DD>

## Direction

**Option <LETTER> — "<Name>"** · <one-line thesis>

- Tokens: `docs/design/option-tokens/<LETTER>-<name>.css` (light + dark, contrast-checked)
- Surface kit: `<flat / outlined / elevated / soft / glass / hard>` — <what it draws: border weight, shadow, blur>. Ported as the seven `--surface-*` vars; components read them instead of inventing shadows
- Brand colour: `<#hex>` — pinned as `--primary` / or: none supplied, this direction proposes `<#hex>`
- Accent strategy: <analogous / complementary / neutral-with-one-accent / …>
- Rejected directions stay on record in `docs/design/ui-options.html`

## Stack

| Layer | Choice | Version |
|---|---|---|
| FE framework | <React / Vue 3 / …> | <major> |
| UI framework | <shadcn/ui / Tailwind only / …> | <major or "copy-in source"> |
| Styling engine | <Tailwind v4 / …> | <major> |

Greenfield — the app does not exist yet, and these commands run only after this plan is approved:

```bash
npm create vite@latest <slug>-app -- --template <template> --no-interactive
# docs/ swap per references/scaffolding.md, then:
npm install <styling engine packages>
npx shadcn@latest init        # if a component library was chosen
```

Existing project — nothing is scaffolded; the majors above were read from `package.json`.

## Tokens & fonts

- Token file: `src/index.css` — both modes, ported from the option CSS (colours + `--surface-*`), **no comments**
- Default colour mode: <light / dark>, toggled by <the `dark` class on `<html>`, persisted>
- Display font: `<Family>` → `--font-serif` / `--font-sans`, loaded via <Google Fonts link / @fontsource-variable / system stack>
- Body font: `<Family>` → `--font-sans`
- Mono: `<Family or "none — omit the token">`

## Scope — <N> items

| # | Component / page | File | Contains |
|---|---|---|---|
| 1 | <App shell> | `src/components/<path>` | <sidebar, topbar, mode toggle> |
| 2 | <Stat card> | `src/components/<path>` | <label, value, delta, icon slot> |
| … | | | |

## Out of scope this pass

- <e.g. auth screens, settings, empty/error states, real data wiring>

## Steps

1. Scaffold + install the stack above; confirm `npm run build` (greenfield only)
2. Write the token file from the option CSS — light + dark, comment-free
3. Wire the fonts and map them to the font tokens
4. Build scope items in order: <1 → 2 → …>, each verified in both modes
5. Motion pass at tier <N> / or: skipped, motion dial is <1-2>
6. Delete the scaffold's demo content and set the real `<title>` (greenfield only)

## Verification

```bash
node "<skill-dir>/scripts/contrast-check.mjs" src/index.css
npm run build
```

Then a reviewer subagent pass against this plan (`reviewer-prompt.md`), and
`docs/design/DECISIONS.md` written last.

## Risks & open questions

- <Theming API of <library>@<major> is confirmed against `package.json` after install, not from memory>
- <anything genuinely undecided — never a question that step 10 should have asked>
````
