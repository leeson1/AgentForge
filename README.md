# AgentForge

A universal framework for long-running AI agent tasks powered by Claude Code CLI.

AgentForge manages parallel agents with template/plugin mechanisms, real-time monitoring via Web UI, and three-level merge conflict resolution.

## Features

- **Parallel Multi-Agent Execution** - Semaphore-based concurrency with Git worktree isolation per worker
- **Topological Scheduling** - Kahn's algorithm groups features into dependency-ordered batches
- **Three-Level Conflict Resolution** - Auto-merge → Resolver Agent → Human intervention
- **Real-Time Web UI** - React + Tailwind dashboard with WebSocket live updates
- **Template System** - Built-in templates (fullstack-web, cli-tool, data-analysis) + custom templates
- **Cost Monitoring** - Per-task token tracking with alert thresholds and hard limits
- **Error Recovery** - Session crash retry, stuck detection, timeout handling
- **Webhook Notifications** - Configurable alerts for task events (Slack/Feishu/WeChat Work)

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Web UI (React)                 │
│  TaskSidebar │ ConversationTab │ StatsPanel      │
│  FeaturesTab │ LogsTab │ GitTab │ StatsTab       │
└──────────────────────┬──────────────────────────┘
                       │ WebSocket + REST API
┌──────────────────────┴──────────────────────────┐
│                  Go Backend                      │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐  │
│  │ Scheduler │  │ BatchMgr  │  │  EventBus    │  │
│  │(Topology) │  │(Semaphore)│  │(Pub/Sub+WS)  │  │
│  └──────────┘  └───────────┘  └──────────────┘  │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐  │
│  │Initializer│ │  Workers  │  │   Resolver   │  │
│  │  Agent    │  │ (Parallel)│  │   Agent      │  │
│  └──────────┘  └───────────┘  └──────────────┘  │
│                Claude Code CLI (stream-json)      │
└──────────────────────────────────────────────────┘
                       │
                  File System Storage
                  ~/.agent-forge/
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- Git

### Build & Run

```bash
# Build everything
make build

# Start the server
./bin/agent-forge serve

# Or development mode (separate terminals)
make dev-backend   # Terminal 1: Go backend
make dev-frontend  # Terminal 2: React dev server
```

### Docker

```bash
# Build Docker image
make docker

# Start with docker-compose
docker-compose up -d

# View at http://localhost:8080
```

### CLI Usage

```bash
# Create a new task
agent-forge init ./my-project --name "My App" --template fullstack-web --workers 4

# Start a task
agent-forge run <task-id>

# View all tasks
agent-forge status

# View logs
agent-forge logs <task-id> --follow

# Stop a task
agent-forge stop <task-id>

# List templates
agent-forge template list
```

## Project Structure

```
.
├── cmd/agent-forge/         # CLI entry point
├── internal/
│   ├── agent/               # Agent orchestration (Initializer, Worker, BatchRunner, Resolver)
│   ├── config/              # Global configuration management
│   ├── notify/              # Webhook notification system
│   ├── recovery/            # Error recovery & cost monitoring
│   ├── server/              # HTTP server, WebSocket hub, REST API handlers
│   ├── session/             # Claude CLI executor, worktree manager, branch merger
│   ├── store/               # File-system persistence (tasks, sessions, logs)
│   ├── stream/              # EventBus pub/sub system
│   ├── task/                # Task model, scheduler (Kahn's), batch manager
│   └── template/            # Template loader (embed.FS), registry, hooks executor
├── web/                     # React + Vite + TypeScript + Tailwind CSS v4
│   ├── src/
│   │   ├── components/      # UI components (ConversationTab, FeaturesTab, LogsTab, etc.)
│   │   ├── pages/           # Global pages (Overview, Settings, Templates)
│   │   ├── stores/          # Zustand state management
│   │   └── lib/             # API client
│   └── ...
├── Dockerfile               # Multi-stage build
├── docker-compose.yml       # Docker deployment
└── Makefile                 # Build automation
```

## Configuration

Configuration file: `~/.agent-forge/config.json`

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "notification": {
    "webhook_url": "https://hooks.example.com/agentforge",
    "enabled_events": {
      "task_complete": true,
      "task_failed": true,
      "merge_conflict": true,
      "cost_alert": true
    }
  },
  "cli": {
    "claude_path": "claude",
    "max_retries": 3,
    "default_timeout": "30m"
  },
  "cost": {
    "alert_threshold": 10.0,
    "hard_limit": 50.0
  }
}
```

## Custom Templates

Create templates in `~/.agent-forge/templates/<name>/`:

```
my-template/
├── template.json       # Template metadata & config
├── initializer.txt     # Initializer agent prompt
├── worker.txt          # Worker agent prompt
├── on_session_start.sh # Optional: pre-session hook
└── validator.sh        # Optional: validation script
```

Template variables: `{{task_name}}`, `{{task_description}}`, `{{feature_id}}`, `{{progress_content}}`, `{{pending_features}}`, `{{session_number}}`, `{{validator_command}}`

## Testing

```bash
# All tests
make test

# Go tests with coverage
make coverage

# TypeScript type check
make test-web
```

## License

MIT
