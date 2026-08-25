package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// SchemaProvider allows an input struct to define its own JSON Schema.
// If implemented, the custom schema is used instead of reflection-based inference.
type SchemaProvider interface {
	Schema() map[string]any
}

// Tool defines a function-callable tool for LLM providers.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
	Execute     func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error)
}

// Manager holds initialized tools. Safe for concurrent reads after creation.
type Manager struct {
	tools map[string]*Tool
	list  []*Tool
}

// NewTool creates a Tool with type-safe handler. Schema is inferred from struct tags,
// or from SchemaProvider interface if implemented.
func NewTool[In any](name, description string, h func(ctx context.Context, input In) (any, error)) *Tool {
	var zero In
	var schema map[string]any
	if sp, ok := any(zero).(SchemaProvider); ok {
		schema = sp.Schema()
	} else {
		schema = schemaFrom(zero)
	}

	return &Tool{
		Name:        name,
		Description: description,
		Schema:      schema,
		Execute: func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error) {
			var input In
			if err := json.Unmarshal(rawArgs, &input); err != nil {
				return nil, fmt.Errorf("parse input: %w", err)
			}
			result, err := h(ctx, input)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	}
}

// NewManager builds a Manager from the given tools.
func NewManager(tools []*Tool) *Manager {
	m := &Manager{tools: make(map[string]*Tool, len(tools))}
	for _, t := range tools {
		m.tools[t.Name] = t
		m.list = append(m.list, t)
	}
	return m
}

// All returns every registered tool definition (for sending to LLM providers).
func (m *Manager) All() []*Tool { return m.list }

// Subset returns the named tools in the order given, so a specialist agent can be
// offered only the tools it declares.
//
// Nil and empty mean different things, deliberately:
//   - nil   → no restriction; returns All(). This is the zero value, so callers
//     that don't care about scoping (the chat path) get every tool.
//   - empty → exactly zero tools. An agent with no tools (a synthesizer that only
//     reasons over prior findings) needs to express this, and "nil means all"
//     leaves no other way to say it.
//
// Unknown names are skipped rather than erroring: a tool may legitimately be
// absent, e.g. a bridged MCP server that failed to start.
func (m *Manager) Subset(names []string) []*Tool {
	if names == nil {
		return m.All()
	}
	out := make([]*Tool, 0, len(names))
	for _, name := range names {
		if t, ok := m.tools[name]; ok {
			out = append(out, t)
		}
	}
	return out
}

// Execute calls the named tool with raw JSON args from the LLM.
func (m *Manager) Execute(ctx context.Context, name string, rawArgs string) (string, error) {
	t, ok := m.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	result, err := t.Execute(ctx, json.RawMessage(rawArgs))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// schemaFrom generates a JSON Schema object from struct tags, using `json` for the
// field name and required-ness and `jsonschema` for the description.
// style: keep — inlining this into NewTool pushes that constructor past ~60 lines, and it shares no locals with it.
func schemaFrom(v any) map[string]any {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	props := map[string]any{}
	var required []string

	for i := range t.NumField() {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]

		// Map the Go kind onto a JSON Schema type.
		jsonType := "string"
		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			jsonType = "integer"
		case reflect.Float32, reflect.Float64:
			jsonType = "number"
		case reflect.Bool:
			jsonType = "boolean"
		}

		prop := map[string]any{"type": jsonType}
		if desc := f.Tag.Get("jsonschema"); desc != "" {
			prop["description"] = desc
		}
		if !strings.Contains(tag, "omitempty") {
			required = append(required, name)
		}
		props[name] = prop
	}

	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
