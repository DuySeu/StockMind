# 004 — Stagger the home page card entrances

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed (needs the API running)
- **Depends on**: 001 (needs `animate-in`, `fade-in`, `slide-in-from-bottom-2` and `--ease-out`)
- **Commit**: 8fcd66a
- **Severity**: LOW
- **Category**: Cohesion (missing stagger) / Missed opportunities
- **Estimated scope**: 1 file, ~8 lines

## Problem

Two sections of the landing page fetch on mount and then replace their placeholder with real content in a single frame.

**Watchlist** — four skeleton cards are swapped for four real cards at once:

```tsx
/* src/pages/HomePage.tsx:232-240 — current */
<div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
  {priceBoard.length > 0
    ? priceBoard.map((d, i) => <TickerCard key={d.listingInfo?.symbol ?? i} {...d} />)
    : Array.from({ length: 4 }).map((_, i) => (
        <Card key={i} className="h-[168px] animate-pulse gap-0 rounded-xl bg-muted/60 p-5" />
      ))}
</div>
```

**News** — a line of text is replaced by three cards:

```tsx
/* src/pages/HomePage.tsx:480-487 — current */
localNewsData.map((news: any, i: number) => (
  <a
    href={news.url || "#"}
    target="_blank"
    rel="noopener noreferrer"
    key={i}
    className="group block rounded-xl transition-transform hover:-translate-y-0.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
  >
```

This is the product's only marketing surface and the first screen most people see. It is also the one place in this app where the delight budget is allowed to be spent — it is seen rarely, not tens of times a day, and nothing on it is data the user is mid-task on. Right now the page's most alive moment (live prices arriving) reads as a repaint.

## Target

Each card fades and rises 8px into place on the strong ease-out curve over 300ms, staggered 60ms apart in DOM order.

**Watchlist** — `TickerCard` takes an `index` so it can offset its own delay:

```tsx
/* src/pages/HomePage.tsx:142 — target signature */
function TickerCard({ listingInfo, matchPrice, index = 0 }: PriceBoard & { index?: number }) {
```

```tsx
/* src/pages/HomePage.tsx:171 — target */
<Card
  className="gap-0 rounded-xl p-5 animate-in fade-in slide-in-from-bottom-2 duration-300 ease-out"
  style={{ animationDelay: `${index * 60}ms`, animationFillMode: "backwards" }}
>
```

```tsx
/* src/pages/HomePage.tsx:234 — target call site */
? priceBoard.map((d, i) => <TickerCard key={d.listingInfo?.symbol ?? i} index={i} {...d} />)
```

**News** — same values on the existing anchor:

```tsx
/* src/pages/HomePage.tsx:481-487 — target */
<a
  href={news.url || "#"}
  target="_blank"
  rel="noopener noreferrer"
  key={i}
  className="group block rounded-xl transition-transform hover:-translate-y-0.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring animate-in fade-in slide-in-from-bottom-2 animation-duration-300 ease-out"
  style={{ animationDelay: `${i * 60}ms`, animationFillMode: "backwards" }}
>
```

Why these exact values:

- `slide-in-from-bottom-2` is `0.5rem` (8px) in this theme's spacing scale — enough to read as arriving, small enough not to look like a page transition.
- `60ms` stagger sits mid-range of the 30–80ms band; with 4 cards the last one starts 180ms in, so the whole group settles inside half a second.
- `animationFillMode: "backwards"` is required. Without it a card is painted at full opacity during its delay and then jumps to `opacity: 0` when the animation starts.
- Inline `style` rather than a `delay-*` class because the index is computed at runtime and Tailwind cannot see dynamic class strings. Inline declarations override the `animation` shorthand set by `animate-in`, which is what makes this work.
- **The two cards take different duration utilities on purpose.** `Card` (`src/components/ui/card.tsx:10`) has no transition, so plain `duration-300` is safe there. The news card is an `<a>` carrying `transition-transform hover:-translate-y-0.5`, and `duration-*` sets `transition-duration` as well as the animation duration — using it there would stretch the hover lift from 150ms to 300ms. `animation-duration-300` sets only `animation-duration`.

**The stagger is decorative and must never block interaction.** These are CSS animations on already-mounted, already-clickable elements — a card is hittable during its own entrance. Do not add `pointer-events: none` or gate rendering on a timer.

## Repo conventions to follow

- `src/pages/HomePage.tsx` is organised as `/* ─── Data ─── */`, `/* ─── Sub-Components ─── */`, `/* ─── Main Page ─── */` and every non-obvious decision carries a short comment explaining the reasoning (see `:235-236`, `:528-529`). Add one comment above the `TickerCard` `<Card>` explaining the fill-mode, in that voice.
- Props are destructured inline in the function signature with types on the parameter (`function TickerCard({ listingInfo, matchPrice }: PriceBoard)`) — extend that pattern rather than introducing an interface.
- Exemplar of an entrance already written in this codebase: `src/pages/MarketResearcherPage.tsx:294`.

## Steps

1. In `src/pages/HomePage.tsx:142`, change the `TickerCard` signature to `function TickerCard({ listingInfo, matchPrice, index = 0 }: PriceBoard & { index?: number }) {`.
2. In `src/pages/HomePage.tsx:171`, replace the `<Card className="gap-0 rounded-xl p-5">` opening tag with the multi-line target above, adding a one-line comment explaining why `animationFillMode` is `backwards`.
3. In `src/pages/HomePage.tsx:234`, add `index={i}` to the `<TickerCard …>` call, before the `{...d}` spread.
4. In `src/pages/HomePage.tsx:486`, append `animate-in fade-in slide-in-from-bottom-2 animation-duration-300 ease-out` to the anchor's existing `className` — `animation-duration-300`, **not** `duration-300` — and add the `style` prop from the target above.

## Boundaries

- Do NOT animate the skeleton placeholders (`src/pages/HomePage.tsx:237-239`) — they already pulse, and animating their exit would double-expose against the real cards.
- Do NOT touch `HeroSection` (`:88-140`), `CapabilitiesSection` (`:245-271`), `StockAnalysisSection` (`:273-401`), `PricingSection` (`:520-568`) or `Footer` (`:570-618`). Static content that is present on first paint has nothing to bridge; scroll-triggered reveals are explicitly out of scope.
- Do NOT add an entrance to the `<Progress>` bar inside `TickerCard` (`:197`) — it is a value the user reads.
- Do NOT change the existing `hover:-translate-y-0.5` on the news card, and do NOT add new hover motion anywhere.
- Do NOT change any data fetching, formatting or classification logic.
- Do NOT add new dependencies.
- If the line numbers above do not match the excerpts, the file has drifted since commit `8fcd66a` — STOP and report.

## Verification

- **Mechanical**: `npx tsc -b` → no errors (the `index?: number` prop must typecheck against the `PriceBoard` spread). `npm run lint` → no new warnings.
- **Feel check**: `npm run dev`, hard-reload `/` with the network throttled to "Fast 3G" so the placeholder state is visible:
  - The four watchlist cards should arrive one after another, roughly 60ms apart, rising slightly as they fade in — not all at once, and not so slowly it reads as a queue.
  - The three news cards should do the same when their fetch lands.
  - No card should ever flash at full opacity and then disappear before animating — if that happens, `animationFillMode: "backwards"` is missing or misspelled.
  - Click a news card while it is still animating; the link must open. The stagger must not block interaction.
  - In DevTools → Animations at 10% playback, confirm the movement decelerates hard into place (strong ease-out) rather than gliding linearly.
- **Done when**: both grids stagger in on a cold load, nothing flashes, and `git diff --stat` shows one file changed.
