package template

import "fmt"

// ErrMissingField 模板缺少必要字段
func ErrMissingField(field string) error {
	return fmt.Errorf("template missing required field: %s", field)
}

// ErrTemplateNotFound 模板未找到
func ErrTemplateNotFound(id string) error {
	return fmt.Errorf("template not found: %s", id)
}

// ErrTemplateLoad 模板加载失败
func ErrTemplateLoad(id, reason string) error {
	return fmt.Errorf("failed to load template %q: %s", id, reason)
}
