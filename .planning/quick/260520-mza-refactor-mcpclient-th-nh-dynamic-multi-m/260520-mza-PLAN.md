# Plan: Refactor mcpclient to Dynamic Multi-MCP Bridge

Implement a robust, extensible MCP client system in StockMind to support dynamic discovery and execution of external MCP server tools without code modifications.

## Tasks

- [x] Create `internal/mcpclient/manager.go` to manage multiple servers.
- [x] Add `ListTools` method to `internal/mcpclient/mcpclient.go`.
- [x] Create `internal/llm/tools/bridge.go` to dynamically bridge tools.
- [x] Modify `internal/llm/tools/schemas.go` to remove static hardcoded AWS docs tools.
- [x] Delete outdated `internal/llm/tools/aws_docs.handler.go`.
- [x] Modify `cmd/main.go` to initialize `mcpclient.Manager`, bridge tools dynamically, and close all connections upon shutdown.
- [x] Add unit test `internal/mcpclient/manager_test.go` to verify correctness.
