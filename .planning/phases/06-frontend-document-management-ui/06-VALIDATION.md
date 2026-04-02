---
phase: 6
slug: frontend-document-management-ui
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-01
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest |
| **Config file** | frontend/vite.config.ts |
| **Quick run command** | `cd frontend && npm test -- src/api/document.ts` |
| **Full suite command** | `cd frontend && npm test` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npm test -- {relevant_file}`
- **After every plan wave:** Run `cd frontend && npm test`
- **Before /gsd-verify-work:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 6-01-01 | 01 | 1 | UPLOAD-01/02 | unit | `npm test -- src/lib/validation.ts` | ❌ W0 | ⬜ pending |
| 6-01-02 | 02 | 1 | UI-01 | component | `npm test -- src/components/DocumentUploadForm.tsx` | ❌ W0 | ⬜ pending |
| 6-01-03 | 03 | 1 | UI-03 | component | `npm test -- src/components/DocumentListTable.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/lib/validation.ts` — stubs for file validation testing
- [ ] `frontend/src/api/__tests__/document.test.ts` — API client mock tests
- [ ] `frontend/src/components/__tests__/DocumentListTable.test.tsx` — table rendering tests

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Upload PDF Processing Status | UI-02 | Async integration | Upload PDF, observe "Processing" badge transition to "Ready". |
| Status Polling Intensity | UI-04 | Time-based behavior | Use Network tab to verify GET calls occur every 3 seconds during processing. |
| Tooltip error message | UI-05 | Visual UX | Manually fail a document in DB, hover over status badge in UI. |
| Confirmation Dialog | UI-06 | User interaction | Click delete, verify dialog appears before deletion. |
| Empty state illustration | UI-07 | Visual UX | Fresh DB state, navigate to /documents, verify layout. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
