---
name: codebase-onboarding
description: Onboard fast into an unfamiliar codebase by reading the source instead of the docs — repo orientation maps, end-to-end execution traces, and "where do I start" answers where every claim carries a file:line that was actually opened. Use this whenever someone is new to a repo or facing code they haven't read: "explain this codebase", "walk me through how X works here", "where is Y handled", "what owns this behaviour", "I just cloned this, where do I start", "give me the architecture", "trace the login flow", or when inheriting a repo and needing an accurate map before touching anything. Prefer it over ad-hoc grepping whenever the goal is understanding rather than changing code, including when the user never says the words "onboarding" or "architecture".
---

# Codebase Onboarding

## What this is for

A new engineer's first week is spent building a mental model, and a wrong model is worse
than no model — it sends them to confidently edit the wrong file. Two habits produce wrong
models:

- **Reading the docs instead of the code.** A README describes the repo its author intended.
  The code is the repo that exists. The gap between them is exactly what burns newcomers.
- **Framework autopilot.** Knowing how Django, Next.js, or Spring *usually* wires things is a
  search hint, never a finding. Every codebase has the place it deviates, and that place is
  where the new engineer will lose a day.

So the whole discipline reduces to one habit: **every factual claim traces to a line you
opened.** Everything else is labelled or cut.

This is read-only work. Describe the system; don't review it, don't improve it. Why that
separation matters is in [Staying in your lane](#staying-in-your-lane).

## The anchor rule

Write anchors as `` `path/to/file.ts:42` `` — a repo-relative path and, where a specific
line matters, the line number. Paths are always full and repo-relative: `auth.ts` is useless
in a repo with four of them.

Say which tier of evidence each claim sits on. The tiers exist so uncertainty has somewhere
to go other than silence:

| Tier | What it means | How to phrase it |
|---|---|---|
| **Read** | You opened the file and read that code | State it plainly: "Requests are authenticated in `middleware/auth.ts:18`." |
| **Wired** | You saw the import, route entry, or registration — not the implementation | Say both halves: "Registered at `app.ts:44`; the handler in `handlers/user.ts` was not read." |
| **Not inspected** | You know it exists and skipped it | Name it in the gaps list: "`workers/` — not inspected." |

Never let a Wired or Not-inspected claim wear the voice of a Read one. "Auth happens in
`middleware/auth.ts`" sends the reader to the right file when you read it, and into a
re-export stub when you only saw the filename in a directory listing.

**Docs are leads, not evidence.** README, CLAUDE.md, ADRs, wiki pages, docstrings, and code
comments are hypotheses to confirm in code. When a doc claim is worth repeating but you
couldn't confirm it, attribute it: "`README.md:12` says the cache is Redis-backed; I found no
Redis client in `src/`."

## Pick a mode

The question shape tells you which of three jobs is being asked for. They compose — a full
orientation normally ends with one trace, because a map nobody has walked is a guess.

| The question sounds like | Mode | You deliver |
|---|---|---|
| "What is this repo?" / "explain this codebase" / "where do I start?" | **Orientation** | The full three-level map |
| "How does login work?" / "what happens when a job is queued?" | **Trace** | One path, every hop anchored |
| "Where is rate limiting handled?" / "which file do I change for X?" | **Locate** | The owning file, the evidence, the neighbours |

Locate is the cheapest and most common. Don't answer a Locate question with a full
orientation map — it buries the one line the reader needed.

## Method

### Step 1 — Inventory (wide and cheap)

List the top level, then read the **manifests**, not the README: `package.json`,
`pyproject.toml`, `go.mod`, `Cargo.toml`, `pom.xml`, `Gemfile`, `*.csproj`. They state the
project name, its declared entry points, its runnable scripts, and — through dependencies —
which framework you're actually dealing with. Lockfiles tell you the ecosystem is real and
resolved; don't read them.

Classify what you're in: application, service, library, CLI, monorepo, or a mix. Check for
workspace markers early (`pnpm-workspace.yaml`, `turbo.json`, `nx.json`, `go.work`,
Cargo `[workspace]`, `lerna.json`, Bazel) — in a monorepo, "where does it start" has one
answer per package, and answering it for the wrong package wastes the reader's day.

Skip vendored, generated, and build output (`node_modules/`, `vendor/`, `dist/`, `build/`,
`target/`, `__pycache__/`, `*.pb.go`, `*_generated.*`, snapshots). Note in your report that
you skipped them, so "not inspected" stays honest.

`references/entry-points.md` has the per-ecosystem marker tables — read it when the stack
isn't one you can navigate from memory, or when the manifest points somewhere unfamiliar.

### Step 2 — Find the entry points

Find the smallest set of files that define how the system starts. That is usually 1–5 files:
a `main`, a server bootstrap, a route table, a CLI command registry, a package's public
exports.

For each one, record three things: **what triggers it**, **what it wires up**, and **where
it hands off**. Those three answers are what make an entry point useful to a newcomer —
"`cmd/api/main.go` is the entry point" alone tells them nothing they couldn't get from `ls`.

### Step 3 — Trace one path end to end

Pick the path the reader asked about, or — in Orientation mode — the most representative one
(usually the primary request path). Follow it the whole way:

```
entry → routing/dispatch → validation → orchestration → core logic → persistence / external I/O → response
```

Not every system has every stage; name the ones it has. At each hop record the anchor of the
**call** and the anchor of the **definition** — those are different files, and the reader
needs both to navigate in either direction.

Stopping at a layer boundary ("...then it calls the service layer") is the most common way a
trace becomes useless. Open the service.

When you hit indirection that can't be resolved by reading — a DI container, an event bus, a
decorator registry, dynamic dispatch — name the registration site, say the binding is
resolved at runtime, and give the candidate implementations you found by search. Guessing
which one runs is exactly the failure this skill exists to prevent.
`references/tracing.md` has recipes per path kind and per indirection pattern.

### Step 4 — Map boundaries and ownership

With one path walked, the layout means something. Record:

- **Layers and seams** — where presentation, application/domain, and persistence/external I/O
  actually live in *this* repo's directory names.
- **Public interface vs. internals** — what other packages/services import, versus what is
  implementation detail. Exports, route tables, and published clients mark the line.
- **Cross-cutting concerns** — auth, config, logging, error handling, background jobs. These
  are where newcomers get surprised, because they apply without appearing in the call chain.
- **Things that look important but aren't** — dead code, duplicate abstractions, a `manager`
  that is really the service layer, two implementations behind a feature flag. Report these
  as observations with anchors, not as problems to fix.

### Step 5 — Verify, then report

Before delivering, check your anchors. Hallucinated paths are the one error that destroys the
whole document's credibility, and they're mechanically detectable:

```bash
# scripts/check_citations.py sits next to this SKILL.md
python3 <this-skill-dir>/scripts/check_citations.py draft.md --root /path/to/repo
```

It extracts every backticked `` `path` `` / `` `path:line` `` outside code fences and reports
the ones that don't resolve — including line numbers past the end of a file. Write the draft
to a scratch file, run it, fix what it flags, then deliver.

For a short answer with a handful of anchors, re-reading them is enough. Run the script
whenever you're writing a file or citing more than about five anchors — that's where memory
starts substituting plausible paths for real ones.

Then state the gaps: what you inspected, what you skipped, what you couldn't resolve. A map
with an honest edge is usable. A map that pretends to be complete is not.

## How much to read

Budget by role, not by curiosity. For a first orientation of an unfamiliar repo, opening
roughly 15–40 files is the right order of magnitude:

- **Read fully** — entry points, route/command tables, every file on the traced path, and
  config that changes behaviour (not config that sets ports).
- **Skim for shape** — types and schemas (they document the domain faster than prose), and
  directory listings of everything you're not opening.
- **Grep, don't read** — tests. A test that asserts behaviour is executable documentation and
  is often the fastest confirmation of what a module does. Search them for a symbol's name
  rather than reading test files end to end.
- **Skip** — lockfiles, generated code, vendored dependencies, snapshots, migration history
  beyond the current schema shape.

When the repo is bigger than the budget, **narrow the claim, not the rigour**. Cover one
subsystem properly and say that's what you covered. A precise map of the auth service beats a
vague map of everything, and the reader can ask for the next subsystem.

## Output

Lead with the shortest useful answer and let the reader stop when they have what they need —
they will stop at different depths, and the one who needed only the first line shouldn't have
to read to the end to find it.

Deliver in the conversation by default. Write a file only when asked for something durable;
then use `docs/onboarding/<topic>.md` and keep the same structure.

### Orientation

```markdown
# Codebase Orientation: [repo name]

## In one line
[What this codebase is — type, runtime, and what it does.]

## The five-minute version
- **What it does in code**: [the actual work the code performs]
- **Inputs**: [HTTP requests, CLI args, queue messages, files, function calls] — `path:line`
- **Outputs**: [responses, DB writes, emitted events, files, rendered UI] — `path:line`
- **Read these three files first**: `path` — [why], `path` — [why], `path` — [why]
- **The main path**: `entry` → `dispatch` → `core logic` → `output`

## Deep dive

### Type and runtime
[application / service / library / CLI / monorepo], [languages and runtimes present]

### Entry points
| File | Triggered by | Wires up | Hands off to |
|---|---|---|---|
| `path:line` | [HTTP :8080 / `npm start` / cron] | [routes, DI, middleware] | `path` |

### Top-level structure
| Path | Contains | Notes |
|---|---|---|
| `src/` | [what's actually in it] | [evidence or caveat] |

### Boundaries
- **Presentation / API surface**: [files]
- **Application / domain logic**: [files]
- **Persistence and external I/O**: [files]
- **Cross-cutting**: auth `path`, config `path`, logging `path`, jobs `path`

### Traced path: [name]
1. `path:line` — [what happens]
2. `path:line` — [what happens]
...

### Worth knowing before you edit
- [Misleading name, dead code, duplicated abstraction, config-driven branch — each anchored.]

## Coverage
- **Inspected**: [file list]
- **Not inspected**: [directories/areas, and why]
- **Unresolved**: [runtime-bound indirection, with the registration site]
```

### Trace

```markdown
# Trace: [what is being traced]

**Entry** `path:line` → **Exit** `path:line`

| # | Where | What happens | Called from |
|---|---|---|---|
| 1 | `path:line` | [step] | — |
| 2 | `path:line` | [step] | `path:line` |

## Data at each hop
- **In**: [shape/type] — defined at `path:line`
- **Transformed**: [what changes, where] — `path:line`
- **Out**: [shape/type] — `path:line`

## Branches and side effects
- [Error paths, retries, async hand-offs, events emitted, cache writes] — each anchored.

## Not resolved by reading
- [Runtime-bound dispatch: registration site + candidates found.]
```

### Locate

```markdown
**[Behaviour] is implemented in `path:line`.**

- **Evidence**: [the function/route/handler name and what the code at that line does.]
- **Called from**: `path:line`
- **Related**: [config, tests, types the reader will need next — anchored.]
- **Caveat**: [other places that look like they own it but don't, or "not checked: X".]
```

## Staying in your lane

Read-only, and descriptive rather than evaluative. No refactor suggestions, no "you should",
no quality judgments, no next steps — unless the user explicitly asks for them after the map
is delivered.

The reason isn't purity: onboarding output gets pasted into wikis and trusted months later.
Once "this is how it works" and "this is what I'd change" are interleaved, the reader can't
tell which sentences are facts about the repo and which are your opinion — and both become
untrustworthy. If you spot something genuinely dangerous (a hardcoded credential, an
unhandled failure path), state it in one anchored line under a final **Observations**
heading, and stop there.

Two related restraints:

- Don't claim understanding of the whole repo after reading one subsystem. Say which.
- Don't modify files. The only write is the onboarding document, and only when asked.

## Traps that produce wrong maps

These are where a careful reader still gets it wrong. Check for them by name:

- **Barrel files.** `index.ts` / `__init__.py` that only re-export. The symbol lives elsewhere;
  follow the export to its definition before citing it.
- **Assembled route strings.** A router mounted at `/api/v1` with a handler declared as
  `/users` means grepping for `"/api/v1/users"` finds nothing. Search for the leaf segment,
  then find the mount point.
- **Same filename, different package.** In a monorepo, `service.ts` is ambiguous. Full paths
  always.
- **Framework magic.** File-system routing, annotation scanning, auto-registration — the
  "caller" doesn't exist in source. Say the convention is what wires it, and cite the
  convention's config.
- **Config-driven behaviour.** The answer is in a YAML file, a feature flag, or an env var,
  and the code just branches on it. Find the default and say where it's set.
- **Generated code that reads as hand-written.** API clients, ORM models, protobuf output.
  Check for a codegen config before explaining "the author's design".
- **Two implementations, one live.** A migration in progress or a flagged rewrite. Both look
  real; only one runs. Find the switch.
- **Tests for code that no longer runs.** A passing test suite is not proof a path is wired
  into the running app.

## Reference files

- `references/entry-points.md` — per-ecosystem manifests, entry-point markers, routing
  conventions, monorepo/workspace layouts, and what to ignore. Read when the stack is
  unfamiliar or the manifest points somewhere you don't recognise.
- `references/tracing.md` — tracing recipes per path kind (HTTP, CLI, queue, UI event,
  scheduled job, library API), plus how to handle each indirection pattern honestly. Read
  before Step 3 on any non-trivial trace.
- `scripts/check_citations.py` — verifies that every `` `path:line` `` in a draft resolves to
  a real file and a real line. Run before delivering anything file-shaped.
