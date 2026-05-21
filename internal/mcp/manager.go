package mcp

import (
	"context"
	"fmt"
	"sync"
)

// ServerConfig stores the parameters required to spawn and configure an external MCP server.
type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// Manager orchestrates a thread-safe registry of multiple external MCP clients.
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
	configs map[string]ServerConfig
}

// NewManager creates a new Manager instance from the provided list of server configurations.
func NewManager(configs []ServerConfig) *Manager {
	cfgMap := make(map[string]ServerConfig)
	for _, cfg := range configs {
		cfgMap[cfg.Name] = cfg
	}
	return &Manager{
		clients: make(map[string]*Client),
		configs: cfgMap,
	}
}

// GetOrStart retrieves an existing running MCP client or lazily starts a new session.
func (m *Manager) GetOrStart(ctx context.Context, name string) (*Client, error) {
	m.mu.RLock()
	client, exists := m.clients[name]
	m.mu.RUnlock()
	if exists {
		return client, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after lock
	if client, exists = m.clients[name]; exists {
		return client, nil
	}

	config, ok := m.configs[name]
	if !ok {
		return nil, fmt.Errorf("no config found for MCP server: %s", name)
	}

	newClient, err := New(ctx, config.Command, config.Args, config.Env)
	if err != nil {
		return nil, fmt.Errorf("failed to start MCP server %s: %w", name, err)
	}

	m.clients[name] = newClient
	return newClient, nil
}

// ActiveServers returns the names of all configured servers.
func (m *Manager) ActiveServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.configs))
	for name := range m.configs {
		names = append(names, name)
	}
	return names
}

// CloseAll cleanly terminates all active MCP client sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, client := range m.clients {
		_ = client.Close()
		delete(m.clients, name)
	}
}
