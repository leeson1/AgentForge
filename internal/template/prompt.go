package template

import (
	"strings"
)

// RenderPrompt 对 prompt 模板进行变量插值
// 支持 {{variable_name}} 语法
func RenderPrompt(tmpl string, vars map[string]string) string {
	result := tmpl
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// DefaultInitializerPrompt 默认的 Initializer Agent prompt 模板
const DefaultInitializerPrompt = `You are an Initializer Agent for the project "{{task_name}}".

## Project Description
{{task_description}}

## Your Task
Analyze the project requirements and create the following files in the working directory:

### 1. feature_list.json
Create a comprehensive feature list in this exact JSON format:
{
  "features": [
    {
      "id": "F001",
      "category": "functional",
      "description": "Brief description of the feature",
      "steps": ["Step 1", "Step 2", "Step 3"],
      "depends_on": [],
      "batch": null,
      "passes": false
    }
  ]
}

Rules for feature_list.json:
- Each feature MUST have a unique "id" (format: F001, F002, ...)
- "depends_on" lists the IDs of features that must be completed first
- Do NOT create circular dependencies
- "passes" must always be false (you are not implementing features)
- "batch" must always be null (the scheduler will assign batches)
- Break down the project into 5-20 granular, independently implementable features
- Order features logically: foundational features first, dependent features later

### 2. init.sh
Create a shell script that:
- Sets up the development environment
- Installs dependencies
- Starts any development servers needed
- Make it executable (it will be run with bash)

### 3. progress.txt
Create an initial progress file with:
- Project name and description
- Date of initialization
- Summary of generated features
- Status: "Initialization complete. Ready for development."

## Important Rules
- Only create the files listed above
- Do NOT implement any features
- Do NOT modify any existing project files beyond what's needed for setup
- Make a git commit with message "Initial setup by AgentForge Initializer"
- Focus on accuracy of the feature decomposition and dependency relationships
`

// DefaultWorkerPrompt 默认的 Worker Agent prompt 模板
const DefaultWorkerPrompt = `You are a Worker Agent for the project "{{task_name}}".
This is Session #{{session_number}}.

## Your Assigned Feature
Feature ID: {{feature_id}}
Description: {{feature_description}}

## Previous Progress
{{progress_content}}

## Pending Features
{{pending_features}}

## Rules
1. Check intervention.txt before each action for user instructions
2. Do NOT delete or modify existing entries in feature_list.json descriptions
3. Only modify files related to YOUR assigned feature
4. If stuck after 3 attempts, record the reason in progress.txt and end the session
5. After completing the feature, run the validation script: {{validator_command}}
6. Git commit after each meaningful step
7. Update progress.txt with what you accomplished
8. When done, update feature_list.json to set your feature's "passes" to true
`
