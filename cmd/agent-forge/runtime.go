package main

import (
	"fmt"

	"github.com/leeson1/agent-forge/internal/config"
	"github.com/leeson1/agent-forge/internal/server"
	"github.com/leeson1/agent-forge/internal/session"
	"github.com/leeson1/agent-forge/internal/store"
	"github.com/leeson1/agent-forge/internal/stream"
	"github.com/leeson1/agent-forge/internal/template"
)

type appRuntime struct {
	cfg              *config.Config
	baseDir          string
	taskStore        *store.TaskStore
	sessionStore     *store.SessionStore
	logStore         *store.LogStore
	eventBus         *stream.EventBus
	executor         *session.Executor
	templateRegistry *template.Registry
	pipeline         *server.Pipeline
	httpServer       *server.Server
}

func bootstrapRuntime() (*appRuntime, error) {
	if err := store.Init(); err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	baseDir := store.BaseDir()
	taskStore := store.NewTaskStore(baseDir)
	sessionStore := store.NewSessionStore(baseDir)
	logStore := store.NewLogStore(baseDir)
	eventBus := stream.NewEventBus(256)

	executor := session.NewExecutor(baseDir, server.ExecutorConfigFromConfig(cfg))

	templateRegistry, err := template.NewRegistryWithBuiltins()
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	pipeline := server.NewPipeline(executor, taskStore, sessionStore, logStore, eventBus, templateRegistry)
	httpServer := server.NewServer(eventBus, taskStore, sessionStore, logStore, executor, templateRegistry, cfg)

	return &appRuntime{
		cfg:              cfg,
		baseDir:          baseDir,
		taskStore:        taskStore,
		sessionStore:     sessionStore,
		logStore:         logStore,
		eventBus:         eventBus,
		executor:         executor,
		templateRegistry: templateRegistry,
		pipeline:         pipeline,
		httpServer:       httpServer,
	}, nil
}
