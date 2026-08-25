# 001 — Enable tw-animate-css and add motion easing tokens

- **Status**: DONE — applied 2026-08-09; mechanical verification passed, browser feel check still owed
- **Commit**: 8fcd66a
- **Severity**: HIGH
- **Category**: Cohesion & tokens / Missed opportunities
- **Estimated scope**: 1 file (`src/index.css`), ~6 lines added

## Problem

`tw-animate-css` is installed (`package.json:54`) but **never imported**. `src/index.css` starts with only:

```css
/* src/index.css:1 — current */
@import "tailwindcss";
```

Every enter/exit animation class in the shadcn/Radix primitives therefore compiles to nothing. Verified by building the app and grepping the emitted CSS: `animate-pulse` and `animate-spin` are present; `animate-in`, `fade-in-0`, `zoom-in-95`, `slide-in-from-*` and `accordion-down` are **absent**.

Dead classes, all currently no-ops:

```tsx
/* src/components/ui/dialog.tsx:61 — current, none of these animate */
"bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] ... duration-200 ..."
```

```tsx
/* src/components/ui/accordion.tsx:56 — current, no height animation */
className="data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down overflow-hidden text-sm"
```

Same dead markup at: `src/components/ui/sheet.tsx:39,61`, `src/components/ui/popover.tsx:31`, `src/components/ui/dropdown-menu.tsx:43,231`, `src/components/ui/context-menu.tsx:86,103`, `src/components/ui/select.tsx:63`, `src/components/ui/tooltip.tsx:47`, `src/pages/MarketResearcherPage.tsx:294`.

Consequence: every dialog, sheet, popover, dropdown, context menu, select and tooltip in the product appears and disappears **instantly**, with no scale-from-trigger and no fade — the exact "teleporting state" the markup was written to prevent. The Settings page accordion (`src/pages/SettingPage.tsx:164`) snaps open with no height transition.

Secondarily, the repo has **no motion vocabulary**: `src/index.css` defines colour, radius, shadow and font tokens but not a single easing or duration token, so any future motion work has nothing to reference.

## Target

```css
/* src/index.css:1-2 — target */
@import "tailwindcss";
@import "tw-animate-css";
```

Plus a motion token block, added immediately after the existing `@theme inline { … }` block closes at `src/index.css:237`:

```css
/* target — new block after line 237 */
/* Motion vocabulary. The built-in Tailwind curves are too weak to read as
   deliberate; these are the three curves the whole app animates on. Names
   match the Tailwind `ease-*` namespace, so `ease-out` resolves to the strong
   curve everywhere rather than cubic-bezier(0, 0, 0.2, 1). */
@theme {
  --ease-out: cubic-bezier(0.23, 1, 0.32, 1);
  --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1);
  --ease-drawer: cubic-bezier(0.32, 0.72, 0, 1);
}
```

After this change these utilities exist and resolve: `animate-in`, `animate-out`, `fade-in`, `fade-in-0`, `fade-out-0`, `zoom-in-95`, `zoom-out-95`, `slide-in-from-{top,bottom,left,right}-N`, `animate-accordion-down`, `animate-accordion-up`, `fill-mode-backwards`, `ease-out`, `ease-in-out`, `ease-drawer`.

**Deliberate override, not an accident:** defining `--ease-out` and `--ease-in-out` in `@theme` replaces Tailwind's built-in curves for those two utility names. Exactly one existing call site uses either: `src/components/ui/sheet.tsx:61` (`ease-in-out` on the mobile sidebar sheet), which is a drawer-style movement and is improved by the stronger curve. No file in `src/` uses `ease-out` as a class today, so nothing else changes.

**Known interaction:** `tw-animate-css` redefines the `delay-*` utility to mean `animation-delay` instead of `transition-delay`. No file in `src/` uses `delay-*` (verified by grep), so nothing breaks. Later plans use inline `style={{ animationDelay: … }}` rather than the utility, because stagger indices are computed at runtime and Tailwind cannot see dynamic class strings.

## Repo conventions to follow

- All design tokens live in `src/index.css`. Colour/radius/shadow tokens sit in `:root` and are exposed to Tailwind through `@theme inline` (`src/index.css:167-237`) — imitate that placement, but use a plain `@theme` block (not `inline`) for the easing curves, since they are literal values rather than aliases of `:root` variables.
- Every token block in this file carries a short comment explaining *why*, e.g. `src/index.css:61-64` and `src/index.css:3-6`. Match that density — one comment above the block, not one per line.
- Exemplar to imitate for block placement and comment style: `src/index.css:167` (`@theme inline`).

## Steps

1. Open `src/index.css`. Insert `@import "tw-animate-css";` as a new line 2, directly under `@import "tailwindcss";`. It must stay above every other rule in the file — CSS requires `@import` first.
2. In the same file, find the closing `}` of the `@theme inline { … }` block (currently line 237, immediately before the blank line and `@layer base {`). After that closing brace, insert a blank line and then the `@theme { … }` block exactly as written in the **Target** section above, including its comment.
3. Change nothing else in this file. In particular, do not touch the `@media (prefers-reduced-motion: reduce)` block at `src/index.css:360-370` — that is plan 007's scope.

## Boundaries

- Do NOT edit any file other than `src/index.css`.
- Do NOT remove, rename or "clean up" the existing animation classes in `src/components/ui/*.tsx` — they are correct and this plan is what makes them work.
- Do NOT add new dependencies; `tw-animate-css` is already in `package.json`.
- Do NOT change the `--radius`, colour, shadow or font tokens.
- If `src/index.css:1` is not `@import "tailwindcss";`, or the `@theme inline` block does not end at line 237, the file has drifted since commit `8fcd66a` — STOP and report instead of improvising.

## Verification

- **Mechanical**: `npx tsc -b` → no errors. Then `npx vite build` → succeeds, and `rg -c "animate-in|zoom-in-95|accordion-down" dist/assets/*.css` now returns a non-zero count (it returns nothing before this change). Delete the `dist/` directory afterwards; it is build output, not a deliverable.
- **Feel check**: `npm run dev`, then:
  - Go to `/documents` and click **Upload**. The dialog should fade in and scale up from 95%, not appear instantly. Press Escape — it should fade and scale back out, not vanish.
  - Open the theme dropdown in the header (`MainLayout.tsx:56`). The menu should scale in *from the corner nearest the trigger*, not from its own centre — that is `origin-(--radix-dropdown-menu-content-transform-origin)` finally taking effect.
  - Go to `/settings` and expand an agent flow. The panel should slide its height open over ~200ms instead of snapping.
  - In DevTools → Animations, set playback speed to 10% and reopen the dialog: confirm opacity and scale move together and nothing overshoots or bounces.
  - In DevTools → Rendering, enable `prefers-reduced-motion: reduce` and reopen the dialog. It will currently snap to instant — that is the existing global rule at `src/index.css:360` and is expected until plan 007 lands.
- **Done when**: the compiled CSS contains the `animate-in` utility, dialogs/menus/tooltips visibly animate in and out, and `git diff --stat` shows exactly one file changed with roughly 7 insertions.
