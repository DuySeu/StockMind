# Phase 2: Database Interface Abstraction - Research

**Objective:** What do I need to know to PLAN this phase well?

## Core Goal
Transition the codebase from tight coupling with `*database.Queries` to using abstract consumer-defined interfaces. The plan operates per decision **D-01: Ad-hoc Interface Definition**, which drops the idea of a centralized `repository` package in favor of defining the required interface directly where it is consumed (`server` and `agent`).

## Interface Consumers & Required Methods

We have two primary packages consuming database methods currently. Each will need its own ad-hoc interface definition. 

### 1. `internal/agent`
The agent service (`internal/agent/service.go` and `internal/agent/session.go`) uses:
- `GetSessionHistoryBySessionID(context.Context, uuid.UUID) ([]database.SessionHistory, error)`
- `SessionAddChatHistory(context.Context, database.SessionAddChatHistoryParams) (database.SessionHistory, error)`
- `UpdateSessionTurnCount(context.Context, database.UpdateSessionTurnCountParams) error`
- `GetAgentFlowById(context.Context, uuid.UUID) (database.AgentFlow, error)`
- `CreateSession(context.Context, database.CreateSessionParams) (database.Session, error)`
- `GetSessionByID(context.Context, uuid.UUID) (database.Session, error)`

### 2. `internal/server`
The HTTP server layer uses a vast array of struct routes across `*.handler.go`:
- **Auth/Users:** `CreateUser`, `GetUsers`, `GetUserByID`, `UpdateUser`, `DeleteUser`
- **Agent/Flow:** `ListAgentFlows`
- **Stock:** `GetWatchlist`, `CreateWatchlistData`
- **Sessions:** `GetSessionsByUserID`, `GetSessionHistoryBySessionID`, `DeleteSessionByID`
- **News/Research:** `GetLatestNews`, `SaveNews`, `CreateResearchReport`, `GetResearchReports`, `GetResearchReportById`

## Structural Requirements for Ad-Hoc interfaces

Because `database.New(dbPool)` returns `*database.Queries` directly, both `agent.NewService` and `server.NewServer` constructors will need to change its argument from `dbPool *pgxpool.Pool` (or `*database.Queries`) to the specific interfaces.

To keep it clean:
1. `internal/agent/service.go` should declare `type AgentStore interface { ... }`.
2. `internal/server/server.go` should declare `type ServerStore interface { ... }`.

### Transaction Validation
Currently, transactions (`*pgx.Tx`) are NOT widely utilized outside of `Queries.WithTx()`, which mostly appears in `internal/database/db.go`. If new transactions are introduced, wrapper functions rather than spreading `pgx.Tx` will be utilized (D-02). For Phase 2, strictly extracting `AgentStore` and `ServerStore` is sufficient.

## Recommended Modifications

1. **internal/agent/service.go**:
   Define `type AgentStore interface { ... }` with the methods required. Change `queries *database.Queries` to `db AgentStore`.

2. **internal/agent/session.go**:
   Update usages from `sm.llm.queries` to `sm.llm.db`.

3. **internal/server/server.go**:
   Define `type ServerStore interface { ... }` with all methods used in handlers. Change `db *database.Queries` to `db ServerStore`. 

4. **cmd/main.go**:
   Update `agent.NewService` and `server.NewServer` initializers. Since `*database.Queries` inherently implements the interfaces, passing `database.New(dbPool)` into both constructors directly satisfies the interface implementations.
