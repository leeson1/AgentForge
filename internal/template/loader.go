package template

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed builtin/*
var builtinFS embed.FS

// LoadBuiltinTemplates 从内嵌文件系统加载所有内置模板
func LoadBuiltinTemplates() ([]*Template, error) {
	var templates []*Template

	entries, err := fs.ReadDir(builtinFS, "builtin")
	if err != nil {
		return nil, fmt.Errorf("read builtin templates dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tmpl, err := loadFromFS(builtinFS, "builtin/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("load builtin template %q: %w", entry.Name(), err)
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// LoadCustomTemplates 从自定义目录加载模板
// 默认路径: ~/.agent-forge/templates/
func LoadCustomTemplates(dir string) ([]*Template, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil // 无法获取 home 目录，跳过
		}
		dir = filepath.Join(home, ".agent-forge", "templates")
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // 目录不存在，返回空
	}

	var templates []*Template
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read custom templates dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tmplDir := filepath.Join(dir, entry.Name())
		tmpl, err := loadFromDisk(tmplDir)
		if err != nil {
			return nil, fmt.Errorf("load custom template %q: %w", entry.Name(), err)
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// loadFromFS 从 embed.FS 加载单个模板
func loadFromFS(fsys embed.FS, dir string) (*Template, error) {
	// 读取 template.json
	configData, err := fs.ReadFile(fsys, dir+"/template.json")
	if err != nil {
		return nil, ErrTemplateLoad(dir, "template.json not found")
	}

	var config TemplateConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, ErrTemplateLoad(dir, fmt.Sprintf("invalid template.json: %v", err))
	}

	tmpl := &Template{
		Config:   config,
		BasePath: dir,
	}

	// 加载 prompt 文件
	if config.Prompts.Initializer != "" {
		data, err := fs.ReadFile(fsys, dir+"/"+config.Prompts.Initializer)
		if err != nil {
			return nil, ErrTemplateLoad(config.ID, "initializer prompt file not found: "+config.Prompts.Initializer)
		}
		tmpl.InitializerPrompt = string(data)
	}

	if config.Prompts.Worker != "" {
		data, err := fs.ReadFile(fsys, dir+"/"+config.Prompts.Worker)
		if err != nil {
			return nil, ErrTemplateLoad(config.ID, "worker prompt file not found: "+config.Prompts.Worker)
		}
		tmpl.WorkerPrompt = string(data)
	}

	// 加载 validator 脚本
	if config.Validator != "" {
		data, err := fs.ReadFile(fsys, dir+"/"+config.Validator)
		if err == nil {
			tmpl.ValidatorScript = string(data)
		}
	}

	// 加载 hook 脚本
	if config.Hooks.OnSessionStart != nil && *config.Hooks.OnSessionStart != "" {
		data, err := fs.ReadFile(fsys, dir+"/"+*config.Hooks.OnSessionStart)
		if err == nil {
			tmpl.HookStartScript = string(data)
		}
	}

	if config.Hooks.OnSessionEnd != nil && *config.Hooks.OnSessionEnd != "" {
		data, err := fs.ReadFile(fsys, dir+"/"+*config.Hooks.OnSessionEnd)
		if err == nil {
			tmpl.HookEndScript = string(data)
		}
	}

	if err := tmpl.Validate(); err != nil {
		return nil, err
	}

	return tmpl, nil
}

// loadFromDisk 从文件系统目录加载单个模板
func loadFromDisk(dir string) (*Template, error) {
	// 读取 template.json
	configPath := filepath.Join(dir, "template.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, ErrTemplateLoad(dir, "template.json not found")
	}

	var config TemplateConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, ErrTemplateLoad(dir, fmt.Sprintf("invalid template.json: %v", err))
	}

	tmpl := &Template{
		Config:   config,
		BasePath: dir,
	}

	// 加载 prompt 文件
	if config.Prompts.Initializer != "" {
		data, err := os.ReadFile(filepath.Join(dir, config.Prompts.Initializer))
		if err != nil {
			return nil, ErrTemplateLoad(config.ID, "initializer prompt file not found")
		}
		tmpl.InitializerPrompt = string(data)
	}

	if config.Prompts.Worker != "" {
		data, err := os.ReadFile(filepath.Join(dir, config.Prompts.Worker))
		if err != nil {
			return nil, ErrTemplateLoad(config.ID, "worker prompt file not found")
		}
		tmpl.WorkerPrompt = string(data)
	}

	// 加载 validator
	if config.Validator != "" {
		data, err := os.ReadFile(filepath.Join(dir, config.Validator))
		if err == nil {
			tmpl.ValidatorScript = string(data)
		}
	}

	// 加载 hooks
	if config.Hooks.OnSessionStart != nil && *config.Hooks.OnSessionStart != "" {
		data, err := os.ReadFile(filepath.Join(dir, *config.Hooks.OnSessionStart))
		if err == nil {
			tmpl.HookStartScript = string(data)
		}
	}

	if config.Hooks.OnSessionEnd != nil && *config.Hooks.OnSessionEnd != "" {
		data, err := os.ReadFile(filepath.Join(dir, *config.Hooks.OnSessionEnd))
		if err == nil {
			tmpl.HookEndScript = string(data)
		}
	}

	if err := tmpl.Validate(); err != nil {
		return nil, err
	}

	return tmpl, nil
}
