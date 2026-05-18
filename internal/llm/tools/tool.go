package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	kb "stockmind/internal/knowledge_base"
)

// ──────── Tool Definition ────────

// HandlerFunc is the execution function signature for a tool.
type HandlerFunc func(ctx context.Context, args map[string]any) (map[string]any, error)

// HandlerFactory creates a HandlerFunc given dependencies. Used for auto-registration.
type HandlerFactory func(deps InternalToolDeps) HandlerFunc

// Tool is a self-contained tool definition with schema and execution logic.
type Tool struct {
	name        string
	description string
	schema      map[string]any
	handler     HandlerFunc
}

// NewTool creates a Tool. Used by internal/llm/tools and mcp_client to build tools.
func NewTool(name, description string, schema map[string]any, handler HandlerFunc) *Tool {
	return &Tool{name: name, description: description, schema: schema, handler: handler}
}

func (t *Tool) Name() string                { return t.name }
func (t *Tool) Description() string         { return t.description }
func (t *Tool) InputSchema() map[string]any { return t.schema }

// Execute runs the tool with the given parsed arguments.
func (t *Tool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.handler(ctx, args)
}

// ──────── Auto-Registration ────────

// ToolSchema holds the static definition of a tool (sent to LLM providers).
type ToolSchema struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// InternalToolDeps holds dependencies needed by internal tools.
type InternalToolDeps struct {
	Retriever kb.Retriever
}

// toolDef is a deferred tool definition (schema + factory).
type toolDef struct {
	schema  ToolSchema
	factory HandlerFactory
}

var registry []toolDef

// Register queues a tool definition for later initialization.
// Call this in init() of each handler file.
func Register(schema ToolSchema, factory HandlerFactory) {
	registry = append(registry, toolDef{schema: schema, factory: factory})
}

// ──────── Tool Manager ────────

// ToolManager registers tools and dispatches execution.
type ToolManager struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewToolManager creates a ToolManager with all registered internal tools.
func NewToolManager(deps InternalToolDeps) *ToolManager {
	mgr := &ToolManager{tools: make(map[string]*Tool)}
	for _, def := range registry {
		handler := def.factory(deps)
		mgr.add(NewTool(def.schema.Name, def.schema.Description, def.schema.InputSchema, handler))
	}
	return mgr
}

// add adds a tool to the registry.
func (m *ToolManager) add(tool *Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name()] = tool
}

// RegisterExternal adds an external tool (e.g. from MCP client) at runtime.
func (m *ToolManager) RegisterExternal(tool *Tool) {
	m.add(tool)
}

// GetDefinitions returns all registered tools for passing to providers.
func (m *ToolManager) GetDefinitions() []*Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Tool, 0, len(m.tools))
	for _, t := range m.tools {
		out = append(out, t)
	}
	return out
}

// Execute runs the named tool, unmarshalling rawArgs from JSON first.
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
