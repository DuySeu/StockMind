---
phase: 3
slug: chunking-embedding-qdrant-storage
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-01
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go Test |
| **Config file** | none — stdlib `go test` |
| **Quick run command** | `go test -v ./internal/rag/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v ./internal/rag/...`
- **After every plan wave:** Run `go test -v ./internal/rag/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 3-01-01 | 01 | 1 | PROC-03 | unit | `go test -v ./internal/rag -run TestRecursiveChunker` | ❌ W0 | ⬜ pending |
| 3-01-02 | 01 | 1 | PROC-04 | unit | `go test -v ./internal/rag -run TestFixedChunker` | ❌ W0 | ⬜ pending |
| 3-01-03 | 01 | 1 | PROC-05 | unit | `go test -v ./internal/rag -run TestSemanticChunker` | ❌ W0 | ⬜ pending |
| 3-02-01 | 02 | 1 | PROC-07 | unit | `go test -v ./internal/rag -run TestEmbedder` | ❌ W0 | ⬜ pending |
| 3-03-01 | 03 | 2 | PROC-06 | unit | `go test -v ./internal/rag -run TestStore` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/rag/chunker_test.go` — stubs for strategies
- [ ] `internal/rag/embedder_test.go` — stubs for embedder
- [ ] `internal/rag/store_test.go` — stubs for Qdrant storage

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real Qdrant connectivity | INFRA-04 | Requires live Docker container | `docker ps` confirm qdrant running, run `go test` with real endpoint |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
