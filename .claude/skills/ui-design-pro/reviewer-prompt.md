# UI Design Reviewer Prompt Template

Use this template when dispatching a UI design reviewer subagent.

**Purpose:** Verify the implemented theme and components match the chosen direction, use tokens consistently, and hold up in both colour modes.

**Dispatch after:** Token file, fonts, and key components are implemented and the contrast check has been run.

```
Task tool (general-purpose):
  description: "Review implemented UI theme"
  prompt: |
    You are a UI design system reviewer. Verify this implementation is consistent,
    accessible, and faithful to the chosen design direction.

    **Chosen direction:** [OPTION_NAME] — [ONE_LINE_THESIS]
    **Surface kit:** [flat|outlined|elevated|soft|glass|hard] — [WHAT_IT_DRAWS]
    **Approved plan:** [PATH_TO_UI-PLAN.md] — read it first; it is the agreed scope
    **Brand colour:** [HEX_OR_"none — the direction proposed its own"]
    **Token file:** [TOKEN_FILE_PATH]
    **Component files:** [COMPONENT_FILE_PATHS]
    **UI framework + version:** [FRAMEWORK]@[VERSION]
    **Styling engine + version:** [ENGINE]@[VERSION]
    **Contrast check output:** [PASTE_OUTPUT]

    Read the files. Do not assume — verify every claim against what is written.

    Before the final pass, read `references/quick-reference.md` sections 1-3
    (Accessibility, Touch & Interaction, Performance — the CRITICAL and HIGH
    categories) and check the implementation against them.

    ## What to Check — Theme Integrity

    | Category | What to Look For |
    |----------|------------------|
    | Token completeness | Every token in the set defined in BOTH light and dark blocks; no key present in one and missing from the other |
    | Token syntax | One colour form throughout the token file — hex (the default), `oklch()`, or Tailwind v3's bare HSL triplets. A wrapped value (`#hex` or `oklch(…)`) in a **v3** setup is a hard failure: `tailwind.config` wraps it in `hsl()` and the colour disappears. Mixed forms within one file is a defect even where both are legal |
    | Hardcoded values | Any hex, `rgb()`, `hsl()`, or named colour literal inside a **component** file — these break theming silently. Hex in the *token* file is expected, not a finding |
    | Token file comments | Any `/* … */` in the CSS token file is a defect — including a ported `/* Option X - … */` header or a section banner. Values only; the reasoning belongs in `docs/design/DECISIONS.md`. This applies to CSS token files, not to JS theme objects or component files |
    | Brand fidelity | If the user supplied a brand colour, `--primary` in light mode is exactly that hex. In dark mode it may be lightened for legibility, but the hue must be unchanged — an off-hue dark primary means the brand shifted |
    | Font wiring | Families loaded AND mapped to token variables; components reference the token, never a family name |
    | Dark mode | Dark values deliberately chosen, not a mechanical inversion; background lifted off pure black; cards distinguishable from background |
    | Contrast | Every text pair ≥ 4.5:1 in both modes — blocking. `ring` must clear 3:1 against its surface, because an invisible focus indicator is an accessibility failure |
    | Border subtlety | `border` and `input` below 3:1 are ADVISORY, not defects. Stock shadcn ships its light border near 1.2:1 — a quiet divider is the convention. Only flag a border if it is absent in one mode while present in the other, or if it is the sole indicator of a control's boundary |
    | Focus states | Visible focus ring on every interactive element, using `ring`; not `outline: none` with no replacement |
    | Consistency | Radius, spacing, and font weights come from tokens — no one-off values |
    | Fidelity | Implementation actually matches the chosen direction's thesis (a "bold editorial" pick that shipped as default grey is a failure) |
    | Surface kit ported | All seven `--surface-*` variables present in BOTH modes, and components consuming them (`var(--surface-shadow)`, `var(--surface-border-width)`) rather than hardcoding their own shadows and 1px borders. Colours ported without the kit is a headline failure, not a detail: it ships a flat app in the right palette, which is not the option the user picked |
    | Surface kit honoured | The rendered treatment matches the kit. `glass` needs a translucent `card` (an opaque hex means the blur does nothing) plus `backdrop-filter` and the page wash; `soft` needs borderless surfaces and debossed inputs; `hard` needs the heavy border in the ink colour on controls too, not just on cards |
    | Framework fit | Theme written against the INSTALLED major version's API |
    | Scope | Exactly the components in the plan's scope table — nothing missing, no unrequested extras. An addition is as much a finding as an omission |
    | Plan fidelity | Stack, default colour mode, font load method, and file paths match what `UI-PLAN.md` says. A justified deviation must be recorded in `DECISIONS.md`; an unrecorded one is a finding |

    ## What to Check — Professional Polish

    These are the details that make an otherwise-correct theme read as unfinished.
    Each one is a specific, greppable failure, not a matter of taste.

    | Category | What to Look For |
    |----------|------------------|
    | Emoji as icons | Any emoji used as a structural icon (nav, settings, status, buttons). Icons must be SVG from one library — Lucide, Phosphor, Heroicons. Emoji are font-dependent, render differently per platform, and cannot take a token colour |
    | Icon family discipline | One icon library, one stroke width per visual layer, and filled vs outline not mixed at the same hierarchy level |
    | Icon sizing | Sizes come from a token scale (`icon-sm`/`icon-md`/`icon-lg`), not arbitrary 20/24/28px chosen per usage |
    | Layout-stable interaction | Hover and press states use colour, opacity, shadow, or border — never a transform that shifts surrounding content |
    | Cursor affordance | `cursor-pointer` on every clickable element, including clickable cards and rows, not just `<button>` |
    | Transition timing | 150-300ms on state changes. Instant (0ms) reads as broken; over 500ms reads as sluggish. Never animate `width`/`height` — use `transform` |
    | Reduced motion | `prefers-reduced-motion` honoured; animation reduced or disabled, not merely shortened |
    | Modal scrim | Overlay behind a dialog/drawer is strong enough to isolate the foreground (roughly 40-60% black). A weak scrim leaves the background competing |
    | Sticky element clearance | No content hidden behind a fixed/sticky header, footer, or CTA bar. Scroll containers have matching top/bottom insets |
    | Spacing rhythm | A consistent 4/8px scale for padding, gaps, and section spacing — not arbitrary increments |
    | Responsive integrity | No horizontal scroll at 320px; layout holds at 320/768/1024/1440 |
    | Form basics | Every input has a real visible label (not placeholder-as-label), errors sit next to the field they belong to |
    | Colour not the only signal | Status, errors, and chart series carry an icon, label, or pattern in addition to colour |
    | Scaffold leftovers | On a newly scaffolded app: no `App.css`, no Vite/framework logo assets in `src/assets/` or `public/`, no counter demo markup, and `index.html` has the real project `<title>` — not "Vite + React" |
    | Project location | The app and its `docs/design/` sit in a normal project folder, not under `.claude/` or a worktree-internal path the user cannot commit |

    ## CRITICAL

    Look especially hard for:
    - Tokens defined in `:root` but missing from `.dark` (or vice versa)
    - Hardcoded colours in component files — grep for `#`, `rgb(`, `hsl(` outside the token file
    - Comments in the CSS token file — grep it for `/*`; expect zero hits
    - `--surface-*` variables missing from the token file, or defined but never referenced by a component
    - `muted-foreground` on `muted` failing contrast (the most common failure)
    - Font `<link>` or import present but no token mapping
    - Theme API syntax belonging to a different major version than the one installed
    - Components that only work in light mode
    - Emoji standing in for icons anywhere in the shipped components
    - `outline: none` with no replacement focus indicator
    - Scaffold demo content still shipping next to the real UI (Vite logo, counter, `App.css`)

    ## Output Format

    ## UI Design Review

    **Status:** ✅ Approved | ❌ Issues Found

    **Issues (if any):**
    - [File:line]: [specific issue] - [why it matters]

    **Recommendations (advisory):**
    - [suggestions that don't block approval]
```

**Reviewer returns:** Status, Issues (if any), Recommendations
