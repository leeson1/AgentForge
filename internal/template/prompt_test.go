package template

import (
	"strings"
	"testing"
)

func TestRenderPrompt_BasicSubstitution(t *testing.T) {
	tmpl := "Hello {{name}}, welcome to {{project}}!"
	vars := map[string]string{
		"name":    "Alice",
		"project": "AgentForge",
	}

	result := RenderPrompt(tmpl, vars)
	expected := "Hello Alice, welcome to AgentForge!"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderPrompt_NoVars(t *testing.T) {
	tmpl := "No variables here"
	result := RenderPrompt(tmpl, nil)
	if result != tmpl {
		t.Errorf("got %q, want %q", result, tmpl)
	}
}

func TestRenderPrompt_UnmatchedPlaceholder(t *testing.T) {
	tmpl := "Hello {{name}}, your role is {{role}}"
	vars := map[string]string{
		"name": "Bob",
	}

	result := RenderPrompt(tmpl, vars)
	// 未匹配的 placeholder 保持原样
	if !strings.Contains(result, "{{role}}") {
		t.Errorf("unmatched placeholder should remain: %q", result)
	}
	if !strings.Contains(result, "Bob") {
		t.Errorf("matched placeholder should be replaced: %q", result)
	}
}

func TestRenderPrompt_MultipleOccurrences(t *testing.T) {
	tmpl := "{{x}} and {{x}} again"
	vars := map[string]string{"x": "foo"}

	result := RenderPrompt(tmpl, vars)
	if result != "foo and foo again" {
		t.Errorf("got %q, want %q", result, "foo and foo again")
	}
}

func TestRenderPrompt_EmptyValue(t *testing.T) {
	tmpl := "prefix-{{key}}-suffix"
	vars := map[string]string{"key": ""}

	result := RenderPrompt(tmpl, vars)
	if result != "prefix--suffix" {
		t.Errorf("got %q, want %q", result, "prefix--suffix")
	}
}

func TestDefaultInitializerPrompt_HasPlaceholders(t *testing.T) {
	if !strings.Contains(DefaultInitializerPrompt, "{{task_name}}") {
		t.Error("DefaultInitializerPrompt should contain {{task_name}}")
	}
	if !strings.Contains(DefaultInitializerPrompt, "{{task_description}}") {
		t.Error("DefaultInitializerPrompt should contain {{task_description}}")
	}
}

func TestDefaultWorkerPrompt_HasPlaceholders(t *testing.T) {
	placeholders := []string{
		"{{task_name}}", "{{session_number}}", "{{feature_id}}",
		"{{feature_description}}", "{{progress_content}}",
		"{{pending_features}}", "{{validator_command}}",
	}
	for _, p := range placeholders {
		if !strings.Contains(DefaultWorkerPrompt, p) {
			t.Errorf("DefaultWorkerPrompt should contain %s", p)
		}
	}
}

func TestRenderPrompt_InitializerTemplate(t *testing.T) {
	vars := map[string]string{
		"task_name":        "MyProject",
		"task_description": "Build a web app",
	}

	result := RenderPrompt(DefaultInitializerPrompt, vars)
	if strings.Contains(result, "{{task_name}}") {
		t.Error("task_name placeholder should be replaced")
	}
	if !strings.Contains(result, "MyProject") {
		t.Error("result should contain task name")
	}
	if !strings.Contains(result, "Build a web app") {
		t.Error("result should contain task description")
	}
}
