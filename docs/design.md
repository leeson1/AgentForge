# AgentForge 设计文档

> 基于 Anthropic 文章 *Effective Harnesses for Long-Running Agents* 的长时间 Agent 运行管理系统

## 1. 项目概述

**AgentForge** 是一个通用的长时间 Agent 运行框架 + Web UI 管理平台。它通过模板/插件机制适配不同类型的复杂项目，并提供全功能的 Web 界面来监控和管理多个长时间运行的 Agent 任务。

### 核心决策

| 维度 | 决定 | 理由 |
|---|---|---|
| AI 引擎 | Claude Code CLI | 直接利用已有工具链（文件读写、Bash、Git 等） |
| 技术栈 | Go 后端 + React 前端 | Go 适合进程管理和高并发，React 生态成熟 |
| 架构 | 模块化单体 | 单一二进制部署极简，内部模块化可扩展 |
| 执行环境 | 本机直接执行 | 简洁高效 |
| 项目适配 | 模板/插件机制 | 内置模板快速起步，支持自定义 |
| 状态存储 | 纯文件系统 | 零外部依赖，与文章思路一致 |
| Agent 调度 | 混合模式（串行+并行） | 默认并行提速，`max_parallel_workers=1` 退化为串行 |
| 部署方式 | 命令行 + Docker Compose | 开发用命令行，生产用 Docker |

## 2. 系统架构

### 2.1 模块化单体架构

单一 Go 进程，内部按清晰的模块边界组织，模块间通过 Go interface 通信。

```
┌─────────────────────────────────────────────────────┐
│                Go 模块化单体服务                       │
│                                                       │
│  ┌─────────────────────────────────────────────────┐ │
│  │            HTTP/WebSocket 层                      │ │
│  │  REST API  │  WebSocket Hub  │  静态文件服务       │ │
│  └──────┬─────────────┬──────────────┬─────────────┘ │
│         │             │              │                │
│  ┌──────▼──────┐ ┌────▼────┐ ┌──────▼────────┐      │
│  │  任务模块    │ │ 流模块  │ │  模板模块      │      │
│  │ (CRUD/调度) │ │(日志/   │ │ (项目模板     │      │
│  │             │ │ 事件流) │ │  加载/管理)   │      │
│  └──────┬──────┘ └────┬────┘ └──────┬────────┘      │
│         │             │              │                │
│  ┌──────▼─────────────▼──────────────▼────────────┐  │
│  │              执行引擎模块                         │  │
│  │  ┌────────┐ ┌────────┐ ┌────────────────┐      │  │
│  │  │Session │ │进程管理 │ │ 文件系统监控    │      │  │
│  │  │管理器  │ │  器    │ │ (fsnotify)    │      │  │
│  │  └───┬────┘ └───┬────┘ └───────┬────────┘      │  │
│  └──────┼──────────┼──────────────┼────────────────┘  │
│         │          │              │                    │
│  ┌──────▼──────────▼──────────────▼────────────────┐  │
│  │          文件系统存储层                             │  │
│  │  ~/.agent-forge/tasks/{task-id}/                   │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────┬────────────────────────────────┘
                       │ exec
          ┌────────────▼─────────────┐
          │    Claude Code CLI        │ x N
          └──────────────────────────┘
```

### 2.2 核心模块职责

| 模块 | 职责 | 对外接口 |
|---|---|---|
| **server** | HTTP/WebSocket 服务、路由、静态文件 | — |
| **task** | 任务全生命周期管理、Coordinator 调度 | `TaskManager` interface |
| **session** | Claude Code CLI 进程的启动/监控/交互 | `SessionExecutor` interface |
| **stream** | 日志监控、事件总线、实时推送 | `EventBus` interface |
| **template** | 项目模板加载、校验、应用 | `TemplateRegistry` interface |
| **store** | 文件系统读写抽象 | `Store` interface |
| **notify** | 告警通知（Webhook） | `Notifier` interface |

### 2.3 项目目录结构

```
agent-forge/
├── cmd/
│   └── agent-forge/
│       └── main.go                 # 入口，CLI 命令（serve / init / run）
├── internal/
│   ├── server/                     # HTTP/WebSocket 服务层
│   │   ├── router.go
│   │   ├── handler_task.go
│   │   ├── handler_session.go
│   │   ├── handler_template.go
│   │   └── ws_hub.go
│   ├── task/                       # 任务模块
│   │   ├── manager.go
│   │   ├── scheduler.go            # Coordinator 逻辑（拓扑排序、批次分配）
│   │   └── model.go
│   ├── session/                    # 会话模块
│   │   ├── executor.go             # Claude Code CLI 进程管理
│   │   ├── initializer.go          # Initializer Agent 逻辑
│   │   ├── worker.go               # Worker Agent 逻辑
│   │   ├── resolver.go             # Resolver Agent 逻辑（冲突解决）
│   │   └── model.go
│   ├── stream/                     # 实时流模块
│   │   ├── log_watcher.go
│   │   ├── event_bus.go
│   │   └── sse.go
│   ├── template/                   # 项目模板模块
│   │   ├── loader.go
│   │   ├── registry.go
│   │   └── builtin/
│   │       ├── fullstack-web/
│   │       ├── cli-tool/
│   │       └── data-analysis/
│   ├── store/                      # 文件系统存储层
│   │   ├── task_store.go
│   │   ├── session_store.go
│   │   └── log_store.go
│   └── notify/                     # 通知告警模块
│       ├── notifier.go
│       └── webhook.go
├── web/                            # React 前端
├── templates/                      # 用户自定义模板
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 3. 多 Agent 调度机制

### 3.1 四种 Agent 角色

| 角色 | 触发时机 | 职责 |
|---|---|---|
| **Initializer** | 任务创建后首次运行 | 分析需求，生成 feature_list.json（含依赖关系）、init.sh、项目脚手架 |
| **Coordinator** | Initializer 完成后（Go 代码，非 Agent） | 拓扑排序，将 features 分成多个 Batch，调度 Worker |
| **Worker** | 每个 Batch 中的每个 feature | 在独立 Git worktree 上实现一个 feature，测试验证，git commit |
| **Resolver** | 合并分支产生冲突且 Level 1 自动合并失败时 | 理解冲突上下文，解决冲突，运行测试验证 |

### 3.2 执行流程

```
Phase 1: 初始化
  Initializer Agent → 生成 feature_list.json（含 id, depends_on 字段）

Phase 2: 规划
  Coordinator（Go 代码）→ 拓扑排序 → 生成 Batch 执行计划
  例：Batch 1: [F001, F002, F005]  Batch 2: [F003, F006]  Batch 3: [F004]

Phase 3: 并行执行（循环每个 Batch）
  1. 为 Batch 内每个 feature 创建 git worktree + 分支
  2. 启动 N 个 Worker Agent 并行工作（N <= max_parallel_workers）
  3. 所有 Worker 完成后，依次合并分支到 main
  4. 冲突处理（三级递进策略）
  5. 集成测试验证
  6. 更新 feature_list.json 和 progress.txt
  7. 进入下一个 Batch

Phase 4: 完成
  所有 Batch 完成 → 任务标记 completed
```

### 3.3 feature_list.json 格式

```json
{
  "features": [
    {
      "id": "F001",
      "category": "functional",
      "description": "用户注册与登录",
      "steps": [
        "创建注册表单",
        "实现JWT认证",
        "添加登录页面"
      ],
      "depends_on": [],
      "batch": null,
      "passes": false
    },
    {
      "id": "F003",
      "category": "functional",
      "description": "购物车功能",
      "steps": [
        "创建购物车页面",
        "实现添加/删除商品",
        "实现数量修改"
      ],
      "depends_on": ["F001", "F002"],
      "batch": null,
      "passes": false
    }
  ]
}
```

- `id`: 唯一标识，用于依赖引用
- `depends_on`: 依赖的 feature ID 列表，空数组表示无依赖
- `batch`: Coordinator 分配的批次号（初始为 null，规划后填充）
- `passes`: 是否通过验证（Agent 只能将 false 改为 true）

### 3.4 Git Worktree 隔离策略

每个 Worker 在独立的 Git worktree 上工作，完全避免文件系统冲突：

```bash
# Coordinator 为每个 Worker 准备工作目录
git worktree add ../wt-F001 -b feat/F001-user-auth
git worktree add ../wt-F002 -b feat/F002-products

# Worker 在各自的 worktree 中执行 Claude Code
claude --prompt "..." --cwd ../wt-F001

# Batch 完成后合并
git merge feat/F001-user-auth
git worktree remove ../wt-F001
```

## 4. 冲突解决：三级递进策略

```
Level 1: 自动合并策略（Go 代码，零 token）
  ├── 纯追加冲突 → 自动合并
  ├── import 语句冲突 → 合并 import 列表
  └── 配置文件冲突 → 智能合并
  └── 失败 → Level 2

Level 2: Resolver Agent（Claude Code，消耗 token）
  ├── 输入：冲突文件、两个分支的 commit log、feature 描述
  ├── 工作：解决冲突 → 运行测试验证
  ├── 最多重试 2 次
  └── 失败 → Level 3

Level 3: 人工介入（最后手段）
  ├── 任务状态 → conflict_wait
  ├── 通知用户（WebSocket + Webhook）
  └── UI 展示冲突详情 + Resolver 尝试记录
```

## 5. 任务状态机

```
pending → initializing → planning → running (Batch 循环) → completed

running 内部状态：
  executing → merging → auto_resolving → agent_resolving → validating

异常分支：
  running → failed（可重试）
  running → paused（可恢复）
  running → cancelled（用户取消）
  merging → conflict_wait → running（用户解决冲突后继续）
```

完整状态列表：

| 状态 | 说明 |
|---|---|
| `pending` | 已创建，等待启动 |
| `initializing` | Initializer Agent 运行中 |
| `planning` | Coordinator 拓扑排序中 |
| `running` | Workers 执行中（含 Batch 信息） |
| `merging` | 合并分支中 |
| `auto_resolving` | Level 1 自动解决冲突中 |
| `agent_resolving` | Level 2 Resolver Agent 解决冲突中 |
| `validating` | 集成测试验证中 |
| `conflict_wait` | 等待用户手动解决冲突 |
| `paused` | 用户暂停 |
| `completed` | 全部完成 |
| `failed` | 失败（可重试） |
| `cancelled` | 用户取消 |

## 6. 项目模板机制

### 6.1 模板结构

```
templates/fullstack-web/
├── template.json              # 模板元信息与配置
├── prompts/
│   ├── initializer.txt        # Initializer Agent prompt
│   └── worker.txt             # Worker Agent prompt
├── hooks/
│   ├── on_session_start.sh    # Session 启动前执行
│   └── on_session_end.sh      # Session 结束后执行
└── validators/
    └── e2e_check.sh           # 端到端验证脚本
```

### 6.2 template.json

```json
{
  "name": "fullstack-web",
  "display_name": "全栈 Web 应用",
  "description": "适用于前后端分离的 Web 应用项目",
  "version": "1.0.0",
  "config": {
    "max_sessions": 50,
    "session_timeout": "30m",
    "max_parallel_workers": 3,
    "auto_continue": true,
    "allowed_tools": ["Bash", "Read", "Write", "Edit", "Glob", "Grep"]
  },
  "prompts": {
    "initializer": "prompts/initializer.txt",
    "worker": "prompts/worker.txt"
  },
  "hooks": {
    "on_session_start": "hooks/on_session_start.sh",
    "on_session_end": "hooks/on_session_end.sh"
  },
  "validators": {
    "e2e": "validators/e2e_check.sh"
  },
  "feature_schema": {
    "required_fields": ["category", "description", "steps", "passes"],
    "categories": ["functional", "ui", "performance", "security"]
  }
}
```

### 6.3 四个扩展点

| 扩展点 | 说明 |
|---|---|
| **Prompts** | 控制 Agent 行为的提示词，支持变量插值（`{{progress}}`、`{{features}}` 等） |
| **Hooks** | Session 生命周期钩子脚本 |
| **Validators** | 验证功能是否完成的脚本 |
| **Feature Schema** | 定义 feature_list.json 的格式约束 |

### 6.4 内置模板

| 模板 | 适用场景 | 特点 |
|---|---|---|
| `fullstack-web` | 前后端分离 Web 应用 | E2E 测试、Puppeteer 验证、dev server 启动 |
| `cli-tool` | 命令行工具 | 单元测试验证、`go test` / `cargo test` |
| `data-analysis` | 数据分析项目 | Jupyter 验证、数据质量检查 |

## 7. Web UI 管理界面

### 7.1 整体布局

三栏式布局：
- **左侧**：任务列表（名称、状态、进度条、当前 Session/Batch）
- **中间**：任务详情（5 个 Tab）
- **右侧**：实时统计面板

### 7.2 五个 Tab

| Tab | 功能 | 数据源 |
|---|---|---|
| **实时对话** | 展示 Agent 思考过程和工具调用，支持手动干预 | Claude Code stream-json 输出 |
| **功能清单** | feature_list.json 可视化，依赖关系图，支持手动调整优先级 | feature_list.json 监控 |
| **日志** | 实时日志流 + 搜索 + 按 Session 筛选 + 级别过滤 | sessions/*.log 监控 |
| **Git 历史** | 提交时间线 + diff 查看 + 回滚操作 | workspace git log |
| **统计** | Token 用量、Session 历史、成本趋势图、Batch 执行耗时 | session-*.json 聚合 |

### 7.3 手动干预

- 中间面板底部有输入框，用户可以给运行中的 Agent 发送指令
- **干预粒度**：
  - 在"实时对话"Tab 中可以看到所有并行 Worker 的实时输出（按 Worker 分栏/标签切换）
  - 干预指令可以针对**特定 Worker**（如"Worker A: 先别做支付"）或**全局**（如"全部暂停"）
  - 针对特定 Worker 的指令写入该 Worker worktree 中的 `intervention.txt`
  - 全局指令写入任务根目录的 `intervention.txt`，所有 Worker 都会检查
- Worker prompt 包含"每次动作前检查 intervention.txt"的规则
- Claude Code CLI 支持通过 stdin 注入消息（如果可用则优先使用）

### 7.4 其他页面

| 页面 | 功能 |
|---|---|
| **模板管理** | 查看内置模板、创建/编辑自定义模板 |
| **全局统计** | 所有任务汇总：总 token、总成本、成功率、平均完成时间 |
| **设置** | 通知配置、默认参数、Claude Code CLI 路径 |

## 8. API 设计

### 8.1 REST API

```
# 任务管理
POST   /api/tasks                    # 创建任务
GET    /api/tasks                    # 任务列表（支持状态筛选）
GET    /api/tasks/:id                # 任务详情
PUT    /api/tasks/:id                # 更新任务配置
DELETE /api/tasks/:id                # 删除任务
POST   /api/tasks/:id/start          # 启动任务
POST   /api/tasks/:id/pause          # 暂停任务
POST   /api/tasks/:id/resume         # 恢复任务
POST   /api/tasks/:id/stop           # 停止任务
POST   /api/tasks/:id/intervene      # 发送干预指令

# Session
GET    /api/tasks/:id/sessions       # Session 列表
GET    /api/tasks/:id/sessions/:sid  # Session 详情

# 功能清单
GET    /api/tasks/:id/features       # 获取 feature list
PUT    /api/tasks/:id/features/:fid  # 更新 feature

# Git
GET    /api/tasks/:id/commits        # Git 提交历史
POST   /api/tasks/:id/rollback       # 回滚到指定 commit

# 模板
GET    /api/templates                # 模板列表
GET    /api/templates/:name          # 模板详情
POST   /api/templates                # 创建自定义模板

# 统计
GET    /api/tasks/:id/stats          # 任务统计
GET    /api/stats/overview           # 全局统计
```

### 8.2 WebSocket 事件

```
WS /api/ws

服务端推送事件：
{type: "task_status",     task_id, status}
{type: "session_start",   task_id, session_id}
{type: "session_end",     task_id, session_id}
{type: "agent_message",   task_id, session_id, msg}
{type: "tool_call",       task_id, session_id, tool}
{type: "feature_update",  task_id, feature_id}
{type: "merge_conflict",  task_id, batch, details}
{type: "alert",           task_id, level, message}
```

### 8.3 CLI 命令

```bash
agent-forge serve [--port 8080]      # 启动后端+前端
agent-forge init <project-dir>       # 交互式创建任务
agent-forge run <task-id>            # 启动任务
agent-forge status                   # 查看所有任务状态
agent-forge logs <task-id> --follow  # 实时查看日志
agent-forge stop <task-id>           # 停止任务
agent-forge template list            # 列出所有模板
agent-forge template create <name>   # 创建自定义模板
```

## 9. 文件系统存储

### 9.1 全局目录结构

```
~/.agent-forge/
├── config.json                      # 全局配置（通知、默认参数等）
├── tasks/
│   ├── {task-id}/
│   │   ├── task.json                # 任务元信息
│   │   ├── feature_list.json        # 功能清单
│   │   ├── progress.txt             # 跨 Session 进度记录
│   │   ├── init.sh                  # 环境初始化脚本
│   │   ├── execution_plan.json      # Coordinator 生成的批次计划（见下）
│   │   ├── intervention.txt         # 用户干预指令（见 7.3 说明）
│   │   ├── prompts/                 # 从模板复制的 prompt 文件
│   │   ├── sessions/                # Session 历史
│   │   │   ├── session-001.json
│   │   │   ├── session-001.log
│   │   │   └── ...
│   │   └── events/                  # 事件日志
│   │       └── events.jsonl
│   └── ...
└── templates/                       # 用户自定义模板
```

### 9.2 execution_plan.json

```json
{
  "generated_at": "2026-04-09T10:05:00Z",
  "batches": [
    {
      "batch": 1,
      "features": ["F001", "F002", "F005"],
      "status": "completed"
    },
    {
      "batch": 2,
      "features": ["F003", "F006", "F007"],
      "status": "running"
    },
    {
      "batch": 3,
      "features": ["F004"],
      "status": "pending"
    }
  ]
}
```

### 9.3 task.json

```json
{
  "id": "abc123",
  "name": "电商网站开发",
  "description": "开发一个完整的电商网站...",
  "template": "fullstack-web",
  "status": "running",
  "created_at": "2026-04-09T10:00:00Z",
  "updated_at": "2026-04-09T14:30:00Z",
  "config": {
    "max_parallel_workers": 3,
    "session_timeout": "30m",
    "workspace_dir": "/workspace/ecommerce"
  },
  "progress": {
    "current_batch": 2,
    "total_batches": 5,
    "features_completed": 12,
    "features_total": 18,
    "total_sessions": 8,
    "total_tokens": 125000,
    "estimated_cost": 3.75
  }
}
```

## 10. 错误处理与容错

### 10.1 错误类型与应对

| 错误类型 | 应对策略 |
|---|---|
| Claude Code 崩溃 | 自动重启 Session，从 progress.txt 恢复上下文。最多重试 3 次 |
| Session 超时 | 强制终止，记录已完成部分，启动新 Session 继续 |
| Agent 陷入死循环 | 检测连续相同工具调用或 token 异常消耗，终止并注入避免提示 |
| Worker 验证失败 | Worker 完成编码但 validator 失败 → 同一 Worker Session 内自动重试（最多 2 次）。仍然失败则结束 Session，下一次重新分配 Worker |
| Feature 反复失败 | 同一 feature 累计 3 个 Worker Session 失败 → 标记 stuck，跳过并通知用户 |
| 磁盘空间不足 | 启动前检查，低于阈值暂停任务并告警 |
| AgentForge 自身重启 | 扫描任务目录恢复状态，通过 PID 文件检测进程存活 |

### 10.2 防呆机制

每个 Worker Session 的 prompt 末尾自动追加：

```
## 防呆规则
1. 每次动作前，检查 intervention.txt 是否有用户指令
2. 不要删除或修改 feature_list.json 中已有条目的描述
3. 只修改你当前负责的 feature，不要碰其他文件
4. 如果卡住超过 3 次尝试，在 progress.txt 中记录原因并结束 Session
5. 完成 feature 后必须运行验证脚本确认通过
6. 每完成一个有意义的步骤就 git commit
```

## 11. 通知告警

### 配置

```json
{
  "notifications": {
    "webhook": {
      "enabled": true,
      "url": "https://hooks.slack.com/xxx"
    },
    "events": {
      "task_completed": true,
      "task_failed": true,
      "feature_stuck": true,
      "conflict_wait": true,
      "session_error": true,
      "cost_threshold": true
    },
    "cost_alert_threshold": 10.00
  }
}
```

支持 Slack、飞书、企业微信等通过 Webhook 接入。

## 12. 部署

### 命令行启动

```bash
# 编译
make build

# 启动
./agent-forge serve --port 8080
```

### Docker Compose

```yaml
version: '3.8'
services:
  agent-forge:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ~/.agent-forge:/root/.agent-forge
      - /workspace:/workspace
    environment:
      - CLAUDE_CODE_PATH=/usr/local/bin/claude
```

## 13. 技术依赖

| 依赖 | 用途 |
|---|---|
| Go 1.22+ | 后端语言 |
| React 18 + TypeScript | 前端框架 |
| gorilla/websocket | WebSocket 实现 |
| fsnotify | 文件系统监控 |
| cobra | CLI 框架 |
| chi / gin | HTTP 路由 |
| tailwindcss | 前端样式 |

## 14. 不在范围内（YAGNI）

- 多机分布式部署
- 用户认证与多租户
- 数据库存储（保持纯文件系统）
- 除 Claude Code CLI 之外的 LLM 后端
- 移动端 UI
