# 005 — Animate the empty-chat welcome state

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed
- **Depends on**: 001 (needs `animate-in`, `fade-in`, `slide-in-from-bottom-1` and `--ease-out`)
- **Commit**: 8fcd66a
- **Severity**: LOW
- **Category**: Missed opportunities (rare, first-run moment with no delight budget spent)
- **Estimated scope**: 1 file, ~4 lines

## Problem

The empty state of the assistant is the first thing a new user sees in the product, and it renders completely flat:

```tsx
/* src/pages/Chatbot.tsx:479-506 — current */
<div className="mt-8 flex flex-1 flex-col items-center justify-center gap-5 text-center">
  <LogoTile className="size-20 drop-shadow-sm" />
  <div className="max-w-md space-y-2">
    <h2 className="text-xl font-bold tracking-tight text-foreground">Welcome to StockMind</h2>
    <p className="text-sm leading-relaxed text-muted-foreground">…</p>
  </div>

  <div className="mt-2 grid w-full max-w-xl grid-cols-1 gap-2 sm:grid-cols-2">
    {[
      "What is FPT stock price?",
      "Should I buy VNM?",
      "Explain P/E ratio",
      "get FPT stock price and report",
    ].map((suggestion, idx) => (
      <Button
        key={idx}
        variant="outline"
        onClick={() => onHandleSuggestion(suggestion)}
        className="h-auto min-h-11 justify-start whitespace-normal px-3.5 py-2.5 text-left text-sm font-normal"
      >
```

Logo, heading, copy and four suggestion chips all appear in the same frame as the app shell. This is the rare/first-time frequency tier — the one place the audit's frequency rule explicitly allows delight — and none of it is spent here.

## Target

The block fades and rises 4px over 200ms; the four suggestions then fade in one after another, 40ms apart, starting 100ms in.

```tsx
/* src/pages/Chatbot.tsx:479 — target */
<div className="mt-8 flex flex-1 flex-col items-center justify-center gap-5 text-center animate-in fade-in slide-in-from-bottom-1 duration-200 ease-out">
```

```tsx
/* src/pages/Chatbot.tsx:496-501 — target */
<Button
  key={idx}
  variant="outline"
  onClick={() => onHandleSuggestion(suggestion)}
  className="h-auto min-h-11 justify-start whitespace-normal px-3.5 py-2.5 text-left text-sm font-normal animate-in fade-in animation-duration-200 ease-out"
  style={{ animationDelay: `${100 + idx * 40}ms`, animationFillMode: "backwards" }}
>
```

Why these exact values:

- The parent gets a 4px rise (`slide-in-from-bottom-1` = `0.25rem`); the buttons get **fade only, no slide**. They are inside the animating parent, so a second translate would compound into an 8px drift and make the chips look detached from the block they sit in.
- `40ms` is the low end of the 30–80ms stagger band — four items at 40ms finish their cascade 120ms after it starts, so the group still reads as one gesture.
- The `100ms` head start lets the parent's own entrance get underway first, so the chips arrive into a block that is already there.
- `animationFillMode: "backwards"` keeps each button invisible during its delay instead of flashing at full opacity first.
- **`animation-duration-200`, not `duration-200`.** `Button` carries `transition-all active:scale-[0.98]` (`src/components/ui/button.tsx:10`), and `duration-*` sets `transition-duration` as well as the animation duration — it would stretch the press feedback to 200ms, past the 100–160ms budget. `animation-duration-*` (from `tw-animate-css`) touches only the animation. The wrapper `<div>` at line 479 has no transition, so plain `duration-200` is correct there.
- Inline `style` rather than a `delay-*` class because `idx` is computed at runtime and Tailwind cannot see dynamic class strings.

**The stagger is decorative and must not block interaction** — the buttons are real, mounted, clickable elements throughout their entrance. Do not gate rendering on a timer or add `pointer-events: none`.

## Repo conventions to follow

- `src/pages/Chatbot.tsx` carries dense "why" comments above non-obvious decisions (see `:355-359`, `:427-429`). Add one short comment above the suggestion `<Button>` explaining the fill-mode and the no-second-slide choice.
- Suggestion buttons use the shared `Button` primitive with `variant="outline"`; keep that — motion classes go on the existing `className`, not into `buttonVariants` (`src/components/ui/button.tsx:7`), which would animate every button in the app.
- Exemplar for an entrance already in this codebase: `src/pages/MarketResearcherPage.tsx:294`.

## Steps

1. In `src/pages/Chatbot.tsx:479`, append `animate-in fade-in slide-in-from-bottom-1 duration-200 ease-out` to the existing `className` of the empty-state wrapper `<div>`.
2. In `src/pages/Chatbot.tsx:500`, append `animate-in fade-in duration-200 ease-out` to the suggestion `<Button>`'s existing `className`, and add the `style` prop from the target above. Add the explanatory comment.
3. Change nothing else in the file.

## Boundaries

- Do NOT touch `src/components/ui/button.tsx` — motion added to `buttonVariants` would animate every button in the product.
- Do NOT animate the `MessageList` branch of this conditional (`src/pages/Chatbot.tsx:477`) — message-bubble entrances were considered and rejected: the user's own bubble echoes text they typed a keystroke ago, and it is a tens-of-times-a-day moment.
- Do NOT touch the auto-scroll effect (`:105-113`). Smooth-scrolling during streaming was considered and rejected — it fires on every token and would lag behind the text.
- Do NOT change `ChatInput`, `Header`, `SideBar` or the `Toaster`.
- Do NOT add a slide to the suggestion buttons.
- Do NOT add new dependencies.
- If the excerpts above do not match the file, it has drifted since commit `8fcd66a` — STOP and report.

## Verification

- **Mechanical**: `npx tsc -b` → no errors. `npm run lint` → no new warnings.
- **Feel check**: `npm run dev`, then navigate to `/c` (or click **New Chat** in the sidebar):
  - The welcome block should lift gently into place, with the four suggestion chips arriving in reading order just behind it.
  - No chip should flash at full opacity before its delay elapses.
  - Click a suggestion mid-entrance — it must send the message. The stagger must not block interaction.
  - Click **New Chat** repeatedly. The entrance should replay cleanly each time and never leave a chip stuck invisible.
  - Send a message, then navigate back to `/c`. The empty state should animate again; the transcript view should not.
  - In DevTools → Animations at 10% playback, confirm the chips fade only — no vertical drift relative to the block around them.
- **Done when**: the empty state cascades in on every new chat, nothing flashes, and `git diff --stat` shows one file changed.
