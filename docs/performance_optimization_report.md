# StockMind - Performance & Scalability Audit

Static review of the Go backend (`cmd/`, `internal/`) and the React 19 SPA (`frontend/src/`) at commit `231998c` plus the uncommitted working tree. Report only: no application code was changed.

## Executive summary

The codebase is well-structured and several hot paths already carry deliberate performance work: `averageVolumeCache` and `symbolDirectory` are TTL caches, `chartClient` reuses connections across the sector fan-out, `common.SharedHTTPClient` pools the LLM SDKs, the watchlist reorder is transactional, and the chat relay drains its producer channel on every exit. That work is real and it shows.

It is not production-ready for concurrent use yet, for five reasons that are structural rather than cosmetic:

1. **`GET /v1/stock/research-reports` issues one upstream VietCap POST per stored report, sequentially**, on `context.Background()`, and 500s the whole list if any one of them fails. Latency grows linearly with the size of the `research` table, forever.
2. **A cold sector price board fans out one chart request per ticker** - over 300 for the largest industry - inside the request, with no single-flight and no cache of the finished board. Two users opening the same sector do the work twice.
3. **An aborted research stream permanently wedges worker-pool workers.** The pool has `MaxWorkers: 3`, so three abandoned SSE streams take the research feature offline until restart.
4. **The BM25 tokenizer reads its `df` map after releasing the read lock.** A retrieval concurrent with an ingest job is a `concurrent map read and map write` fatal throw, which kills the process - not a recoverable panic.
5. **Every SSE token in chat re-renders the entire message list and re-parses every bubble's markdown**, and forces a layout read/write on the scroll container. Cost per token scales with conversation length.

There is also no index on `messages(conversation_id, created_at)`, which is read on every chat turn, and no route-level code splitting in the SPA, so `jspdf` and all twelve pages ship in the first chunk.

Assumptions used throughout: single process, single instance, unless a finding says otherwise; PostgreSQL local (`local-ms` latency class); `pgxpool.MaxConns = 10` (`cmd/main.go:88`); VietCap and Tavily reachable over the public internet with per-call latency in the hundreds of milliseconds; no cache or request coalescing anywhere except where the code shows one; result sets unbounded unless a `LIMIT` exists.

## Path Capacity

Selection was capped at 15 rows. All Critical-linked and Medium-linked paths are kept; `GetSessionsHandler`, `AddSymbolInPriceBoardHandler`, `UploadDocumentHandler` and `spaHandler` are Low-only and were dropped. The transport helpers `FetchIQInsight`, `fetchPriceBoardJSON` and `HandleGetStockPrice` are not entrypoints and share the rows of the handlers that call them.

| Path / symbol | Verdict | Band | Assumptions |
|---------------|---------|------|-------------|
| `GetResearchReportsHandler` (`GET /v1/stock/research-reports`) | Won't scale | Needs horizontal scale | One upstream HTTP call per stored report, sequential, unpaginated; no cache; any single failure fails the response |
| `GetSectorPriceBoardHandler` (`GET /v1/stock/sectors/{code}/price-board`) | Won't scale | Needs horizontal scale | Cold cache costs 1 upstream request per ticker (300+ for the largest ICB industry); no single-flight; finished board not cached |
| `averageVolumeCache.averages` | Won't scale | Single-instance | Process-local TTL cache; correctness of the liquidity filter depends on this process's warm state; concurrency bounded at 16 |
| `MarketResearchStreamHandler` / `stockDigestAgent` | Won't scale | Needs horizontal scale | `progressCh` not drained after client abort; pool is `MaxWorkers: 3`, `QueueSize: 20`, per-process; Tavily poll up to 2 min per ticker |
| `Tokenizer.Vectorize` / `Tokenizer.AddDocument` (BM25) | Won't scale | Single-instance | IDF state is process-local and never persisted; map read outside the lock; concurrent ingest + retrieval is a fatal throw |
| `MessageList` / `ChatbotPage` stream render | Won't scale | Single-instance | Per-token full-list re-render plus per-bubble markdown re-parse; browser main thread; cost grows with turn count |
| `Orchestrator.Run` via `ChatHandler` max mode (`POST /v1/chat`) | Won't scale | Needs horizontal scale | 8-minute total budget per HTTP request, steps strictly sequential, one goroutine and one open connection held throughout; `WriteTimeout: 10m` |
| `ChatHandler` (`POST /v1/chat`, single-agent) | Limited | Modest concurrent | Stateless across instances (session in DB); soft ceiling is provider latency plus history re-sent per tool round |
| `LLMService.LLMChat` / `runToolRound` | Limited | Modest concurrent | Tools in one round executed strictly sequentially; history grows by one row per round; no per-round cap |
| `symbolDirectory.load` | Limited | Single-instance | `sync.Mutex` held across a ~1 MB network fetch; every caller for 12 h serves from one process's copy |
| `GetConversationWithMessages` / `GetMessagesBySessionIdHandler` | Limited | Modest concurrent | No index on `messages(conversation_id, created_at)`; 20-row page carries full JSONB metadata (thinking capped at 24 KB per row) |
| `GetNewsHandler` (`GET /v1/news`) | Limited | Modest concurrent | `DATE(created_at) = DATE($1)` is non-sargable and unindexed; N sequential INSERTs; no guard against concurrent Tavily fan-out |
| `GetPriceBoardHandler` (`GET /v1/stock/price-board`) | Limited | Modest concurrent | One DB read plus one upstream POST per request; no cache, so upstream rate limits are the ceiling |
| `MoveWatchlistSymbolHandler` (`PATCH /v1/stock/watchlist/{id}/move`) | Limited | Modest concurrent | Read-modify-write of the whole list; one UPDATE round trip per row inside the transaction; no row locking |
| `router.tsx` initial bundle (all SPA routes) | Limited | Modest concurrent | No `lazy()` anywhere; `jspdf` and all twelve pages in the entry chunk; served by `spaHandler` off disk |

## Findings

### Critical

### P-001 - `GET /v1/stock/research-reports` makes one sequential upstream call per report
- **Severity:** Critical
- **Track:** api
- **Location:** `internal/server/researcher.handler.go:278`, price fetch at `:286`
- **Path:** `GetResearchReportsHandler`
- **Impact:** Response time grows linearly and without bound with the number of stored research reports, and one flaky upstream call turns the whole list into a 500.
- **Why:** `GetResearchReports` is unpaginated (`schema/queries/research_report.sql`: `SELECT ... FROM research ORDER BY created_at DESC`), and the loop calls `GetLatestMatchPrice` once per row. Each call is a full POST to VietCap's price board (`fetchPriceBoardJSON`, `internal/server/stock.handler.go:55`) requesting exactly one symbol. Fifty reports is fifty round trips in series. Three problems compound it: the loop passes `context.Background()` rather than `r.Context()`, so a client that gave up cannot cancel the remaining calls; a single error `return`s a 500 after work is already done; and `reports` is appended without a preallocated capacity.
- **Suggestion:** Paginate the query (`LIMIT`/`OFFSET`, or keyset on `created_at`), then collect the page's distinct tickers and make **one** `fetchPriceBoardJSON` call for all of them - the endpoint already takes a symbol list, which is what `GetPriceBoardHandler` does. Pass `r.Context()`. Treat a failed price lookup as a missing price on that row rather than as a failed request; a report without a live price is still useful.

### P-002 - Cold sector price board fans out one upstream request per ticker, with no single-flight
- **Severity:** Critical
- **Track:** api
- **Location:** `internal/server/sector.handler.go:158`, fan-out at `internal/server/liquidity.go:80`
- **Path:** `GetSectorPriceBoardHandler`, `averageVolumeCache.averages`
- **Impact:** The first request for a large industry issues 300+ upstream HTTP calls inside the request; concurrent requests for the same industry each issue their own full set.
- **Why:** The comment at `liquidity.go:38` states the case plainly: VietCap's chart endpoint takes one symbol per call, so the largest industry is over 300 requests. `averages` bounds concurrency at 16, so a cold industry is roughly 20 serialized batches. The cache is keyed per ticker with a 12 h TTL, which does help across sectors - but unlike `symbolDirectory.load`, `averages` has no single-flight gate: N concurrent requests for the same cold sector all read the cache, all miss, and all fan out. The finished board is not cached either, so even a fully warm volume cache still costs one price-board POST plus a sort on every request. Note also that all 300 goroutines are spawned before any acquires a slot (`liquidity.go:105`, the `slots <- struct{}{}` is inside the goroutine), and none checks `ctx.Err()` before its HTTP call, so an abandoned request runs to completion.
- **Suggestion:** Add a single-flight gate keyed by ticker so concurrent cold requests share one in-flight fetch (`golang.org/x/sync/singleflight`, or a `map[string]chan struct{}` under the existing mutex). Warm the cache for the tradable universe on a background ticker at startup, so the request path is warm-only. Cache the projected `[]SectorBoardRow` per sector code for a short TTL (30-60 s is well inside a trading session's tolerance). Check `ctx.Err()` at the top of each goroutine.

### P-003 - An aborted research stream wedges the worker pool permanently
- **Severity:** Critical
- **Track:** runtime
- **Location:** `internal/server/researcher.handler.go:197` (the `return` on `r.Context().Err()`), producer at `:349`
- **Path:** `MarketResearchStreamHandler`, `stockDigestAgent`
- **Impact:** Each abandoned SSE stream leaks goroutines and blocks pool workers forever; with `MaxWorkers: 3` (`internal/service/worker/research.go`), three abandoned streams take market research offline until the process restarts.
- **Why:** The handler exits the `for evt := range progressCh` loop with a bare `return` when the client disconnects. Nobody drains `progressCh` after that. `ProcessResearchJob`'s `emit` writes into the proxy channel unconditionally (`researcher.handler.go` `emit` closure), the proxy goroutine writes into `progressCh`; both channels are buffered at 50 and then both writers block for good. The worker goroutine is stuck inside `emit`, so it never reaches `job.ResultCh <- ...`, so `stockDigestAgent` never returns from its `<-resultCh` loop, so `close(progressCh)` at the end of `stockDigestAgent` never runs. Nothing times out. The `Pool.activeWorkers` counter is never decremented, so the pool believes it is at capacity.
- **Suggestion:** Make every `emit` send non-blocking or context-aware - a `select` with a `default` (progress events are advisory; dropping one is correct) or with `<-ctx.Done()`. Give `ProcessResearchJob` a cancellable context derived from the request rather than `context.Background()`. On the handler side, drain `progressCh` in a deferred goroutine the way `ChatHandler` does for `streamCh` (`internal/server/chat.handler.go`, the `defer func()` before `relay:`), which is exactly the right pattern and is already in the codebase.

### P-004 - BM25 tokenizer reads its document-frequency map after releasing the lock
- **Severity:** Critical
- **Track:** runtime
- **Location:** `internal/knowledge/bm25.go:70` (`df := t.df`), `:71` (`RUnlock`), read at `:85`
- **Path:** `Tokenizer.Vectorize` / `Tokenizer.AddDocument`
- **Impact:** A retrieval running concurrently with document ingestion is a `concurrent map read and map write` fatal throw, which terminates the process and cannot be recovered by `middleware.Recoverer`.
- **Why:** `Vectorize` copies the map *header* under `RLock`, releases the lock, then indexes `df[idx]` in the scoring loop. `AddDocument` writes `t.df[idx]++` under the write lock. The two are reachable at the same time by construction: the ingest worker pool (`MaxWorkers: 2`) calls `pipeline.Process` → `tokenizer.AddDocuments` (`internal/knowledge/ingest.go`), while any chat turn using `retrieve_knowledge` calls `VectorizeQuery` → `Vectorize`. Both hold references to the same `*Tokenizer` built once in `kb.New`. Secondary scale point in the same file: `docCount` and `df` live only in this process's memory and are never persisted, so a restart resets every IDF weight to its cold value, and a second instance would score the same query differently. `AddDocuments` (`:49`) also takes and releases the write lock once per document instead of once per batch.
- **Suggestion:** Do the scoring loop's map reads inside the `RLock`, or snapshot only the `df` values the query's terms need (`for idx := range tf { dfs[idx] = t.df[idx] }`) while still holding it. Then take the batch lock once in `AddDocuments`. Separately, persist the IDF table (a small table or a Qdrant payload) if BM25 scores are meant to be stable across restarts and instances - as written, the hybrid retrieval's sparse half is single-instance by design.

### P-005 - Every chat token re-renders the whole message list and re-parses every bubble's markdown
- **Severity:** Critical
- **Track:** web
- **Location:** `frontend/src/components/containers/MessageList.tsx:126`, markdown at `:176` and `:312`; scroll effect at `frontend/src/pages/Chatbot.tsx:105-113`
- **Path:** `MessageList` / `ChatbotPage` stream render
- **Impact:** Typing-speed jank on the primary journey; the cost of each streamed token grows with the number of messages already on screen, so long conversations degrade progressively.
- **Why:** Each `text` / `thinking` delta calls `setMessages` with a fresh array (`Chatbot.tsx` `handleStreamEvent`, `case "text"`). `MessageList` is a plain function component with no `memo` boundary and no per-message component, so it maps over every message and re-runs each one's `<Markdown remarkPlugins={[remarkGfm]}>` on every delta - the remark parse for the *whole* transcript, dozens of times a second, not just for the bubble that changed. `markdownComponents` is module-level (good), but the `components` prop is passed to a component that re-renders anyway. On top of that, the effect on `[messages, isLoadingMore]` runs a `querySelector` and then reads `scrollHeight` and writes `scrollTop` on every delta, forcing a synchronous layout each time.
- **Suggestion:** Extract a `MessageBubble` wrapped in `React.memo`, keyed by message id rather than array index, so an unchanged bubble neither re-renders nor re-parses. Coalesce deltas: buffer incoming text in a ref and flush to state on a `requestAnimationFrame` or a ~50 ms timer, so the render rate is bounded by the frame rate rather than by the token rate. Move the auto-scroll into the same rAF and skip it when the user has scrolled away from the bottom.

### Medium

### P-006 - No index on `messages(conversation_id, created_at)`, and none on the other list paths
- **Severity:** Medium
- **Track:** api
- **Location:** `schema/migrations/0001_init.sql:52` (`conversations`), `:62` (`messages`), `:72` (`research`), `:88` (`news`)
- **Path:** `GetConversationWithMessages` / `GetMessagesBySessionIdHandler`
- **Impact:** Chat history load and message count degrade as the `messages` table grows, on a query that runs on every single chat turn.
- **Why:** Only `watchlist(position)` has a secondary index in the whole schema (`0002_watchlist_position.sql:16`). Every hot filter-and-sort pattern is unindexed: `messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT 20` (run by `ChatHandler` at `internal/server/chat.handler.go:142` **and** by `GetMessagesBySessionIdHandler`), `COUNT(*) FROM messages WHERE conversation_id = $1` (run twice per turn - once by the handler, once by `maybeSummarize`), `conversations WHERE user_id = $1 ORDER BY created_at DESC`, `research ORDER BY created_at DESC`, `news WHERE DATE(created_at) = ...`. Postgres will seq-scan and sort.
- **Suggestion:** Add `messages(conversation_id, created_at DESC)`, `conversations(user_id, created_at DESC)`, `research(created_at DESC)` and `news(created_at DESC)` in a new migration. Verify each with `EXPLAIN (ANALYZE, BUFFERS)` against a realistically sized table - the composite on `messages` should turn both the page query and the count into index-only work.

### P-007 - Chat history is over-fetched and re-sent in full on every tool round
- **Severity:** Medium
- **Track:** api
- **Location:** `internal/llm/service.go:65` (`toolDefs`), `:135` (`runToolRound`), history read at `internal/server/chat.handler.go:142`
- **Path:** `LLMService.LLMChat` / `runToolRound`
- **Impact:** Per-turn latency and token cost grow with the number of tool rounds; a multi-tool round takes the sum of its tools' latencies rather than the max.
- **Why:** `runToolRound` executes `s.tools.Execute` strictly in sequence (`for _, tc := range pending`). The tools it dispatches are network-bound - `get_stock_price`, `get_report`, `piotroski_evaluation`, `get_news` are all remote HTTP - and a model that asks for three of them in one round pays for all three serially. Each round then appends an assistant row carrying the full text of every tool result and re-sends the whole `history` to the provider on the next round. The history itself arrives pre-inflated: `GetConversationWithMessages` `jsonb_agg`s 20 rows including each row's complete `metadata`, which holds up to 24 KB of persisted thinking (`chat.handler.go:29`) plus every historical tool result.
- **Suggestion:** Run one round's tool calls concurrently with a small bounded fan-out and preserve result order by index - the shape already used in `GetMarketIndicesHandler` (`internal/server/index.handler.go`). Project the history query to the fields the prompt needs and leave `thinking` out of it: it is stored for replay in the UI, not for the model. Consider truncating persisted tool results for history rows beyond the most recent turn.

### P-008 - Max-mode orchestration holds one request open for up to eight minutes, steps strictly sequential
- **Severity:** Medium
- **Track:** api
- **Location:** `internal/orchestration/orchestrator.go:26-27` (budgets), `:execute` step loop; `WriteTimeout` at `internal/server/server.go`
- **Path:** `Orchestrator.Run` via `ChatHandler` max mode
- **Impact:** A handful of concurrent max-mode turns is enough to occupy the server for minutes at a time, with no queue, no admission control and no visibility to the client beyond the plan.
- **Why:** `DefaultTotalBudget` is 8 minutes and `DefaultStepTimeout` 150 s, sized deliberately against a `WriteTimeout` of 10 minutes - the comments are explicit that a real four-step run took 237 s. `execute` runs steps one at a time because later steps consume earlier outputs via `UseOutputOf`, which is correct for a dependent chain but means independent steps (a market-data lookup and a news lookup that both feed the synthesizer) still serialize. Nothing bounds how many pipelines run at once. Each one holds a goroutine, an HTTP connection, and - through its agents' tools - upstream quota.
- **Suggestion:** Execute the plan as a dependency DAG rather than a list: steps whose `UseOutputOf` is already satisfied can run concurrently, which is the single biggest latency win available here and needs no protocol change. Add a semaphore bounding concurrent pipelines, and return 503 with a clear message rather than accepting work the box cannot do. Longer term, a max-mode turn is a job, not a request - persisting it and letting the client poll or reconnect removes the 10-minute write window entirely.

### P-009 - `GET /v1/news` uses a non-sargable date filter and inserts one row at a time
- **Severity:** Medium
- **Track:** api
- **Location:** `internal/server/news.handler.go:16`, insert loop at `:30`; query at `schema/queries/news.sql:3`
- **Path:** `GetNewsHandler`
- **Impact:** Full scan of `news` on every request, and on the first request of the day every concurrent visitor independently calls Tavily and writes duplicate rows.
- **Why:** `WHERE DATE(created_at) = DATE($1::timestamptz)` wraps the column in a function, so no plain index on `created_at` can be used. The fallback path is entered on `err != nil || len(latestNews) == 0`, with nothing to coalesce concurrent entries: ten simultaneous requests at 09:00 make ten Tavily searches and insert ten copies of the same articles, one `SaveNews` round trip per article. There is no unique constraint on `news(url)` to stop the duplicates.
- **Suggestion:** Rewrite as a half-open range (`created_at >= $1 AND created_at < $1 + interval '1 day'`) so the index from P-006 applies. Insert the batch in one statement (`unnest` or a `pgx` batch) with `ON CONFLICT (url) DO NOTHING`, backed by a unique index on `url`. Guard the refresh with a single-flight gate or an advisory lock so only one request per day does the fetching.

### P-010 - A fresh `http.Client` per call on the VietCap paths caps effective pooling at two connections
- **Severity:** Medium
- **Track:** runtime
- **Location:** `internal/common/iq_insight.go:61`, `internal/server/stock.handler.go:75`, `internal/tools/implementations/get_stock_price.handler.go:81`
- **Path:** shares the rows of `GetSectorPriceBoardHandler`, `GetPriceBoardHandler` and `GetResearchReportsHandler`
- **Impact:** TLS handshake churn under exactly the fan-out patterns the code performs, adding hundreds of milliseconds per extra concurrent call.
- **Why:** These three sites construct `&http.Client{Timeout: ...}` with no `Transport`, so they fall through to `http.DefaultTransport`. Connections are pooled - but `DefaultTransport`'s `MaxIdleConnsPerHost` is 2, so beyond two concurrent calls to `iq-insight-service` the surplus connections are closed rather than kept. `HandlePiotroskiEvaluation` (`internal/tools/implementations/piotroski_evaluation.handler.go`) fires three `FetchIQInsight` calls concurrently; `FundamentalAnalysisHandler` fires three more. This is the exact problem the comment above `chartClient` (`internal/server/liquidity.go:52`) describes and solves for the chart endpoint - the fix just was not applied to the other two hosts.
- **Suggestion:** Give each upstream host one package-level client with a tuned `Transport`, following `chartClient`. `common.SharedHTTPClient` (`internal/common/configs.go:165`) is already configured with 100 idle connections per host and is the natural home for the iq-insight and price-board calls.

### P-011 - Watchlist reorder does a read-modify-write of the whole list, one UPDATE per row
- **Severity:** Medium
- **Track:** api
- **Location:** `internal/server/stock.handler.go:345`, UPDATE loop at `:410`
- **Path:** `MoveWatchlistSymbolHandler`
- **Impact:** One DB round trip per watchlist entry per move, and two concurrent moves can interleave into an order neither user asked for.
- **Why:** The handler reads the full watchlist, swaps two elements in memory, then issues `SetWatchlistPosition` once per row inside a transaction. The renumbering is justified in the comment (positions drift into gaps and ties), but the cost is N round trips for a two-row change, and the read at `:369` is not part of the transaction and takes no row locks, so the classic lost-update window is open between read and write.
- **Suggestion:** Renumber in one statement - `UPDATE watchlist SET position = v.pos FROM (SELECT ... ) v WHERE watchlist.id = v.id`, or a `WITH ordered AS (SELECT id, row_number() OVER (ORDER BY position, created_at DESC) ...)`. Do the read inside the transaction with `FOR UPDATE` so a concurrent move serializes rather than interleaves. Better still, express the move as a single SQL swap of the two positions and leave a periodic (or triggered) renumber to close gaps.

### P-012 - No route-level code splitting; `jspdf` and all twelve pages ship in the entry chunk
- **Severity:** Medium
- **Track:** web
- **Location:** `frontend/src/router.tsx:1-12`, `frontend/src/lib/pdf-export.ts:1`
- **Path:** `router.tsx` initial bundle
- **Impact:** Every visitor downloads, parses and evaluates the PDF generator and all twelve routes before the landing page can paint, on a market where mobile is dominant.
- **Why:** `router.tsx` statically imports every page component; `grep` finds no `lazy(` or `Suspense` anywhere in `frontend/src`. `pdf-export.ts` imports `jspdf` at module scope and is imported statically by `ResearchResultPage.tsx:5`, so `jspdf` is in the entry graph even for a user who never opens a research report. `react-markdown` + `remark-gfm` are likewise pulled in for the landing page, which renders no markdown. There is no `build.rollupOptions.manualChunks` in `vite.config.ts`.
- **Suggestion:** Convert the router's children to `lazy(() => import(...))` with a `Suspense` fallback - React Router's data router supports it directly. Move the `jspdf` import inside the export handler (`await import("jspdf")`) so it is fetched on click. Then run `vite build` and check the chunk report; `manualChunks` for the Radix and markdown vendor groups is worth it once the routes are split.

### P-013 - Price board renders every row unvirtualized, and re-derives its filter, sort and lookup map each render
- **Severity:** Medium
- **Track:** web
- **Location:** `frontend/src/pages/WatchList.tsx:276-285`, row loop at `frontend/src/components/PriceBoardTable.tsx:350`
- **Path:** `PriceBoardTable`
- **Impact:** A large ICB industry produces hundreds of rows of thirteen cells each, all mounted at once, and every unrelated state change (a dropdown opening, an error clearing) rebuilds the derived arrays and re-renders the whole table.
- **Why:** `WatchList.tsx` computes `watchlistByTicker` (a fresh `Map`), the dedupe `filter`, the exchange `filter` and the sort inline in the component body - no `useMemo` - so all four run on every render, and `visibleRows` is a new array identity every time, which defeats memoization downstream. `PriceBoardTable` then maps over the entire `visibleRows` with no windowing; each row mounts two `PriceCell`s, a `DropdownMenu` and several formatters. The sector endpoint accepts a `limit` (`sector.handler.go:107`) but the frontend never sends one (`api/stock.ts` `getSectorPriceBoard` is always called with `undefined`).
- **Suggestion:** Wrap the four derived values in `useMemo` with precise dependencies, and memoize the row component. Pass a `limit` on the sector fetch, or virtualize the table body - `@tanstack/react-virtual` is the light option, but a limit plus pagination avoids the dependency entirely and matches how the endpoint is already built.

### P-014 - Research SSE client re-slices the cumulative XHR response on every progress event
- **Severity:** Medium
- **Track:** web
- **Location:** `frontend/src/api/stock.ts:84-89`, `:122-123`
- **Path:** shares the `MarketResearchStreamHandler` row
- **Impact:** Quadratic string work over the life of the stream and the entire response - including the full report JSON - retained in memory twice.
- **Why:** `responseType: "text"` with `onDownloadProgress` reads `XMLHttpRequest.responseText`, which is the whole response accumulated so far. Each event slices from `parsedLength` to the end; the underlying buffer keeps growing, so total work is O(n²) in the response size. The `result` event carries the complete multi-ticker research report, which is the largest payload the app handles. The chat path already does this correctly - `sendChatMessage` (`frontend/src/api/chat.ts`) uses `fetch` with a `ReadableStream` reader and a bounded frame buffer.
- **Suggestion:** Switch `streamMarketResearch` to the same `fetch` + `getReader()` + `TextDecoder` pattern as `sendChatMessage`, and reuse its `parseFrame`. That removes the accumulation entirely and drops the axios dependency from this path.

### P-015 - Document embedding issues its batches strictly in sequence
- **Severity:** Medium
- **Track:** api
- **Location:** `internal/knowledge/embedder.go:87`, batch size at `:19`
- **Path:** shares the `Tokenizer` / ingest row
- **Impact:** Ingest time for a large document is the sum of all embedding round trips; a 2,000-sentence PDF is 100 sequential API calls at 20 inputs each.
- **Why:** `Embed` loops over `inputs` in `batchSize` slices and awaits each `s.embed` call before starting the next. This is called twice per document under semantic chunking - once by `SemanticChunker.Chunk` to embed every sentence (`internal/knowledge/chunker.go`), then again by `IngestPipeline.Process` to embed the merged chunks. It is background work behind a 2-worker pool, so it does not block a request, but it sets how long a document sits in `processing` and how quickly the queue (`QueueSize: 20`) backs up.
- **Suggestion:** Run the batches with a bounded concurrency (4-8 is usually where embedding providers' rate limits bite) writing into the preallocated `results` slice by index, which keeps the existing ordering guarantee. Respect `ctx` between batches so a cancelled job stops promptly.

### Low

### P-016 - `persistReports` refetches a price per ticker on `context.Background()`
- **Severity:** Low
- **Track:** api
- **Location:** `internal/server/researcher.handler.go:324-326`
- **Path:** shares the `MarketResearchStreamHandler` row
- **Impact:** One extra sequential upstream call per ticker on a fire-and-forget goroutine that no context can cancel.
- **Why:** Same one-symbol-per-call pattern as P-001, at most five tickers here (the payload caps at 5), so the blast radius is small. It runs after the response is sent, so nothing user-visible waits on it - but a hung upstream call keeps the goroutine alive indefinitely.
- **Suggestion:** Fetch the prices for all of the report's tickers in one `fetchPriceBoardJSON` call, and give the goroutine a `context.WithTimeout(context.Background(), ...)`.

### P-017 - `POST /v1/stock/add-symbol` loads the whole watchlist and linearly scans the company directory
- **Severity:** Low
- **Track:** api
- **Location:** `internal/server/stock.handler.go:267`, watchlist read at `:295`, directory scan at `internal/server/sector.handler.go:221`
- **Impact:** Two avoidable O(n) passes per insert; harmless at watchlist scale, wrong shape if watchlists ever become per-user.
- **Why:** `distinctTickers(watchlist)` is rebuilt just to run `slices.Contains`, and `symbolDirectory.lists` does `slices.ContainsFunc` over ~1,700 entries. The duplicate check is also not atomic with the insert - two concurrent adds of the same symbol both pass.
- **Suggestion:** A unique index on `watchlist(ticker)` plus `ON CONFLICT DO NOTHING` makes the check both cheaper and correct. Build a `map[string]struct{}` alongside `entries` in `symbolDirectory.load` for the membership test.

### P-018 - `symbolDirectory.load` holds its mutex across the network fetch
- **Severity:** Low
- **Track:** runtime
- **Location:** `internal/server/sector.handler.go:233-234`
- **Path:** `symbolDirectory.load`
- **Impact:** Once per 12 hours, every concurrent request touching sectors or add-symbol blocks for the full duration of a ~1 MB upstream fetch (up to the 10 s timeout).
- **Why:** This is deliberate - the comment at `sector.handler.go:56` says the mutex doubles as a single-flight gate, and coalescing one refetch is the right call. The cost is that it is an exclusive lock held across I/O, so readers with a still-valid cache also wait, and a slow upstream stalls `GetSectorsHandler`, `GetSectorPriceBoardHandler` and `AddSymbolInPriceBoardHandler` together.
- **Suggestion:** Keep the single-flight, but serve the stale copy to readers while one goroutine refreshes: swap to `sync.RWMutex` for the read of `entries`/`fetchedAt`, and hold a separate refresh mutex around the fetch. Stale-while-revalidate is safe here - the directory only changes on a listing day.

### P-019 - `GET /v1/sessions` is unpaginated
- **Severity:** Low
- **Track:** api
- **Location:** `internal/server/session.handler.go:27`
- **Impact:** Payload and query cost grow without bound as a user accumulates conversations; the sidebar renders all of them.
- **Why:** `GetConversationsByUserID` has no `LIMIT`, and `SideBar` fetches the whole list on mount and on every `sessionVersion` bump. With one hardcoded user today the list is small; the shape does not survive real usage.
- **Suggestion:** Add `LIMIT`/`OFFSET` (or keyset on `created_at`) and select only `id`, `title`, `created_at` - the sidebar reads nothing else. Pairs with the `conversations(user_id, created_at DESC)` index from P-006.

### P-020 - `spaHandler` builds a new `FileServer` per request and opens each file twice
- **Severity:** Low
- **Track:** api
- **Location:** `internal/server/routes.go:126-135`
- **Impact:** One redundant `open`/`close` and one allocation per static asset request; measurable only under sustained static load, which a CDN or reverse proxy would normally absorb anyway.
- **Why:** The handler opens the file to test for existence, closes it via `defer`, then constructs `http.FileServer(http.FS(spaFS))` - a fresh handler value - which opens the same file again to serve it.
- **Suggestion:** Build the `FileServer` once outside the returned closure, and use `fs.Stat` for the existence check instead of `Open`. Add a `Cache-Control` header for hashed asset paths while you are there.

### P-021 - `tradingSessionsSince` walks the calendar day by day per ticker
- **Severity:** Low
- **Track:** runtime
- **Location:** `internal/server/liquidity.go:254`
- **Path:** shares the `averageVolumeCache.averages` row
- **Impact:** Negligible on its own - ~30 iterations per ticker - but it is inside the 300-way fan-out of P-002 and the result is identical for every ticker in a given request.
- **Why:** Each `fetchAverageVolume` call recomputes the same weekday count for the same 30-day window.
- **Suggestion:** Compute it once per `averages` call and pass it in, or replace the loop with arithmetic on whole weeks plus the remainder.

## What to measure next

Optional, and not a gate on acting on the above. Roughly in order of what would confirm the most:

- `go test -bench` plus `pprof` (cpu and block profiles) around `GetSectorPriceBoardHandler` with a stubbed `chartURL` - `internal/server/liquidity_test.go` already stubs it, so the harness exists. The block profile is the one that will show P-002's serialization.
- `go test -race ./internal/knowledge/...` with an ingest and a retrieval running concurrently. This should reproduce P-004 directly; the race detector will name both sites.
- `EXPLAIN (ANALYZE, BUFFERS)` on `GetConversationWithMessages`, `GetMessageCountByConversationID` and `GetLatestNews` against a `messages` table seeded to ~10^5 rows, before and after the P-006 indexes.
- React Profiler on `/c` during a streaming turn, with a 20-message conversation loaded. Record commit counts and durations per delta - that quantifies P-005 and confirms whether the `memo` boundary lands where expected.
- `vite build` and the resulting chunk size report, as the baseline for P-012.
- A `pprof` goroutine dump taken after aborting a research stream mid-flight, to confirm the wedged workers in P-003 (`/debug/pprof/goroutine?debug=2` would need to be wired up; the leak is also visible as `Pool.activeWorkers` never returning to zero).
