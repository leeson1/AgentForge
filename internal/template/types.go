package template

// TemplateConfig 模板配置（对应 template.json）
type TemplateConfig struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Prompts     PromptPaths       `json:"prompts"`
	Hooks       HookPaths         `json:"hooks"`
	Validator   string            `json:"validator"`
	Variables   map[string]string `json:"variables"`
}

// PromptPaths prompt 文件路径配置
type PromptPaths struct {
	Initializer string `json:"initializer"`
	Worker      string `json:"worker"`
}

// HookPaths hook 脚本路径配置
type HookPaths struct {
	OnSessionStart *string `json:"on_session_start"`
	OnSessionEnd   *string `json:"on_session_end"`
}

// Template 加载后的完整模板
type Template struct {
	Config            TemplateConfig
	InitializerPrompt string
	WorkerPrompt      string
	ValidatorScript   string
	HookStartScript   string
	HookEndScript     string
	BasePath          string // 模板文件所在目录
}

// Validate 校验模板必要字段
func (t *Template) Validate() error {
	if t.Config.ID == "" {
		return ErrMissingField("id")
	}
	if t.Config.Name == "" {
		return ErrMissingField("name")
	}
	if t.InitializerPrompt == "" {
		return ErrMissingField("initializer prompt")
	}
	if t.WorkerPrompt == "" {
		return ErrMissingField("worker prompt")
	}
	return nil
}
