# Animation improvement plans

Seven self-contained plans, written against commit `8fcd66a`. Each is executable by an agent with no context from the conversation that produced it: exact file paths, current-code excerpts, exact target values, hard boundaries, and a feel check.

**Context that produced these:** a read-only sweep of `frontend/src` found that this UI is already restrained and mostly right — press feedback, spinners, skeletons and reduced-motion handling all exist. The dominant finding was not missing design but a missing import: `tw-animate-css` is in `package.json` and never imported, so every enter/exit class in nine shadcn/Radix primitives compiles to nothing. Verified by building the app and grepping the emitted CSS.

## Plans

| # | Title | Severity | Files | Status |
| --- | --- | --- | --- | --- |
| [001](001-enable-tw-animate-and-motion-tokens.md) | Enable tw-animate-css and add motion easing tokens | HIGH | `src/index.css` | DONE |
| [002](002-animate-thought-process-disclosure.md) | Animate the thought-process disclosure open and closed | MEDIUM | `src/index.css`, `MessageList.tsx` | DONE |
| [003](003-tool-chip-entrance-and-status-transition.md) | Give tool chips an entrance and a colour transition | LOW | `MessageList.tsx` | DONE |
| [004](004-stagger-home-page-card-entrances.md) | Stagger the home page card entrances | LOW | `HomePage.tsx` | DONE |
| [005](005-animate-empty-chat-welcome-state.md) | Animate the empty-chat welcome state | LOW | `Chatbot.tsx` | DONE |
| [006](006-research-progress-row-and-completion.md) | Animate research progress rows and their completion | LOW | `MarketResearcherPage.tsx` | DONE |
| [007](007-reduced-motion-gentler-not-zero.md) | Make reduced motion gentler, not zero | LOW | `src/index.css`, `HomePage.tsx` | DONE |

All seven were applied on 2026-08-09 in the order below, touching five source files:
`src/index.css`, `src/components/containers/MessageList.tsx`, `src/pages/Chatbot.tsx`,
`src/pages/HomePage.tsx`, `src/pages/MarketResearcherPage.tsx`.

**Verified mechanically**: `npx tsc -b` clean; `npm run lint` shows 24 errors, all pre-existing
`no-explicit-any` / `react-refresh` / `no-control-regex` findings on lines none of these plans
touched — zero new; `npx vite build` succeeds and the emitted CSS now contains `animate-in`,
`fade-in`, `zoom-in-95`, `slide-in-from-*`, `accordion-down`, `.disclosure::details-content`,
`interpolate-size`, and `.ease-out{--tw-ease:var(--ease-out)}` resolving to
`cubic-bezier(.23,1,.32,1)`. The reduced-motion carve-out compiles after the universal rule with
the specificity needed to win.

**Not yet verified**: every feel check in the plans. They need the app running against a live
backend (`make run` from the repo root) and, for plan 002, a browser with `::details-content`
support. Nothing here has been watched in motion.

### Post-implementation review (2026-08-10)

A motion review of the applied diff found two regressions, both from the same mistake, both fixed:

- `HomePage.tsx:491` and `Chatbot.tsx:504` used `duration-*` for an entrance on elements that
  already carried a transition. `duration-*` compiles to `--tw-duration` **and**
  `transition-duration`, so it stretched the news card's hover lift to 300ms and the suggestion
  buttons' `active:scale-[0.98]` press feedback to 200ms (budget: 100–160ms). Both now use
  `animation-duration-*`, which sets only `animation-duration`. Plans 003, 004, 005 and 007 have
  been corrected so the pattern is not repeated; plan 003 carries the warning.

Checked and found correct, contrary to expectation: Tailwind v4 already wraps every `hover:`
variant in `@media (hover:hover)`, so hover motion is gated against touch without extra work; the
`motion-reduce:hover:translate-y-0` override compiles after `hover:-translate-y-0.5` and wins the
cascade; and `--default-transition-timing-function` stays `cubic-bezier(.4,0,.2,1)`, so overriding
`--ease-out`/`--ease-in-out` affects only explicit `ease-*` classes, not every transition in the app.

Accepted with reason rather than fixed: `.disclosure::details-content` animates `block-size`, a
layout property. There is no transform equivalent for a disclosure that does not clip its content,
and the repo's own `animate-accordion-down` animates height the same way.

Open recommendation, not applied: the tool chip (`MessageList.tsx:235`) sits on the tens-per-day
frequency tier, where the bar is "remove or drastically reduce". A 200ms fade plus `zoom-in-95` on
every chip of every turn may be more than that tier warrants — decide by watching a real multi-tool
turn, then either drop `zoom-in-95` or cut to 150ms.

## Execution order used

**001 first, then stop and look at the app.** It is one import plus three easing tokens, and it restores intended motion across dialogs, sheets, popovers, dropdowns, context menus, selects, tooltips and the settings accordion. It is also the highest-leverage change here by a wide margin. Everything else is small polish on top of a product that will already feel different.

After 001 has landed and been looked at:

1. **002** — the only remaining case where the system makes content vanish under the user's eyes.
2. **003** — the core chat screen; deliberately near-imperceptible.
3. **004**, **005**, **006** — independent of each other, any order.
4. **007** — last, so the reduced-motion carve-out is written against the animations that actually exist by then.

## Dependencies

- **001 blocks 002–007.** Every later plan uses utilities (`animate-in`, `fade-in`, `zoom-in-95`, `slide-in-from-*`) or tokens (`--ease-out`) that do not exist until 001 lands.
- **004 must land before 007.** Both edit `src/pages/HomePage.tsx:486`; 007's step assumes 004's classes are already on that line.
- 003, 005 and 006 touch one file each and share no files with anything else.

## Deliberately not planned

These were considered during the sweep and rejected. Recorded here so they are not "discovered" and fixed later by someone reading only the code:

- **Smooth-scrolling the chat transcript while streaming** (`Chatbot.tsx:105-113`) — fires on every token; a smooth behaviour would lag behind the text and fight itself.
- **Entrance on message bubbles** (`MessageList.tsx:304`) — tens of times a day, and the user's own bubble echoes text they typed a keystroke ago. Nothing to bridge.
- **Flash-on-change for updated prices** (`WatchList.tsx:201`) — refresh is manual and repaints the whole board; flashing every cell is noise, and it is functional data being read.
- **View Transitions on route changes** (`Navbar.tsx:28`) — core navigation, 100+ times a day. Never animate.
- **Crossfading the send ↔ stop ↔ dictate swap** (`ChatInput.tsx:185-232`) — the highest-frequency control in the product; motion there puts perceived latency in front of send.
- **Hold-to-confirm on document delete** (`DocumentListTable.tsx:110`) — already gated by a confirmation dialog; a 2s hold on top of that is redundant friction.

## Open finding with no plan written

`src/components/containers/Header.tsx:138` applies `animate-fade-right` to the conversation title — a 1.5s `clip-path` reveal (`src/index.css:319-333`) that plays every time a saved conversation is opened. That is five times the upper duration budget for UI, on an element seen many times a day, and it is an existing animation rather than a missing one, so it falls outside what was asked for here. Say the word and it becomes plan 008 (the fix is either deleting it or cutting it to a 200ms opacity fade).
