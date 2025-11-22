# ToolBridge Secrets Reference

This document details all secrets and environment variables required for ToolBridge deployments across different environments.

## Overview

ToolBridge uses dual authentication:
1. **JWT tokens** for user identity validation
2. **Signed tenant headers** for tenant isolation and cross-service authentication

**Critical:** The `TENANT_HEADER_SECRET` must match across all components (Go API in K8s and MCP proxy in Fly.io).

## Kubernetes Deployment (Go API + PostgreSQL)

### Required Secrets

Location: `homelab-k8s/apps/toolbridge-api/production-overlays/toolbridge-secret.sops.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: toolbridge-secret
  namespace: toolbridge
type: Opaque
stringData:
  # PostgreSQL configuration
  username: toolbridge
  password: <generated-postgres-password>
  database-url: postgres://toolbridge:<password>@toolbridge-api-postgres-rw.toolbridge.svc.cluster.local:5432/toolbridge?sslmode=require

  # JWT configuration (HS256 for backend tokens, OIDC RS256 for production)
  jwt-secret: <generated-hs256-secret>  # Required for backend token signing

  # Tenant header authentication (CRITICAL - must match Fly.io)
  tenant-header-secret: <generated-tenant-secret>
```

### Generating Secrets

```bash
# PostgreSQL password
openssl rand -base64 32

# JWT HS256 secret (required for backend token signing)
openssl rand -base64 32

# Tenant header secret (SAVE THIS - needed for Fly.io)
openssl rand -base64 32
```

### Managing K8s Secrets

```bash
# Decrypt and edit with SOPS
cd homelab-k8s/apps/toolbridge-api/production-overlays
sops toolbridge-secret.sops.yaml

# After editing, commit (file stays encrypted)
git add toolbridge-secret.sops.yaml
git commit -m "chore: update secrets"
git push

# Retrieve secret value (for Fly.io setup)
kubectl get secret toolbridge-secret -n toolbridge \
  -o jsonpath='{.data.tenant-header-secret}' | base64 -d
```

### Helm Values

Location: `homelab-k8s/apps/toolbridge-api/helm-values-production.yaml`

Non-secret configuration:

```yaml
api:
  env: production
  oidc:
    issuer: "https://svelte-monolith-27-staging.authkit.app"
    jwksUrl: "https://svelte-monolith-27-staging.authkit.app/oauth2/jwks"
    audience: "https://toolbridgeapi.erauner.dev"

secrets:
  existingSecret: "toolbridge-secret"  # References SOPS secret above
```

## Fly.io Deployment (MCP Proxy Only)

### Required Secrets

Set via `fly secrets set -a <app-name>`:

```bash
# External Go API endpoint (K8s ingress)
TOOLBRIDGE_GO_API_BASE_URL="https://toolbridgeapi.erauner.dev"

# Tenant identification
TOOLBRIDGE_TENANT_ID="staging-tenant-001"  # or per-tenant ID

# Tenant header authentication (MUST match K8s secret)
TOOLBRIDGE_TENANT_HEADER_SECRET="<same-as-k8s-tenant-header-secret>"
```

### Optional Secrets

```bash
# Logging level
TOOLBRIDGE_LOG_LEVEL="INFO"  # DEBUG, INFO, WARNING, ERROR

# Custom timeouts (rarely needed)
TOOLBRIDGE_MAX_TIMESTAMP_SKEW_SECONDS="300"  # default: 5 minutes
```

### Setting Secrets

```bash
# Initial setup
fly secrets set -a toolbridge-mcp-staging \
  TOOLBRIDGE_GO_API_BASE_URL="https://toolbridgeapi.erauner.dev" \
  TOOLBRIDGE_TENANT_ID="staging-tenant-001" \
  TOOLBRIDGE_TENANT_HEADER_SECRET="<from-k8s-secret>"

# Update a single secret
fly secrets set TOOLBRIDGE_LOG_LEVEL="DEBUG" -a toolbridge-mcp-staging

# List configured secrets (values hidden)
fly secrets list -a toolbridge-mcp-staging

# Unset a secret
fly secrets unset TOOLBRIDGE_LOG_LEVEL -a toolbridge-mcp-staging
```

## Local Development

### Go API (.env)

Location: `toolbridge-api/.env`

```bash
# Database
DATABASE_URL=postgres://toolbridge:dev-password@localhost:5432/toolbridge?sslmode=disable

# JWT authentication
JWT_HS256_SECRET=dev-secret-change-in-production
ENV=dev  # Enables X-Debug-Sub header bypass

# Tenant header authentication
TENANT_HEADER_SECRET=dev-tenant-secret

# HTTP server
HTTP_ADDR=:8080
```

### Python MCP Service (mcp/.env)

Location: `toolbridge-api/mcp/.env`

```bash
# Tenant configuration
TOOLBRIDGE_TENANT_ID=test-tenant-123
TOOLBRIDGE_TENANT_HEADER_SECRET=dev-tenant-secret  # Must match Go API

# Go API connection
TOOLBRIDGE_GO_API_BASE_URL=http://localhost:8080

# Logging
TOOLBRIDGE_LOG_LEVEL=DEBUG

# Server config (default values)
TOOLBRIDGE_HOST=0.0.0.0
TOOLBRIDGE_PORT=8001
TOOLBRIDGE_MAX_TIMESTAMP_SKEW_SECONDS=300
```

## Secret Rotation

### When to Rotate

- **Tenant header secret:** Every 90 days or on suspected compromise
- **JWT HS256 secret:** Every 90 days (if using HS256)
- **PostgreSQL password:** Every 180 days or on suspected compromise

### Rotation Process

#### 1. Rotate Tenant Header Secret

**Critical:** Must be coordinated across K8s and Fly.io to avoid downtime.

```bash
# Step 1: Generate new secret
NEW_SECRET=$(openssl rand -base64 32)
echo "New secret: $NEW_SECRET"  # Save securely

# Step 2: Update K8s secret
cd homelab-k8s/apps/toolbridge-api/production-overlays
sops toolbridge-secret.sops.yaml
# Edit tenant-header-secret value, save, commit, push

# Step 3: Wait for ArgoCD to sync and redeploy Go API
kubectl rollout status deployment/toolbridge-api -n toolbridge

# Step 4: Update Fly.io immediately after K8s deployment completes
fly secrets set TOOLBRIDGE_TENANT_HEADER_SECRET="$NEW_SECRET" -a toolbridge-mcp-staging

# Step 5: Verify both services work
curl https://toolbridgeapi.erauner.dev/healthz
curl https://toolbridge-mcp-staging.fly.dev/
```

#### 2. Rotate PostgreSQL Password

**Note:** Requires brief downtime or connection pool drain.

```bash
# Step 1: Generate new password
NEW_PG_PASSWORD=$(openssl rand -base64 32)

# Step 2: Update PostgreSQL user password
kubectl exec -it -n toolbridge toolbridge-api-postgres-1 -- psql -U postgres
# In psql:
ALTER USER toolbridge PASSWORD 'new-password-here';
\q

# Step 3: Update K8s secret
sops toolbridge-secret.sops.yaml
# Update both 'password' and 'database-url', commit, push

# Step 4: Restart Go API pods to pick up new connection string
kubectl rollout restart deployment/toolbridge-api -n toolbridge
```

#### 3. Rotate JWT HS256 Secret (if using)

**Note:** Invalidates all existing tokens. Users must re-authenticate.

```bash
# Step 1: Generate new secret
NEW_JWT_SECRET=$(openssl rand -base64 32)

# Step 2: Update K8s secret
sops toolbridge-secret.sops.yaml
# Update jwt-secret, commit, push

# Step 3: Wait for deployment
kubectl rollout status deployment/toolbridge-api -n toolbridge

# Step 4: Notify users to re-authenticate (all tokens now invalid)
```

## Secret Validation

### Test K8s → Go API

```bash
# Test with debug header (dev only)
curl -H "X-Debug-Sub: test-user" https://toolbridgeapi.erauner.dev/healthz

# Test with JWT token
TOKEN=$(python -c "import jwt; print(jwt.encode({'sub': 'test-user'}, 'your-jwt-secret', algorithm='HS256'))")
curl -H "Authorization: Bearer $TOKEN" https://toolbridgeapi.erauner.dev/v1/notes
```

### Test Fly.io → K8s (Tenant Headers)

```bash
# Get current tenant header secret from K8s
TENANT_SECRET=$(kubectl get secret toolbridge-secret -n toolbridge \
  -o jsonpath='{.data.tenant-header-secret}' | base64 -d)

# Generate signed headers (Python)
python3 << EOF
import hmac, hashlib, time
tenant_id = "staging-tenant-001"
secret = "$TENANT_SECRET"
timestamp_ms = int(time.time() * 1000)
message = f"{tenant_id}:{timestamp_ms}"
signature = hmac.new(secret.encode(), message.encode(), hashlib.sha256).hexdigest()
print(f"X-TB-Tenant-ID: {tenant_id}")
print(f"X-TB-Timestamp: {timestamp_ms}")
print(f"X-TB-Signature: {signature}")
EOF

# Test request with headers
curl -H "X-TB-Tenant-ID: staging-tenant-001" \
     -H "X-TB-Timestamp: <timestamp>" \
     -H "X-TB-Signature: <signature>" \
     -H "Authorization: Bearer test-token" \
     https://toolbridgeapi.erauner.dev/v1/notes
```

### Test End-to-End (MCP → K8s)

```bash
# Test via Fly.io MCP proxy
# The MCP proxy should automatically add tenant headers
curl -H "Authorization: Bearer test-token" \
  https://toolbridge-mcp-staging.fly.dev/

# Check logs for successful Go API calls
fly logs -a toolbridge-mcp-staging | grep "200 OK"
```

## Security Best Practices

### Secret Storage

- ✅ **K8s:** Use SOPS with age encryption
- ✅ **Fly.io:** Use `fly secrets` (encrypted at rest)
- ✅ **Local:** Use `.env` files (never commit to git)
- ❌ **Never:** Store secrets in code, config files, or CI logs

### Secret Sharing

- ✅ Use password managers (1Password, Bitwarden) for team sharing
- ✅ Use encrypted channels (Signal, encrypted email) for one-time sharing
- ❌ Never share via Slack, email, or text messages

### Secret Access Control

- **K8s secrets:** Only cluster admins and ArgoCD
- **Fly.io secrets:** Only app deployers
- **SOPS age keys:** Only platform team members

### Monitoring

Set up alerts for:
- Failed JWT validation attempts (potential token theft)
- Failed tenant header validation (potential MITM)
- Unusual secret access patterns
- Expired/expiring secrets

## Troubleshooting

### "Invalid tenant headers" Error

```bash
# Verify secrets match
# K8s side:
kubectl get secret toolbridge-secret -n toolbridge \
  -o jsonpath='{.data.tenant-header-secret}' | base64 -d

# Fly.io side (can't view, but can update):
fly secrets set TOOLBRIDGE_TENANT_HEADER_SECRET="<k8s-value>" -a toolbridge-mcp-staging
```

### "JWT validation failed" Error

```bash
# Check JWT secret is correct
kubectl get secret toolbridge-secret -n toolbridge \
  -o jsonpath='{.data.jwt-secret}' | base64 -d

# For Auth0 tokens, verify domain and audience in helm values
```

### "Connection refused" to Go API

```bash
# Verify TOOLBRIDGE_GO_API_BASE_URL is correct
fly secrets list -a toolbridge-mcp-staging

# Test Go API directly
curl https://toolbridgeapi.erauner.dev/healthz
```

## References

- **SOPS Documentation:** https://github.com/mozilla/sops
- **Fly.io Secrets:** https://fly.io/docs/reference/secrets/
- **JWT Best Practices:** https://tools.ietf.org/html/rfc8725
- **OWASP Secret Management:** https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
