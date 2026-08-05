# Entry point discovery by ecosystem

Use this to answer two questions fast: *what is the manifest telling me*, and *which file
does the runtime actually start at*. Everything here is a search hint. Confirm each one by
opening the file — a project that follows every convention in this table still deviates
somewhere, and that deviation is the fact worth reporting.

## Contents

- [Reading order](#reading-order)
- [JavaScript / TypeScript](#javascript--typescript)
- [Python](#python)
- [Go](#go)
- [Java / Kotlin](#java--kotlin)
- [Ruby](#ruby)
- [PHP](#php)
- [Rust](#rust)
- [C# / .NET](#c--net)
- [Mobile](#mobile)
- [Infrastructure as the entry point](#infrastructure-as-the-entry-point)
- [Monorepos and workspaces](#monorepos-and-workspaces)
- [What to skip](#what-to-skip)

## Reading order

1. The manifest — name, declared entry, scripts, dependencies.
2. The process starter — `main`, `app`, `server`, `wsgi`, `Program.cs`.
3. The dispatch table — routes, commands, subscriptions, exports.
4. One handler, all the way down.

Dependencies in the manifest are the fastest framework detector available. `fastapi` in
`pyproject.toml` tells you more about the shape of the repo than the README's first paragraph.

## JavaScript / TypeScript

| Signal | Where |
|---|---|
| Manifest | `package.json` — `main`, `module`, `exports`, `bin`, `scripts`, `workspaces` |
| Real start command | `scripts.start` / `scripts.dev` — follow what it actually runs |
| Server bootstrap | `src/index.ts`, `src/server.ts`, `src/main.ts`, `app.ts` |
| CLI | `bin` field → the referenced file (often has a shebang) |
| Library | `exports` map — the public surface, and the only thing consumers can reach |

Framework markers:

- **Next.js** — `next.config.*`; routes are the file tree under `app/` (route handlers in
  `route.ts`, pages in `page.tsx`) or `pages/`. `middleware.ts` at the root runs before
  matched requests. No central router file exists; the convention *is* the router.
- **NestJS** — `main.ts` bootstraps, `*.module.ts` declares providers, `*.controller.ts`
  carries route decorators. Wiring is dependency injection: read the module before the
  controller.
- **Express / Fastify / Koa / Hono** — `app.use(...)` and `app.<method>(...)` calls, usually
  in `routes/` or `src/routes`. Mount prefixes matter (see the assembled-route trap).
- **Remix / SvelteKit / Nuxt / Astro** — file-system routing again; the config file names the
  routes directory.
- **Vite / Webpack SPA** — `index.html` → the module it loads → the app root component.
  `vite.config.*` / `webpack.config.*` for aliases, which explain otherwise unresolvable
  imports like `@/components/Foo`.

Also read `tsconfig.json` `paths` — import aliases are a frequent reason a search for a
module's path returns nothing.

## Python

| Signal | Where |
|---|---|
| Manifest | `pyproject.toml` (`[project.scripts]`, `[project.entry-points]`), `setup.py`, `requirements.txt` |
| CLI | `[project.scripts]` mapping, or `__main__.py` in the package |
| Module run | `if __name__ == "__main__":` blocks |
| WSGI/ASGI | `wsgi.py`, `asgi.py`, or the `app` object named in a Procfile / Dockerfile CMD |

Framework markers:

- **Django** — `manage.py`, `<project>/settings.py`, `<project>/urls.py` (the root URLconf),
  then per-app `urls.py` → `views.py` → `models.py`. `INSTALLED_APPS` in settings is the
  authoritative list of what's active; a directory that looks like an app but isn't listed
  is dead.
- **FastAPI** — `FastAPI()` instantiation, `@app.<method>` and `@router.<method>` decorators,
  `include_router(...)` calls for prefixes. Pydantic models are the request/response contract.
- **Flask** — `Flask(__name__)`, `@app.route`, blueprints registered via
  `register_blueprint(..., url_prefix=...)`.
- **Celery / RQ / Dramatiq** — `celery.py` or an app object, `@task`/`@shared_task`
  decorators. Producers call `.delay()` / `.apply_async()`; that's the seam between web and
  worker.

## Go

| Signal | Where |
|---|---|
| Manifest | `go.mod` — module path is the import prefix for everything internal |
| Entry | `func main()` in `main.go`, conventionally under `cmd/<binary>/` |
| Workspace | `go.work` lists the modules in play |
| Internal-only code | `internal/` — importable only within the module, a real boundary |

Routing is explicit: `http.HandleFunc`, `mux.HandleFunc`, `r.Get(...)` (chi), `e.GET(...)`
(echo), `router.GET(...)` (gin). Grep the router variable name. `init()` functions register
things without any visible caller — grep for `func init()` before concluding something isn't
wired up.

## Java / Kotlin

| Signal | Where |
|---|---|
| Manifest | `pom.xml` (Maven), `build.gradle[.kts]` (Gradle), `settings.gradle` for modules |
| Entry | `public static void main`, or `@SpringBootApplication` |
| Config | `application.yml` / `application.properties`, profile-specific variants |

Spring wiring is annotation-driven: `@RestController` + `@RequestMapping`/`@GetMapping` for
routes, `@Service` / `@Component` / `@Repository` for beans, `@Configuration` classes for
manual wiring. Component scanning means a class is active because of its package location and
annotation, with no import from the entry point — say so rather than hunting for a caller.

## Ruby

| Signal | Where |
|---|---|
| Manifest | `Gemfile`, `*.gemspec` |
| Rack entry | `config.ru` |
| Rails | `config/routes.rb` is the authoritative route table; `config/application.rb` boots |

Rails paths: `config/routes.rb` → `app/controllers/*_controller.rb` → `app/models` →
`app/views`. `rails routes` output, if present in the repo docs, is generated — verify
against `routes.rb`. Concerns in `app/controllers/concerns` and `app/models/concerns` add
behaviour by inclusion, invisibly at the call site.

## PHP

| Signal | Where |
|---|---|
| Manifest | `composer.json` (`autoload.psr-4` maps namespaces to directories) |
| Web entry | `public/index.php` |
| Laravel routes | `routes/web.php`, `routes/api.php`, then `app/Http/Controllers/` |
| Symfony routes | `config/routes.yaml` or `#[Route]` attributes on controllers |

Laravel service providers (`app/Providers/`) bind interfaces to implementations — that's
where an interface-typed constructor argument gets its concrete class.

## Rust

| Signal | Where |
|---|---|
| Manifest | `Cargo.toml` — `[[bin]]`, `[lib]`, `[workspace]` members |
| Binary entry | `src/main.rs` (`fn main()`) |
| Library surface | `src/lib.rs` — `pub mod` / `pub use` lines are the public API |

Module tree follows the filesystem via `mod` declarations; a directory with no `mod`
statement pointing at it is not compiled in.

## C# / .NET

| Signal | Where |
|---|---|
| Manifest | `*.csproj`, `*.sln` |
| Entry | `Program.cs` (top-level statements in modern versions) |
| Routing | `[ApiController]` + `[Route]` attributes, or minimal-API `app.MapGet(...)` calls |
| Config | `appsettings.json` + environment overrides |

`Program.cs` also holds the DI registrations (`builder.Services.Add...`) — that's where an
injected interface resolves to a class.

## Mobile

- **iOS/Swift** — `@main` struct (SwiftUI) or `AppDelegate` / `SceneDelegate` (UIKit);
  `*.xcodeproj` / `Package.swift` for structure.
- **Android/Kotlin** — `AndroidManifest.xml` declares the launcher activity; that's the entry.
  `build.gradle` modules define the boundaries.
- **React Native** — `index.js` → `App.tsx`; native shells under `ios/` and `android/`.
- **Flutter** — `lib/main.dart`, `void main()`; `pubspec.yaml` is the manifest.

## Infrastructure as the entry point

When the manifest doesn't say how the thing runs, the deployment config does — and it is the
authority, because it's what actually executes:

- `Dockerfile` — `CMD` / `ENTRYPOINT` names the real start command.
- `docker-compose.yml` — the set of services, their commands, and the dependencies between
  them. Frequently the fastest architecture diagram in the repo.
- `Procfile` — one line per process type: web, worker, scheduler.
- Kubernetes manifests / Helm charts — `command`/`args` per container.
- `Makefile`, `Taskfile.yml`, `justfile` — the commands the team actually types.
- CI workflows (`.github/workflows/`) — how it's built, tested, and deployed, which reveals
  which directories matter.

## Monorepos and workspaces

Detect first, because every "where does it start" answer becomes per-package:

| Marker | Tool |
|---|---|
| `pnpm-workspace.yaml`, `workspaces` in `package.json` | pnpm / npm / yarn workspaces |
| `turbo.json` | Turborepo |
| `nx.json`, `project.json` | Nx |
| `lerna.json` | Lerna |
| `go.work` | Go workspaces |
| `[workspace]` in `Cargo.toml` | Cargo |
| `WORKSPACE`, `BUILD.bazel`, `MODULE.bazel` | Bazel |
| `settings.gradle` with `include` | Gradle multi-project |
| `pyproject.toml` per subdirectory + a root tool config | uv / Poetry / Rye workspaces |

Then separate **applications** (have an entry point, get deployed) from **libraries**
(only imported). The dependency edges between packages are usually declared in each
package's own manifest — that's the real internal dependency graph, and it's cheaper to read
than to infer from imports.

## What to skip

Don't open, and say you skipped them: `node_modules/`, `vendor/`, `.venv/`, `dist/`, `build/`,
`out/`, `target/`, `__pycache__/`, `.next/`, `coverage/`, lockfiles (`package-lock.json`,
`pnpm-lock.yaml`, `poetry.lock`, `Cargo.lock`, `go.sum`), `*.min.js`, snapshots
(`__snapshots__/`), and generated code (`*.pb.go`, `*_pb2.py`, `*.generated.*`,
`schema.graphql` when a codegen config exists).

Migration directories deserve a rule of their own: read the *latest* migration or the schema
dump for the current shape, not the history. The history is a changelog, not a description of
the system as it exists.
