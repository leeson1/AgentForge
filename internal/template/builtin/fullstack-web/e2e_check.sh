#!/bin/bash
# Fullstack Web: E2E validation script
# Performs basic HTTP health checks

set -e

WORKSPACE="${WORKSPACE_DIR:-.}"
DEV_PORT="${DEV_SERVER_PORT:-3000}"
API_PORT="${API_SERVER_PORT:-8080}"

echo "[AgentForge] Running E2E checks..."

ERRORS=0

# Check if frontend dev server responds
if lsof -i :${DEV_PORT} > /dev/null 2>&1; then
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${DEV_PORT}" 2>/dev/null || echo "000")
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "304" ]; then
        echo "[PASS] Frontend dev server responding (HTTP ${STATUS})"
    else
        echo "[WARN] Frontend dev server returned HTTP ${STATUS}"
    fi
else
    echo "[INFO] Frontend dev server not running on port ${DEV_PORT}"
fi

# Check if API server responds
if lsof -i :${API_PORT} > /dev/null 2>&1; then
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/health" 2>/dev/null || echo "000")
    if [ "$STATUS" = "200" ]; then
        echo "[PASS] API server health check passed (HTTP ${STATUS})"
    else
        echo "[WARN] API server returned HTTP ${STATUS}"
    fi
else
    echo "[INFO] API server not running on port ${API_PORT}"
fi

# Check if build succeeds
cd "${WORKSPACE}"
if [ -f "package.json" ]; then
    if npm run build > /dev/null 2>&1; then
        echo "[PASS] Frontend build succeeded"
    else
        echo "[FAIL] Frontend build failed"
        ERRORS=$((ERRORS + 1))
    fi
fi

if [ $ERRORS -gt 0 ]; then
    echo "[AgentForge] E2E check completed with ${ERRORS} error(s)"
    exit 1
fi

echo "[AgentForge] E2E check passed"
exit 0
