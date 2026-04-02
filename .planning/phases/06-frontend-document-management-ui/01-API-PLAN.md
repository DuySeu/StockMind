---
phase: 6
plan: 1
name: document-api-types-validation
wave: 1
depends_on: []
requirements: [UPLOAD-01, UPLOAD-02, UPLOAD-03]
files_modified: [frontend/src/types/document.ts, frontend/src/api/document.ts, frontend/src/lib/validation.ts]
autonomous: true
---

# Plan 01: Document API, Types & Validation

Scaffold the foundational code necessary for document management, including type definitions, API client functions, and client-side file validation logic.

## Tasks

<task id="6-01-01" requirements="UPLOAD-01">
<action>
Create `frontend/src/types/document.ts`.
Define `DocumentStatus` as union: `'pending' | 'processing' | 'ready' | 'failed'`.
Define `Document` interface:
- id: string
- name: string
- file_type: string
- size_bytes: number
- status: DocumentStatus
- chunk_count: number
- strategy: string
- error_msg?: string
- created_at: string
- updated_at: string
</action>
<read_first>
- frontend/src/types/stock.ts (pattern reference)
</read_first>
<acceptance_criteria>
- `frontend/src/types/document.ts` exists.
- Exported `Document` type matches the backend `documents` table schema.
</acceptance_criteria>
</task>

<task id="6-01-02" requirements="UPLOAD-01">
<action>
Create `frontend/src/api/document.ts`.
Implement using `api` axios instance:
- `getDocuments()`: `GET /api/documents`
- `uploadDocument(file: File, strategy: string)`: `POST /api/documents` with `FormData`.
- `deleteDocument(id: string)`: `DELETE /api/documents/:id`
</action>
<read_first>
- frontend/src/api/stock.ts (pattern reference)
- frontend/src/api/index.ts (axios instance)
</read_first>
<acceptance_criteria>
- `frontend/src/api/document.ts` allows fetching and uploading.
- `uploadDocument` correctly sets `Content-Type: multipart/form-data`.
</acceptance_criteria>
</task>

<task id="6-01-03" requirements="UPLOAD-02">
<action>
Create `frontend/src/lib/validation.ts`.
Implement `validateDocumentFile(file: File): { valid: boolean; error?: string }`:
- Check size: `file.size <= 10 * 1024 * 1024` (10MB).
- Check extension: `.pdf, .docx, .md, .txt`.
</action>
<read_first>
- .planning/REQUIREMENTS.md (UPLOAD-02 spec)
</read_first>
<acceptance_criteria>
- File > 10MB returns `valid: false`.
- Invalid extension returns `valid: false`.
- CSV/JPG returns `valid: false`.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- TypeScript types strictly match backend schema from Phase 1.
- API client handles `multipart/form-data` for uploads.
- Client-side validation prevents unnecessary server hits for large/invalid files.
</must_haves>

<automated>
- `cd frontend && npm test -- src/lib/validation.ts`
- `cd frontend && npm test -- src/api/document.ts`
</automated>
