# MCP Bridge - Pre-Deployment Testing Checklist

Use this checklist before deploying the MCP bridge to production.

## Local Docker Testing

### ✅ Automated Test Suite

Run the full automated test suite:

```bash
make test-mcp-docker
```

**Expected Results:**
- [ ] Docker image builds successfully
- [ ] PostgreSQL starts and becomes healthy
- [ ] Database migrations complete
- [ ] REST API starts and becomes healthy
- [ ] MCP bridge starts and becomes healthy
- [ ] REST API health check passes
- [ ] REST API session creation works
- [ ] MCP bridge health check passes
- [ ] MCP bridge readiness check passes
- [ ] MCP initialize endpoint works
- [ ] Dev mode is correctly reported
- [ ] Logs show proper startup messages
- [ ] Graceful shutdown works without hanging
- [ ] All resources clean up properly

**Time:** ~30-60 seconds

### ✅ Manual Verification (Optional)

For interactive testing:

```bash
# 1. Start environment
make test-mcp-docker-up

# 2. Test health endpoints
curl http://localhost:8082/healthz    # Should return {"status":"ok"}
curl http://localhost:8082/readyz     # Should return {"status":"ready","devMode":true}

# 3. Test MCP initialize
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

# 4. View logs (check for errors)
docker-compose -f docker-compose.mcp-test.yml logs mcpbridge-dev

# 5. Test graceful shutdown
docker-compose -f docker-compose.mcp-test.yml stop mcpbridge-dev
docker-compose -f docker-compose.mcp-test.yml logs mcpbridge-dev | tail -20
# Should see "Shutting down MCP server..."

# 6. Clean up
make test-mcp-docker-down
```

**Checklist:**
- [ ] Health endpoint responds immediately
- [ ] Readiness shows `devMode: true`
- [ ] Initialize creates a session
- [ ] No errors in logs
- [ ] Shutdown is clean (no hangs)

## Unit Tests

Run unit tests to verify code correctness:

```bash
make test-mcp
```

**Expected Results:**
- [ ] All config loader tests pass
- [ ] Environment variable precedence works
- [ ] CLI flag overrides work correctly
- [ ] Validation logic is correct
- [ ] No race conditions detected

**Time:** ~5-10 seconds

## Smoke Tests

Quick verification of the binary:

```bash
make test-mcp-smoke
```

**Expected Results:**
- [ ] Binary runs in dev mode without config
- [ ] Debug logging works
- [ ] Version flag works
- [ ] No Auth0 errors in dev mode

**Time:** ~10 seconds

## Production Readiness Checks

### Configuration

- [ ] Auth0 tenant is set up
- [ ] API audience is configured
- [ ] Client IDs are created (web/native/macOS as needed)
- [ ] Helm values file is prepared
- [ ] Secrets are configured (via external secret manager)
- [ ] Allowed origins are set (not empty!)

### Docker Image

- [ ] Multi-arch image builds: `make docker-build-mcp-multiarch`
- [ ] Image size is reasonable (< 50MB)
- [ ] Image runs on both amd64 and arm64
- [ ] Health check is configured in Dockerfile
- [ ] Image is pushed to registry: `make docker-release-mcp`

### Helm Chart

- [ ] Chart lints successfully: `make helm-mcp-lint`
- [ ] Values file has all required fields
- [ ] Secrets reference is correct
- [ ] Resource limits are appropriate
- [ ] Health/readiness probes are configured
- [ ] Chart packages successfully: `make helm-mcp-package`

### Kubernetes Environment

- [ ] Gateway API is available in cluster
- [ ] Cert-manager is installed
- [ ] ClusterIssuer exists for TLS
- [ ] Namespace is created
- [ ] External secrets operator is configured (if used)
- [ ] DNS is configured for ingress hostname

## Pre-Deployment Test Matrix

| Test Type | Command | Duration | Required |
|-----------|---------|----------|----------|
| Unit Tests | `make test-mcp` | ~5s | ✅ Yes |
| Smoke Tests | `make test-mcp-smoke` | ~10s | ✅ Yes |
| Docker Integration | `make test-mcp-docker` | ~60s | ✅ Yes |
| Manual Verification | `make test-mcp-docker-up` | Variable | ⚪ Optional |

## Common Issues and Solutions

### Issue: Docker build fails

**Solution:**
```bash
# Clean Docker cache and rebuild
docker system prune -f
make docker-build-mcp-local
```

### Issue: Services don't become healthy

**Solution:**
```bash
# Check logs
docker-compose -f docker-compose.mcp-test.yml logs

# Check port conflicts
lsof -i :5432
lsof -i :8081
lsof -i :8082

# Clean up and retry
make test-mcp-docker-down
make test-mcp-docker
```

### Issue: Tests fail intermittently

**Solution:**
```bash
# Increase wait times in test script (edit scripts/test-mcp-bridge-docker.sh)
# Or add DEBUG=1 to see more details:
DEBUG=1 make test-mcp-docker
```

### Issue: Graceful shutdown hangs

**Solution:**
This was a known issue that's been fixed. If you still see this:
1. Check logs for goroutine leaks
2. Verify signal handlers are registered
3. Check context cancellation is working

## Sign-Off

Before proceeding to deployment:

```
Date: __________
Tester: __________

✅ All automated tests passed
✅ Manual verification completed
✅ Configuration reviewed
✅ Docker image built and tested
✅ Helm chart validated
✅ Production environment prepared

Ready for deployment: [ ] Yes  [ ] No

Notes:
_____________________________________
_____________________________________
_____________________________________
```

## Next Steps After Testing

1. **Commit changes:**
   ```bash
   git add .
   git commit -m "Add MCP bridge local testing infrastructure"
   git push
   ```

2. **Create PR** with test results in description

3. **After PR approval:**
   ```bash
   # Build and push multi-arch image
   make docker-release-mcp VERSION=v0.1.0

   # Deploy to Kubernetes
   make helm-mcp-release
   ```

4. **Post-deployment verification:**
   - Check pod status: `kubectl get pods -n toolbridge-mcp`
   - Verify health: `kubectl exec -it <pod> -- wget -qO- http://localhost:8082/healthz`
   - Check logs: `kubectl logs -f <pod>`
   - Test from external client

## References

- [Local Testing Guide](./mcp-bridge-local-testing.md)
- [Production Setup Guide](../cmd/mcpbridge/docs/production-setup.md)
- [MCP Bridge README](../cmd/mcpbridge/README.md)
