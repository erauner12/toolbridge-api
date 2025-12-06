#!/bin/bash
# Development helper script for MCP-UI testing
# Usage: ./dev.sh [start|stop|restart|status]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Ports
TEST_SERVER_PORT=8099
NANOBOT_PORT=8080

start_test_server() {
    echo "Starting test server on port $TEST_SERVER_PORT..."
    .venv/bin/python test_ui_server.py > /tmp/test_ui_server.log 2>&1 &
    echo $! > /tmp/test_ui_server.pid
    sleep 2
    if lsof -i :$TEST_SERVER_PORT >/dev/null 2>&1; then
        echo "✓ Test server running: http://localhost:$TEST_SERVER_PORT/mcp"
    else
        echo "✗ Test server failed to start. Check /tmp/test_ui_server.log"
        return 1
    fi
}

start_nanobot() {
    if [ -z "$OPENAI_API_KEY" ]; then
        echo "✗ OPENAI_API_KEY not set. Export it first:"
        echo "  export OPENAI_API_KEY=sk-..."
        return 1
    fi
    echo "Starting Nanobot on port $NANOBOT_PORT..."
    nanobot run ./nanobot-test.yaml > /tmp/nanobot.log 2>&1 &
    echo $! > /tmp/nanobot.pid
    sleep 3
    if lsof -i :$NANOBOT_PORT >/dev/null 2>&1; then
        echo "✓ Nanobot running: http://localhost:$NANOBOT_PORT"
    else
        echo "✗ Nanobot failed to start. Check /tmp/nanobot.log"
        return 1
    fi
}

stop_services() {
    echo "Stopping services..."

    # Kill by PID files
    for pidfile in /tmp/test_ui_server.pid /tmp/nanobot.pid; do
        if [ -f "$pidfile" ]; then
            kill $(cat "$pidfile") 2>/dev/null
            rm "$pidfile"
        fi
    done

    # Also kill by process name (backup)
    pkill -f "test_ui_server.py" 2>/dev/null
    pkill -f "nanobot run" 2>/dev/null

    # Kill anything on the ports
    lsof -ti :$TEST_SERVER_PORT | xargs kill -9 2>/dev/null
    lsof -ti :$NANOBOT_PORT | xargs kill -9 2>/dev/null

    sleep 1
    echo "✓ Services stopped"
}

show_status() {
    echo "Service Status:"
    echo "---------------"
    if lsof -i :$TEST_SERVER_PORT >/dev/null 2>&1; then
        echo "✓ Test server: running on http://localhost:$TEST_SERVER_PORT/mcp"
    else
        echo "✗ Test server: not running"
    fi

    if lsof -i :$NANOBOT_PORT >/dev/null 2>&1; then
        echo "✓ Nanobot: running on http://localhost:$NANOBOT_PORT"
    else
        echo "✗ Nanobot: not running"
    fi
}

show_logs() {
    echo "=== Test Server Log (last 20 lines) ==="
    tail -20 /tmp/test_ui_server.log 2>/dev/null || echo "(no log)"
    echo ""
    echo "=== Nanobot Log (last 20 lines) ==="
    tail -20 /tmp/nanobot.log 2>/dev/null || echo "(no log)"
}

case "${1:-help}" in
    start)
        stop_services
        start_test_server && start_nanobot
        echo ""
        show_status
        ;;
    stop)
        stop_services
        ;;
    restart)
        stop_services
        sleep 1
        start_test_server && start_nanobot
        echo ""
        show_status
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    server)
        # Start only test server (useful during development)
        # Also ensures Nanobot is running if OPENAI_API_KEY is set
        lsof -ti :$TEST_SERVER_PORT | xargs kill -9 2>/dev/null
        sleep 1
        start_test_server
        # Check if Nanobot is down and restart it
        if ! lsof -i :$NANOBOT_PORT >/dev/null 2>&1; then
            if [ -n "$OPENAI_API_KEY" ]; then
                echo "Nanobot was down, restarting..."
                start_nanobot
            else
                echo "⚠ Nanobot is not running (set OPENAI_API_KEY to auto-start)"
            fi
        else
            echo "✓ Nanobot still running on http://localhost:$NANOBOT_PORT"
        fi
        ;;
    *)
        echo "MCP-UI Development Helper"
        echo ""
        echo "Usage: ./dev.sh <command>"
        echo ""
        echo "Commands:"
        echo "  start    - Start test server + Nanobot"
        echo "  stop     - Stop all services"
        echo "  restart  - Stop then start all services"
        echo "  status   - Show what's running"
        echo "  logs     - Show recent logs"
        echo "  server   - Start only test server (for dev iterations)"
        echo ""
        echo "Requirements:"
        echo "  - export OPENAI_API_KEY=sk-... (for Nanobot)"
        echo ""
        echo "URLs:"
        echo "  - Test server: http://localhost:8099/mcp"
        echo "  - Nanobot UI:  http://localhost:8080"
        ;;
esac
