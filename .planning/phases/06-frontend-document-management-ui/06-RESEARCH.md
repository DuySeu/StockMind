# Phase 06: Frontend Document Management UI - Research

**Date:** 2026-04-01
**Status:** Complete
**Objective:** Research how to implement a document management interface in the existing React + Vite + TypeScript frontend.

## 1. Architectural Alignment

### Current Stack
- **Framework:** React 19.1.1
- **Icons:** `lucide-react`
- **UI Components:** Radix UI primitives, localized `shadcn/ui` in `src/components/ui`.
- **Routing:** `react-router-dom` 7.8
- **Form Handling:** `react-hook-form` + `zod`
- **API Client:** `axios` configured in `src/api/index.ts` (baseURL: `http://localhost:8080/v1`)

### Navigation & Routing
- New route `/documents` should be added to `src/router.tsx` as a child of `MainLayout`.
- `src/components/layout/Navbar.tsx` needs to be updated with a "Documents" item.

## 2. API Integration Strategy

### API Client (`src/api/document.ts`)
Needs 3 main functions:
1. `listDocuments()`: `GET /api/documents`
2. `uploadDocument(file, strategy)`: `POST /api/documents` (multipart/form-data)
3. `deleteDocument(id)`: `DELETE /api/documents/:id`

### Type Definitions (`src/types/document.ts`)
```typescript
export type DocumentStatus = 'pending' | 'processing' | 'ready' | 'failed';

export type Document = {
  id: string;
  name: string;
  file_type: string;
  size_bytes: number;
  status: DocumentStatus;
  chunk_count: number;
  strategy: string;
  error_msg?: string;
  created_at: string;
  updated_at: string;
};
```

## 3. UI/UX Patterns

### Document List (`DocumentListTable.tsx`)
- Uses `src/components/ui/table.tsx`.
- Status Badges:
  - `pending`/`processing`: Blue badge with a spinner or "Processing..." text.
  - `ready`: Green "Ready" badge.
  - `failed`: Red "Failed" badge with a tooltip showing `error_msg`.
- Actions: Delete button with `lucide-react/Trash2`.

### Upload Form (`DocumentUploadForm.tsx`)
- Radix-based `Dialog` or `Sheet`.
- Form Fields:
  - File picker (styled input file).
  - Select dropdown for chunking strategies:
    - `Recursive` -> "Smart Split (Recursive)" (Recommended)
    - `Fixed` -> "Fixed Size"
    - `Paragraph` -> "By Paragraph"
    - `Semantic` -> "By Topic (Semantic)"
- Validation (Zod):
  - File size ≤ 10MB (`UPLOAD-02`).
  - File type: PDF, DOCX, MD, TXT (`UPLOAD-01`).

### Status Polling (`useDocumentPolling.tsx`)
- Custom hook that triggers `setInterval` when at least one document in the list is in `pending` or `processing` state.
- Refreshes the list every 3-5 seconds.

## 4. Risks & Considerations

- **File Upload Progress:** Simple `axios.onUploadProgress` can be used to show a progress bar during the initial upload.
- **Empty States:** Specifically mentioned in `UI-07`. Needs a nice illustration or descriptive placeholder.
- **Responsiveness:** Ensure the table doesn't break on mobile by using horizontal scrolling or a card layout for mobile view.

## 5. Implementation Roadmap Recommendation

1. **Scaffold:** Types and API client.
2. **Layout:** Add route and nav item.
3. **List:** Build table with dummy data first, then connect to `GET`.
4. **Upload:** Create Dialog + form.
5. **Polite Polling:** Implement the status refresh logic.
6. **Polish:** Errors handling, tooltips, and empty states.

## 6. Validation Strategy (Nyquist Dimension 8)

### Manual Verification
- Upload 5MB PDF, watch status transition from "Processing" -> "Ready".
- Attempt 11MB upload, verify client-side validation error.
- Delete a document, verify it vanishes from UI.
- Mock "failed" status in API, verify tooltip shows error reason.

### Automated Testing
- Mock API responses with MSW during component tests.
- Unit tests for file validation logic (size, extension).
- Snapshot tests for status badges.
