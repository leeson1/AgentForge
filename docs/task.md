# AgentForge 实现任务拆分

> 每个 Phase 设计为一个独立的 Claude Code 会话可完成的工作量。
> 前后 Phase 有依赖关系，必须按顺序执行。

## 总览

```
Phase 1:  项目骨架 + 数据模型 + 文件存储层
Phase 2:  Session 执行器（Claude Code CLI 集成）
Phase 3:  Initializer Agent 流程
Phase 4:  Coordinator 调度器（拓扑排序 + 批次规划）
Phase 5:  Worker 并行执行 + Git Worktree 隔离
Phase 6:  三级冲突解决机制
Phase 7:  事件流 + WebSocket 实时推送
Phase 8:  REST API 层
Phase 9:  React 前端 — 项目脚手架 + 整体布局
Phase 10: React 前端 — 实时对话 + 功能清单
Phase 11: React 前端 — 日志/Git历史/统计
Phase 12: 项目模板系统
Phase 13: 通知告警 + 错误恢复
Phase 14: CLI 命令完善
Phase 15: Docker 部署 + 集成测试 + 收尾
```

---

## Phase 1: 项目骨架 + 数据模型 + 文件存储层

**目标**：搭建 Go 项目结构，定义所有核心数据模型，实现文件系统存储层。

**具体任务**：
- [ ] `go mod init github.com/leeson1/agent-forge`
- [ ] 创建完整目录结构（cmd/, internal/, web/, templates/）
- [ ] 定义数据模型（`internal/task/model.go`）
  - Task 结构体（含所有状态字段）
  - TaskStatus 枚举（pending/initializing/planning/running/...共 13 个状态）
  - TaskConfig 结构体
  - TaskProgress 结构体
- [ ] 定义数据模型（`internal/session/model.go`）
  - Session 结构体
  - SessionType 枚举（initializer/worker/resolver）
  - SessionResult 结构体
- [ ] 定义 Feature 模型
  - Feature 结构体（id/category/description/steps/depends_on/batch/passes）
  - FeatureList 结构体
  - ExecutionPlan 结构体（batches）
- [ ] 实现 Store 接口（`internal/store/`）
  - TaskStore：Create/Get/List/Update/Delete
  - SessionStore：Save/Get/List
  - LogStore：Append/Read/Tail
  - 所有操作基于 `~/.agent-forge/tasks/{task-id}/` 文件读写
- [ ] 编写 Store 层单元测试
- [ ] 入口文件 `cmd/agent-forge/main.go`（cobra 空壳）

**验收标准**：
- `go build ./...` 编译通过
- `go test ./internal/store/...` 测试通过
- Store 能正确读写 task.json, feature_list.json, session-*.json

**预计文件数**：~15 个

---

## Phase 2: Session 执行器（Claude Code CLI 集成）

**目标**：实现启动、监控、停止 Claude Code CLI 子进程的核心能力。

**具体任务**：
- [ ] 实现 SessionExecutor 接口（`internal/session/executor.go`）
  - Start(prompt, workDir, allowedTools) → 启动 claude 子进程
  - Stop(sessionID) → 优雅终止进程
  - IsRunning(sessionID) → 检查进程是否存活
- [ ] Claude Code CLI 调用封装
  - 构建命令行参数：`claude --print --output-format stream-json --verbose --max-turns N`
  - stdin 写入 prompt
  - stdout stream-json 实时解析
  - stderr 捕获错误
- [ ] stream-json 输出解析器（`internal/session/parser.go`）
  - 解析 assistant 消息、tool_use、tool_result 等事件类型
  - 提取 token 使用量（input_tokens, output_tokens）
  - 生成结构化的 SessionEvent
- [ ] 进程生命周期管理
  - PID 文件写入/读取（`sessions/session-N.pid`）
  - 超时检测与强制终止
  - 进程崩溃检测与自动重试（最多 3 次）
- [ ] Session 日志持久化
  - 实时写入 `sessions/session-N.log`
  - Session 结束后生成 `sessions/session-N.json`（元数据摘要）
- [ ] 编写单元测试（mock claude CLI）

**验收标准**：
- 能启动一个 Claude Code CLI 进程并实时捕获 stream-json 输出
- 能正确解析 token 用量
- PID 文件正确管理
- 超时终止功能正常

**预计文件数**：~8 个

---

## Phase 3: Initializer Agent 流程

**目标**：实现任务初始化阶段，Initializer Agent 分析需求并生成 feature_list.json。

**具体任务**：
- [ ] 实现 Initializer 逻辑（`internal/session/initializer.go`）
  - 构建 Initializer prompt（注入用户需求描述 + 模板规则）
  - Prompt 模板变量插值引擎（`{{task_name}}`、`{{task_description}}` 等）
  - 调用 SessionExecutor 启动 Claude Code
  - 监控输出，等待 feature_list.json 生成
- [ ] feature_list.json 校验器
  - 校验必须字段（id, description, depends_on, passes）
  - 校验依赖关系无循环（检测有向图环）
  - 校验 id 唯一性
- [ ] init.sh 校验
  - 检查文件是否生成
  - 检查是否可执行
- [ ] progress.txt 初始化
- [ ] 首次 git commit 自动执行
- [ ] Task 状态流转：pending → initializing → planning

**验收标准**：
- 给定任务描述，Initializer 能生成合法的 feature_list.json
- 依赖关系循环检测正常工作
- progress.txt 和 init.sh 正确生成
- 工作目录有首次 git commit

**预计文件数**：~6 个

---

## Phase 4: Coordinator 调度器

**目标**：实现拓扑排序算法，将 features 分成有序的 Batch 执行计划。

**具体任务**：
- [ ] 拓扑排序实现（`internal/task/scheduler.go`）
  - 读取 feature_list.json 的 depends_on 字段
  - Kahn 算法实现拓扑排序
  - 将同层级（无互相依赖）的 features 归入同一 Batch
  - 生成 execution_plan.json
- [ ] Batch 执行控制（`internal/task/manager.go`）
  - 按 Batch 顺序执行
  - 当前 Batch 所有 Worker 完成后才进入下一个 Batch
  - 跟踪每个 Batch 的状态（pending/running/completed/failed）
- [ ] Task 状态机完整实现
  - 所有 13 种状态的合法转换
  - 状态转换时触发事件（为后续 EventBus 预留接口）
- [ ] feature_list.json 的 batch 字段回写
- [ ] execution_plan.json 持久化
- [ ] 编写调度器单元测试（多种依赖图场景）

**验收标准**：
- 线性依赖链正确分成多个 Batch
- 无依赖的 features 归入同一 Batch
- 循环依赖正确报错
- 空依赖列表（全部并行）正确处理
- 状态机转换全覆盖测试

**预计文件数**：~5 个

---

## Phase 5: Worker 并行执行 + Git Worktree 隔离

**目标**：实现多个 Worker Agent 在独立 Git worktree 上并行执行 features。

**具体任务**：
- [ ] Git Worktree 管理器（`internal/session/worktree.go`）
  - CreateWorktree(taskDir, featureID) → 创建 worktree + 分支
  - RemoveWorktree(path) → 清理 worktree
  - ListWorktrees(taskDir) → 列出活跃 worktree
- [ ] Worker Agent 实现（`internal/session/worker.go`）
  - 构建 Worker prompt（注入 feature 描述、progress.txt、防呆规则）
  - 在指定 worktree 目录启动 Claude Code
  - 监控 feature 完成状态
  - intervention.txt 检查机制
- [ ] 并行执行控制器
  - 根据 max_parallel_workers 控制并发数
  - 使用 goroutine + WaitGroup 管理并行 Worker
  - 单个 Worker 完成/失败不影响其他 Worker
  - 所有 Worker 完成后触发合并阶段
- [ ] Worker 验证失败重试
  - Validator 脚本执行
  - 同 Session 内最多重试 2 次
  - Feature stuck 检测（累计 3 次失败）
- [ ] 编写集成测试

**验收标准**：
- 能创建多个 Git worktree 并在各自分支上工作
- max_parallel_workers=1 时退化为串行
- max_parallel_workers=3 时确实有 3 个 claude 进程并行
- Worker 完成后 worktree 被正确清理
- stuck feature 被正确标记跳过

**预计文件数**：~6 个

---

## Phase 6: 三级冲突解决机制

**目标**：实现 Batch 完成后的分支合并和三级冲突解决策略。

**具体任务**：
- [ ] 分支合并控制器（`internal/session/merger.go`）
  - 依次合并每个 Worker 分支到 main
  - 检测是否有冲突
- [ ] Level 1: 自动合并策略
  - 纯追加冲突检测与自动解决
  - import 语句冲突合并
  - JSON/YAML 配置文件智能合并
- [ ] Level 2: Resolver Agent（`internal/session/resolver.go`）
  - 构建 Resolver prompt（注入冲突文件、commit log、feature 描述）
  - 调用 Claude Code 解决冲突
  - 运行测试验证合并结果
  - 最多重试 2 次
- [ ] Level 3: 人工介入
  - 设置 task 状态为 conflict_wait
  - 保存冲突详情到文件（供 UI 展示）
  - 提供"应用某个分支版本"的 API
- [ ] 合并后集成测试
  - 运行 init.sh 启动服务
  - 执行 validator 脚本
  - 验证所有已完成 feature 仍然正常
- [ ] 状态流转：running → merging → auto_resolving → agent_resolving → validating

**验收标准**：
- 无冲突的分支合并正常
- 简单冲突 Level 1 能自动解决
- 复杂冲突 Level 2 Resolver Agent 能处理
- Resolver 失败后正确进入 conflict_wait 状态
- 集成测试在合并后执行

**预计文件数**：~6 个

---

## Phase 7: 事件流 + WebSocket 实时推送

**目标**：实现内部事件总线和 WebSocket 实时推送，为前端提供实时数据。

**具体任务**：
- [ ] EventBus 实现（`internal/stream/event_bus.go`）
  - Publish(event) → 发布事件
  - Subscribe(filter) → 订阅事件（支持按 task_id 过滤）
  - 事件类型定义（task_status, session_start, agent_message, tool_call, feature_update, merge_conflict, alert）
- [ ] 日志文件监控器（`internal/stream/log_watcher.go`）
  - 使用 fsnotify 监控 sessions/*.log 文件变化
  - 增量读取新内容 → 发布到 EventBus
- [ ] 文件变化监控
  - 监控 feature_list.json → 发布 feature_update 事件
  - 监控 progress.txt → 发布进度事件
  - 监控 task.json → 发布 task_status 事件
- [ ] WebSocket Hub（`internal/server/ws_hub.go`）
  - 客户端连接管理（注册/注销）
  - 订阅 EventBus → 广播给所有连接的客户端
  - 支持按 task_id 过滤推送
  - 心跳检测与断线重连
- [ ] events.jsonl 持久化
  - 所有事件追加写入 events/events.jsonl
  - 供历史查询和 UI 时间线使用
- [ ] 将 SessionExecutor 的输出接入 EventBus

**验收标准**：
- 事件发布/订阅正常工作
- 文件变化能触发对应事件
- WebSocket 客户端能收到实时事件
- events.jsonl 正确记录所有事件

**预计文件数**：~7 个

---

## Phase 8: REST API 层

**目标**：实现所有 REST API 端点，连接前后端。

**具体任务**：
- [ ] HTTP 服务器搭建（`internal/server/router.go`）
  - 选用 chi 或 gin 路由框架
  - CORS 配置
  - 静态文件服务（嵌入 React 构建产物）
- [ ] 任务 API（`internal/server/handler_task.go`）
  - POST /api/tasks — 创建任务
  - GET /api/tasks — 任务列表（支持 ?status= 筛选）
  - GET /api/tasks/:id — 任务详情
  - PUT /api/tasks/:id — 更新任务配置
  - DELETE /api/tasks/:id — 删除任务
  - POST /api/tasks/:id/start — 启动任务
  - POST /api/tasks/:id/pause — 暂停
  - POST /api/tasks/:id/resume — 恢复
  - POST /api/tasks/:id/stop — 停止
  - POST /api/tasks/:id/intervene — 干预指令
- [ ] Session API（`internal/server/handler_session.go`）
  - GET /api/tasks/:id/sessions — Session 列表
  - GET /api/tasks/:id/sessions/:sid — Session 详情
- [ ] Feature API
  - GET /api/tasks/:id/features — 获取 feature list
  - PUT /api/tasks/:id/features/:fid — 更新 feature
- [ ] Git API
  - GET /api/tasks/:id/commits — 提交历史
  - POST /api/tasks/:id/rollback — 回滚
- [ ] 模板 API（`internal/server/handler_template.go`）
  - GET /api/templates — 模板列表
  - GET /api/templates/:name — 模板详情
  - POST /api/templates — 创建模板
- [ ] 统计 API
  - GET /api/tasks/:id/stats — 任务统计
  - GET /api/stats/overview — 全局统计
- [ ] WebSocket 端点
  - WS /api/ws — 连接 WebSocket Hub
- [ ] 编写 API 集成测试（httptest）

**验收标准**：
- 所有端点返回正确的 JSON 响应
- 错误情况返回合适的 HTTP 状态码
- WebSocket 连接可建立并收到事件
- API 集成测试覆盖核心流程

**预计文件数**：~8 个

---

## Phase 9: React 前端 — 项目脚手架 + 整体布局

**目标**：搭建 React 项目，实现三栏式整体布局和任务列表。

**具体任务**：
- [ ] 创建 React 项目（Vite + TypeScript + Tailwind）
- [ ] 安装依赖：react-router-dom, zustand（状态管理）, lucide-react（图标）
- [ ] 整体布局组件
  - AppLayout：顶部导航栏 + 三栏式内容区
  - TopNav：Logo、页面切换（任务/模板/统计/设置）、运行状态指示
- [ ] 左侧任务列表（TaskSidebar）
  - 任务卡片（名称、状态标签、进度条、当前 Batch）
  - "新建任务"按钮
  - 状态筛选
- [ ] 右侧统计面板（StatsPanel）
  - 进度百分比
  - 当前 Session/Batch
  - Token 消耗
  - Git Commits 数
  - 运行时长
  - 最近事件列表
- [ ] 中间任务详情区域（TaskDetail）
  - Tab 栏（实时对话/功能清单/日志/Git历史/统计）
  - Tab 切换路由
  - 任务操作按钮（暂停/停止/干预）
- [ ] 新建任务对话框（CreateTaskModal）
  - 任务名称、描述输入
  - 模板选择下拉框
  - 工作目录选择
  - 并行度配置
- [ ] API 客户端封装（api.ts）
  - 封装所有 REST API 调用
  - WebSocket 连接管理
- [ ] Zustand 全局状态
  - tasks store
  - activeTask store
  - websocket store

**验收标准**：
- 页面正常渲染三栏式布局
- 能通过 API 获取并展示任务列表
- 新建任务对话框能创建任务
- WebSocket 连接成功建立

**预计文件数**：~15 个

---

## Phase 10: React 前端 — 实时对话 + 功能清单

**目标**：实现最核心的两个 Tab：实时对话（含干预功能）和功能清单。

**具体任务**：
- [ ] 实时对话 Tab（ConversationTab）
  - Agent 消息气泡（思考过程）
  - Tool Call 折叠面板（命令 + 输出）
  - 按 Worker 分标签展示（并行模式下）
  - 自动滚动到底部
  - 时间戳显示
- [ ] 干预输入框（InterventionInput）
  - Worker 选择（针对特定 Worker 或全局）
  - 文本输入 + 发送按钮
  - 发送后显示在对话流中（用户消息样式）
- [ ] 功能清单 Tab（FeaturesTab）
  - Feature 列表（卡片式），显示状态图标（✅ 🔄 ⬚ ❌）
  - 每个 Feature 可展开查看 steps 详情
  - 依赖关系可视化（简单的 Batch 分组展示）
  - Batch 进度条（当前 Batch 高亮）
  - stuck feature 红色高亮
- [ ] WebSocket 事件处理
  - agent_message → 追加到对话流
  - tool_call → 追加工具调用记录
  - feature_update → 更新功能清单状态

**验收标准**：
- 实时对话能展示 Agent 的完整工作过程
- 干预消息能发送成功并反映在对话中
- 功能清单正确显示所有 feature 状态
- Batch 分组展示清晰
- WebSocket 实时更新正常

**预计文件数**：~8 个

---

## Phase 11: React 前端 — 日志/Git历史/统计

**目标**：实现剩余三个 Tab 和其他管理页面。

**具体任务**：
- [ ] 日志 Tab（LogsTab）
  - 实时日志流（类似终端，等宽字体，深色背景）
  - Session 下拉选择器（查看历史 Session 日志）
  - 日志级别过滤（info/warn/error，颜色区分）
  - 全文搜索框
  - 自动滚动 + 暂停滚动按钮
- [ ] Git 历史 Tab（GitTab）
  - 时间线展示（垂直时间线，每个 commit 一个节点）
  - 每个 commit 显示：消息、时间、所属 Session/Batch
  - 点击 commit 展开 diff 详情
  - "回滚到此版本"按钮（带确认对话框）
- [ ] 统计 Tab（StatsTab）
  - Token 用量折线图（按 Session）
  - 成本累计柱状图
  - Session 执行时长对比
  - Batch 执行耗时
  - 图表使用 recharts 库
- [ ] 模板管理页面（TemplatesPage）
  - 模板卡片列表（名称、描述、类型标签）
  - 模板详情查看（配置、prompt 预览）
- [ ] 全局统计页面（OverviewPage）
  - 总任务数、运行中、已完成、失败
  - 总 token 消耗 / 总成本
  - 任务成功率
- [ ] 设置页面（SettingsPage）
  - 通知 Webhook URL 配置
  - 事件开关
  - 成本告警阈值
  - Claude Code CLI 路径

**验收标准**：
- 日志实时流畅滚动
- Git 时间线正确展示 commit 历史
- 统计图表正确渲染
- 模板管理能查看内置模板
- 设置页面能保存配置

**预计文件数**：~12 个

---

## Phase 12: 项目模板系统

**目标**：实现模板加载、校验、应用机制和 3 个内置模板。

**具体任务**：
- [ ] 模板加载器（`internal/template/loader.go`）
  - 从内置目录（embed.FS）加载内置模板
  - 从 ~/.agent-forge/templates/ 加载自定义模板
  - template.json 解析与校验
- [ ] 模板注册表（`internal/template/registry.go`）
  - 注册/查询模板
  - 按名称获取模板
  - 列出所有可用模板
- [ ] Prompt 变量插值引擎
  - 支持变量：`{{task_name}}`、`{{task_description}}`、`{{progress_content}}`、`{{pending_features}}`、`{{session_number}}`、`{{validator_command}}` 等
  - 插值时自动读取对应文件内容
- [ ] Hooks 执行器
  - on_session_start.sh 执行（Session 启动前）
  - on_session_end.sh 执行（Session 结束后）
  - 环境变量注入（TASK_ID, SESSION_ID, WORKSPACE_DIR 等）
- [ ] 内置模板：fullstack-web
  - initializer.txt / worker.txt prompt
  - on_session_start.sh（启动 dev server）
  - e2e_check.sh（基础 HTTP 健康检查）
- [ ] 内置模板：cli-tool
  - prompt 侧重命令行参数、单元测试
  - validator 执行 `go test` / `cargo test` / `npm test`
- [ ] 内置模板：data-analysis
  - prompt 侧重数据处理、可视化
  - validator 检查输出文件是否生成
- [ ] 自定义模板创建 CLI 流程
  - 交互式提问 → 生成模板目录结构

**验收标准**：
- 3 个内置模板能正确加载
- Prompt 变量插值正确替换
- Hooks 脚本在正确时机执行
- 自定义模板能创建并使用
- 模板缺失必要文件时报错提示清晰

**预计文件数**：~15 个（含模板文件）

---

## Phase 13: 通知告警 + 错误恢复

**目标**：实现 Webhook 通知、错误检测与自动恢复机制。

**具体任务**：
- [ ] Notifier 接口与 Webhook 实现（`internal/notify/`）
  - Webhook POST 请求（支持 Slack/飞书/企业微信格式）
  - 事件过滤（config.json 中配置哪些事件通知）
  - 发送失败重试（最多 3 次，指数退避）
- [ ] 全局配置管理（`~/.agent-forge/config.json`）
  - 读取/写入全局配置
  - 配置热加载（fsnotify 监控变化）
- [ ] 错误恢复机制
  - Claude Code 崩溃 → 自动重启 Session（最多 3 次）
  - Session 超时 → 强制终止 + 记录部分进度 + 新 Session 继续
  - 死循环检测：连续 N 次相同 tool_call 或 token 异常消耗无进度
  - Feature stuck 检测与自动跳过
- [ ] AgentForge 自身恢复
  - 启动时扫描 tasks/ 目录
  - 检测 running 状态的 task → 检查 PID 文件 → 进程存活则恢复监控，否则重启
  - 清理孤立的 worktree
- [ ] 成本监控
  - 按任务累计 token 用量
  - 超过阈值时触发告警
  - 可选：超过硬限制自动暂停任务
- [ ] 磁盘空间检查（启动 Session 前）

**验收标准**：
- Webhook 通知能发送到指定 URL
- 进程崩溃后自动重试正常
- 超时终止正常
- AgentForge 重启后能恢复运行中的任务
- 成本超过阈值时收到告警

**预计文件数**：~8 个

---

## Phase 14: CLI 命令完善

**目标**：完善所有 CLI 命令，让用户无需 UI 也能操作。

**具体任务**：
- [ ] `agent-forge serve` 命令
  - 启动 HTTP 服务器 + WebSocket
  - --port 参数
  - --host 参数
  - 启动时打印访问 URL
- [ ] `agent-forge init <project-dir>` 命令
  - 交互式创建任务（inquire 风格）
  - 选择模板、输入名称/描述、配置并行度
  - 创建任务并自动启动
- [ ] `agent-forge run <task-id>` 命令
  - 启动指定任务
  - --follow 参数：启动后实时输出日志
- [ ] `agent-forge status` 命令
  - 表格展示所有任务状态
  - 显示：名称、状态、进度、当前 Batch、运行时长
- [ ] `agent-forge logs <task-id>` 命令
  - --follow 实时尾随日志
  - --session N 查看指定 Session 日志
  - --level 过滤日志级别
- [ ] `agent-forge stop <task-id>` 命令
  - 优雅停止（等待当前 Worker 完成）
  - --force 强制终止
- [ ] `agent-forge template list` / `agent-forge template create <name>`
- [ ] 命令输出美化（颜色、表格、进度条）

**验收标准**：
- 所有命令可用且输出格式美观
- `agent-forge serve` 能启动完整服务
- `agent-forge init` 能交互式创建并启动任务
- `agent-forge status` 正确显示任务状态
- `agent-forge logs --follow` 实时输出

**预计文件数**：~6 个

---

## Phase 15: Docker 部署 + 集成测试 + 收尾

**目标**：完成部署配置、端到端集成测试和项目收尾。

**具体任务**：
- [ ] Dockerfile
  - 多阶段构建（Go 编译 + React 构建 + 最终镜像）
  - 包含 Claude Code CLI 安装
  - 包含 Git 安装
- [ ] docker-compose.yml
  - 卷映射（~/.agent-forge、workspace 目录）
  - 端口映射
  - 环境变量配置
- [ ] Makefile
  - `make build` — 编译 Go + 构建 React
  - `make dev` — 开发模式（前后端热重载）
  - `make test` — 运行所有测试
  - `make docker` — 构建 Docker 镜像
  - `make clean` — 清理构建产物
- [ ] 端到端集成测试
  - 创建任务 → 初始化 → 执行 → 完成的完整流程
  - 使用 cli-tool 模板（最简单）做 E2E 测试
  - 测试暂停/恢复/停止
  - 测试干预功能
- [ ] README.md 更新
  - 项目介绍
  - 快速开始
  - 配置说明
  - 模板开发指南
- [ ] 代码清理
  - 去除 TODO 注释
  - 补充关键函数注释
  - go vet / golangci-lint 检查
  - eslint 检查前端代码

**验收标准**：
- `make build` 一键构建成功
- `docker-compose up` 能正常启动
- E2E 测试通过完整生命周期
- README 内容完整
- 无 lint 警告

**预计文件数**：~10 个

---

## 依赖关系图

```
Phase 1 ──▶ Phase 2 ──▶ Phase 3 ──▶ Phase 4 ──▶ Phase 5 ──▶ Phase 6
  │                                                             │
  │           Phase 7 ◀─────────────────────────────────────────┘
  │             │
  │           Phase 8
  │             │
  └──────▶ Phase 9 ──▶ Phase 10 ──▶ Phase 11
              │
Phase 12 (可与 Phase 9-11 并行)
Phase 13 (依赖 Phase 7-8)
Phase 14 (依赖 Phase 8)
Phase 15 (依赖所有 Phase)
```

## 预估总量

| 指标 | 预估 |
|---|---|
| 总 Phase 数 | 15 |
| 总文件数 | ~135 个 |
| Go 代码文件 | ~70 个 |
| React 代码文件 | ~50 个 |
| 模板/配置文件 | ~15 个 |
