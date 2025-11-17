# MCP Bridge Local Testing Guide

This guide explains how to test the MCP bridge Docker image locally before deploying to Kubernetes.

## Overview

The local testing setup includes:

- **docker-compose.mcp-test.yml**: Full-stack test environment with PostgreSQL, REST API, and MCP bridge
- **scripts/test-mcp-bridge-docker.sh**: Comprehensive automated test suite
- **Makefile targets**: Easy-to-use commands for testing

## Quick Start

### Option 1: Automated Full Test Suite (Recommended)

Run the complete automated test suite:

```bash
make test-mcp-docker
```

This will:
1. Build the MCP bridge Docker image
2. Start the full stack (PostgreSQL, REST API, MCP bridge)
3. Run comprehensive integration tests
4. Test graceful shutdown
5. Clean up all resources automatically

**Expected output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MCP Bridge Docker Integration Tests
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

━━━ Step 1: Building MCP Bridge Docker image
✓ Docker image built successfully

━━━ Step 2: Starting test environment
✓ Services started

━━━ Step 3: Waiting for services to be healthy
✓ PostgreSQL is ready
✓ REST API is healthy
✓ MCP Bridge (dev) is healthy

━━━ Step 4: Testing REST API
✓ REST API health check passed
✓ REST API session creation works (id=...)

━━━ Step 5: Testing MCP Bridge (Dev Mode)
✓ MCP health check passed
✓ MCP readiness check passed
✓ Dev mode correctly reported in readiness response
✓ MCP initialize endpoint works
✓ MCP session created (id=...)

━━━ Step 6: Verifying logs and behavior
✓ Found startup message in logs
✓ Dev mode warning present in logs
✓ No Auth0 validation errors in dev mode (correct)

━━━ Step 7: Testing graceful shutdown
✓ Graceful shutdown completed

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ All Docker Integration Tests Passed!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Option 2: Manual Testing (Interactive)

Start the test environment manually for interactive testing:

```bash
# Start the environment
make test-mcp-docker-up

# Wait for services to be ready (shown in output)
# Then test manually:

# Test health
curl http://localhost:8082/healthz

# Test readiness
curl http://localhost:8082/readyz

# Test MCP initialize
curl -X POST http://localhost:8082/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Protocol-Version: 2025-03-26" \
  -H "X-Debug-Sub: test-user" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "capabilities": {},
      "clientInfo": {"name": "test", "version": "1.0"}
    }
  }'

# View logs
docker-compose -f docker-compose.mcp-test.yml logs -f mcpbridge-dev

# When done, stop the environment
make test-mcp-docker-down
```

## What Gets Tested

The automated test suite verifies:

### 1. Docker Image Build
- ✅ Image builds successfully from Dockerfile
- ✅ All required files are included
- ✅ Binary is executable

### 2. Service Health
- ✅ PostgreSQL starts and becomes ready
- ✅ Database migrations complete successfully
- ✅ REST API starts and becomes healthy
- ✅ MCP bridge starts and becomes healthy

### 3. REST API Functionality
- ✅ Health endpoint responds correctly
- ✅ Session creation works
- ✅ Authentication works (X-Debug-Sub in dev mode)

### 4. MCP Bridge Functionality
- ✅ Health endpoint (`/healthz`) responds
- ✅ Readiness endpoint (`/readyz`) reports correct status
- ✅ Dev mode is correctly enabled
- ✅ MCP initialize endpoint works
- ✅ Sessions are created and tracked

### 5. Logging and Behavior
- ✅ Startup messages appear in logs
- ✅ Dev mode warning is present
- ✅ No Auth0 errors in dev mode
- ✅ Log level is respected

### 6. Graceful Shutdown
- ✅ Container stops cleanly without hanging
- ✅ Shutdown messages appear in logs
- ✅ No goroutine leaks or deadlocks

## Test Scenarios

### Scenario 1: Dev Mode (Default)

**What it tests:**
- MCP bridge starts without Auth0 configuration
- Uses `X-Debug-Sub` header for authentication
- All endpoints work correctly
- Readiness is always "ready" in dev mode

**Configuration:**
```yaml
environment:
  MCP_DEV_MODE: "true"
  MCP_DEBUG: "true"
  MCP_API_BASE_URL: http://toolbridge-api:8081
```

### Scenario 2: Production Mode with Retry Logic (Optional)

**What it tests:**
- Background JWKS retry mechanism
- Health passes even when JWT validator isn't ready
- Readiness reports "not ready" when JWT validator unavailable
- Retry logic runs in background

**To enable:**
```bash
TEST_RETRY_LOGIC=true make test-mcp-docker
```

This starts an additional service (`mcpbridge-prod-retry`) with invalid Auth0 config to test the retry mechanism.

## Architecture

The test environment consists of:

```
┌─────────────────────────────────────────┐
│ PostgreSQL (postgres:16-alpine)         │
│ Port: 5432                              │
│ Healthcheck: pg_isready                 │
└────────────┬────────────────────────────┘
             │
             │ (after healthy)
             ▼
┌─────────────────────────────────────────┐
│ Migration Job (toolbridge-api:latest)   │
│ Runs: /app/scripts/migrate.sh           │
│ Waits for: postgres healthy             │
└────────────┬────────────────────────────┘
             │
             │ (after completed)
             ▼
┌─────────────────────────────────────────┐
│ REST API (toolbridge-api:latest)        │
│ Port: 8081                              │
│ Healthcheck: /healthz                   │
│ Waits for: migrations complete          │
└────────────┬────────────────────────────┘
             │
             │ (after healthy)
             ▼
┌─────────────────────────────────────────┐
│ MCP Bridge (toolbridge-mcpbridge:latest)│
│ Port: 8082                              │
│ Mode: Dev (no Auth0)                    │
│ Healthcheck: /healthz                   │
│ Waits for: REST API healthy             │
└─────────────────────────────────────────┘
```

## Troubleshooting

### Services don't start

**Problem:** `docker-compose up` fails

**Solutions:**
1. Check if ports are already in use:
   ```bash
   lsof -i :5432  # PostgreSQL
   lsof -i :8081  # REST API
   lsof -i :8082  # MCP Bridge
   ```

2. Clean up previous test runs:
   ```bash
   make test-mcp-docker-down
   docker system prune -f
   ```

3. Check Docker daemon is running:
   ```bash
   docker info
   ```

### Tests fail

**Problem:** Test script reports failures

**Debug steps:**
1. Check service logs:
   ```bash
   docker-compose -f docker-compose.mcp-test.yml logs postgres
   docker-compose -f docker-compose.mcp-test.yml logs toolbridge-api
   docker-compose -f docker-compose.mcp-test.yml logs mcpbridge-dev
   ```

2. Inspect container status:
   ```bash
   docker-compose -f docker-compose.mcp-test.yml ps
   ```

3. Test endpoints manually:
   ```bash
   curl -v http://localhost:8082/healthz
   curl -v http://localhost:8082/readyz
   ```

### Image build fails

**Problem:** Docker build fails

**Solutions:**
1. Check Dockerfile syntax:
   ```bash
   docker build -f cmd/mcpbridge/Dockerfile --no-cache .
   ```

2. Verify Go dependencies:
   ```bash
   go mod tidy
   go mod verify
   ```

3. Check disk space:
   ```bash
   df -h
   docker system df
   ```

## Next Steps

After successful local testing:

1. **Review test results** - Ensure all tests passed
2. **Test manually** - Try different scenarios interactively
3. **Build multi-arch image** - `make docker-build-mcp-multiarch`
4. **Push to registry** - `make docker-release-mcp`
5. **Deploy to Kubernetes** - `make helm-mcp-release`

## Advanced Testing

### Test with Real Auth0 Configuration

To test with actual Auth0 (not dev mode):

1. Create a test configuration file:
   ```json
   {
     "apiBaseUrl": "http://toolbridge-api:8081",
     "devMode": false,
     "debug": true,
     "auth0": {
       "domain": "your-tenant.us.auth0.com",
       "clients": {
         "native": {
           "clientId": "YOUR_CLIENT_ID"
         }
       },
       "syncApi": {
         "audience": "https://api.toolbridge.com"
       }
     }
   }
   ```

2. Mount the config in docker-compose:
   ```yaml
   mcpbridge-auth0-test:
     image: toolbridge-mcpbridge:latest
     volumes:
       - ./test-auth0-config.json:/config.json:ro
     environment:
       MCP_CONFIG_PATH: /config.json
   ```

3. Obtain a valid JWT token from Auth0
4. Test with the token:
   ```bash
   curl -X POST http://localhost:8082/mcp \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     -H "Content-Type: application/json" \
     ...
   ```

### Load Testing

Test concurrent connections:

```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Start test environment
make test-mcp-docker-up

# Run load test
hey -n 1000 -c 10 http://localhost:8082/healthz
```

## CI/CD Integration

To run these tests in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Test MCP Bridge Docker Image
  run: |
    make test-mcp-docker
```

The test script automatically:
- Builds the image
- Starts all services
- Runs tests
- Cleans up resources (even on failure)

## Reference

### Make Targets

| Target | Description |
|--------|-------------|
| `make test-mcp-docker` | Run full automated test suite |
| `make test-mcp-docker-up` | Start test environment (manual) |
| `make test-mcp-docker-down` | Stop test environment |
| `make test-mcp-smoke` | Run basic smoke tests (binary only) |
| `make build-mcp` | Build MCP bridge binary |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TEST_RETRY_LOGIC` | `false` | Enable production retry logic tests |

### Service Ports

| Service | Port | Endpoint |
|---------|------|----------|
| PostgreSQL | 5432 | - |
| REST API | 8081 | http://localhost:8081 |
| MCP Bridge (dev) | 8082 | http://localhost:8082 |
| MCP Bridge (retry test) | 8083 | http://localhost:8083 |

## FAQ

**Q: How long do the tests take?**
A: Typically 30-60 seconds for the full suite, including image build and service startup.

**Q: Can I run tests in parallel?**
A: No, the tests use fixed ports. Run them sequentially or on different machines.

**Q: Do I need to rebuild the image for every test?**
A: No. If the image is already built, the test script will use it. Use `docker rmi toolbridge-mcpbridge:latest` to force a rebuild.

**Q: Can I keep the environment running for manual testing?**
A: Yes! Use `make test-mcp-docker-up` to start it manually, then `make test-mcp-docker-down` when done.

**Q: How do I test with my changes?**
A: Just run `make test-mcp-docker` - it always builds a fresh image from your current code.

## Support

If you encounter issues:

1. Check this guide's troubleshooting section
2. Review service logs: `docker-compose -f docker-compose.mcp-test.yml logs`
3. Open an issue on GitHub with logs and error messages
