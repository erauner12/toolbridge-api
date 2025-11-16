#!/bin/bash
#
# Smoke test for MCP bridge --dev flag
# Verifies that the bridge can start without any config or environment variables
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_PATH="$PROJECT_ROOT/bin/mcpbridge"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "MCP Bridge Dev Mode Smoke Test"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Test 1: --dev flag without any config
echo "Test 1: Running with --dev flag (no config or env vars)"
echo "────────────────────────────────────────────"

# Clear all MCP and Auth0 env vars
unset MCP_API_BASE_URL
unset MCP_DEV_MODE
unset MCP_DEBUG
unset MCP_LOG_LEVEL
unset AUTH0_DOMAIN
unset AUTH0_CLIENT_ID_NATIVE
unset AUTH0_CLIENT_ID_WEB
unset AUTH0_CLIENT_ID_NATIVE_MACOS
unset AUTH0_SYNC_API_AUDIENCE
unset AUTH0_SYNC_API_SCOPE

# Start bridge in background
timeout 3 "$BIN_PATH" --dev > /tmp/mcp-test-dev.log 2>&1 || true

# Check if it started successfully (should contain "Starting MCP Bridge Server")
if grep -q "Starting MCP Bridge Server" /tmp/mcp-test-dev.log; then
    echo "✓ Bridge started successfully"
else
    echo "✗ Bridge failed to start"
    cat /tmp/mcp-test-dev.log
    exit 1
fi

# Check if dev mode warning is present
if grep -q "Dev mode is enabled" /tmp/mcp-test-dev.log; then
    echo "✓ Dev mode warning displayed"
else
    echo "✗ Dev mode warning missing"
    cat /tmp/mcp-test-dev.log
    exit 1
fi

# Check that it didn't fail with auth0 validation error
if grep -q "auth0.domain is required" /tmp/mcp-test-dev.log; then
    echo "✗ Auth0 validation error should not appear in dev mode"
    cat /tmp/mcp-test-dev.log
    exit 1
else
    echo "✓ No Auth0 validation errors"
fi

echo ""
echo "Test 2: Running with --dev --debug flags"
echo "────────────────────────────────────────────"

# Start bridge with debug flag
timeout 3 "$BIN_PATH" --dev --debug > /tmp/mcp-test-debug.log 2>&1 || true

# Check for debug level logs (should contain "DBG" or "debug" level)
if grep -qE "(DBG|\"level\":\"debug\")" /tmp/mcp-test-debug.log; then
    echo "✓ Debug logging enabled"
else
    echo "✗ Debug logging not enabled"
    cat /tmp/mcp-test-debug.log
    exit 1
fi

# Check for console formatter (colored output indicators)
if grep -qE "(\[90m|\[32m|\[33m)" /tmp/mcp-test-debug.log; then
    echo "✓ Console formatter active (pretty logs)"
else
    echo "✗ Console formatter not active"
    cat /tmp/mcp-test-debug.log
    exit 1
fi

echo ""
echo "Test 3: Running with --version flag"
echo "────────────────────────────────────────────"

# Test version flag
VERSION_OUTPUT=$("$BIN_PATH" --version)
if echo "$VERSION_OUTPUT" | grep -q "mcpbridge version"; then
    echo "✓ Version flag works: $VERSION_OUTPUT"
else
    echo "✗ Version flag failed"
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✓ All smoke tests passed!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Cleanup
rm -f /tmp/mcp-test-dev.log /tmp/mcp-test-debug.log
