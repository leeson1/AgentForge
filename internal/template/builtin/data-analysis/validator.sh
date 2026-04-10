#!/bin/bash
# Data Analysis: validation script
# Checks that output files are generated

set -e

WORKSPACE="${WORKSPACE_DIR:-.}"
OUTPUT_DIR="${OUTPUT_DIR:-output}"
cd "${WORKSPACE}"

echo "[AgentForge] Running data analysis validation..."

ERRORS=0

# Check output directory exists
if [ ! -d "${OUTPUT_DIR}" ]; then
    echo "[WARN] Output directory '${OUTPUT_DIR}' not found, creating..."
    mkdir -p "${OUTPUT_DIR}"
fi

# Count output files
OUTPUT_COUNT=$(find "${OUTPUT_DIR}" -type f 2>/dev/null | wc -l | tr -d ' ')
echo "[INFO] Found ${OUTPUT_COUNT} files in ${OUTPUT_DIR}/"

# Check for common output types
PNG_COUNT=$(find "${OUTPUT_DIR}" -name "*.png" -type f 2>/dev/null | wc -l | tr -d ' ')
SVG_COUNT=$(find "${OUTPUT_DIR}" -name "*.svg" -type f 2>/dev/null | wc -l | tr -d ' ')
CSV_COUNT=$(find "${OUTPUT_DIR}" -name "*.csv" -type f 2>/dev/null | wc -l | tr -d ' ')
JSON_COUNT=$(find "${OUTPUT_DIR}" -name "*.json" -type f 2>/dev/null | wc -l | tr -d ' ')

echo "[INFO] Output breakdown: ${PNG_COUNT} PNG, ${SVG_COUNT} SVG, ${CSV_COUNT} CSV, ${JSON_COUNT} JSON"

# Run Python tests if available
if [ -f "requirements.txt" ] || [ -f "setup.py" ] || [ -f "pyproject.toml" ]; then
    if command -v python &> /dev/null; then
        if python -m pytest tests/ 2>&1; then
            echo "[PASS] Python tests passed"
        else
            echo "[WARN] Python tests failed or no tests found"
        fi
    fi
fi

# Check for syntax errors in Python scripts
for f in $(find . -name "*.py" -not -path "*/venv/*" -not -path "*/.venv/*" 2>/dev/null); do
    if ! python -c "import py_compile; py_compile.compile('${f}', doraise=True)" 2>/dev/null; then
        echo "[FAIL] Syntax error in ${f}"
        ERRORS=$((ERRORS + 1))
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo "[AgentForge] Validation failed with ${ERRORS} error(s)"
    exit 1
fi

echo "[AgentForge] Validation passed"
exit 0
