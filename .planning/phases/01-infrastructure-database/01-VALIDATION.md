---
phase: 1
slug: infrastructure-database
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-01
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go test (`go test`) / Docker Compose |
| **Config file** | `docker-compose.yml`, `docker-compose.dev.yml` |
| **Quick run command** | `docker compose config` / `go build ./internal/database/...` |
| **Full suite command** | `go test ./internal/database/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick syntax check (`go build` / `docker compose config`)
- **After every plan wave:** Run `make itest` (integration tests) or `make build`
- **Before `/gsd-verify-work`:** Full build and `docker compose up -d` must succeed cleanly
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01-01 | 1 | INFRA-01 | integration | `docker compose config -q` | ✅ | ⬜ pending |
| 1-01-02 | 01-01 | 1 | INFRA-01 | integration | `docker compose -f docker-compose.dev.yml config -q` | ✅ | ⬜ pending |
| 1-02-01 | 01-02 | 1 | INFRA-04 | unit | `cat schema/migrations/0002_add_documents.sql \| grep "goose"` | ✅ | ⬜ pending |
| 1-02-02 | 01-02 | 1 | INFRA-04 | unit | `go build ./internal/database/...` | ✅ | ⬜ pending |
| 1-03-01 | 01-03 | 2 | INFRA-02 | unit | `go list -m github.com/qdrant/go-client` | ✅ | ⬜ pending |
| 1-03-02 | 01-03 | 2 | INFRA-02 | unit | `go build ./internal/rag/...` | ❌ W0 | ⬜ pending |
| 1-03-03 | 01-03 | 2 | INFRA-03 | integration | `go build ./cmd/main.go` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Create `internal/rag/client.go` package structure (handled in Plan 01-03)

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| DB Startup Connect | INFRA-03 | Requires full stack state | `docker compose up` and watch backend logs to retry connection until Qdrant is up. |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-01
