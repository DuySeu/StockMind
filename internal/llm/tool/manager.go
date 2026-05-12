package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandler is the function signature for custom tool implementations.
type ToolHandler func(ctx context.Context, args string) (string, error)

// ToolManager registers tool definitions and dispatches execution to handlers.
type ToolManager struct {
	mu       sync.RWMutex
	tools    map[string]mcp.Tool
	handlers map[string]ToolHandler
	defCache []mcp.Tool // rebuilt when nil after Register
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:    make(map[string]mcp.Tool),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds a tool definition along with its execution handler.
func (m *ToolManager) Register(def mcp.Tool, handler ToolHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[def.Name] = def
	m.handlers[def.Name] = handler
	m.defCache = nil
}

// GetDefinitions returns all registered tool definitions for passing to the LLM.
func (m *ToolManager) GetDefinitions() []mcp.Tool {
	m.mu.RLock()
	if m.defCache != nil {
		out := m.defCache
		m.mu.RUnlock()
		return out
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.defCache == nil {
		m.defCache = make([]mcp.Tool, 0, len(m.tools))
		for _, d := range m.tools {
			m.defCache = append(m.defCache, d)
		}
	}
	return m.defCache
}

// Execute runs the handler for the given tool name.
func (m *ToolManager) Execute(ctx context.Context, name string, args string) (string, error) {
	m.mu.RLock()
	handler, ok := m.handlers[name]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return handler(ctx, args)
}
