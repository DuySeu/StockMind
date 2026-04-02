---
phase: 6
plan: 2
name: core-components-upload-list
wave: 2
depends_on: [6-01-01, 6-01-02]
requirements: [UI-01, UI-03, UI-05, UI-06]
files_modified: [frontend/src/components/DocumentUploadForm.tsx, frontend/src/components/DocumentListTable.tsx, frontend/src/components/StatusBadge.tsx]
autonomous: true
---

# Plan 02: Core Components - Upload & List

Build the essential UI components for document management: an upload form, a tabular display for existing documents, and a status tracking badge.

## Tasks

<task id="6-02-01" requirements="UI-01">
<action>
Create `frontend/src/components/DocumentUploadForm.tsx`.
Use `react-hook-form` + `zod`.
Components:
- File picker (styled `input type="file"`) with `onBlur` validation from Plan 1.
- `Select` dropdown for strategies:
  - `recursive`: "Smart Split (Recursive)" (Recommended)
  - `fixed`: "Fixed Size"
  - `paragraph`: "By Paragraph"
  - `semantic`: "By Topic (Semantic)"
- Submit button with `lucide-react/Upload` icon.
</action>
<read_first>
- frontend/src/components/ui/select.tsx
- frontend/src/components/ui/form.tsx
- frontend/src/lib/validation.ts (for client-side size check)
</read_first>
<acceptance_criteria>
- `DocumentUploadForm` renders a file input and a dropdown.
- Button shows loading state during upload.
</acceptance_criteria>
</task>

<task id="6-02-02" requirements="UI-03, UI-05">
<action>
Create `frontend/src/components/StatusBadge.tsx`.
Uses `src/components/ui/badge.tsx`.
Props: `status: DocumentStatus`, `errorMessage?: string`.
Variants:
- `pending`: `variant="secondary"` + "Pending"
- `processing`: `variant="secondary"` + "Processing" (with spinner spinning)
- `ready`: `variant="secondary" className="bg-green-500/10 text-green-500"` + "Ready"
- `failed`: `variant="destructive"` + "Failed" (with `Tooltip` showing `errorMessage`)
</action>
<read_first>
- frontend/src/components/ui/badge.tsx
- frontend/src/components/ui/tooltip.tsx
</read_first>
<acceptance_criteria>
- `failed` status shows tooltip on hover.
- `processing` status shows a loading spinner or pulsing effect.
</acceptance_criteria>
</task>

<task id="6-02-03" requirements="UI-03, UI-06">
<action>
Create `frontend/src/components/DocumentListTable.tsx`.
Use `src/components/ui/table.tsx`.
Columns:
- Name (with icon based on `file_type`)
- Date (formatted `created_at`)
- Chunks (number)
- Status (`StatusBadge` component)
- Actions (`Button` with `Trash2` icon)
Include `AlertDialog` (via `shadcn/ui/dialog.tsx` or similar) for delete confirmation.
</action>
<read_first>
- frontend/src/components/ui/table.tsx
- frontend/src/components/ui/dialog.tsx
- frontend/src/pages/WatchList.tsx (table reference)
</read_first>
<acceptance_criteria>
- Deleting triggers a confirmation dialog.
- Table handles empty state by displaying a message/placeholder.
</acceptance_criteria>
</task>

## Verification Criteria

<must_haves>
- Component styling matches existing dashboard aesthetics.
- Labels are clear in both English/Vietnamese if required (standardize on Vietnamese if applicable, otherwise English).
- Table is responsive and handles long filenames gracefully (truncation).
</must_haves>

<automated>
- `cd frontend && npm test -- src/components/__tests__/DocumentListTable.test.tsx`
- `cd frontend && npm test -- src/components/__tests__/StatusBadge.test.tsx`
</automated>
