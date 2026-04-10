package template

import (
	"fmt"
	"sync"
)

// Registry 模板注册表
type Registry struct {
	mu        sync.RWMutex
	templates map[string]*Template
}

// NewRegistry 创建空注册表
func NewRegistry() *Registry {
	return &Registry{
		templates: make(map[string]*Template),
	}
}

// NewRegistryWithBuiltins 创建带有内置模板的注册表
func NewRegistryWithBuiltins() (*Registry, error) {
	reg := NewRegistry()

	// 加载内置模板
	builtins, err := LoadBuiltinTemplates()
	if err != nil {
		return nil, fmt.Errorf("load builtin templates: %w", err)
	}
	for _, tmpl := range builtins {
		reg.Register(tmpl)
	}

	// 注册默认模板（使用 prompt.go 中的默认 prompt）
	reg.registerDefault()

	// 加载自定义模板（不报错，只跳过）
	custom, _ := LoadCustomTemplates("")
	for _, tmpl := range custom {
		reg.Register(tmpl)
	}

	return reg, nil
}

// registerDefault 注册默认模板
func (r *Registry) registerDefault() {
	// 仅当 "default" 尚未注册时
	if _, exists := r.templates["default"]; exists {
		return
	}
	r.templates["default"] = &Template{
		Config: TemplateConfig{
			ID:          "default",
			Name:        "Default",
			Description: "General-purpose template for any project type",
			Category:    "general",
		},
		InitializerPrompt: DefaultInitializerPrompt,
		WorkerPrompt:      DefaultWorkerPrompt,
	}
}

// Register 注册模板
func (r *Registry) Register(tmpl *Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[tmpl.Config.ID] = tmpl
}

// Get 按 ID 获取模板
func (r *Registry) Get(id string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.templates[id]
	if !ok {
		return nil, ErrTemplateNotFound(id)
	}
	return tmpl, nil
}

// GetOrDefault 获取模板，不存在则返回默认模板
func (r *Registry) GetOrDefault(id string) *Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if tmpl, ok := r.templates[id]; ok {
		return tmpl
	}
	if tmpl, ok := r.templates["default"]; ok {
		return tmpl
	}
	// 兜底
	return &Template{
		Config:            TemplateConfig{ID: "default", Name: "Default"},
		InitializerPrompt: DefaultInitializerPrompt,
		WorkerPrompt:      DefaultWorkerPrompt,
	}
}

// List 列出所有模板
func (r *Registry) List() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Template, 0, len(r.templates))
	for _, tmpl := range r.templates {
		result = append(result, tmpl)
	}
	return result
}

// Count 返回已注册模板数
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.templates)
}
