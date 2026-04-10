#!/bin/bash
# CLI Tool: validation script
# Auto-detects project type and runs appropriate tests

set -e

WORKSPACE="${WORKSPACE_DIR:-.}"
cd "${WORKSPACE}"

echo "[AgentForge] Running CLI tool validation..."

ERRORS=0

# Detect project type and run tests
if [ -f "go.mod" ]; then
    echo "[AgentForge] Detected Go project"
    if go test ./... 2>&1; then
        echo "[PASS] Go tests passed"
    else
        echo "[FAIL] Go tests failed"
        ERRORS=$((ERRORS + 1))
    fi
    if go build ./... 2>&1; then
        echo "[PASS] Go build succeeded"
    else
        echo "[FAIL] Go build failed"
        ERRORS=$((ERRORS + 1))
    fi
elif [ -f "Cargo.toml" ]; then
    echo "[AgentForge] Detected Rust project"
    if cargo test 2>&1; then
        echo "[PASS] Cargo tests passed"
    else
        echo "[FAIL] Cargo tests failed"
        ERRORS=$((ERRORS + 1))
    fi
elif [ -f "package.json" ]; then
    echo "[AgentForge] Detected Node.js project"
    if npm test 2>&1; then
        echo "[PASS] npm tests passed"
    else
        echo "[FAIL] npm tests failed"
        ERRORS=$((ERRORS + 1))
    fi
elif [ -f "setup.py" ] || [ -f "pyproject.toml" ]; then
    echo "[AgentForge] Detected Python project"
    if python -m pytest 2>&1; then
        echo "[PASS] Python tests passed"
    else
        echo "[FAIL] Python tests failed"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "[WARN] Unknown project type, skipping tests"
fi

if [ $ERRORS -gt 0 ]; then
    echo "[AgentForge] Validation failed with ${ERRORS} error(s)"
    exit 1
fi

echo "[AgentForge] Validation passed"
exit 0
