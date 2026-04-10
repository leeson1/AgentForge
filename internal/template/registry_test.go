package template

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg.Count() != 0 {
		t.Errorf("Expected 0 templates, got %d", reg.Count())
	}
}

func TestNewRegistryWithBuiltins(t *testing.T) {
	reg, err := NewRegistryWithBuiltins()
	if err != nil {
		t.Fatalf("NewRegistryWithBuiltins() error: %v", err)
	}

	// 应该有 default + 3 个内置模板 = 4
	if reg.Count() < 4 {
		t.Errorf("Expected at least 4 templates, got %d", reg.Count())
	}

	// 检查 default 模板
	tmpl, err := reg.Get("default")
	if err != nil {
		t.Fatalf("Get(default) error: %v", err)
	}
	if tmpl.InitializerPrompt == "" {
		t.Error("Default template should have InitializerPrompt")
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	tmpl := &Template{
		Config:            TemplateConfig{ID: "test", Name: "Test"},
		InitializerPrompt: "init",
		WorkerPrompt:      "worker",
	}
	reg.Register(tmpl)

	got, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Get(test) error: %v", err)
	}
	if got.Config.Name != "Test" {
		t.Errorf("Expected name 'Test', got %q", got.Config.Name)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent template")
	}
}

func TestRegistryGetOrDefault(t *testing.T) {
	reg := NewRegistry()
	reg.registerDefault()

	// 获取不存在的模板应返回 default
	tmpl := reg.GetOrDefault("nonexistent")
	if tmpl.Config.ID != "default" {
		t.Errorf("Expected default template, got %q", tmpl.Config.ID)
	}

	// 注册新模板后可以获取到
	reg.Register(&Template{
		Config:            TemplateConfig{ID: "custom", Name: "Custom"},
		InitializerPrompt: "custom init",
		WorkerPrompt:      "custom worker",
	})
	tmpl = reg.GetOrDefault("custom")
	if tmpl.Config.ID != "custom" {
		t.Errorf("Expected custom template, got %q", tmpl.Config.ID)
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Template{
		Config:            TemplateConfig{ID: "a", Name: "A"},
		InitializerPrompt: "init",
		WorkerPrompt:      "worker",
	})
	reg.Register(&Template{
		Config:            TemplateConfig{ID: "b", Name: "B"},
		InitializerPrompt: "init",
		WorkerPrompt:      "worker",
	})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 templates in list, got %d", len(list))
	}
}

func TestRegistryOverwrite(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Template{
		Config:            TemplateConfig{ID: "test", Name: "V1"},
		InitializerPrompt: "init",
		WorkerPrompt:      "worker",
	})
	reg.Register(&Template{
		Config:            TemplateConfig{ID: "test", Name: "V2"},
		InitializerPrompt: "init",
		WorkerPrompt:      "worker",
	})

	tmpl, _ := reg.Get("test")
	if tmpl.Config.Name != "V2" {
		t.Errorf("Expected name 'V2', got %q", tmpl.Config.Name)
	}
	if reg.Count() != 1 {
		t.Errorf("Expected 1 template after overwrite, got %d", reg.Count())
	}
}
