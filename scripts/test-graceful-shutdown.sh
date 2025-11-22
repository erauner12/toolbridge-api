#!/usr/bin/env bash
#
# Test MCP server graceful shutdown behavior
#
# Verifies that the server:
# 1. Starts successfully
# 2. Responds to SIGTERM within the configured timeout
# 3. Shuts down cleanly without CancelledError tracebacks
#
# This guards the graceful shutdown implementation (server.py + config.py)
# and ensures the uvicorn timeout_graceful_shutdown < Fly kill_timeout invariant.
#
# Usage:
#   ./scripts/test-graceful-shutdown.sh
#
# Exit codes:
#   0 - Success (clean shutdown within timeout)
#   1 - Failure (timeout, unclean shutdown, or error)

set -euo pipefail

# Configuration
TIMEOUT_SECONDS="${TOOLBRIDGE_SHUTDOWN_TIMEOUT_SECONDS:-7}"  # Match default from config.py
MAX_STARTUP_WAIT=10  # Max seconds to wait for server startup
LOG_FILE="/tmp/mcp-graceful-shutdown-test-$$.log"
SERVER_PORT="${TOOLBRIDGE_PORT:-8001}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $*"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

cleanup() {
    if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
        log_warn "Cleaning up: Killing server (PID $SERVER_PID)"
        kill -9 "$SERVER_PID" 2>/dev/null || true
    fi

    if [[ -f "$LOG_FILE" ]]; then
        rm -f "$LOG_FILE"
    fi
}

trap cleanup EXIT

main() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║     ToolBridge MCP Graceful Shutdown Test                   ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""

    log_info "Configuration:"
    log_info "  - Shutdown timeout: ${TIMEOUT_SECONDS}s"
    log_info "  - Max startup wait: ${MAX_STARTUP_WAIT}s"
    log_info "  - Server port: ${SERVER_PORT}"
    log_info "  - Log file: ${LOG_FILE}"
    echo ""

    # Step 1: Start MCP server
    log_info "Starting MCP server..."
    cd "$(git rev-parse --show-toplevel)/mcp"

    # Detect Python command (prefer uv run, then python3, then python)
    if command -v uv &> /dev/null && [[ -f ".venv/bin/python" || -f "pyproject.toml" ]]; then
        PYTHON_CMD="uv run python"
    elif command -v python3 &> /dev/null; then
        PYTHON_CMD="python3"
    elif command -v python &> /dev/null; then
        PYTHON_CMD="python"
    else
        log_error "No Python interpreter found (tried: uv, python3, python)"
        return 1
    fi

    log_info "Using Python command: $PYTHON_CMD"

    # Start server in background, redirecting output to log file
    $PYTHON_CMD -m toolbridge_mcp.server > "$LOG_FILE" 2>&1 &
    SERVER_PID=$!

    log_info "Server started (PID: $SERVER_PID)"

    # Step 2: Wait for server to be ready
    log_info "Waiting for server startup..."
    STARTUP_START=$(date +%s)

    while true; do
        if ! kill -0 "$SERVER_PID" 2>/dev/null; then
            log_error "Server process died during startup"
            cat "$LOG_FILE"
            return 1
        fi

        if grep -q "Uvicorn running on" "$LOG_FILE" 2>/dev/null; then
            log_success "Server is ready"
            break
        fi

        ELAPSED=$(($(date +%s) - STARTUP_START))
        if [[ $ELAPSED -ge $MAX_STARTUP_WAIT ]]; then
            log_error "Server startup timed out after ${MAX_STARTUP_WAIT}s"
            cat "$LOG_FILE"
            return 1
        fi

        sleep 0.5
    done

    # Step 3: Send SIGTERM
    log_info "Sending SIGTERM to server (PID: $SERVER_PID)..."
    SHUTDOWN_START=$(date +%s)
    kill -TERM "$SERVER_PID"

    # Step 4: Wait for clean shutdown
    log_info "Waiting for graceful shutdown (max ${TIMEOUT_SECONDS}s)..."

    while kill -0 "$SERVER_PID" 2>/dev/null; do
        ELAPSED=$(($(date +%s) - SHUTDOWN_START))

        if [[ $ELAPSED -ge $TIMEOUT_SECONDS ]]; then
            log_error "Server did not shut down within ${TIMEOUT_SECONDS}s timeout"
            kill -9 "$SERVER_PID" 2>/dev/null || true
            cat "$LOG_FILE"
            return 1
        fi

        sleep 0.2
    done

    SHUTDOWN_DURATION=$(($(date +%s) - SHUTDOWN_START))
    log_success "Server shut down cleanly in ${SHUTDOWN_DURATION}s"

    # Step 5: Verify clean shutdown in logs
    echo ""
    log_info "Verifying shutdown behavior in logs..."

    # Check for graceful shutdown message
    if grep -q "Received signal 15, initiating graceful shutdown" "$LOG_FILE"; then
        log_success "✓ Signal handler invoked correctly"
    else
        log_error "✗ Missing graceful shutdown signal message"
        cat "$LOG_FILE"
        return 1
    fi

    # Check for application shutdown complete
    if grep -q "Application shutdown complete" "$LOG_FILE"; then
        log_success "✓ Application shutdown completed"
    else
        log_error "✗ Missing 'Application shutdown complete' message"
        cat "$LOG_FILE"
        return 1
    fi

    # Check for finished server process
    if grep -q "Finished server process" "$LOG_FILE"; then
        log_success "✓ Server process finished cleanly"
    else
        log_error "✗ Missing 'Finished server process' message"
        cat "$LOG_FILE"
        return 1
    fi

    # Check for CancelledError (should NOT be present)
    if grep -q "CancelledError" "$LOG_FILE"; then
        log_error "✗ Found CancelledError traceback (unclean shutdown)"
        log_error "Log excerpt:"
        grep -A 10 "CancelledError" "$LOG_FILE"
        return 1
    else
        log_success "✓ No CancelledError tracebacks"
    fi

    # Check for asyncio errors (should NOT be present)
    if grep -qE "asyncio.*Error|Task.*exception" "$LOG_FILE"; then
        log_warn "Found asyncio warnings/errors in logs:"
        grep -E "asyncio.*Error|Task.*exception" "$LOG_FILE" || true
    fi

    # Summary
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_success "All graceful shutdown checks passed!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    log_info "Summary:"
    log_info "  ✓ Server started successfully"
    log_info "  ✓ SIGTERM handled gracefully"
    log_info "  ✓ Shutdown completed in ${SHUTDOWN_DURATION}s (< ${TIMEOUT_SECONDS}s timeout)"
    log_info "  ✓ No CancelledError tracebacks"
    log_info "  ✓ Clean uvicorn shutdown"
    echo ""

    return 0
}

main "$@"
