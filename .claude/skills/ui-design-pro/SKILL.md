---
name: ui-design-pro
description: "Use when starting a new frontend project from scratch, or designing, building, reviewing, or improving UI — pages, components, colour theme, brand colour, index.css or theme config, design tokens, typography, font pairing, layout, spacing, radius, density, accessibility, animation, dark mode, data visualization — for a website, landing page, dashboard, admin panel, e-commerce, SaaS, portfolio, or blog, in any style (glassmorphism, brutalism, neumorphism, minimalism, bento grid)."
---

# UI Design Pro: Direction Into Themed Components

Turn a vague "make it look good" into a confirmed design direction and a working theme: token file, fonts, and the key components — in the user's actual stack.

**Two phases, and they never happen in the same turn.**

| Phase | You do | Ends with | Artifacts |
|---|---|---|---|
| **Planning phase** | Probe, confirm the stack, ask the four concept questions, seed 3-5 real options, gate on contrast, get a pick, write the plan | The user approving the plan | `docs/design/ui-options.html`, `option-tokens/*.css`, `UI-PLAN.md` |
| **Implementation phase** | Scaffold if new, then build exactly what the plan says | A verified, reviewed theme | the app, and a closing report in the chat |

The planning phase writes no code. The implementation phase makes no new decisions.

<HARD-GATE>
**Gate 1 — the pick.** Do NOT write a plan file until BOTH are true:
1. The user confirmed the FE framework AND the UI framework (never inferred, never defaulted).
2. The user picked one of the presented options (or a named hybrid).

Presenting options is not approval. Enthusiasm is not approval. "Looks nice" is not approval — ask which option, by letter.

**Gate 2 — the plan.** Do NOT write `index.css`, theme config, tokens, any component, **or scaffold an app** until the user has read `docs/design/UI-PLAN.md` and said to implement it.

A pick approves a *direction*. It does not approve a scope, a file list, or a stack install — those are the plan, and the plan gets its own yes. **The scope is not asked for, it is proposed**: you write what you think should be built first into the plan's scope table, and Gate 2 is where the user corrects it. After writing the plan you **end your turn** and wait.

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

Create a task per item and complete them in order. Steps 1-10 are the planning phase; 11-17 are the implementation phase and do not start until 10 came back approved.

### Planning phase

1. **Probe the frontend context** — run `scripts/probe-context.py`, read its verdict, and do what that verdict requires. On `THEMED` that includes asking refresh vs replacement; on `GREENFIELD` it means creating `./<project-slug>/docs/design/` and nothing else. See Context Probe.
2. **Ask the stack questions** — FE framework, UI framework, styling engine + version. REQUIRED. Never assume.
3. **Check compatibility** — validate the FE × UI pair against the matrix; resolve conflicts first.
4. **Ask the four concept questions** — what the product is, whether they have a main colour, whether any site inspired them, and whether it should animate. Nothing else.
5. **Query the database** — `--domain product`, then `color` / `style` / `typography`.
6. **Seed 3-5 options** — `scripts/seed-options.py`, with `--brand`, `--inspiration`, `--archetype` and `--no-animation` as the answers dictate.
7. **Contrast-gate every seed** — `contrast-check.mjs` per option CSS. Fix or drop before the user sees anything.
8. **Sharpen the seeds** — rewrite generated names and theses; check fonts, radius, surface kits, motion, and the archetype against the brief and the reference sites.
9. **GATE 1 — user picks an option.** Iterate on the preview for a hybrid or a tweak.
10. **Write `docs/design/UI-PLAN.md`, then GATE 2** — propose the scope in it, tell the user where the file is, ask them to review it, ask whether to implement. **End the turn.**

### Implementation phase

Entry condition: the plan was approved. Build what it says; if something in it turns out wrong, say so and amend the plan rather than improvising past it.

11. **Scaffold the app** — greenfield only, with the command the plan names. See Scaffolding. Skip on an existing project.
12. **Implement the token file** — `index.css` or the framework's theme config, light + dark. Port the winning option's CSS — colours **and** its `--surface-*` kit. No comments.
13. **Wire the fonts** — as the plan says, mapped to the token variables. `CSS Import` and `Tailwind Config` come from `typography.csv`.
14. **Implement the scoped components and pages** — the plan's list, in its order, tokens only.
15. **Motion pass** — build the option's motion personality, `--domain gsap` at the plan's tier. Skip entirely when animation was declined; state changes still get their 150-300ms transition.
16. **Verify** — `contrast-check.mjs` on the real token file, `npm run build`, both modes render, motion matches the picked personality.
17. **Review loop** — dispatch the reviewer subagent (`reviewer-prompt.md`); fix and re-dispatch until approved (max 5, then surface to the user). Then **report in the chat**: what was built, and every point where the result differs from the plan and why. No `DECISIONS.md` — the token file holds the values, the plan holds the agreement, and a third document only drifts from both.

Four steps loop rather than advance: the contrast gate reseeds (7 → 6), Gate 1 iterates the preview (9 → 8) until a letter is named, Gate 2 revises the plan in place (10, edited and re-asked) until the scope is approved, and the review loop re-dispatches (17) until approved or five iterations are spent.

## Review Priority

When reviewing existing UI rather than building new, work in this order — ranked by what each costs a user when wrong, not by visibility:

1. **Accessibility** (CRITICAL, `ux`) · 2. **Touch & interaction** (CRITICAL, `ux`) · 3. **Performance** (HIGH, `ux`) · 4. **Style selection** (HIGH, `style`+`product`) · 5. **Layout & responsive** (HIGH, `ux`) · 6. **Typography & colour** (MEDIUM, `typography`+`color`) · 7. **Animation** (MEDIUM, `ux`+`gsap`) · 8. **Forms & feedback** (MEDIUM, `ux`) · 9. **Navigation** (HIGH, `ux`) · 10. **Charts & data** (LOW, `chart`)

All 98 rules, with must-haves and anti-patterns per category, are in `references/quick-reference.md` — read the section you need, not the file.

## Context Probe

Find out what is already true before asking anything. **Run the probe first** — it reads every file below in one pass, so the interview only spends the user's attention on what genuinely isn't written down:

```bash
python "<skill-dir>/scripts/probe-context.py"        # or a path; --json for machine output
```

It reports the stack with **resolved** versions, the token files and what form their colours are in, the shadcn config, the component directory, the screen files, any prior design work, and any conflict it can see. It ends with a **verdict** and the steps that verdict requires — read that block, it is the instruction for the rest of step 1.

| Verdict | Means | You must |
|---|---|---|
| `GREENFIELD` | No `package.json`, no `composer.json` | Go to Greenfield. Stack questions are load-bearing, not confirmatory |
| `FRESH` | Project exists, no design tokens yet | Report, confirm the detected stack, run the normal planning phase |
| `THEMED` | A token file already carries design decisions | **Ask refresh vs replacement** before anything else — see below |
| `RESUME` | `docs/design/UI-PLAN.md` exists | Read the plan, restate its direction and scope in one line, ask whether it still stands, then resume at step 11 instead of reseeding |

### What the probe cannot do — and you must

The script reports file facts. Three things are judgement, and it hands you the inputs rather than a conclusion:

1. **Existing screens.** It lists the page/route files; reading them for density and layout conventions worth preserving is yours. A convention the user already lives with beats a fresh one.
2. **Refresh vs replacement**, on a `THEMED` verdict. This is a question for the **user**, never a decision you make from the file listing:
   - **refresh** — keep the token *names* and the component structure, change the values. The option set is seeded to fit what already exists.
   - **replacement** — a new direction. The option gate runs normally and the existing values are discarded.

   It changes what gets seeded, so it has to be answered **before step 5**. Do not proceed on silence, and do not infer it from how bad the current theme looks.
3. **Confirmation.** Detected is not confirmed. State the versions back to the user in the stack questions — a probe that read `react@19.2.8` tells you what is installed, not what they intend to keep.

Report what the probe found in one short paragraph — the token file, the component count, the stack — then ask only what is genuinely undecided.

**Conflicts block everything.** If the probe prints a `CONFLICTS` section — a Tailwind directive that disagrees with the installed major, two component libraries installed at once — resolve it with the user before theming anything. Both of those silently produce a theme that compiles and does nothing.

## Greenfield: No Project Yet

The checklist still applies; only step 1 changes and step 11 switches on.

```bash
mkdir -p ./<project-slug>/docs/design
```

`<project-slug>` is kebab-case from what they are building — `clinic-portal`, `nail-salon-booking`. Ask for the name if the concept doesn't hand you one.

**For all of the planning phase the project folder holds exactly this:**

```
<project-slug>/docs/design/
├── ui-options.html
├── option-tokens/{A,B,…}-<name>.css
└── UI-PLAN.md          # step 10, after the pick
```

No `package.json`, no `src/`, no framework, no `index.css` — the app is scaffolded at step 11, after the plan is approved. Nothing loose in the current directory and nothing in `/tmp`: the preview is a deliverable the user opens and keeps as the record of the rejected directions.

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

Four questions, asked one at a time. Every one of them changes what gets seeded; nothing else does.

1. **What is this product, and who uses it?** One or two sentences in their words. "A patient portal where clinic staff and patients both log in" is better than "a healthcare app". This answer *is* the database query, and it decides density, motion, accent loudness, which styles are candidates, and **which product the preview renders as** (`--archetype`).
2. **Do you have a main colour?** If yes, take the hex (or read it off the logo) and pass `--brand`. Every option then pins *their* colour and the choice becomes which treatment of it they want. If no, say "I'll propose one per direction — you'll pick a colour by picking a direction" and omit the flag.
3. **Is there a site whose look you want to borrow from?** Ask them to paste the links into the chat — as many as they have, and one line each on *what* they like about it ("the density", "the type", "the way it moves"). Then:
   - **Fetch each one** and read what is actually there — the type pairing, the surface treatment, the density, the motion. A link the user pasted and you never opened is worse than no link, because they will assume it landed.
   - Turn it into seeding input, not into a copy: the style keywords go into the query, a sampled brand colour can become `--brand`, and the links go through `--inspiration "Label=url=what they liked"` so they sit above the options in the preview.
   - **At least one option must visibly answer the reference**, and at least one must not — a reference is a data point about taste, not a specification. Say which is which when you present the set.
   - Cannot reach the site (offline, login wall, blocked)? Say so in one line and ask them to describe it instead. Never invent what a page you could not open looks like.
   - No reference? Fine — that is the common case, and the option set is what generates the taste signal instead.
4. **Should it animate?** One question, yes or no, with the trade-off in one line: "motion makes a product feel alive and makes a dense one feel slower". Then:
   - **Yes** → each option carries its own motion personality (`still` / `calm` / `crisp` / `springy` / `cinematic`), the preview plays it, and the implementation phase gets a real motion pass.
   - **No** → pass `--no-animation`. Every option previews static, step 15 is skipped, and the plan says so. State changes still get a 150-300ms transition — that is not animation, that is a control not looking broken.
   - Do **not** ask which animations, how long, or how much. That is the same delegation trap as asking about fonts; the option set shows five answers instead.

### Do not ask about

**Fonts. Corner radius. Density. Spacing. Mood adjectives. Style names. Light vs dark. Which components come first.**

Asking "sharp or rounded corners?" hands the work back to the person who came here to avoid it. Most users have no defensible answer, so they guess — and then you have built to a guess and called it taste. These are consequences of what the product is, and the option set makes them visible: five screens answer "do you want rounded corners" better than the question does.

Scope and default colour mode are the same: you **propose** them in the plan and the user corrects them at Gate 2. A scope question asked before the plan exists gets answered without the plan's context, which is how a five-component ask becomes a fifteen-component build.

Volunteered constraints ("our brand font is Söhne", "must be dark") are different — take them and thread them through.

### Dials come from the concept

`--density`, `--variance`, `--motion` and `--archetype` are properties of the product, not the user's mood. `seed-options.py` infers all four and prints what it inferred — read those lines; override only when you can say why.

| The concept | Inferred |
|---|---|
| Dashboard, admin, analytics, console, CRM, ERP, trading, monitoring | `--density 9`, `--motion 2`, `--archetype dashboard` |
| Landing, marketing, portfolio, agency, spa, hotel, luxury | `--density 3`, `--motion 8`, `--archetype landing` |
| Shop, storefront, catalog, checkout, retail, fashion, marketplace | `--archetype ecommerce` |
| Blog, magazine, editorial, publication, docs, knowledge base | `--density 3`, `--archetype editorial` |
| Bank, fintech, insurance, healthcare, government, legal, enterprise | `--variance 2` |
| Creative, agency, fashion, gaming, entertainment, art, experimental | `--variance 8` |
| Anything the words don't settle | dials `5`; archetype `landing` |

`--density` rewrites the `--space-*` scale, `--variance` biases which styles fill open slots, `--motion` sets the centre the options' motion personalities spread around and picks the GSAP tier, `--archetype` decides which miniature product every option is demonstrated on.

The user's answer to question 4 overrides the motion dial in one direction only: `--no-animation` forces every option to `still` no matter what the concept implied. A *yes* leaves the dial to the concept — "yes, some animation" on a trading terminal still means tier 2, not tier 8.

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
  --inspiration "Linear=https://linear.app=the density and the calm" \
  --out <project-slug>/docs/design/ui-options.html \
  --token-dir <project-slug>/docs/design/option-tokens
```

Paths are relative to the current directory; the script creates missing parents. On an existing project drop the `<project-slug>/` prefix and write to `docs/design/` at the repo root.

| Answer | Flag |
|---|---|
| They gave a brand colour | `--brand "#hex"` — omit when they did not |
| They pasted reference sites | `--inspiration "Label=url=why"`, repeated per link |
| They said no to animation | `--no-animation` |
| The concept's product type was misread | `--archetype dashboard\|landing\|ecommerce\|editorial` |
| A dial is wrong and you can say why | `--density` / `--variance` / `--motion` |

Each option is a distinct direction, not a hue variant: no repeated style, no repeated font pairing, ≥40° OKLCH hue separation, a different **surface kit** per slot where the query allows, a **motion personality** spread around the motion dial, then a **derived** dark theme (`colors.csv` has no dark values) with every WCAG text pair nudged clear of 4.5:1 in both modes. It reports every fallback and adjustment — read those lines. With `--brand` the hue axis is spent, so distinctness moves to surface, motion, type, shape, and accent strategy.

Three surface kits also adjust the palette and say so: `glass` (translucent `card`/`popover`, contrast enforced against the composited surface), `soft` (card = background, page off pure white), `hard`/`outlined` (border darkened toward the ink). Mechanics: `references/seeding-internals.md` — read it when a seed looks wrong.

### Step 7 — the contrast gate, and why it runs before anyone looks

```bash
for f in <project-slug>/docs/design/option-tokens/*.css; do node "<skill-dir>/scripts/contrast-check.mjs" "$f" || echo "FAILED: $f"; done
```

Any FAIL means reseed or hand-fix. **Zero failures is the entry condition for showing the preview** — not a checkbox afterwards, and not something to defer to the implementation phase.

Three reasons it sits here and not later:

- **A failing option is not an option.** If the user picks the direction whose `muted-foreground` sits at 3.9:1, one of two things follows: you ship it and the product is inaccessible, or you fix it afterwards and the colour they picked is no longer the colour they got. Both are worse than never showing it.
- **Fixing it here is a nudge; fixing it later is a rebuild.** At this point a lightness value moves by a few percent inside a generated file. After the pick, the same fix means editing the token file, the components built against it, and the screenshot the user remembers agreeing to.
- **It is the one part of taste that is not taste.** Everything else in this phase is a judgement the user makes. 4.5:1 is arithmetic, so it is settled before the judgement starts — that is exactly why it is automated and why "it looks fine to me" does not clear it.

WARNs on `border`/`input` are advisory and can ship; a WARN on `ring` cannot — an invisible focus indicator is an accessibility failure wearing a different name.

### Step 8 — sharpening the seeds, and why generated output is not shippable

The seeder is a database query with maths on top. It is very good at the parts that are mechanical (contrast, hue separation, dark derivation, kit selection) and incapable of the parts that are editorial. Step 8 is where the second half gets done, and skipping it is what makes an option set read as machine output — which the user then rejects wholesale instead of picking from.

What is wrong by construction until you fix it:

- **Name** — the script names an option after its style row, so it ships "Neumorphism" and "SaaS Mobile (High-Tech Boutique)". Those name a *technique*, and a user cannot pick between techniques. Give each one character: "Editorial Warm", "Signal", "Slate Precision". A direction needs a name someone can say out loud in a meeting.
- **Thesis** — generated by joining keywords, so it reads as a tag list. Replace with one line on what the option commits to and who it is for. The thesis is what the user actually compares; five keyword dumps are five ways of saying nothing.
- **Fonts** — the ranked pairing comes from text similarity, which occasionally lands a CJK family on an English spa site or a display face on a data table. Re-pick with `--domain typography` when it looks wrong.
- **Radius** — often stepped off a ladder rather than read from the style row, and the script says which per option. A data-dense console with 1rem corners is wrong even though the ladder produced it.
- **Surface kit** — the script reports whether the kit was read from the row or stepped off a ladder (32 of 84 style rows declare no technique). Check it suits the brief (`soft` on a trading terminal is wrong) *and* that the set varies — five options on `flat` is the hue-only failure one level down.
- **Motion** — the personalities are spread mechanically around the dial. Check each one against its own option: `cinematic` on the brutalist direction is a contradiction, and two adjacent options that both landed on `crisp` waste a slot.
- **Archetype and references** — this is the step where you check the preview is rendering the right product, and where you make sure at least one option answers the sites the user pasted while at least one deliberately does not.

Also check the set *as a set*, not option by option: if the tab strip reads "flat · flat · flat" or every thesis says "modern and clean", the seeder produced hue variants and the choice is fake.

Edit `ui-options.html` directly — the `OPTIONS` array is formatted for hand-editing. Change names, theses, fonts, kits, and motion; if you change a colour or a `--surface-*` value, change the matching `option-tokens/*.css` too, so the file the gate reads stays the file the preview shows.

## Rendering the Preview

`seed-options.py` writes `ui-options.html` filled in — one self-contained file, no server, no build, no app needed to view it.

**One option on screen at a time, switched by a tab bar.** Each tab carries its letter, name, surface kit, radius, motion style, and a five-swatch strip; the active option's header adds role, surface description, density, accent strategy, motion tier, and primary hex. Arrow keys and `1`–`5` switch tabs, `d` toggles dark, `r` replays the entrance animation, and the active letter is in the URL hash so an option can be linked or reopened. A stacked page answers "how do these differ" — the question that decides a pick is "does *this* screen work", and that needs the screen undivided.

**The brief sits above the options.** The concept sentence and the reference links the user pasted are rendered in the page header, because an option can only be judged against what it was seeded for — "is this right?" needs the "for what" in the same viewport.

### A direction is four things, and the preview shows all four

Colour and type are the two everybody expects. The other two are where an option set stops being a palette picker:

- **Surface kit** — `flat`, `outlined`, `elevated`, `soft`, `glass`, `hard`: border weight, shadow geometry, translucency, blur and sheen, applied to every surface. Glass renders frosted over a tinted page, brutalist draws 3px ink borders with hard offset shadows, neumorphic extrudes borderless surfaces out of the page. It is seven `--surface-*` variables in the same token file as the colours (`shadcn-tokens.md`), so what the user picks is what gets ported. The popover is open on purpose — elevation and blur only show over content.
- **Motion** — `still`, `calm`, `crisp`, `springy`, `cinematic`: entrance distance, easing curve, stagger, and hover lift, moving together. A screenshot cannot show it and a still option set silently decides it. Each option carries its own personality so the user picks how the product *moves* at the same time as how it looks; `Motion: on/off` compares against static, and `↻ Replay` re-runs the entrance. When the user declined animation, every option renders `still` and the controls are visibly disabled — nothing is sold that was not asked for. `prefers-reduced-motion` is honoured, with a banner saying so, because the shipped app must honour it too.

### The preview renders the user's product, not a generic screen

`ARCHETYPE` picks one of four miniature products — **dashboard** (sidebar, stat cards, chart, data table, form), **landing** (nav, hero, logo strip, features, pricing, CTA band), **ecommerce** (chips, filter rail, product grid, open cart), **editorial** (masthead, article, pull quote, figure, related, newsletter). It is inferred from the concept and overridable with `--archetype`.

This matters more than it sounds: a sidebar and a data table say "admin console" far louder than a palette says anything. Preview a boutique storefront as a dashboard and the user judges the wrong screen — they reject a direction that would have been right for their product, or accept one that only worked as a console.

Every option renders against the *same* archetype so the comparison stays fair, with light/dark and focus-state toggles because both modes and keyboard states ship. The three non-dashboard archetypes get a **component rail** appended — buttons, badges, an input, an open popover, the chart colours — so token coverage is identical whichever layout the concept selected.

Tell the user the path and end your turn:

> "Five directions are in `<project-slug>/docs/design/ui-options.html` — click through the tabs, toggle dark on each, and hit ↻ to watch how each one moves. B and D lean on the Linear reference you sent; A deliberately doesn't. Tell me which letter, or which parts to combine. Nothing else is created yet; once you've picked I'll write the implementation plan for you to review."

Offline or a system-font stack? Remove the Google Fonts `<link>` from the preview too.

**Iterate before advancing.** Hybrids are the common outcome ("B's colours, D's typography"). Build the hybrid as a new option in `ui-options-v2.html` and get a clean pick on it. Never plan around a hybrid you have not shown.

## The Plan File

Step 10 — the last thing the planning phase produces and the thing the implementation phase executes. Written **by hand** to `docs/design/UI-PLAN.md`.

It does two jobs at once, and both are load-bearing:

1. **It is the review surface.** The user should catch a wrong stack version, a missing page, or an over-broad scope from a page of markdown instead of from a diff. Everything in the plan is cheap to change while it is still a sentence.
2. **It is the work order.** The implementation phase makes no new decisions — it reads this file and builds it. Anything genuinely undecided when the plan is written stays undecided when the code is written, which is why "open questions" is a section rather than something to resolve silently.

**Copy the skeleton in `references/plan-template.md`** rather than inventing a layout. It also carries what must stay *out* of each section.

### What each section is for

Every section exists because something specific goes wrong without it. None of them is a summary of the others.

| Section | What it holds | Why it is there | What it feeds in the implementation phase |
|---|---|---|---|
| **Direction** | Letter, name, thesis, the option's token file path, surface kit, brand colour, accent strategy | Names *which* of the presented options was picked, unambiguously. "The blue one" is not a record. | Step 12 ports from the named CSS file — no re-deriving colours, no reopening the choice |
| **Stack** | FE framework, UI framework, styling engine, each with its major version, plus the exact scaffold and install commands on greenfield | Version is where a plan silently goes wrong: Tailwind 3 vs 4 and Chakra 2 vs 3 are different products with the same name. Writing the commands out is what lets the user veto an install *before* it runs | Step 11 runs these commands verbatim; steps 12-14 write against these majors |
| **Motion** | Whether animation was requested, the option's personality and tier, and what actually moves | Motion is the axis the user just picked in the preview and the one most likely to evaporate on the way to code | Step 15 builds this — or is skipped outright when it says animation was declined |
| **Tokens & fonts** | Token file path, default colour mode, how the mode is toggled, each font family mapped to its token, and the load method | These are the four decisions that are painful to reverse once components exist. Default mode in particular is proposed here, not asked earlier | Steps 12-13 are this section, executed |
| **Scope** | A table: each component/page, its file path, what it contains | The single most valuable section to review. Scope is where "add a theme" becomes a fortnight, and a table makes an over-broad build visible in ten seconds. It is **proposed** by you — the user's job is to cut it | Step 14 builds exactly these rows, in this order. A component not in the table is unapproved work |
| **Out of scope** | What is deliberately not in this pass | Turns silence into a decision. Without it "no auth screens" reads as an oversight and gets built anyway | Prevents step 14 from expanding; the reviewer flags additions as findings |
| **Steps** | The running order, as a numbered list | Makes the sequence reviewable — a user can see that scaffolding happens before tokens and tokens before components | The implementation phase's actual running order |
| **Verification** | The exact commands that must pass | Defines done before work starts, so "done" is not a judgement call at the end | Step 16 runs these |
| **Risks & open questions** | Anything genuinely undecided, and any version API to confirm after install | This is the pressure valve. Without it, an uncertainty gets resolved by guessing mid-build and nobody finds out | Answered by the user at Gate 2, or verified at the step that needs it |

Keep it to roughly one screen. It is a document the user reads in a minute, not a specification: no token dumps, no restated checklists, every file path concrete.

### Gate 2 — asking for the plan review

Say where it is, what it commits to in two or three lines, and ask both questions. Then **end the turn.**

> "The plan is in `docs/design/UI-PLAN.md` — direction **B "Signal"**, React + Vite + Tailwind v4 + shadcn/ui, dark by default, springy motion, and five components to start with: app shell, stat card, data table, form row, button set.
>
> The scope is my proposal, not your request — have a read and cut or add. Nothing is generated yet; say the word and I'll scaffold the Vite app and build from the plan."

- **Revisions edit the plan in place.** Restate what changed, ask again. One copy — never a `UI-PLAN-v2.md`.
- **A comment is not an approval.** "Looks good, though maybe fewer components" is a revision request.
- **A direction change loops further back** — "actually, C's typography" reopens Gate 1. Rebuild the option, get a clean pick, rewrite the plan.
- **Nothing in the implementation phase starts in the plan's turn.** Not the scaffold, not `npm install`, not the token file.

## Scaffolding the App

Step 11, **greenfield only** — skip entirely when a `package.json` exists. First action of the implementation phase, so only after the plan is approved, using the command the plan names.

**Read `references/scaffolding.md` first** — per-framework commands, the Vite swap dance that preserves `docs/design/`, the styling-engine install. Three rules that do not wait for that file:

- **Always the non-interactive flag** (`--no-interactive` or its equivalent). A TTY prompt hangs the turn.
- **Never `--overwrite`.** It deletes `docs/design/` — the preview and token files the pick was made from.
- **Never scaffold into the folder that holds the design docs.** Vite prints `Operation cancelled`, creates nothing, and exits 0 — a silent no-op that reads as success.

Confirm `npm run build` passes before touching the theme. Then `src/index.css` is the token file.

## Implementing

The implementation phase, after the plan was approved and (on a new project) after the app exists. The plan's Steps section is the running order; this is how each step is done. File locations and syntax per framework: `framework-recipes.md`. Token reference: `shadcn-tokens.md`.

1. **Tokens first, in one file** — complete light and dark blocks before any component. The winning option's CSS exists at `docs/design/option-tokens/<letter>-*.css`; port it, don't retype it. On a fresh Vite app, replace `src/index.css` outright — its demo styles are not a starting point.
2. **Port the whole option, `--surface-*` included.** The kit carries the visual style; colours alone ship a flat app in the right palette, which is not what the user picked. Components read `var(--surface-shadow)` / `var(--surface-border-width)` instead of inventing their own, the kit name goes on the app root for the few per-kit rules (`framework-recipes.md` → *Porting the surface kit*), and a translucent `card` stays translucent — flattening it deletes a glass direction.
3. **No comments in the token file.** `index.css` (and `app.css` / `globals.css` / `styles.css`) ships zero `/* … */` — not the seeder's `/* Option B … */` header, not section banners, not a note on a nudged lightness. Comments there go stale on the first value change, and the file is machine-read. Reasoning goes in the closing report, not the file. CSS token files only: JS theme objects and components keep the project's normal conventions.
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

## Closing Report — in the chat, not in a file

The last thing the implementation phase does is **say what happened**, in the conversation. No `DECISIONS.md`, no design-system doc: the token file already holds every value, `UI-PLAN.md` already holds what was agreed, and a third document restating both starts drifting from both on the first edit.

Keep it short and put the deviations first — that is the part the user cannot get anywhere else:

- **Where the result differs from the plan, and why.** Every difference: a component that turned out to need a sibling, a version whose theming API was not what the plan assumed, a token deliberately off the seeded value. None? Say "built as planned" in one line.
- **What was built and where** — the file paths, so the next message can point at one.
- **Anything a lightness nudge or a fallback changed** about the direction the user picked — if their brand colour had to move in dark mode, they need to hear it from you, not notice it later.
- **What is still out of scope**, one line, from the plan's own list.

Then stop. Do not restate the palette, the spacing scale, or the plan's step list back to someone who just read it.

If the project already carries a `DECISIONS.md` from an older run — or a legacy `design-system/*/MASTER.md` — **update it** rather than leaving it stale, and say you did. Prior decisions are not yours to discard. Just do not create one where none exists.

## Situations → What To Do

Rules stated once above are not repeated here. These are the judgement calls that live nowhere else.

| Situation | Do this |
|---|---|
| Only one option feels worth showing | Show three anyway. The choice *is* the deliverable. |
| The options differ only in hue, or all share one surface kit or one motion style | Not a choice. Vary structure, type, density, surface, and motion — check the tab strip, which prints all three per option. If the query only ranked one style family, broaden it or hand-swap a kit. |
| User pastes a reference site | Fetch it and read it before answering. Then make one option answer it and one deliberately not, and say which. A link you never opened is worse than no link. |
| User pastes a site you cannot open (login wall, offline, blocked) | Say so in one line and ask them to describe what they like about it. Never invent what a page you could not read looks like. |
| User wants "exactly like site X" | That is a build request, not a direction — say so. Seed the set anyway with X's style keywords in the query; the closest option becomes the starting point and the others show what was given up. |
| User says no to animation | `--no-animation`, and the plan's Motion section says animation was declined. State-change transitions still ship — they are not animation. |
| The preview renders the wrong kind of product | The archetype was misinferred. Re-run with `--archetype`; do not let the user judge a storefront by looking at a dashboard. |
| Only 2 options survive the seeder | Broaden the query and rerun. Two is not a choice. |
| Seeder reports "taken from the widened pool" | Check that direction against the brief by hand before showing it. |
| User has a brand colour | `--brand "#hex"`. Never hand-edit an option's `primary` afterwards — it breaks the "one colour, five ways" premise. |
| User picks an option and says "go ahead" in the same message | Still the planning phase. Write the plan, show it, ask. "Go ahead" came before the scope existed to agree to. |
| User skips the preview: "just implement something reasonable" | The options are the deliverable, not a formality. Offer the fast path: seed 3, pick in one turn — then plan, then build. |
| Project already has shadcn installed | The probe reports `style`, `baseColor` and `cssVariables` from `components.json`. Keep the token names, replace values only. |
| Probe says `THEMED` and the user just says "make it better" | That is not an answer to refresh-vs-replacement. Ask again with the two options named — the choice decides whether the existing token names survive. |
| Probe reports a token file with comments in it | Note it; the no-comments rule applies to the file you write. Do not clean up an existing file that is out of scope. |
| Probe finds no FE framework but a `package.json` exists | A backend-only or tooling repo. Say so and ask what the frontend actually is — never guess from a lockfile. |
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
- Starting the implementation phase in the plan's turn, on the strength of an earlier "go ahead"
- Building a component that is not in the plan's scope table
- Thinking "the default theme is fine for now" or "dark mode can come later"
- About to ask "rounded or sharp corners?" — that is what the options are for
- About to ask for a brand colour a second time, or design around one you never got
- About to ask which components to build first instead of proposing them in the plan
- About to seed options against a reference link you were given and never opened
- About to preview a storefront or an article as a dashboard because the archetype was never checked
- About to ship animation the user declined, or drop the motion personality they picked
- About to write a `DECISIONS.md` instead of reporting the deviations in the chat
- About to reseed over a project the probe called `THEMED` without asking refresh vs replacement
- Treating the probe's detected versions as confirmed instead of stating them back
- Leaving a Vite logo or counter demo next to a themed app
- About to present a 0-result search as data instead of naming the fallback

**All of these mean: stop, back up to the checklist, get the missing confirmation.**

## Supporting Files

Read on demand, at the step that needs them — never up front:

| File | For | Step |
|---|---|---|
| `framework-recipes.md` | where the theme lives, what to write, porting the surface kit | 10, 12 |
| `shadcn-tokens.md` | token reference, `--surface-*` family, v4/v3 forms, OKLCH | 12 |
| `references/plan-template.md` | the `UI-PLAN.md` skeleton | 10 |
| `references/scaffolding.md` | scaffold commands, the Vite swap dance, stack install | 11 (skim at 10) |
| `references/seeding-internals.md` | how an option is built and derived | 8, when a seed looks wrong |
| `references/quick-reference.md` | all 98 UX rules by category | review passes |
| `reviewer-prompt.md` | reviewer subagent prompt | 17 |

Run, don't read — these never enter context:

- `scripts/probe-context.py` — step 1: the project's frontend state, a verdict, and what that verdict requires
- `scripts/seed-options.py` — database → preview-ready options with derived dark mode, surface kits, motion personalities, and the archetype
- `scripts/search.py` — database search, `--design-system`, dials (`design_system.py` is its generator)
- `scripts/contrast-check.mjs` — WCAG gate for a token file
- `scripts/mockup-template.html` — the tabbed preview harness the seeder fills in: four archetypes, the motion layer, the concept banner
- `scripts/validate_data.py` — data integrity; run after editing any CSV
- `data/` — 12 domain CSVs + 12 web stack CSVs
