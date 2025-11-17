# Trust-Proxy Mode Deprecation Path

This document outlines code that can be deprecated/removed once trust-proxy mode (via ToolHive or similar proxies) becomes the primary deployment pattern.

## Architecture Comparison

### Current: Standalone Mode
```
Client → ToolBridge (JWT validation, OAuth metadata) → REST API
         ├─ Validates JWT signatures via JWKS
         ├─ Serves OAuth metadata (RFC 8414/9728)
         └─ Manages token lifecycle
```

### Future: Trust-Proxy Mode
```
Client → ToolHive Proxy → ToolBridge → REST API
         ├─ Validates JWT           ├─ Parses JWT claims (no validation)
         ├─ Serves OAuth metadata   ├─ Trusts proxy authentication
         └─ Manages tokens          └─ Focuses on MCP protocol
```

## Code to Remove

### 1. JWT Validation (`internal/mcpserver/server/jwt.go`)
**Entire file can be removed** (~300 lines)

**What it does:**
- Fetches JWKS from Auth0
- Validates JWT signatures (RS256)
- Handles key rotation
- Background retry logic for JWKS failures

**Why it's obsolete:**
- ToolHive proxy handles all JWT validation
- ToolBridge only needs to parse claims (no crypto validation)
- `parseJWTClaimsWithoutValidation()` is sufficient

**Migration path:**
1. Deploy all ToolBridge instances behind ToolHive
2. Verify all use `TrustToolhiveAuth=true`
3. Remove `JWTValidator` type
4. Remove JWKS fetching logic
5. Remove JWT validation from handlers

### 2. OAuth Metadata (`internal/mcpserver/server/oauth_metadata.go`)
**Entire file can be removed** (~80 lines)

**What it does:**
- Serves `GET /.well-known/oauth-authorization-server` (RFC 8414)
- Serves `GET /.well-known/oauth-protected-resource` (RFC 9728)
- Advertises Auth0 OAuth endpoints to MCP clients

**Why it's obsolete:**
- ToolHive proxy serves its own OAuth metadata
- Clients discover OAuth from ToolHive, not ToolBridge
- Already guarded to return 501 in trust-proxy mode

**Migration path:**
1. Verify clients connect to ToolHive for OAuth discovery
2. Confirm no clients directly access ToolBridge OAuth metadata
3. Remove both endpoint handlers
4. Remove routes from `setupRoutes()`

### 3. Auth0 Config Fields (`internal/mcpserver/config/config.go`)

**Fields to remove:**
```go
type Auth0Config struct {
    Domain  string                  // REMOVE: only for JWT validation
    Clients map[string]ClientConfig // REMOVE: only for OAuth flow
    SyncAPI *SyncAPIConfig          // KEEP: needed for REST API calls
}

type ClientConfig struct { ... } // REMOVE ENTIRE STRUCT
```

**Why Domain/Clients are obsolete:**
- `Domain`: Only used for JWT validation and OAuth metadata endpoints
- `Clients`: Only used for OAuth client configuration
- ToolHive handles both

**Why SyncAPI.Audience must stay:**
- ToolBridge still needs to call the REST API
- REST API requires valid audience in token
- This is backend-to-backend auth, not client auth

**Migration path:**
1. Make `Domain` and `Clients` optional
2. Update `Config.Validate()` to not require them in trust-proxy mode
3. Eventually remove fields entirely
4. Keep only `SyncAPI.Audience` for REST API calls

### 4. JWT Validation in Handlers

**Code to remove from handlers:**
```go
// In handleMCPPost, handleMCPGet, handleMCPDelete
if userID == "" {
    if s.jwtValidator == nil {
        s.sendError(w, nil, InternalError, "authentication not configured")
        return
    }
    // ... JWT validation code
}
```

**Why it's obsolete:**
- Only runs when `TrustToolhiveAuth=false` and `DevMode=false`
- If all deployments are behind ToolHive, this path is never taken
- Can simplify to just: trust-proxy OR dev-mode OR error

**Migration path:**
1. Deploy all instances with `TrustToolhiveAuth=true`
2. Remove standalone JWT validation path
3. Keep only trust-proxy and dev-mode paths

### 5. NewMCPServer JWT Validator Creation

**Code to simplify:**
```go
func NewMCPServer(cfg *config.Config) *MCPServer {
    var jwtValidator *JWTValidator

    // REMOVE THIS ENTIRE BLOCK
    if !cfg.DevMode && !cfg.TrustToolhiveAuth && cfg.Auth0.SyncAPI != nil {
        jwtValidator = NewJWTValidator(cfg.Auth0.Domain, cfg.Auth0.SyncAPI.Audience)
    }

    return &MCPServer{
        jwtValidator: jwtValidator, // REMOVE: always nil
        // ...
    }
}
```

### 6. WarmUp Method

**Code to remove:**
```go
func (s *MCPServer) WarmUp(ctx context.Context) error {
    if s.jwtValidator != nil {
        return s.jwtValidator.WarmUp(ctx)
    }
    return nil
}
```

**Why it's obsolete:**
- Only pre-fetches JWKS keys
- No JWT validator means no warmup needed

## Code to Keep

### 1. parseJWTClaimsWithoutValidation()
**KEEP** - Core function for trust-proxy mode

Parses JWT claims without cryptographic validation. Safe because proxy already validated.

### 2. Trust-Proxy Authentication Paths
**KEEP** - Core trust-proxy logic in all handlers

```go
if s.config.TrustToolhiveAuth {
    // Parse JWT from Authorization header
    claims, _, err := parseJWTClaimsWithoutValidation(tokenString)
    userID = claims["sub"].(string)
}
```

### 3. DevMode Authentication
**KEEP** - Useful for local development

```go
if s.config.DevMode {
    debugSub := r.Header.Get("X-Debug-Sub")
    if debugSub != "" {
        userID = debugSub
    }
}
```

### 4. SyncAPI.Audience Config
**KEEP** - Still needed for REST API calls

Even in trust-proxy mode, ToolBridge needs to know the audience for backend API tokens.

### 5. Session Management
**KEEP** - Core MCP functionality

Session management is independent of authentication method.

## Migration Timeline

### Phase 1: Add Trust-Proxy Support (✅ Done)
- Add `TrustToolhiveAuth` config flag
- Implement `parseJWTClaimsWithoutValidation()`
- Add trust-proxy paths to all handlers
- Guard OAuth metadata endpoints

### Phase 2: Deploy via ToolHive (In Progress)
- Deploy ToolBridge as MCPServer CR
- Set `TRUST_TOOLHIVE_AUTH=true`
- Verify authentication works
- Monitor for issues

### Phase 3: Sunset Standalone Mode (Future)
- Make standalone JWT validation opt-in (not default)
- Add deprecation warnings to logs
- Document migration path
- Support existing standalone deployments for 1-2 versions

### Phase 4: Remove JWT Validation (Future)
- Remove `jwt.go` entirely
- Remove `oauth_metadata.go` entirely
- Remove `jwtValidator` field from `MCPServer`
- Simplify `Auth0Config` to just `SyncAPI.Audience`
- Update tests

## Testing Strategy

Before removing any code:
1. ✅ Deploy trust-proxy mode to production
2. ✅ Verify all authentication flows work
3. ✅ Confirm no standalone deployments remain
4. ✅ Check logs for any JWT validator usage
5. ✅ Run full integration test suite
6. ✅ Document breaking changes in CHANGELOG

## Breaking Changes

Removing JWT validation will be a **breaking change** for:
- Standalone deployments (direct client → ToolBridge)
- Self-hosted instances without ToolHive
- Development environments without proxy

**Migration requirement:**
- Must deploy behind ToolHive or similar proxy
- Must set `TRUST_TOOLHIVE_AUTH=true`
- Must configure proxy for JWT validation

## Benefits of Simplification

1. **Less Code** - Remove ~400 lines of JWT/OAuth logic
2. **Fewer Dependencies** - No JWKS fetching, no crypto validation
3. **Better Separation of Concerns** - Auth in proxy, MCP in ToolBridge
4. **Simpler Testing** - No need to mock JWKS endpoints
5. **Faster Startup** - No JWKS warmup needed
6. **Less Surface Area** - Fewer auth-related bugs

## Questions to Answer Before Deprecation

1. Do any users run ToolBridge standalone (without proxy)?
2. Are there any integration tests that depend on JWT validation?
3. Does the REST API deployment also use trust-proxy mode?
4. Can we provide a migration guide for standalone users?
5. Should we keep JWT validation as an optional feature (behind flag)?

## Related Documentation

- [Trust-Proxy Architecture](./ARCHITECTURE.md) (TODO)
- [Migration Guide](./MIGRATION.md) (TODO)
- [ToolHive Integration](./TOOLHIVE_INTEGRATION.md) (TODO)
