package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltinTemplates(t *testing.T) {
	templates, err := LoadBuiltinTemplates()
	if err != nil {
		t.Fatalf("LoadBuiltinTemplates() error: %v", err)
	}

	if len(templates) < 3 {
		t.Fatalf("Expected at least 3 builtin templates, got %d", len(templates))
	}

	// 检查已知模板
	ids := map[string]bool{}
	for _, tmpl := range templates {
		ids[tmpl.Config.ID] = true
	}

	for _, expected := range []string{"fullstack-web", "cli-tool", "data-analysis"} {
		if !ids[expected] {
			t.Errorf("Missing builtin template: %s", expected)
		}
	}
}

func TestLoadBuiltinTemplate_FullstackWeb(t *testing.T) {
	templates, err := LoadBuiltinTemplates()
	if err != nil {
		t.Fatalf("LoadBuiltinTemplates() error: %v", err)
	}

	var web *Template
	for _, tmpl := range templates {
		if tmpl.Config.ID == "fullstack-web" {
			web = tmpl
			break
		}
	}

	if web == nil {
		t.Fatal("fullstack-web template not found")
	}

	if web.Config.Name == "" {
		t.Error("Name should not be empty")
	}
	if web.Config.Category != "web" {
		t.Errorf("Expected category 'web', got %q", web.Config.Category)
	}
	if web.InitializerPrompt == "" {
		t.Error("InitializerPrompt should not be empty")
	}
	if web.WorkerPrompt == "" {
		t.Error("WorkerPrompt should not be empty")
	}
	if web.ValidatorScript == "" {
		t.Error("ValidatorScript should not be empty")
	}
	if web.HookStartScript == "" {
		t.Error("HookStartScript should not be empty")
	}
}

func TestLoadBuiltinTemplate_CLITool(t *testing.T) {
	templates, err := LoadBuiltinTemplates()
	if err != nil {
		t.Fatalf("LoadBuiltinTemplates() error: %v", err)
	}

	var cli *Template
	for _, tmpl := range templates {
		if tmpl.Config.ID == "cli-tool" {
			cli = tmpl
			break
		}
	}

	if cli == nil {
		t.Fatal("cli-tool template not found")
	}

	if cli.Config.Category != "cli" {
		t.Errorf("Expected category 'cli', got %q", cli.Config.Category)
	}
	if cli.ValidatorScript == "" {
		t.Error("ValidatorScript should not be empty")
	}
	// cli-tool 没有 hook
	if cli.HookStartScript != "" {
		t.Error("HookStartScript should be empty for cli-tool")
	}
}

func TestLoadCustomTemplates_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	templates, err := LoadCustomTemplates(dir)
	if err != nil {
		t.Fatalf("LoadCustomTemplates() error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("Expected 0 templates from empty dir, got %d", len(templates))
	}
}

func TestLoadCustomTemplates_NonExistentDir(t *testing.T) {
	templates, err := LoadCustomTemplates("/nonexistent/path/templates")
	if err != nil {
		t.Fatalf("Should not error on nonexistent dir: %v", err)
	}
	if templates != nil {
		t.Error("Expected nil for nonexistent dir")
	}
}

func TestLoadCustomTemplates_ValidTemplate(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "test-template")
	os.MkdirAll(tmplDir, 0755)

	// 写入 template.json
	config := TemplateConfig{
		ID:          "test-template",
		Name:        "Test Template",
		Description: "A test template",
		Category:    "test",
		Prompts: PromptPaths{
			Initializer: "init.txt",
			Worker:      "worker.txt",
		},
		Validator: "check.sh",
	}
	configJSON, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmplDir, "template.json"), configJSON, 0644)
	os.WriteFile(filepath.Join(tmplDir, "init.txt"), []byte("Init prompt {{task_name}}"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "worker.txt"), []byte("Worker prompt {{feature_id}}"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "check.sh"), []byte("#!/bin/bash\nexit 0"), 0755)

	templates, err := LoadCustomTemplates(dir)
	if err != nil {
		t.Fatalf("LoadCustomTemplates() error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template, got %d", len(templates))
	}

	tmpl := templates[0]
	if tmpl.Config.ID != "test-template" {
		t.Errorf("Expected ID 'test-template', got %q", tmpl.Config.ID)
	}
	if tmpl.InitializerPrompt != "Init prompt {{task_name}}" {
		t.Errorf("Unexpected initializer prompt: %q", tmpl.InitializerPrompt)
	}
}

func TestLoadCustomTemplates_MissingPrompt(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "broken")
	os.MkdirAll(tmplDir, 0755)

	config := TemplateConfig{
		ID:   "broken",
		Name: "Broken",
		Prompts: PromptPaths{
			Initializer: "missing.txt",
			Worker:      "worker.txt",
		},
	}
	configJSON, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmplDir, "template.json"), configJSON, 0644)

	_, err := LoadCustomTemplates(dir)
	if err == nil {
		t.Fatal("Expected error for missing prompt file")
	}
}

func TestTemplateValidate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		wantErr bool
	}{
		{
			name: "valid",
			tmpl: Template{
				Config:            TemplateConfig{ID: "test", Name: "Test"},
				InitializerPrompt: "prompt",
				WorkerPrompt:      "prompt",
			},
			wantErr: false,
		},
		{
			name: "missing id",
			tmpl: Template{
				Config:            TemplateConfig{Name: "Test"},
				InitializerPrompt: "prompt",
				WorkerPrompt:      "prompt",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			tmpl: Template{
				Config:            TemplateConfig{ID: "test"},
				InitializerPrompt: "prompt",
				WorkerPrompt:      "prompt",
			},
			wantErr: true,
		},
		{
			name: "missing initializer",
			tmpl: Template{
				Config:       TemplateConfig{ID: "test", Name: "Test"},
				WorkerPrompt: "prompt",
			},
			wantErr: true,
		},
		{
			name: "missing worker",
			tmpl: Template{
				Config:            TemplateConfig{ID: "test", Name: "Test"},
				InitializerPrompt: "prompt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
