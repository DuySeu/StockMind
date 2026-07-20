# Code Conventions

## Go Backend

### Import Grouping
Three groups separated by blank lines:
1. Standard library
2. Internal packages (`stockmind/internal/...`)
3. Third-party packages

```go
import (
    "context"
    "fmt"
    "net/http"

    "stockmind/internal/common"
    "stockmind/internal/database"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
)
```

### Package Aliases
- `kb` for `knowledge_base`
- `core` for `llm`
- `pb` for Qdrant protobuf client

### Naming
- Exported: PascalCase (`LLMService`, `ToolManager`, `WriteJSON`)
- Unexported: camelCase (`completionFunc`, `defCache`)
- Acronyms fully capitalized: `LLM`, `SSE`, `DB`, `MCP`, `RRF`
- Constructors: `New*` prefix (`NewServer`, `NewLLMService`, `NewToolManager`)
- Handlers: `*Handler` suffix (`UploadDocumentHandler`, `ListDocumentsHandler`)
- Verb-prefixed functions: `Write*`, `Get*`, `Create*`, `Flush*`

### Error Handling
- Wrap with context: `fmt.Errorf("stage description: %w", err)`
- HTTP handlers: `http.Error(w, msg, statusCode)` or `common.WriteJSONError(w, statusCode, msg)`
- Streaming: send `StreamEvent{Type: EventError}` on channel
- Fatal init errors: `log.Fatalf`
- Non-critical: `log.Printf` and continue
- No custom error types — use `fmt.Errorf` wrapping

### Struct Patterns
- Dependency injection via interfaces in struct fields
- Unexported fields for internal state, exported for service-level access
- Constructor returns `(*T, error)` when fallible, `*T` when infallible

```go
type Server struct {
    queries        *database.Queries
    dbPool         *pgxpool.Pool
    agent          *core.LLMService
    knowledgeStore kb.Store
    objectStore    storage.ObjectStore
    service        *service.Service
}
```

### Interfaces
- Used for dependency injection: `Store`, `Embedder`, `ObjectStore`, `Retriever`
- Function types as lightweight interfaces: `ToolHandler`, `completionFunc`
- Define interfaces where consumed, not where implemented

### Concurrency
- Goroutines + channels for streaming responses
- `sync.RWMutex` for shared state (ToolManager)
- Unbuffered channels for backpressure
- Context threaded through all operations

### Logging
- `log.Printf` / `log.Println` / `log.Fatalf` (stdlib only)
- No structured logging library
- Log non-fatal errors, propagate fatal ones

### Comments
- Doc comments on all exported types and functions
- Inline `//` for implementation notes
- Section separators: `// ──────── Section Name ────────`

---

## React Frontend

### Import Ordering
Grouped by category (no strict alphabetical enforcement):
1. React / react-router-dom
2. Local API modules (`@/api/*`)
3. Internal components (`@/components/*`)
4. UI primitives (`@/components/ui/*`)
5. Types (`@/types/*`)
6. External libraries (lucide-react, react-hook-form, etc.)
7. Utilities (`@/lib/*`)

### Path Aliases
- `@/` maps to `src/` — use consistently for all internal imports

### Naming
- Components: PascalCase (`ChatbotPage`, `MessageList`, `DocumentUploadForm`)
- Hooks: camelCase with `use` prefix (`useDocumentPolling`)
- Handlers: camelCase with `handle` prefix (`handleFileClick`, `handleSubmit`)
- State variables: camelCase (`isLoading`, `messages`, `serverError`)
- Types/Interfaces: PascalCase (`Message`, `ChatEvent`, `ValidationResult`)

### Type Definitions
- `type` for unions and inferred types: `type ChatEventType = "start" | "text" | ...`
- `interface` for object shapes (API contracts): `interface ChatMessage { ... }`
- Zod-inferred types: `type FormValues = z.infer<typeof formSchema>`
- Discriminated unions with `type` field for metadata

### Exports
- Named exports for utilities, hooks, API functions, types
- Default exports for page components (consumed by router)
- `export const` for functions, `export type`/`export interface` for types

### Component Structure
```tsx
// 1. Schema/constants (if any)
const formSchema = z.object({ ... })

// 2. Component function
export default function PageName() {
  // 3. Hooks (navigation, params, state, refs, form)
  const navigate = useNavigate()
  const [state, setState] = useState<Type>(initial)

  // 4. Effects
  useEffect(() => { ... }, [deps])

  // 5. Handlers
  const handleAction = () => { ... }

  // 6. Return JSX
  return (...)
}
```

### State Management
- Local `useState` + `useRef` only — no global state library
- `useRef` for mutable values that shouldn't trigger re-renders (streaming flags, intervals)
- Saved callback ref pattern for intervals/polling

### Forms
- react-hook-form + zod resolver for validation
- Vietnamese error messages in validation schemas

### Error Handling
- API calls: try/catch with `toast.error()` for user-facing errors
- Validation: structured `{ valid, error }` result objects
- API modules: let errors propagate or use callback pattern (`onError`)
- Swallow `AbortError` intentionally for cancelled requests

### Styling
- Tailwind CSS utility classes exclusively
- `cn()` helper (clsx + tailwind-merge) for conditional classes
- No CSS modules or styled-components

### SSE Handling
- Chat: native `fetch` + `ReadableStream` with manual frame parsing
- Research: Axios `onDownloadProgress` with incremental text parsing
- No `EventSource` API used

### File Organization
```
api/          → HTTP client functions (one file per domain)
types/        → Shared TypeScript type definitions
lib/          → Pure utility functions
hooks/        → Custom React hooks
components/
  ui/         → shadcn/ui primitives (don't modify)
  containers/ → Composed business components
  layout/     → Layout wrappers (MainLayout, Navbar)
pages/        → Route-level page components
```

---

## Shared Conventions

- **i18n**: Mixed Vietnamese and English (no i18n library). UI labels often Vietnamese, code comments mostly English.
- **Dead code**: Commented-out code preserved for reference (acceptable during active development).
- **Config**: All configuration via environment variables, loaded at startup.
- **Database changes**: Edit SQL in `schema/queries/*.sql`, then run `sqlc generate`.
