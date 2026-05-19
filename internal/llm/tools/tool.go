package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	kb "stockmind/internal/knowledge_base"
)

// Deps holds shared dependencies available to tool handlers.
type Deps struct {
	Retriever kb.Retriever
}

// Tool defines a function-callable tool for LLM providers.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
	// Run executes the tool. rawArgs is the JSON string from the LLM.
	Execute func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error)
}

// ──────── Registration ────────

type registration struct {
	name, description string
	schema            map[string]any
	build             func(deps Deps) func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error)
}

var registrations []registration

// AddTool registers a typed tool. Schema is inferred from In struct tags.
// Handler receives pre-parsed input and returns any JSON-serializable value.
func AddTool[In any](name, description string, h func(ctx context.Context, deps Deps, input In) (any, error)) {
	var zero In
	schema := schemaFrom(zero)
	registrations = append(registrations, registration{
		name:        name,
		description: description,
		schema:      schema,
		build: func(deps Deps) func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error) {
			return func(ctx context.Context, rawArgs json.RawMessage) (json.RawMessage, error) {
				var input In
				if err := json.Unmarshal(rawArgs, &input); err != nil {
					return nil, fmt.Errorf("parse input: %w", err)
				}
				result, err := h(ctx, deps, input)
				if err != nil {
					return nil, err
				}
				return json.Marshal(result)
			}
		},
	})
}

// ──────── Manager ────────

// Manager holds initialized tools. Safe for concurrent reads after creation.
type Manager struct {
	tools map[string]*Tool
	list  []*Tool
}

// NewManager builds all registered tools with the given dependencies.
func NewManager(deps Deps) *Manager {
	m := &Manager{tools: make(map[string]*Tool, len(registrations))}
	for _, r := range registrations {
		t := &Tool{
			Name:        r.name,
			Description: r.description,
			Schema:      r.schema,
			Execute:     r.build(deps),
		}
		m.tools[r.name] = t
		m.list = append(m.list, t)
	}
	return m
}

// All returns every registered tool definition (for sending to LLM providers).
func (m *Manager) All() []*Tool { return m.list }

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

// ──────── Schema Inference ────────

// schemaFrom generates a JSON Schema object from struct tags.
// Uses `json` tag for field name/required, `jsonschema` tag for description.
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

		prop := map[string]any{"type": goTypeToJSON(f.Type.Kind())}
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

func goTypeToJSON(k reflect.Kind) string {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		return "string"
	}
}
