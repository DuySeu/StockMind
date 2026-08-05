# UI-PLAN.md skeleton

The shape of the step 10 plan file. Copy it, fill every placeholder, delete the rows and
sections that do not apply — an unfilled `<…>` reaching the user is a defect.

Keep the whole thing to roughly one screen. It is a document the user reads in a minute
to catch a wrong stack or a wrong scope, not a specification.

It has two jobs and they pull in opposite directions, which is what the section list is
balancing:

- **Reviewable before it exists.** Everything here is cheap to change while it is still a
  sentence. A wrong Tailwind major, a scope three times bigger than the ask, a page that
  was never mentioned — all of them are a one-line edit now and a rebuild later.
- **Executable without further decisions.** The implementation phase reads this file and
  builds it. Anything left undecided here gets decided by guessing at the keyboard, which
  is why *Risks & open questions* is a real section and not a formality.

**What each section is for** — the full table (what it holds, why it exists, what it
feeds in the implementation phase) is in SKILL.md → *The Plan File*. Read it once; this
file is the shape, that table is the reasoning.

**What stays out**, per section — a plan that restates the code is a second copy to keep
in sync:

| Section | Keep out |
|---|---|
| Direction | The token values — they are in the option CSS |
| Stack | Alternatives already ruled out |
| Motion | A keyframe spec — the personality and the tier are the decision |
| Tokens & fonts | A restated token table |
| Scope | Anything the user did not ask for, padded out to look thorough |
| Steps | Prose restating the skill's checklist |
| Risks | Taste questions the option set already answered |

One page, no token dumps, no invented components, every file path concrete. If a scoped
item needs a decision you do not have, it goes in Risks — not silently resolved.

**The scope table is a proposal.** It is the section most worth the user's attention, and
they have not been asked about it before now — say so at Gate 2 so they know it is theirs
to cut.

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
- Reference sites the user gave: <label + url, and which options answered them> / or: none
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

## Motion

- Animation: **<requested / declined>**
- Personality: `<still / calm / crisp / springy / cinematic>` at tier <N> — <what it does: entrance distance, easing, hover lift>
- What moves: <e.g. card entrance on first paint, hover lift on cards and rows, page transitions>
- What does not: <e.g. no scroll-triggered reveals, no parallax, no loading skeleton animation>
- `prefers-reduced-motion` is honoured — animation disabled, not merely shortened

Or, when animation was declined:

- Animation: **declined.** No entrance, scroll, or hover-transform motion. State changes still
  transition at 150-300ms — that is a control not looking broken, not animation. Step 15 is skipped.

## Tokens & fonts

- Token file: `src/index.css` — both modes, ported from the option CSS (colours + `--surface-*`), **no comments**
- Default colour mode: <light / dark> — proposed, not asked; say so, toggled by <the `dark` class on `<html>`, persisted>
- Display font: `<Family>` → `--font-serif` / `--font-sans`, loaded via <Google Fonts link / @fontsource-variable / system stack>
- Body font: `<Family>` → `--font-sans`
- Mono: `<Family or "none — omit the token">`

## Scope — <N> items

Proposed, not requested. Cut or add anything.

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
5. Motion pass — <personality> at tier <N> / or: skipped, animation was declined
6. Delete the scaffold's demo content and set the real `<title>` (greenfield only)

## Verification

```bash
node "<skill-dir>/scripts/contrast-check.mjs" src/index.css
npm run build
```

Then a reviewer subagent pass against this plan (`reviewer-prompt.md`), and a closing
report in the chat naming every deviation from this file. No separate decisions document.

## Risks & open questions

- <Theming API of <library>@<major> is confirmed against `package.json` after install, not from memory>
- <anything genuinely undecided — never a taste question the option set already answered>
````
