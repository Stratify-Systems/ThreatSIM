package plugins

import (
	"context"
	"testing"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

// mockPlugin is a minimal plugin implementation for testing the registry.
type mockPlugin struct {
	id   string
	name string
}

func (p *mockPlugin) ID() string                { return p.id }
func (p *mockPlugin) Name() string              { return p.name }
func (p *mockPlugin) Description() string       { return "mock plugin" }
func (p *mockPlugin) DefaultConfig() core.PluginConfig { return core.PluginConfig{} }
func (p *mockPlugin) Execute(ctx context.Context, config core.PluginConfig, sink core.EventSink) error {
	return nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	plugin := &mockPlugin{id: "test_plugin", name: "Test Plugin"}
	if err := r.Register(plugin); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := r.Get("test_plugin")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID() != "test_plugin" {
		t.Errorf("expected ID 'test_plugin', got '%s'", got.ID())
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	plugin := &mockPlugin{id: "dup", name: "Duplicate"}
	r.Register(plugin)

	err := r.Register(plugin)
	if err == nil {
		t.Fatal("expected error when registering duplicate plugin")
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockPlugin{id: "a", name: "A"})
	r.Register(&mockPlugin{id: "b", name: "B"})
	r.Register(&mockPlugin{id: "c", name: "C"})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(list))
	}
}

func TestRegistry_IDs(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockPlugin{id: "alpha", name: "Alpha"})
	r.Register(&mockPlugin{id: "beta", name: "Beta"})

	ids := r.IDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet["alpha"] || !idSet["beta"] {
		t.Errorf("expected IDs alpha and beta, got %v", ids)
	}
}
