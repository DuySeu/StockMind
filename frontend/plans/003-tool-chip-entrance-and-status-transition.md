# 003 — Give tool chips an entrance and a colour transition

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed (needs a live stream)
- **Depends on**: 001 (needs the `animate-in`/`fade-in`/`zoom-in-95` utilities and `--ease-out`)
- **Commit**: 8fcd66a
- **Severity**: LOW
- **Category**: Missed opportunities (state indication)
- **Estimated scope**: 1 file, ~4 lines

## Problem

During a turn, each tool the model invokes appears as a chip, and each chip changes appearance twice: it is inserted mid-answer, then it flips from `running` (grey, spinner) to `succeeded` (green, check) or `failed` (red, cross). Both changes happen in a single frame.

```tsx
/* src/components/containers/MessageList.tsx:232-248 — current */
<span
  key={tc.id || `tc-${idx}`}
  className={`inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs ${chipClass}`}
  title={tc.arguments ? `${tc.name}(${tc.arguments})` : tc.name}
>
```

```ts
/* src/components/containers/MessageList.tsx:58-81 — current, no transition on any variant */
running: { chipClass: "border-border bg-muted text-muted-foreground", … },
success: { chipClass: "border-emerald-600/30 bg-emerald-500/10 text-emerald-800 dark:border-emerald-400/30 dark:text-emerald-300", … },
error:   { chipClass: "border-red-600/30 bg-red-500/10 text-red-800 dark:border-red-400/30 dark:text-red-300", … },
```

The chip strip is the running commentary of what the assistant is doing, and it grows while the user watches. A chip snapping into existence pushes the answer text down with no bridge, and the grey→green flip reads as a repaint rather than a result arriving.

**Frequency note, and why this stays deliberately small:** this is the app's core screen, seen tens of times a day. Per the audit's frequency rule, that tier gets "near-imperceptible motion or none". 150ms with a 95% scale is at the imperceptible end on purpose. Do not increase these numbers; a chip strip that visibly animates on every turn would be worse than no animation at all.

## Target

One `duration-200` covers both the entrance and the colour change. In Tailwind v4, `duration-*` compiles to `--tw-duration: .2s; transition-duration: .2s` — it sets the transition duration *and* the value `animate-in` falls back to for its animation duration. Here that coupling is wanted: the chip's only transition is `transition-colors`, and 200ms is the right value for both it and the entrance.

> **Trap — read before copying this pattern elsewhere.** `duration-*` silently retimes any transition the element already has. On an element that carries `transition-transform`, `transition-all`, or a `Button` with `active:scale-[0.98]`, adding `duration-300` for an entrance also stretches the hover or press feedback to 300ms, well past the 100–160ms press budget. When the element already has a transition on a different budget, use `animation-duration-*` (from `tw-animate-css`) instead — it compiles to `animation-duration` only and leaves `transition-duration` at the 150ms default. Plans 004 and 005 both need that form.

```tsx
/* src/components/containers/MessageList.tsx:232-236 — target */
<span
  key={tc.id || `tc-${idx}`}
  className={`inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs transition-colors animate-in fade-in zoom-in-95 duration-200 ease-out ${chipClass}`}
  title={tc.arguments ? `${tc.name}(${tc.arguments})` : tc.name}
>
```

Behaviour this produces:

- **On insert**: the chip fades in from `opacity: 0` and scales up from 95% over 200ms on the strong ease-out curve. Never from `scale(0)` — nothing appears from nothing.
- **On status flip**: `transition-colors` glides the border, background and text between the grey, green and red variants over 200ms. The icon still swaps instantly (spinner → check); that is correct, an icon crossfade would blur the one thing carrying the outcome.

No change is needed to `toolStatusStyle` — its `chipClass` strings are the transition's start and end states.

## Repo conventions to follow

- This file builds `className` with template literals and a lookup object of variant strings (`toolStatusStyle`, `src/components/containers/MessageList.tsx:58`). Keep the shared/static classes in the template literal and the variant classes in `chipClass` — do not move motion classes into `toolStatusStyle`.
- The codebase already writes `transition-colors` on interactive elements: `src/components/containers/MessageList.tsx:169` (summary hover) and `:293` (retry button). Same utility, same intent.
- Exemplar for an entrance on a streamed element after plan 001: `src/pages/MarketResearcherPage.tsx:294` (`animate-in fade-in duration-300`).

## Steps

1. In `src/components/containers/MessageList.tsx`, at the chip `<span>` (line 234-235), add `transition-colors animate-in fade-in zoom-in-95 duration-200 ease-out` to the static portion of the `className` template literal, before the interpolated `${chipClass}`. Use the **Target** block above verbatim.
2. Change nothing else in the file.

## Boundaries

- Do NOT add motion to the "Tool run summary" line (`src/components/containers/MessageList.tsx:185-219`) — its text and icon change on every tool completion, and animating it would flicker.
- Do NOT animate the inline tool-error paragraphs (`:255-264`); an error message must be readable the instant it appears.
- Do NOT change `toolStatusStyle` (`:58-81`) or `turnErrorStyle` (`:90-107`).
- Do NOT add an entrance to the assistant or user message bubbles (`:304-315`) — that was considered and rejected: the user's own bubble echoes text they typed a keystroke ago, and there is nothing jarring to bridge.
- Do NOT increase the duration past 200ms or the scale below 95%.
- Do NOT add new dependencies.
- If line 234-235 does not match the "current" excerpt above, the file has drifted since commit `8fcd66a` — STOP and report.

## Verification

- **Mechanical**: `npx tsc -b` → no errors. `npm run lint` → no new warnings.
- **Feel check**: `npm run dev`, open `/c`, and send a question that calls tools (e.g. "get FPT stock price and report" — it is one of the built-in suggestions):
  - Each chip should fade and scale in as the model invokes it, not pop.
  - When a tool finishes, the chip should glide grey → green rather than repainting. The spinner→check icon swap stays instant; that is intended.
  - Send a second question in the same conversation. The chips from the *previous* turn must not re-animate — they are keyed by `tc.id` and the DOM node is reused, so a re-render must not restart the entrance. If old chips flicker on every stream delta, STOP and report; that means the key or the render path changed.
  - In DevTools → Animations at 10% playback, confirm the chip grows from 95%, not from zero, and that it never overshoots past 100%.
- **Done when**: chips animate in once each, status colour changes glide, and `git diff --stat` shows one file changed with one modified line.
