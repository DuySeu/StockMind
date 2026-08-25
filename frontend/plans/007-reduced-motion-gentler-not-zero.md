# 007 — Make reduced motion gentler, not zero

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed
- **Depends on**: 001 (the `--tw-enter-*` / `--tw-exit-*` variables only exist once `tw-animate-css` is imported); run **after** 004, which edits the same line in `HomePage.tsx`
- **Commit**: 8fcd66a
- **Severity**: LOW
- **Category**: Accessibility
- **Estimated scope**: 2 files, ~16 lines

## Problem

The global reduced-motion rule collapses every animation and transition in the product to effectively zero:

```css
/* src/index.css:360-370 — current */
@media (prefers-reduced-motion: reduce) {

  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```

Reduced motion means *fewer and gentler* animations, not none: transitions that aid comprehension should stay, and only position changes should go. Under the current rule a user with the preference set gets a product where every dialog, menu and tooltip teleports — which is harder to follow, not easier, and is exactly the jarring-change problem plans 001–006 exist to fix. Those users would receive none of the benefit.

Someone already noticed this and worked around it for one component:

```tsx
/* src/components/ui/button.tsx:8-10 — current */
// active:scale — every button gets a physical press, not just a hover colour.
// motion-reduce opts out; the global reduced-motion rule only kills duration.
"… transition-all active:scale-[0.98] motion-reduce:active:scale-100 …"
```

Separately, one hover effect moves with no reduced-motion or pointer gating:

```tsx
/* src/pages/HomePage.tsx:486 — current */
className="group block rounded-xl transition-transform hover:-translate-y-0.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
```

## Target

Keep the existing blanket rule as the floor — it correctly freezes decorative loops such as `.animate-fade-right` (a 1.5s `clip-path` reveal at `src/index.css:331-333`) — then carve out a single exception: enter/exit surfaces keep a short fade with **all travel and scale removed**.

```css
/* src/index.css:360-… — target, replacing the current block */
@media (prefers-reduced-motion: reduce) {

  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }

  /* Gentler, not none. Surfaces that enter and exit keep a short fade with no
     travel and no scale: a dialog that teleports in is harder to follow than
     one that fades, and the preference is for less movement, not less
     information. tw-animate-css drives all of its motion through these
     variables, so zeroing them leaves the opacity change and nothing else. */
  .animate-in,
  .animate-out {
    --tw-enter-translate-x: 0 !important;
    --tw-enter-translate-y: 0 !important;
    --tw-enter-scale: 1 !important;
    --tw-enter-rotate: 0 !important;
    --tw-exit-translate-x: 0 !important;
    --tw-exit-translate-y: 0 !important;
    --tw-exit-scale: 1 !important;
    --tw-exit-rotate: 0 !important;
    animation-duration: 150ms !important;
  }

  /* Colour still carries state changes — a tool chip going grey to green is
     information, not decoration. */
  .transition-colors {
    transition-duration: 150ms !important;
  }
}
```

```tsx
/* src/pages/HomePage.tsx:486 — target (after plan 004 has appended its own classes) */
className="group block rounded-xl transition-transform hover:-translate-y-0.5 motion-reduce:hover:translate-y-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring animate-in fade-in slide-in-from-bottom-2 animation-duration-300 ease-out"
```

Why it is built this way:

- `.animate-in` / `.animate-out` (specificity 0,1,0) beat the universal selector (0,0,0), so the carve-out wins even though both use `!important`.
- Spinners (`animate-spin`), skeletons (`animate-pulse`) and the sidebar width transition are **deliberately left frozen exactly as today**. They are status indicators and layout, not entrance choreography, and changing them is out of scope.
- 150ms is inside the tooltip/small-popover budget and short enough that a fade with no movement never feels like waiting.

## Repo conventions to follow

- Media-query blocks live at the bottom of `src/index.css`, after the utilities layer: `@media (prefers-reduced-transparency: reduce)` at `:335-358` is the immediate neighbour and the exemplar for structure and comment voice.
- The codebase prefers `motion-reduce:` Tailwind variants for per-component opt-outs — `src/components/ui/button.tsx:10` is the exemplar to imitate for the `HomePage` change.

## Steps

1. In `src/index.css`, inside the existing `@media (prefers-reduced-motion: reduce)` block (starting line 360), leave the universal `*, *::before, *::after` rule exactly as it is and add the two new rules — `.animate-in, .animate-out { … }` and `.transition-colors { … }` — after it, with the comments shown in the target.
2. In `src/pages/HomePage.tsx:486`, add `motion-reduce:hover:translate-y-0` directly after the existing `hover:-translate-y-0.5` class. Do not reorder or remove any other class on that line.

## Boundaries

- Do NOT delete or weaken the universal `*, *::before, *::after` rule — it is what keeps decorative loops and the 1.5s `.animate-fade-right` reveal from running under the preference.
- Do NOT add reduced-motion handling for `animate-spin`, `animate-pulse` or the sidebar transitions; leaving them as they are today is deliberate.
- Do NOT touch `src/components/ui/button.tsx` — its `motion-reduce:active:scale-100` opt-out is already correct.
- Do NOT introduce `useReducedMotion()` or any JavaScript branching; this is a CSS-only change.
- Do NOT add new dependencies.
- If `src/index.css:360-370` does not match the "current" excerpt above, the file has drifted since commit `8fcd66a` — STOP and report.

## Verification

- **Mechanical**: `npx vite build` → succeeds. `npx tsc -b` → no errors.
- **Feel check**: `npm run dev`, then in DevTools → Rendering set `Emulate CSS media feature prefers-reduced-motion: reduce`, and confirm:
  - Opening the upload dialog on `/documents` **fades** in over ~150ms and does **not** scale or slide. Escape fades it out the same way.
  - The theme dropdown in the header fades with no scale-from-corner.
  - On `/c`, a tool chip still glides grey → green (colour is information) but does not scale in.
  - Loading spinners and skeleton pulses behave exactly as they did before this change — verify by toggling the emulation off and on; nothing about them should differ from the pre-change behaviour.
  - The conversation title on `/c/<id>` does **not** play the 1.5s left-to-right reveal.
  - Hovering a news card on `/` does not lift it.
  - Turn the emulation off and re-check `/`: the news card lift returns and card entrances slide again.
- **Done when**: with the preference on, entrances fade without moving and nothing teleports; with it off, behaviour is identical to before this plan; `git diff --stat` shows exactly two files changed.
