# Local Tool Schema Design

Remove `mcp-go` dependency from `internal/llm/` by defining a local tool type with builder pattern.

## Motivation

- Clean architecture: `internal/llm` should not import an external protocol library for a struct shape
- Decoupling: enables swapping or removing `mcp-go` without touching the core LLM orchestration

## Design

### File Layout

```
internal/llm/
  tool.go           → Tool struct, ToolInputSchema, HandlerFunc type, methods
  tools.go          → All tool definitions (constructs Tool instances with schemas + handlers)
  tool_handlers.go  → Handler factory functions (one per tool)
  tool_manager.go   → ToolManager (registry + dispatch, updated to use new Tool type)
```

### `tool.go` — Core Types

```go
// HandlerFunc is the execution function signature for a tool.
type HandlerFunc func(ctx context.Context, args map[string]any) (map[string]any, error)

// ToolInputSchema describes expected parameters as JSON Schema.
type ToolInputSchema struct {
    Type       string
    Properties map[string]any
    Required   []string
}

// Tool is a self-contained tool definition with schema and execution logic.
type Tool struct {
    name        string
    description string
    schema      ToolInputSchema
    handler     HandlerFunc
}

func (t *Tool) Name() string                 { return t.name }
func (t *Tool) Description() string          { return t.description }
func (t *Tool) InputSchema() ToolInputSchema { return t.schema }
func (t *Tool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
    return t.handler(ctx, args)
}
```

### `tool_handlers.go` — Handler Factories

Each handler factory takes dependencies and returns a `HandlerFunc`:

```go
func newRetrieveKnowledgeHandler(retriever kb.Retriever) HandlerFunc {
    return func(ctx context.Context, args map[string]any) (map[string]any, error) {
        query, _ := args["query"].(string)
        if strings.TrimSpace(query) == "" {
            return nil, fmt.Errorf("query is required")
        }

        results, err := retriever.Search(ctx, query, kb.SearchHybrid, 5)
        if err != nil {
            return nil, fmt.Errorf("knowledge base search failed: %w", err)
        }

        if len(results) == 0 {
            return map[string]any{"content": "No relevant information found in knowledge base."}, nil
        }

        var sb strings.Builder
        sb.WriteString(fmt.Sprintf("Found %d relevant chunks:\n\n", len(results)))
        for i, res := range results {
            sb.WriteString(fmt.Sprintf("--- Source %d | Doc: %s, Chunk: %d ---\n", i+1, res.DocID, res.ChunkIndex))
            sb.WriteString(res.Text)
            sb.WriteString("\n\n")
        }

        return map[string]any{"content": sb.String()}, nil
    }
}
```

### `tools.go` — Tool Definitions

Each tool has a constructor that wires schema + handler:

```go
func NewRetrieveKnowledgeTool(retriever kb.Retriever) *Tool {
    return &Tool{
        name:        "retrieve_knowledge",
        description: "Retrieve detailed financial knowledge, concepts, definitions, or internal document information from the knowledge base. Use this for general queries, not for real-time stock prices or latest news.",
        schema: ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "query": map[string]any{
                    "type":        "string",
                    "description": "Query related to financial knowledge or concepts",
                },
            },
            Required: []string{"query"},
        },
        handler: newRetrieveKnowledgeHandler(retriever),
    }
}
```

### `tool_manager.go` — Registry + Dispatch

```go
type ToolManager struct {
    mu    sync.RWMutex
    tools map[string]*Tool
}

func NewToolManager() *ToolManager {
    return &ToolManager{tools: make(map[string]*Tool)}
}

func (m *ToolManager) Register(tool *Tool) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.tools[tool.Name()] = tool
}

func (m *ToolManager) GetDefinitions() []*Tool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    out := make([]*Tool, 0, len(m.tools))
    for _, t := range m.tools {
        out = append(out, t)
    }
    return out
}

func (m *ToolManager) Execute(ctx context.Context, name string, rawArgs string) (string, error) {
    m.mu.RLock()
    tool, ok := m.tools[name]
    m.mu.RUnlock()
    if !ok {
        return "", fmt.Errorf("tool not found: %s", name)
    }

    var args map[string]any
    if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
        return "", fmt.Errorf("invalid arguments: %w", err)
    }

    result, err := tool.Execute(ctx, args)
    if err != nil {
        return "", err
    }

    out, err := json.Marshal(result)
    if err != nil {
        return "", fmt.Errorf("failed to marshal tool result: %w", err)
    }
    return string(out), nil
}
```

### Provider Adaptation

All providers change signature from `[]mcp.Tool` to `[]*Tool` and access fields via methods:

```go
// completionFunc type
type completionFunc func(context.Context, []database.Message, []*Tool) (<-chan database.StreamEvent, error)

// Provider functions
func OpenRouterCompletion(..., tools []*Tool) ...
func OpenAICompletion(..., tools []*Tool) ...
func AnthropicCompletion(..., tools []*Tool) ...

// Accessing tool data
t.Name(), t.Description(), t.InputSchema()
```

`schemaToMap` in `openrouter.go` simplifies:

```go
func schemaToMap(schema ToolInputSchema) map[string]any {
    m := map[string]any{"type": schema.Type}
    if schema.Properties != nil {
        m["properties"] = schema.Properties
    }
    if len(schema.Required) > 0 {
        m["required"] = schema.Required
    }
    return m
}
```

### Registration in `service.go`

```go
toolMgr := NewToolManager()
toolMgr.Register(NewRetrieveKnowledgeTool(knowledgeBase.Retriever))
```

## Files Changed

| File | Change |
|------|--------|
| `internal/llm/tool.go` (new) | Tool struct, ToolInputSchema, HandlerFunc, methods |
| `internal/llm/tools.go` (new) | NewRetrieveKnowledgeTool and future constructors |
| `internal/llm/tool_handlers.go` (new) | newRetrieveKnowledgeHandler and future factories |
| `internal/llm/tool_manager.go` | Simplified to store `map[string]*Tool` |
| `internal/llm/service.go` | Remove mcp import, use NewRetrieveKnowledgeTool, update completionFunc |
| `internal/llm/openrouter.go` | `[]mcp.Tool` → `[]*Tool`, simplify schemaToMap |
| `internal/llm/openai.go` | `[]mcp.Tool` → `[]*Tool`, access via methods |
| `internal/llm/anthropic.go` | `[]mcp.Tool` → `[]*Tool`, access via methods |
| `internal/server/server.go` | Remove duplicate retrieveKnowledgeHandler |

## Adding New Tools

1. Add handler factory in `tool_handlers.go`: `func newXxxHandler(deps) HandlerFunc`
2. Add constructor in `tools.go`: `func NewXxxTool(deps) *Tool`
3. Register in `service.go`: `toolMgr.Register(NewXxxTool(deps))`
