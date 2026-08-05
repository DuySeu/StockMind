# Tracing execution paths

A trace is only useful if it doesn't stop early. "The controller calls the service layer" is
where most explanations end and where the reader's real question begins. This file is the
mechanics of going the rest of the way, plus how to stay honest at the points where reading
alone can't answer it.

## Contents

- [The general loop](#the-general-loop)
- [HTTP request](#http-request)
- [CLI command](#cli-command)
- [Queue / event consumer](#queue--event-consumer)
- [Scheduled job](#scheduled-job)
- [UI interaction](#ui-interaction)
- [Library public API](#library-public-api)
- [Indirection you cannot read through](#indirection-you-cannot-read-through)
- [Search recipes](#search-recipes)
- [Knowing when to stop](#knowing-when-to-stop)

## The general loop

At each hop, answer four things and then move one level down:

1. **Where am I** — `path:line` of the definition.
2. **What does this code do to the data** — validate, transform, branch, persist, emit.
3. **Who does it call next** — `path:line` of the call site, then find that definition.
4. **What else fires here that isn't in the call chain** — middleware, decorators,
   interceptors, ORM hooks, signals, error handlers.

Point 4 is what separates a trace from a call-stack dump. Cross-cutting behaviour is invisible
at the call site by design, and it's usually what surprises the newcomer: the request was
already authenticated, the transaction was already open, the response was already serialised.

Record the data shape as it changes. "A `CreateUserRequest` (`schemas.py:14`) becomes a `User`
row (`models.py:31`) and is returned as a `UserResponse` (`schemas.py:28`)" tells a reader
more than three paragraphs of narrative.

## HTTP request

1. **Find the route.** Grep the leaf path segment, not the full URL — routers mount prefixes,
   so the complete string rarely appears anywhere in source. In file-system-routed frameworks
   (Next.js, SvelteKit, Nuxt, Remix), translate the URL to a directory path instead.
2. **Find the mount point.** `app.use("/api/v1", router)`, `include_router(prefix=...)`,
   `scope "/api"`, `RoutePrefix`. This gives you the real external URL and often a second
   middleware stack.
3. **List the middleware chain in order.** Auth, CORS, body parsing, rate limiting, request
   IDs, transaction wrapping. Order is behaviour: an auth middleware registered after the
   route doesn't protect it.
4. **Open the handler.** Note request validation and where the parsed input's type is defined.
5. **Follow into the service/domain layer**, then into persistence. Don't stop at the
   repository interface — find the implementation, or say the binding is runtime-resolved.
6. **Find the response path.** Serializers, response models, status codes, and the error
   handler that catches what the handler throws.

Worth checking every time: is there an API gateway, reverse proxy, or BFF in front? A route
that seems unauthenticated in source is often authenticated a layer up, and that layer may be
in a different repo — say so rather than reporting the endpoint as open.

## CLI command

1. Start from the manifest's declared binary (`bin`, `[project.scripts]`, `[[bin]]`,
   `cmd/<name>/main.go`).
2. Find the argument parser and the command registry: `argparse`/`click`/`typer` decorators,
   `commander`/`yargs` chains, `cobra.Command` structs, `clap` derives.
3. Map each subcommand to its handler function, and note global flags applied before dispatch
   (verbosity, config path, dry-run).
4. Follow the chosen subcommand into its implementation.
5. Note the exit paths: return codes, what gets written to stdout vs stderr, and what side
   effects happen on the filesystem or network.

Config resolution order (flag → env → file → default) is usually the thing a newcomer gets
wrong. Find where it's decided and cite it.

## Queue / event consumer

Trace both halves separately; they're connected by a name, not a call.

- **Producer** — grep for the publish call: `.delay(`, `.apply_async(`, `.send(`, `.publish(`,
  `.enqueue(`, `emit(`. Record what triggers it and the payload shape.
- **The name in between** — queue name, topic, event type, routing key. This string is the
  join between the two halves and belongs in the trace explicitly.
- **Consumer** — grep the same string, or the handler registration:
  `@task`, `@app.task`, `@EventHandler`, `subscribe(`, `consumer.on(`.
- **Delivery semantics visible in code** — retries, backoff, dead-letter config, idempotency
  keys, `ack` placement. These change what "it works" means and are usually in the consumer's
  setup, not the handler.

Say plainly that the producer→consumer link is by name rather than by call. That's the fact
that stops a newcomer looking for a caller that doesn't exist.

## Scheduled job

Find the schedule definition first — `crontab`, `CronJob` manifests, `celery beat` schedule,
`@Scheduled`, `node-cron`, a cloud scheduler config — because the schedule lives outside the
handler and is often outside the repo. Then trace the handler like any other entry point.
Note whether the job is idempotent-by-design and whether concurrent runs are prevented, if the
code says so; don't infer it if it doesn't.

## UI interaction

1. **Find the element** by its visible text or test id, which lands you in the component.
2. **Find the handler** — the `onClick`/`onSubmit`/`@click` binding.
3. **Follow the state change** — local state, store dispatch (Redux/Zustand/Pinia/signals), or
   a form library submit.
4. **Find the network call** — the fetch/axios/query hook, and the API path it targets. That
   path is the seam into the backend trace; if the backend is in the same repo, continue.
5. **Follow the response back** — cache updates, re-render triggers, error/loading states.

Note where state actually lives: server-cache libraries (React Query, SWR, Apollo) mean the
component isn't the source of truth, and a newcomer looking for a `useState` will never find
it.

## Library public API

There is no runtime entry point; the entry point is whatever consumers can import. Read the
`exports` map / `lib.rs` / `__init__.py` / package-level exports, then trace one public
function inward. Separate the published surface from internals — for a library that
distinction *is* the architecture.

## Indirection you cannot read through

Each of these breaks the call chain. The honest move is the same in all of them: name the
registration site, say how the binding is resolved, list the candidates you found, and don't
assert which one runs.

| Pattern | Where the binding actually happens | How to report it |
|---|---|---|
| DI container (Spring, NestJS, .NET, Laravel providers) | Module/config class or provider registration | "Injected as `UserRepo`; bound to `SqlUserRepo` at `config/di.ts:22`" |
| Interface + single implementation | Nowhere — it's a compile-time choice | Say there's exactly one implementer and name it |
| Interface + several implementations | A factory, config value, or env var | Name the selector and its default |
| Event bus / pub-sub | Subscription registration | Give the event name and every subscriber found |
| Decorator / annotation registry | The decorator's own definition | Explain what the decorator does, then list the decorated functions |
| File-system routing | The framework convention | Cite the config that sets the routes directory |
| Component scanning (Spring, Symfony) | Package location + annotation | Say the class is active by location, not by import |
| Dynamic dispatch (`getattr`, reflection, `eval`, string→handler maps) | Runtime only | Show the map or the lookup expression; list candidate keys |
| Plugin/hook systems | A registry populated at startup | Name the registry and where plugins are discovered |
| Monkey patching / method overriding | Wherever the patch is applied | Grep for the patched name; report both definitions |
| Code generation | The codegen config | Say the file is generated and cite the source of truth |
| Middleware order | Registration order in the bootstrap file | List the chain in registration order |

## Search recipes

`rg` is assumed; `grep -rn` works the same way.

```bash
# Where is this symbol defined vs. merely used?
rg -n "(function|def|class|func|const|type) +Foo\b"
rg -n "\bFoo\b" --type ts

# Who imports this module?
rg -n "from ['\"].*user-service|require\(['\"].*user-service"
rg -n "import .*userservice" -i

# Find a route without knowing the prefix
rg -n "['\"/]users['\"]" -g '!node_modules'
rg -n "@(Get|Post|Put|Delete|RequestMapping)" -g '*.ts' -g '*.java'

# Find the producer/consumer pair for a queue name
rg -n "user\.created"

# What does this env var control?
rg -n "FEATURE_NEW_CHECKOUT"

# Confirm behaviour cheaply — tests as executable documentation
rg -n "describe\(|it\(|def test_" -l | rg -i "checkout"
```

Two habits that pay off: search for the *string* a user would see (a URL segment, an error
message, a queue name) rather than the abstraction's name, and when you find a definition,
immediately search for its usages to learn whether it's on the live path at all.

## Knowing when to stop

Stop descending when the next hop is one of:

- **Third-party library internals.** Name the library and the version from the manifest;
  don't read into `node_modules`.
- **The database or an external service.** The query or the client call is the edge. Cite the
  SQL/ORM call and stop.
- **Language runtime or standard library.**
- **A different repository.** Name it and say the trace continues there.

And say which of these you hit. "The trace ends at `repo.Save()` (`store/user.go:88`), which
issues an INSERT against Postgres" is a finished trace. "It saves the user somewhere" is not.
