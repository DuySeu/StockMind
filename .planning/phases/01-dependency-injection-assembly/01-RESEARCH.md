# Phase 1: Dependency Injection Assembly - Research

## Context Summary
The goal is to eliminate global states, specifically `DefaultOpenAIConfig`, `DefaultAnthropicConfig`, and `GlobalStreamManager`, ensuring they are loaded centrally via `main.go` and explicitly injected downstream.

## Codebase Analysis
1. **Tool Configurations (`internal/agent/configs.go`)**: 
   `DefaultOpenAIConfig` and `DefaultAnthropicConfig` are initialized as package-level globals. They read `os.Getenv` dynamically, making them impossible to mock easily in tests.
2. **Agent Service Init (`internal/agent/service.go`)**:
   `NewService` wraps the globals into `config := LLMProviderConfig{ OpenAI: DefaultOpenAIConfig, Anthropic: DefaultAnthropicConfig }`. 
3. **Stream State Singleton (`internal/server/stream.go`)**:
   `var GlobalStreamManager = &StreamManager{...}` acts as a singleton.
4. **WebSocket Routes (`internal/server/routes.go`)**:
   Code accesses the singleton directly (e.g., `GlobalStreamManager.CreateStream(...)`, `GlobalStreamManager.GetStream(...)`).
5. **Entry Point (`cmd/main.go`)**:
   Bootstraps environments and calls `agent.NewService` and `server.NewServer`.

## Implementation Strategy for Planners

### 1. Abstracting Configuration
Planners should introduce a `LoadConfig()` mechanism (can live inside `internal/agent/configs.go` since `LLMProviderConfig` exists).
- Return a strongly typed struct replacing the `Default*` globals.
- Globals MUST be deleted.

### 2. Refactoring `agent.NewService`
- **Current Signature:** `NewService(ctx context.Context, dbPool *pgxpool.Pool)`
- **Target Signature:** `NewService(ctx context.Context, dbPool *pgxpool.Pool, cfg LLMProviderConfig)`

### 3. Refactoring `GlobalStreamManager`
- Move `StreamManager` initialization out of global variable scope.
- Initialize `NewStreamManager()` within `server.NewServer` or pass it in from `main.go`.
- Ensure all handler functions in `routes.go` use receptor methods (e.g., `func (s *Server) myHandler`) to access `s.streamManager` rather than referencing the global variable.

## Validation Architecture

**Dimension 8: Verification Goals**
- Must prove `DefaultOpenAIConfig` does not exist in `internal/agent/configs.go`.
- Must prove `GlobalStreamManager` does not exist in `internal/server/stream.go`.
- Server boot in `main.go` must instantiate `LLMProviderConfig` explicitly.
- The compiled code must run perfectly under `make run` without panic.
