---
wave: 1
depends_on: []
files_modified:
  - "internal/agent/configs.go"
  - "internal/server/stream.go"
  - "internal/agent/service.go"
  - "internal/server/routes.go"
  - "internal/server/server.go"
  - "cmd/main.go"
autonomous: true
requirements_addressed: ["ARCH-01"]
---

# Phase 1: Dependency Injection Assembly Plan

<objective>
Refactor `DefaultOpenAIConfig`, `DefaultAnthropicConfig`, and `GlobalStreamManager` globally scoped variables out of packages. Establish explicitly wired Dependency Injection at `cmd/main.go`.
</objective>

<must_haves>
- `DefaultOpenAIConfig` variable must not exist.
- `DefaultAnthropicConfig` variable must not exist.
- `GlobalStreamManager` variable must not exist.
- `cmd/main.go` must instantiate configurations natively and pass them sequentially to `agent.NewService` and `server.NewServer`.
</must_haves>

<task>
<id>1</id>
<title>Refactor LLM Config Globals</title>
<action>
Modify `internal/agent/configs.go`. Delete the variables `DefaultOpenAIConfig` and `DefaultAnthropicConfig`.
Add a new public function `func LoadLLMConfig() LLMProviderConfig` that natively constructs and returns these configurations, preserving their exact current defaults (`os.Getenv("OPENROUTER_API_KEY")`, `RoleARN`, etc.).
</action>
<read_first>
- internal/agent/configs.go
</read_first>
<acceptance_criteria>
- `grep "var DefaultOpenAIConfig" internal/agent/configs.go` exits 1.
</acceptance_criteria>
</task>

<task>
<id>2</id>
<title>Refactor StreamManager Singleton</title>
<action>
In `internal/server/stream.go`, delete `var GlobalStreamManager = &StreamManager{...}`.
Create a strict constructor: `func NewStreamManager() *StreamManager { return &StreamManager{...} }`.
In `internal/server/server.go`: Update `Server` struct to embed or include `StreamManager *StreamManager`. 
Modify `NewServer` signature to accept `*StreamManager`: `func NewServer(db *pgxpool.Pool, a *agent.AgentService, streamMap *StreamManager, port string) *http.Server`. Assign `streamMap` to the returned pointer struct instances.
</action>
<read_first>
- internal/server/stream.go
- internal/server/server.go
</read_first>
<acceptance_criteria>
- `grep "var GlobalStreamManager" internal/server/stream.go` exits 1.
</acceptance_criteria>
</task>

<task>
<id>3</id>
<title>Update Agent Service Constructor</title>
<action>
In `internal/agent/service.go`: Update `NewService` to accept `LLMProviderConfig` explicitly:
`func NewService(ctx context.Context, dbPool *pgxpool.Pool, config LLMProviderConfig) (*AgentService, error)`
Drop the internal initialization mapping (`config := LLMProviderConfig{...}`) and instead bind `s.config = config` from the parameter.
</action>
<read_first>
- internal/agent/service.go
</read_first>
<acceptance_criteria>
- `grep "config LLMProviderConfig" internal/agent/service.go` correctly exits 0, indicating proper parameter placement.
</acceptance_criteria>
</task>

<task>
<id>4</id>
<title>Update WebSocket Routes</title>
<action>
In `internal/server/routes.go`: Search for all 2 instances of `GlobalStreamManager`. 
Replace them with `s.StreamManager`, utilizing the embedded struct's state on the receiving pointer.
Ensure the `RegisterRoutes` callbacks execute identically but utilizing the server instance explicitly over the imported un-namespaced context.
</action>
<read_first>
- internal/server/routes.go
</read_first>
<acceptance_criteria>
- `grep "GlobalStreamManager" internal/server/routes.go` exits 1.
</acceptance_criteria>
</task>

<task>
<id>5</id>
<title>Update Entry Point</title>
<action>
In `cmd/main.go`, inside `runServer()`:
1. Initialize `llmConfig := agent.LoadLLMConfig()`.
2. Update the agent initialization call to: `agentService, err := agent.NewService(ctx, dbPool, llmConfig)`.
3. Initialize `streamManager := server.NewStreamManager()`.
4. Update the server initialization call to: `server := server.NewServer(dbPool, agentService, streamManager, port)`.
Ensure paths line up correctly and the compile errors vanish.
</action>
<read_first>
- cmd/main.go
</read_first>
<acceptance_criteria>
- `go build ./cmd/server/main.go` succeeds (or equivalently `make build`).
</acceptance_criteria>
</task>
