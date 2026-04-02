---
phase: 6
plan: 3
name: document-page-router-polling
wave: 3
depends_on: [6-02-01, 6-02-02, 6-02-03]
requirements: [UI-02, UI-04, UI-07]
files_modified: [frontend/src/pages/DocumentPage.tsx, frontend/src/router.tsx, frontend/src/components/layout/Navbar.tsx, frontend/src/hooks/useDocumentPolling.ts]
autonomous: true
---

# Plan 03: Document Page, Router & Polling

Finalize the document management interface by creating the main page, registering the route, updating the global navigation, and implementing reliable status polling.

## Tasks

<task id="6-03-01" requirements="UI-04">
<action>
Create `frontend/src/hooks/useDocumentPolling.ts`.
Params: `documents: Document[], refreshCallback: () => Promise<void>`.
Logic:
- Boolean `shouldPoll`: `documents.some(d => d.status === 'pending' || d.status === 'processing')`.
- Effect: `setInterval(refreshCallback, 3000)` if `shouldPoll` is true, otherwise `clearInterval`.
- Cleanup: `clearInterval` on unmount.
</action>
<read_first>
- frontend/src/hooks/use-mobile.ts (hook structure)
</read_first>
<acceptance_criteria>
- Polling starts only when a document is in processing.
- Polling stops once all documents are in `ready` or `failed` state.
</acceptance_criteria>
</task>

<task id="6-03-02" requirements="UI-01, UI-07">
<action>
Create `frontend/src/pages/DocumentPage.tsx`.
Structure:
- Header with "Document Management" title and subtitle.
- "Upload Document" button (triggers `DocumentUploadForm` in a Dialog).
- `DocumentListTable` (receives document list from API state).
- Handle `GET /api/documents` on mount.
- Integration with `useDocumentPolling` to keep processing states fresh.
- Empty State: If list size is 0, show "Upload tài liệu để tăng cường khả năng trả lời của AI" with a plus icon.
</action>
<read_first>
- frontend/src/pages/WatchList.tsx (header reference)
- .planning/REQUIREMENTS.md (UI-07 text)
</read_first>
<acceptance_criteria>
- Page loads initial list on mount correctly.
- Empty state message displays when no docs are available.
</acceptance_criteria>
</task>

<task id="6-03-03" requirements="v1-nav">
<action>
Update `frontend/src/router.tsx` and `frontend/src/components/layout/Navbar.tsx`.
- Router: Add route `{ path: "documents", element: <DocumentPage /> }` to the main children array.
- Navbar: Add `{ label: "Knowledge", path: "/documents" }` to `navItems`. (Label changed to "Knowledge" for better user alignment).
</action>
<read_first>
- frontend/src/router.tsx
- frontend/src/components/layout/Navbar.tsx
</read_first>
<acceptance_criteria>
- Link appears in the navigation bar.
- Clicking the link navigates to the Document Page.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Navigation links stay highlighted when the page is active.
- Polling does not cause excessive re-renders (optimized hooks).
- Error messages from failed uploads are clearly displayed using a toast or banner.
</must_haves>

<automated>
- `cd frontend && npm run build` (verify build succeeds with new routes/types)
</automated>
