# 002 — Animate the thought-process disclosure open and closed

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed (incl. `::details-content` support)
- **Depends on**: 001 (needs `--ease-out` from the new `@theme` block)
- **Commit**: 8fcd66a
- **Severity**: MEDIUM
- **Category**: Missed opportunities (state change that teleports)
- **Estimated scope**: 2 files, ~12 lines

## Problem

The assistant's "Thought process" block is a native `<details>` with no transition:

```tsx
/* src/components/containers/MessageList.tsx:166-182 — current */
{thinkingBlocks.map((block, idx) => (
  <div key={`think-${idx}`} className="mb-2 w-full">
    <details open={block.is_open} className="glass group overflow-hidden rounded-lg">
      <summary className="flex cursor-pointer select-none items-center gap-2 px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring">
        <Brain className="size-3.5 shrink-0" aria-hidden="true" />
        <span>Thought process</span>
      </summary>
      <div className="border-t border-border p-3 text-xs leading-relaxed text-muted-foreground">
```

Two things make this worse than a normal disclosure the user chooses to open:

1. It opens **by itself** while reasoning streams in — `is_open: true` is set on the first `thinking` delta (`src/pages/Chatbot.tsx:229`).
2. It closes **by itself** when the turn ends — `src/pages/Chatbot.tsx:311-315` maps every thinking block to `{ ...m, is_open: false }` on the `done` event.

So at the end of every reasoning turn a multi-paragraph block of text disappears in one frame, with everything below it jumping up to fill the gap, while the user is still reading. That is the textbook "preventing a jarring change" case, and it is triggered by the system, not by a click.

## Target

A height transition on the disclosure content, 200ms on the strong ease-out curve, matching the 200ms the repo's own Accordion primitive uses (`animate-accordion-down` from `tw-animate-css` is `0.2s ease-out`).

New utility in `src/index.css`, inside the existing `@layer utilities` block:

```css
/* target — added inside @layer utilities in src/index.css */
/* A <details> that opens and closes over its own height instead of snapping.
   The thought-process block opens and closes on its own while a turn streams,
   so a whole paragraph would otherwise appear and vanish in one frame.
   Height is the one layout property worth animating here — it is what the
   repo's Accordion primitive animates too. Browsers without
   ::details-content support keep today's instant behaviour. */
.disclosure::details-content {
  block-size: 0;
  overflow: hidden;
  transition:
    block-size 200ms var(--ease-out),
    content-visibility 200ms allow-discrete;
}

.disclosure[open]::details-content {
  block-size: auto;
}
```

Plus one line in the `@layer base` block so `block-size: auto` is interpolatable:

```css
/* target — added inside @layer base in src/index.css, next to the html rule */
:root {
  interpolate-size: allow-keywords;
}
```

And one class added to the element:

```tsx
/* src/components/containers/MessageList.tsx:168 — target */
<details open={block.is_open} className="disclosure glass group overflow-hidden rounded-lg">
```

**Browser support is the known risk and is unverified in this plan.** `::details-content` transitions and `interpolate-size` are Chromium-first; Safari support is recent and Firefox may not have shipped it. This is written as progressive enhancement on purpose: where the pseudo-element is unsupported, the whole rule block is dropped by the parser and the disclosure behaves exactly as it does today. Nothing regresses. Confirm the animation in Chrome during the feel check; do not "fix" it if Firefox still snaps.

**Alternative considered and rejected:** replacing `<details>` with the repo's Radix `Accordion` (`src/components/ui/accordion.tsx`) would animate in every browser, but the block's open state is driven by streaming metadata (`block.is_open`), so it would need controlled `value`/`onValueChange` state plus an effect syncing from props — a behavioural rewrite of live streaming UI in exchange for broader support of a decorative transition. Not worth the regression risk. If broad support later becomes a requirement, that is the path.

## Repo conventions to follow

- Custom utility classes live in the `@layer utilities` block of `src/index.css` (currently `src/index.css:285-317`, holding `.num`, `.glass`, `.glass-raised`, `.glass-chrome`, `.surface-solid`). Add `.disclosure` there, after `.surface-solid`.
- Exemplar for comment style and placement: `.glass` at `src/index.css:291-297` — a short "why" comment above the rule, not per-declaration.
- The `@layer base` block is `src/index.css:239-283`; the `html { scroll-behavior: smooth; }` rule at `:244-246` is the exemplar for a bare root-level declaration.
- Class order in this codebase puts semantic/custom classes before Tailwind utilities (`"glass group overflow-hidden rounded-lg"`), so `disclosure` goes first.

## Steps

1. In `src/index.css`, inside `@layer base { … }`, add a `:root { interpolate-size: allow-keywords; }` rule directly after the existing `html { scroll-behavior: smooth; }` rule.
2. In `src/index.css`, inside `@layer utilities { … }`, after the `.surface-solid` rule (currently ends at line 316), add the `.disclosure::details-content` and `.disclosure[open]::details-content` rules exactly as written in the **Target** section, including the comment.
3. In `src/components/containers/MessageList.tsx:168`, add `disclosure` as the first class of the `<details>` element's `className`. Change nothing else on that line — `open={block.is_open}` stays as it is.

## Boundaries

- Do NOT change `src/pages/Chatbot.tsx`. The auto-open on first delta and auto-close on `done` are deliberate product behaviour; this plan animates them, it does not alter them.
- Do NOT convert `<details>` to Radix `Accordion`, and do NOT add React state to control the disclosure.
- Do NOT touch the `<summary>` element, its hover transition, or the markdown rendering inside.
- Do NOT add a `@supports` block or a JavaScript fallback — unsupported browsers keeping today's behaviour is the accepted outcome.
- Do NOT add new dependencies.
- If `src/components/containers/MessageList.tsx:168` does not read `<details open={block.is_open} className="glass group overflow-hidden rounded-lg">`, the file has drifted since commit `8fcd66a` — STOP and report.

## Verification

- **Mechanical**: `npx tsc -b` → no errors. `npm run lint` → no new warnings. `npx vite build` → succeeds.
- **Feel check**: `npm run dev` in Chrome, open `/c`, and send a question that produces reasoning (e.g. "Compare VIC and VHM fundamentals"):
  - When the thought-process block first opens mid-stream, its height should grow over ~200ms rather than the answer below it jumping down in one frame.
  - When the turn finishes and the block auto-closes, the content should collapse smoothly — this is the moment the plan exists for. Watch the message below it: it should slide up, not teleport.
  - Click the summary to reopen it manually; the same 200ms transition should play.
  - Toggle it open/closed rapidly. Because this is a CSS *transition* rather than keyframes, it must retarget from the current height — it must never jump to full height and restart.
  - In DevTools → Animations, set playback to 10% and confirm the height eases out (fast start, soft landing) rather than moving linearly.
- **Done when**: the block animates open and closed in Chrome, no console warnings appear, and `git diff --stat` shows exactly two files changed.
