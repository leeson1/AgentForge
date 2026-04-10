#!/bin/bash
# Fullstack Web: on_session_start hook
# Starts dev server before each worker session

set -e

echo "[AgentForge] Starting dev server for task: ${TASK_ID}"
echo "[AgentForge] Workspace: ${WORKSPACE_DIR}"
echo "[AgentForge] Session: ${SESSION_ID}"

cd "${WORKSPACE_DIR}" || exit 1

# Detect and start appropriate dev server
if [ -f "package.json" ]; then
    # Check if dev server is already running
    if ! lsof -i :${DEV_SERVER_PORT:-3000} > /dev/null 2>&1; then
        echo "[AgentForge] Starting npm dev server..."
        npm run dev > /tmp/agentforge-dev-server-${TASK_ID}.log 2>&1 &
        echo $! > /tmp/agentforge-dev-server-${TASK_ID}.pid
        sleep 3
        echo "[AgentForge] Dev server started (PID: $(cat /tmp/agentforge-dev-server-${TASK_ID}.pid))"
    else
        echo "[AgentForge] Dev server already running on port ${DEV_SERVER_PORT:-3000}"
    fi
fi

echo "[AgentForge] Session start hook completed"
