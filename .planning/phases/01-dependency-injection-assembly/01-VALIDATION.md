# Phase 1 Nyquist Validation Strategy

**Date:** 2026-03-31
**Phase:** 01-dependency-injection-assembly

## Goal-Backward Validation

1. Is `DefaultOpenAIConfig` removed from the global package scope?
   - **Verification:** `grep -r "var DefaultOpenAIConfig" internal/agent` should return no results.
2. Is `GlobalStreamManager` removed?
   - **Verification:** `grep -r "var GlobalStreamManager" internal/server` should return no results.
3. Is `LLMProviderConfig` successfully injected?
   - **Verification:** Check that `cmd/main.go` instantiates the config and passes it to `agent.NewService`.
4. Does the app compile?
   - **Verification:** `go build -o /dev/null ./cmd/main.go`

## Pre-execution Checks (Dimension 2)
- Inspect `internal/server/routes.go` handler attachments to ensure they are strictly methods off `*Server` containing the `StreamManager`.

## Step-by-step Verifications (Dimension 4)
- **Step 1:** Modify configs. Verify tests pass.
- **Step 2:** Modify Service. Verify `main.go` compile fixes.
- **Step 3:** Modify Server. Update routes. Verify HTTP compile fixes.
