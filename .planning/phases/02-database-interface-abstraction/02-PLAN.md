---
wave: 1
depends_on: []
files_modified:
  - internal/agent/service.go
  - internal/agent/session.go
  - internal/server/server.go
  - cmd/main.go
requirements:
  - ARCH-02
autonomous: true
---

# Plan 02: Database Interface Abstraction

## Objective
Implement Ad-hoc Interface Definition for the database bindings. Extract `*database.Queries` pointers out of the `internal/agent` and `internal/server` packages, replacing them with locally defined interfaces `AgentStore` and `ServerStore`.

## Context
Per `02-CONTEXT.md` (Decision D-01) and subsequent research, rather than a centralized repository, we are implementing local interfaces inside `service.go` and `server.go` respectively to fully isolate the generated `sqlc` models from application business logic coupling.

## Tasks

```xml
<task>
  <id>1</id>
  <description>Extract AgentStore Interface</description>
  <read_first>
    <file>internal/agent/service.go</file>
    <file>internal/agent/session.go</file>
  </read_first>
  <action>
    - In `internal/agent/service.go`, define a new interface:
      ```go
      type AgentStore interface {
          GetSessionHistoryBySessionID(context.Context, uuid.UUID) ([]database.SessionHistory, error)
          SessionAddChatHistory(context.Context, database.SessionAddChatHistoryParams) (database.SessionHistory, error)
          UpdateSessionTurnCount(context.Context, database.UpdateSessionTurnCountParams) error
          GetAgentFlowById(context.Context, uuid.UUID) (database.AgentFlow, error)
          CreateSession(context.Context, database.CreateSessionParams) (database.Session, error)
          GetSessionByID(context.Context, uuid.UUID) (database.Session, error)
      }
      ```
    - Modify `AgentService` struct: replace `queries *database.Queries` with `db AgentStore`.
    - Modify `NewService` signature to accept `db AgentStore` instead of `dbPool *pgxpool.Pool`. Update struct initialization to assign `db: db`.
  </action>
  <acceptance_criteria>
    - `internal/agent/service.go` contains `type AgentStore interface`.
    - `internal/agent/service.go` no longer imports `*database.Queries` directly for structural assignment.
  </acceptance_criteria>
</task>

<task>
  <id>2</id>
  <description>Update Agent Database Usages</description>
  <depends_on>1</depends_on>
  <read_first>
    <file>internal/agent/session.go</file>
  </read_first>
  <action>
    - In `internal/agent/session.go`, change all instances of `sm.llm.queries` to `sm.llm.db` (e.g., lines 70, 227, 287, 371, 412, 452).
    - In `internal/agent/service.go`, change all instances of `s.queries` to `s.db`.
  </action>
  <acceptance_criteria>
    - `grep sm.llm.queries internal/agent/session.go` returns no results.
    - `grep s.queries internal/agent/service.go` returns no results.
  </acceptance_criteria>
</task>

<task>
  <id>3</id>
  <description>Extract ServerStore Interface</description>
  <read_first>
    <file>internal/server/server.go</file>
  </read_first>
  <action>
    - In `internal/server/server.go`, define a new interface:
      ```go
      type ServerStore interface {
          ListAgentFlows(context.Context) ([]database.AgentFlow, error)
          GetWatchlist(context.Context, int32) ([]database.Watchlist, error)
          CreateWatchlistData(context.Context, string) (database.Watchlist, error)
          GetSessionsByUserID(context.Context, uuid.UUID) ([]database.Session, error)
          GetSessionHistoryBySessionID(context.Context, uuid.UUID) ([]database.SessionHistory, error)
          DeleteSessionByID(context.Context, uuid.UUID) error
          GetLatestNews(context.Context, string) ([]database.News, error)
          SaveNews(context.Context, database.SaveNewsParams) (database.News, error)
          CreateResearchReport(context.Context, database.CreateResearchReportParams) (database.ResearchReport, error)
          GetResearchReports(context.Context) ([]database.ResearchReport, error)
          GetResearchReportById(context.Context, uuid.UUID) (database.ResearchReport, error)
          CreateUser(context.Context, database.User) (database.User, error)
          GetUsers(context.Context) ([]database.User, error)
          GetUserByID(context.Context, uuid.UUID) (database.User, error)
          UpdateUser(context.Context, database.User) error
          DeleteUser(context.Context, uuid.UUID) error
      }
      ```
    - Modify `Server` struct: replace `db *database.Queries` with `db ServerStore`.
    - Modify `NewServer` signature: `NewServer(dbPool *pgxpool.Pool, db ServerStore, agent *agent.AgentService, streamManager *StreamManager, port string) *http.Server`.
    - Note: Update struct assignment `db: db`.
  </action>
  <acceptance_criteria>
    - `internal/server/server.go` contains `type ServerStore interface`.
    - `Server` struct contains `db ServerStore` instead of `db *database.Queries`.
  </acceptance_criteria>
</task>

<task>
  <id>4</id>
  <description>Update Server Bootstrapping</description>
  <depends_on>1,3</depends_on>
  <read_first>
    <file>cmd/main.go</file>
  </read_first>
  <action>
    - In `cmd/main.go`, create the database abstraction concretely via `database.New(dbPool)` and assign it to a variable `dbQueries := database.New(dbPool)`.
    - Inject `dbQueries` into the agent: `agent.NewService(ctx, dbQueries, llmConfig)`.
    - Inject `dbQueries` into the server: `server.NewServer(dbPool, dbQueries, agent, streamManager, port)`.
  </action>
  <acceptance_criteria>
    - `cmd/main.go` instantiates `database.New(dbPool)` exactly once and passes it explicitly to `AgentService` and `Server` as an interface.
    - Application compiles: `go build -o /dev/null ./cmd/main.go` returns 0.
  </acceptance_criteria>
</task>
```

## Verification

### Must Haves
- The application must compile successfully without any missing method errors, confirming the `ServerStore` and `AgentStore` interfaces perfectly match generated signatures.
- All direct dependency on `*database.Queries` in business logic struct members must be eradicated.

### Steps
1. Execute `go build -o /dev/null ./cmd/main.go`.
2. Ensure no output is thrown.
3. Validate memory references explicitly.
