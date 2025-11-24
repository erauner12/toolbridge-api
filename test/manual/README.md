# Manual Tests

This directory contains manual integration tests that demonstrate and verify the OIDC tenant resolution flow.

## Tests Overview

### `test_full_tenant_flow.go` ⭐ **START HERE**

**Purpose**: Complete end-to-end reference implementation of tenant resolution flow

**What it demonstrates**:
1. OIDC Discovery (`.well-known/openid-configuration`)
2. PKCE authentication flow (code verifier/challenge)
3. Authorization via browser
4. Token exchange (authorization code → ID token)
5. Backend tenant resolution (`/v1/auth/tenant` endpoint)
6. Complete flow with proper error handling

**Usage**:
```bash
# Test against local backend
go run test/manual/test_full_tenant_flow.go

# Test against production
BACKEND_URL=https://toolbridgeapi.erauner.dev go run test/manual/test_full_tenant_flow.go
```

**Output**:
```
=== Complete Tenant Resolution Flow Test ===
Step 1: Discovering OIDC endpoints...
   ✓ Authorization: https://...
Step 2: Generating PKCE parameters...
   ✓ Code verifier: ...
Step 3: Building authorization URL...
   ✓ Authorization URL ready
Step 4: Performing authorization (browser will open)...
   ✓ Authorization code: ...
Step 5: Exchanging authorization code for tokens...
   ✓ ID token obtained: ...
Step 6: Calling backend /v1/auth/tenant endpoint...
   Backend URL: https://toolbridgeapi.erauner.dev
   ✓ Tenant resolved successfully!

=== Tenant Resolution Result ===
Tenant ID: org_01KABXHNF45RMV9KBWF3SBGPP0
Organization Name: Test Organization
Requires Selection: false

=== Flow Complete ===
✅ SUCCESS - Full tenant resolution flow completed
```

**Use this as reference** when implementing the Flutter client!

---

### `standard_oidc_tenant.go`

**Purpose**: Prove that standard OIDC tokens do NOT contain `organization_id` claim

**What it demonstrates**:
- Standard OIDC authentication flow
- Token inspection showing missing `organization_id`
- Why backend resolution is necessary

**Usage**:
```bash
go run test/manual/standard_oidc_tenant.go
```

**Key Finding**:
```
--- ID Token ---
   Claims: {
     "aud": "client_01KAPCBQNQBWMZE9WNSEWY2J3Z",
     "email": "raunerevan@gmail.com",
     "exp": 1764004121,
     "iat": 1764000521,
     "iss": "https://svelte-monolith-27-staging.authkit.app",
     "name": "Evan Rauner",
     "sub": "user_01KAHS4J1W6TT5390SR3918ZPF"
   }

✗ FAIL: No organization_id claim found in ID token
```

This proves that client-side token inspection cannot determine tenant ID.

---

### `test_tenant_resolution.go`

**Purpose**: Simple endpoint testing utility

**What it does**:
- Accepts an ID token via environment variable
- Calls `/v1/auth/tenant` endpoint
- Displays the response

**Usage**:
```bash
# 1. Get ID token from standard_oidc_tenant.go
go run test/manual/standard_oidc_tenant.go
# Copy the export ID_TOKEN=... line

# 2. Test endpoint
export ID_TOKEN='eyJhbGc...'
export BACKEND_URL='https://toolbridgeapi.erauner.dev'  # optional
go run test/manual/test_tenant_resolution.go
```

**Output**:
```json
{
  "tenant_id": "org_01KABXHNF45RMV9KBWF3SBGPP0",
  "organization_name": "Test Organization",
  "requires_selection": false
}
```

---

## Configuration

All tests use the following WorkOS AuthKit configuration:

```go
const (
    issuerURL      = "https://svelte-monolith-27-staging.authkit.app"
    clientID       = "client_01KAPCBQNQBWMZE9WNSEWY2J3Z"
    redirectURI    = "http://localhost:3000/callback"
    organizationID = "org_01KABXHNF45RMV9KBWF3SBGPP0"
)
```

### Environment Variables

- `BACKEND_URL`: Backend API base URL (default: `http://localhost:8080`)
  - Production: `https://toolbridgeapi.erauner.dev`
  - Local: `http://localhost:8080`

- `ID_TOKEN`: ID token for testing tenant resolution endpoint
  - Get from `standard_oidc_tenant.go` output

## Expected Behavior

### Single Organization User

When a user belongs to one organization:
```json
{
  "tenant_id": "org_01KABXHNF45RMV9KBWF3SBGPP0",
  "organization_name": "Test Organization",
  "requires_selection": false
}
```

### Multi-Organization User

When a user belongs to multiple organizations:
```json
{
  "organizations": [
    {"id": "org_01...", "name": "Acme Corp"},
    {"id": "org_02...", "name": "Globex Inc"}
  ],
  "requires_selection": true
}
```

Client must present selection UI (not yet implemented in Flutter).

## Troubleshooting

### "Token is expired" Error

ID tokens expire after 1 hour. Get a fresh token:
```bash
go run test/manual/standard_oidc_tenant.go
```

### "WorkOS tenant resolution not configured"

Backend missing `WORKOS_API_KEY` environment variable. Check:
```bash
kubectl get secret toolbridge-secret -n toolbridge -o jsonpath='{.data.workos-api-key}' | base64 -d
```

### "User not in any organization"

The test user (`raunerevan@gmail.com`) is not a member of the test organization. Check WorkOS dashboard.

### "Invalid token"

Token validation failed. Possible causes:
- Token expired (get fresh token)
- Token audience mismatch (check JWT_AUDIENCE config)
- JWKS fetch failed (check network/DNS)

## Implementation Notes for Flutter

Key takeaways from these tests:

1. **Use flutter_appauth** for OIDC/PKCE flow (mirrors Go implementation)
2. **Store tokens securely** using flutter_secure_storage
3. **Call `/v1/auth/tenant`** immediately after authentication
4. **Cache tenant_id** for use in subsequent API requests
5. **Handle multi-org case** by showing selection UI (future work)

See `docs/tenant-resolution.md` for complete implementation guide.

## Related Documentation

- Main documentation: `/docs/tenant-resolution.md`
- WorkOS AuthKit: https://workos.com/docs/authkit
- OIDC Spec: https://openid.net/specs/openid-connect-core-1_0.html
- PKCE RFC: https://datatracker.ietf.org/doc/html/rfc7636
