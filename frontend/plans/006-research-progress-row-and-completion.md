# 006 — Animate research progress rows and their completion

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed (needs a live digest run)
- **Depends on**: 001 (needs `animate-in`, `fade-in`, `slide-in-from-top-1`, `zoom-in-95` and `--ease-out`)
- **Commit**: 8fcd66a
- **Severity**: LOW
- **Category**: Missed opportunities (state indication on a rare, high-attention screen)
- **Estimated scope**: 1 file, ~3 lines

## Problem

Running a digest on `/research` is a slow, streamed operation the user sits and watches — up to five tickers, each moving through *building prompt → submitting → polling → parsing → completed*. Two moments in that screen have no motion at all.

**Rows appear instantly** as each ticker's first progress event arrives:

```tsx
/* src/pages/MarketResearcherPage.tsx:300-304 — current */
return (
  <div
    key={ticker}
    className="flex items-center gap-3 rounded-lg border border-border bg-muted/40 p-3"
  >
```

**The completion tick replaces the spinner in one frame** — the payoff of a multi-second wait, rendered as a repaint:

```tsx
/* src/pages/MarketResearcherPage.tsx:307-310 — current */
{isCompleted ? (
  <span className="flex size-6 items-center justify-center rounded-full bg-status-ok-bg">
    <Check className="size-3.5 text-status-ok" aria-hidden="true" />
  </span>
) : isFailed ? (
```

Note that the surrounding container already asks for an entrance that currently does nothing:

```tsx
/* src/pages/MarketResearcherPage.tsx:294 — current, dead until plan 001 lands */
<div className="space-y-4 animate-in fade-in duration-300">
```

This screen is on the rare frequency tier — a user runs a digest occasionally, not dozens of times a day — so it can carry standard animation without ever becoming noise.

## Target

```tsx
/* src/pages/MarketResearcherPage.tsx:300-304 — target */
return (
  <div
    key={ticker}
    className="flex items-center gap-3 rounded-lg border border-border bg-muted/40 p-3 animate-in fade-in slide-in-from-top-1 duration-200 ease-out"
  >
```

```tsx
/* src/pages/MarketResearcherPage.tsx:307-310 — target */
{isCompleted ? (
  <span className="flex size-6 items-center justify-center rounded-full bg-status-ok-bg animate-in fade-in zoom-in-95 duration-200 ease-out">
    <Check className="size-3.5 text-status-ok" aria-hidden="true" />
  </span>
) : isFailed ? (
```

Why these exact values:

- Rows enter **from the top** (`slide-in-from-top-1` = 4px) because they are appended below the header of a growing list — the direction matches where they come from.
- 200ms on the strong ease-out curve, matching plans 003–005, so the whole app decelerates the same way.
- The completion badge scales from **95%, not smaller**. A bigger celebratory pop was considered and rejected: nothing in the real world appears from nothing, the recommended entrance range is 0.9–0.97, and on a 24px badge a larger scale reads as a glitch rather than a flourish. The spinner→check swap is what carries the news; the scale just softens it.
- No stagger. Rows arrive on their own schedule from the SSE stream, so real timing already separates them; an artificial delay would fight it.

Nothing needs to change at `:294` — that container's existing `animate-in fade-in duration-300` starts working the moment plan 001 lands.

## Repo conventions to follow

- This file writes conditional variant classes with template literals and ternaries (see the `<Progress>` className at `:333-339`) — keep the static motion classes in the plain string portion.
- Existing entrance in this same file to imitate: `src/pages/MarketResearcherPage.tsx:294`.
- Comments in this file explain layout and colour decisions in one or two lines above the element (`:212-214`, `:284`, `:359-360`). Add one short comment above the row `<div>` noting why rows enter from the top and are not staggered.

## Steps

1. In `src/pages/MarketResearcherPage.tsx:303`, append `animate-in fade-in slide-in-from-top-1 duration-200 ease-out` to the row `<div>`'s `className`. Add the explanatory comment above the `<div>`.
2. In `src/pages/MarketResearcherPage.tsx:308`, append `animate-in fade-in zoom-in-95 duration-200 ease-out` to the completed badge `<span>`'s `className`.
3. Change nothing else in the file.

## Boundaries

- Do NOT animate the failed badge (`:311-316`). A failure must be legible instantly; a pop on an error reads as celebration.
- Do NOT touch the `<Progress>` bars (`:330-340`). The primitive already transitions its indicator width (`src/components/ui/progress.tsx:22`), and the bar is a value the user reads.
- Do NOT animate the past-reports `<Table>` rows (`:378-409`) — that is a data table being read, and rows arrive all at once from one fetch.
- Do NOT change the streaming logic, the `tickerLatestSteps` grouping (`:156-164`), or the cancel path.
- Do NOT add a stagger delay to the rows.
- Do NOT add new dependencies.
- If the excerpts above do not match the file, it has drifted since commit `8fcd66a` — STOP and report.

## Verification

- **Mechanical**: `npx tsc -b` → no errors. `npm run lint` → no new warnings.
- **Feel check**: `npm run dev`, go to `/research`, add two or three tickers and run a digest:
  - Each ticker row should fade and settle down from ~4px above as its first event lands, rather than snapping into the list.
  - When a ticker reaches `completed`, the green check badge should scale up subtly from 95% — noticeable as a beat, not as a bounce.
  - Rows must animate **once**. Progress events fire repeatedly per ticker and re-render the list; because rows are keyed by `ticker`, the DOM node is reused and the entrance must not replay on every update. If rows flicker as the percentage climbs, STOP and report — the key or the grouping has changed.
  - Click **Cancel** mid-run and start again; rows should re-enter cleanly.
  - In DevTools → Animations at 10% playback, confirm the badge never overshoots past 100%.
- **Done when**: rows enter once each, the completion badge pops subtly, and `git diff --stat` shows one file changed.
