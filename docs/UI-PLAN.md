# UI Implementation Plan — StockMind

Status: **partially implemented** · written and approved 2026-07-30

Implementation was narrowed on request to **tokens, fonts, the shell, HomePage and
Chatbot** (items 1, 2, 3, 8-in-part). Items 4–7 are unstarted and still valid as written.
What shipped, and why, is recorded in [`DECISIONS.md`](./DECISIONS.md).

## Direction

**Option E — "Đêm"** · Dark first and unapologetic: luminous indigo over near-black, translucent panels that layer over the price board instead of boxing it in.

- Tokens: `docs/design/option-tokens-round2/E-dem.css` (light + dark, both gate-passed)
- Surface kit: **glass** — 1px border, 14px backdrop blur, lit top edge (`--surface-shadow-inset`), diffuse shadow, indigo page wash. Ported as the seven `--surface-*` vars; components read those instead of inventing shadows
- Brand colour: none supplied. This direction proposes `#9B93F5` (dark) / `#4338CA` (light). Negotiable, but changing it means re-running both contrast gates
- Accent strategy: the accent is a *cooled step of the same indigo* (`#2C2C52` dark), used for hover and selection. No second hue enters the palette — the only other colours in the app belong to the price board
- Round-1 and round-2 rejected directions stay on record in `ui-options.html` and `ui-options-round2.html`

**Why the primary is indigo:** the Vietnamese board already owns violet (`trần`), green (`tăng`), yellow (`tham chiếu`), red (`giảm`) and cyan (`sàn`). Indigo is the nearest thing to "brand blue" that does not collide with a price state.

## Stack

| Layer | Choice | Version |
| --- | --- | --- |
| FE framework | React + Vite | 19.1 / Vite 7.1 |
| UI framework | shadcn/ui `new-york`, `cssVariables: true` | copy-in source, 23 primitives in `src/components/ui/` |
| Styling engine | Tailwind | **v4** (`@import "tailwindcss"`, `@tailwindcss/vite`) |

Existing project — nothing is scaffolded and nothing is installed. Majors read from `package.json`, not memory. Token *names* are frozen: all 23 primitives depend on them, so only values change.

## Tokens & fonts

- Token file: `src/index.css` — both modes, ported from `E-dem.css` (colours **and** `--surface-*`), **comments stripped** (the file currently carries block comments; their reasoning moves to `DECISIONS.md`)
- The seven `--surface-*` vars stay as plain custom properties, consumed by four utilities (`.glass`, `.glass-raised`, `.glass-chrome`, `.surface-solid`) plus a `data-slot` rule in `@layer base`; `@theme inline` gains only `--color-card-solid`. The existing `--color-price-*` and shadow mappings are unchanged. *(Amended during implementation: mapping blur/shadow into `@theme` would have generated utilities nothing uses.)*
- Default colour mode: **follows the OS**, manual toggle — the existing `prefers-color-scheme` + `localStorage` code in `SideBar.tsx` is not touched
- Display + body font: **Manrope** → `--font-sans`, replacing Be Vietnam Pro
- Mono: **JetBrains Mono** → `--font-mono`, replacing IBM Plex Mono (carries all prices and figures)
- Both verified to ship the Google Fonts `vietnamese` subset. Loaded via the existing `<link>` in `index.html` — see Risks

## Scope — 8 items

| # | Item | File(s) | Contains |
| --- | --- | --- | --- |
| 1 | Token file | `src/index.css` | E's light + dark, `--surface-*`, `@theme` mapping, comment-free |
| 2 | Fonts | `index.html` | Manrope + JetBrains Mono swap |
| 3 | Glass on the shell | `layout/MainLayout.tsx`, `layout/Navbar.tsx`, `containers/SideBar.tsx`, `containers/Header.tsx` | translucency, blur, lit edge, page wash |
| 4 | Token-drift fixes | `pages/MarketResearcherPage.tsx` (8), `pages/SettingPage.tsx` (4), `components/StatusBadge.tsx` (3) | kill 15 raw palette classes that ignore the toggle |
| 5 | Research flow | `pages/MarketResearcherPage.tsx` (343 LOC), `pages/ResearchResultPage.tsx`, `containers/ResearchReport.tsx` | layout + density, not just recolour |
| 6 | Documents | `pages/DocumentPage.tsx`, `components/DocumentListTable.tsx`, `components/DocumentUploadForm.tsx`, `components/StatusBadge.tsx` | table density, upload affordance, status states |
| 7 | Settings + auth + small pages | `pages/SettingPage.tsx`, `pages/LoginPage.tsx`, `pages/ErrorPage.tsx`, `pages/PendingPage.tsx` | the least-finished screens in the app |
| 8 | Re-check the 8 already-restyled screens | `HomePage`, `WatchList`, `Chatbot`, `MessageList`, `Header`, `SideBar`, `Navbar`, `MainLayout` | they inherit tokens for free; verify glass reads right and the rail still works |

**One craft decision I am making rather than asking:** glass goes on chrome, panels and popovers — **not on the price board**. Blur behind five hue-coded columns of small tabular figures costs legibility, and the board is the one thing in this app that must never be hard to read. The board card stays opaque.

## Out of scope this pass

- **All performance work**, per your call: the 744 kB `stockmind.png`, the 1.24 MB unsplit bundle. Re-noted in `DECISIONS.md` as inherited, untouched
- Any logic, state, data fetching, routing, handler, API or `lib/` change — including `lib/stock.ts`, which the previous pass rewrote. `getPriceState` / `PRICE_STATE` are used as they are
- Changing the colour-mode default; migrating to the installed-but-unused `next-themes`
- Auth (login stays non-functional), the hardcoded sidebar user, the commented-out WebSocket route
- GSAP or any new dependency. Motion dial is 5, so motion is CSS transitions only, inside the existing `prefers-reduced-motion` block

## Steps

1. `src/index.css` — E's tokens, both modes, `--surface-*`, `@theme` mapping, comments stripped
2. `index.html` — swap the font link, map to `--font-sans` / `--font-mono`
3. Shell glass (item 3), verified in both modes
4. Drift fixes (item 4) — after this, `grep` for raw palette classes returns nothing outside `ui/`
5. Research flow → documents → settings/auth/small pages (items 5 → 6 → 7), each finished in both modes before the next
6. Re-check the 8 inherited screens (item 8)

## Verification

```bash
node /Users/duyseu/PersonalPaper/StockMind/.claude/skills/ui-design-pro/scripts/contrast-check.mjs src/index.css
npm run build
npm run lint     # expect exactly 30 pre-existing no-explicit-any errors, none new
```

Plus the price-token pass the official gate cannot do (15 pairs × 2 modes: each board colour on its own fill, on `--card`, on `--background`), a grep for `#`/`rgb(` in components and `/*` in the token file, and a check that alpha-composited text (`text-*/NN`) is not used on text. Then `docs/design/DECISIONS.md`, updated rather than replaced.

## Risks & open questions

- **Manrope is not purpose-designed for Vietnamese.** It ships the `vietnamese` subset, but Be Vietnam Pro was chosen last round specifically because its stacked diacritics sit clear of the line above — a real concern at 12–13px on a dense board. My plan: ship Manrope as E specifies, and if diacritics crowd at board sizes, keep Manrope for headings and return Be Vietnam Pro to `--font-sans` for body. Flagging rather than silently deciding.
- **Nothing can be visually verified this session.** The Chrome extension was declined, so as in the previous pass, verification is contrast maths, build output and grep only. Glass and blur are exactly the sort of thing that needs eyes — a look at both modes before shipping is on you.
- **Light mode is secondary for a dark-first direction.** It is fully built and gate-passed, but E is designed to be seen dark; light mode will read as the alternate rather than the intent.
- `backdrop-filter` behind a large blurred surface can cost frames on low-end Android. If the board area feels sluggish, blur comes off the largest surfaces first — the kit degrades to `elevated` cleanly.
