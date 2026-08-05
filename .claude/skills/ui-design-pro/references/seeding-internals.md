# What `seed-options.py` actually does

Read this when a seed looks wrong and you need to know whether the script or the query
produced it — not on every run. The command itself, the `--brand` rule, and the
contrast gate are in SKILL.md → Seeding the Options.

## Building the option set

- Pulls ranked palettes, styles, and font pairings for the query, then **extends each
  pool with the rest of the CSV**. A narrow query legitimately matches one hue family,
  and three near-identical options are not a choice. Anything drawn from beyond the
  ranked matches is reported — read those lines and confirm the direction still suits
  the brief.
- Anchors the set: one **safe** (the database's own top match), one **bolder** (highest
  complexity), one **structurally different** (opposite corner family from safe, serif-
  or mono-led type). Remaining slots are filled by `--variance`.
- Enforces distinctness on three axes at once — no repeated style, no repeated font
  pairing, and ≥40° OKLCH hue separation between primaries. One axis alone still ships
  the same design twice.
- **With `--brand`,** `primary` and `ring` are the user's colour in *every* option, so
  the hue axis is spent. Distinctness moves to the surface strategy (pure white /
  off-white / warm paper / tinted / dark, one each), the type, the shape, and the
  **accent strategy** — tonal, analogous, complementary, counter-analogous, triadic.
  Each option names which relationship it uses; that is a real design decision, and it
  is what the user is choosing between.
- Caps **dark-first pages at one** per set. Every option already carries a derived dark
  mode, so a second dark page buys nothing and costs a light option someone might have
  wanted. Put "dark" in the query to lift the cap.

## Colour maths

- Maps `colors.csv` onto shadcn token names. Colour maths runs in OKLCH; **values are
  written as hex** — `--format oklch` if a Tailwind v4 project prefers that form.
- **Derives the dark set.** `colors.csv` has no dark values — 0 of 192 rows — so every
  dark theme is computed: background lifted to 0.145–0.215 lightness depending on the
  style's own character (never pure black), cards raised above it, primary lightened
  with chroma eased back, hues carried over from light. A pinned brand colour is **not**
  re-lightened wholesale — it moves only if it fails 3:1 against the dark page, and the
  move is reported.
- Nudges foreground lightness until every WCAG text pair clears 4.5:1 in **both** modes,
  and reports each adjustment. An option that cannot be fixed is dropped, not shipped
  failing. On a pinned brand fill the *text* adapts, never the brand.
- Derives `chart-1..5` from each option's own primary, so data viz belongs to the theme
  instead of arriving as a stock rainbow.
- Leaves `border` and `input` alone at their palette values. Those are advisory at 3:1
  — a quiet divider is a legitimate choice. `ring` is pushed up, because an invisible
  focus ring is an accessibility failure, not a style.

## Why the radius often needs a human

Only 6 of 84 style rows declare a radius, so most are inferred from the style's
keywords or stepped off a ladder. The script says which, per option. A data-dense
console with 1rem corners is wrong even if the ladder produced it.

## The surface kit

Each option also gets one of six surface treatments — `flat`, `outlined`, `elevated`,
`soft`, `glass`, `hard` — which is what makes the styles visibly different from each
other rather than differently coloured. It becomes the seven `--surface-*` variables
documented in `shadcn-tokens.md`.

How the kit is chosen, in order:

1. **The style's name**, for the styles whose whole identity is their surface —
   Glassmorphism, Claymorphism, Neumorphism, Brutalism, Memphis, Material, Bento. A
   name match wins outright, because several rows describe a technique they are
   contrasting themselves *against*: Neo Brutalism's own cell reads "hard offset
   shadows (4–8px, no blur)", and matching the word "blur" there classified it as glass.
2. **The technique cells** — `Effects & Animation`, `CSS/Technical Keywords`,
   `Design System Variables`. This is where the database actually records shadows,
   border weights and blur radii.
3. **A stepped ladder** (`flat` → `elevated` → `outlined`) when the row names no
   technique at all. 32 of 84 rows land here, and the script says so per option.

A row that names a heavy border *and* rules shadows out ("thick 4px borders, no
shadows, strictly 2D") gets `outlined` rather than `hard`. The database is being
precise there, so the shadow half of the kit is dropped instead of overridden.

Kit distinctness is a *bias*, not a guarantee: distinct style categories are not
automatically distinct surfaces — Minimalism, Flat Design and Swiss Style all draw a
hairline and no shadow — so an unused kit is preferred when filling each slot. If the
query only ranks flat styles, the set can still repeat one; that shows up in the tab
strip, where every tab prints its kit.

### Kits that change the palette

Three treatments are not paint on top of arbitrary colours, so the script adjusts the
tokens and reports each adjustment:

- **`glass`** makes `card` and `popover` translucent, and derives the page wash the blur
  samples. Contrast is then enforced against the *composited* surface, not the opaque
  one — otherwise the seeder's "passes by construction" promise would not survive the
  gate reading the same file.
- **`soft`** sets card = popover = background (soft UI extrudes the surface out of the
  page) and drops a pure-white page to 0.95 lightness, because a white surface has no
  headroom for the highlight above it. Real neumorphic themes sit on #E8E8E8 for the
  same reason.
- **`hard`** and **`outlined`** move `border` and `input` toward the foreground. A 3px
  border in a hairline colour is a thick smudge; weight and colour have to move together.

## The motion personality

Each option also carries one of five motion personalities — `still`, `calm`, `crisp`,
`springy`, `cinematic` — chosen the same way the surface kit is: spread across the set so
the user is picking motion at the same time as colour, not inheriting it.

- The `--motion` dial (inferred from the concept, overridable) sets the **centre**. The
  slot offsets `[0, +2, -2, +1, -1]` spread the set around it, clamped to 1-10, and each
  step off the dial is reported. Near an extreme the spread compresses and several
  options share a personality — that is deliberate. A clinical dashboard at dial 2
  should not be offered a cinematic direction just so the tab strip looks varied, and
  sliding the window upward would hand a tier-5 option to a concept that asked for 1.
- The tier maps to a personality: 1-2 `still`, 3-4 `calm`, 5-6 `crisp`, 7-8 `springy`,
  9-10 `cinematic`. Each is a complete feel — entrance distance, easing curve, stagger,
  and hover lift move together, because 700ms with a linear curve is sluggish while
  700ms with an ease-out is cinematic.
- **`--no-animation` overrides everything**: every option becomes `still` at tier 1, and
  the preview disables its motion controls rather than hiding them, so the user can see
  that the decision was theirs.
- Motion is **not a token**. It ships as transitions/GSAP in the implementation phase, so
  it appears in the option CSS only as a line in the provenance header — which is
  stripped when the file is ported, like the rest of that header.

Check each personality against its own option at step 8: `cinematic` on the brutalist
direction is a contradiction the ladder cannot see.

## The archetype

The preview renders one of four miniature products — `dashboard`, `landing`, `ecommerce`,
`editorial` — inferred from the query and overridable with `--archetype`. It is a property
of the whole set, not of an option: every option renders the same product so the
comparison stays fair.

The hint patterns are ordered most-specific first, so "e-commerce landing page" resolves
to `ecommerce` — the storefront is the harder thing to get right, and its landing page is
one screen of it. A query that names no product type falls back to `landing` and says so.

This matters more than its size suggests: a sidebar and a data table say "admin console"
far louder than a palette says anything. Preview a storefront as a dashboard and the user
judges the wrong screen.
