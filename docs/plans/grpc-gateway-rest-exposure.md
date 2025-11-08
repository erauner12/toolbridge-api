# gRPC Gateway: Future REST API Exposure

**Status**: Future Consideration (Post Phase 1)
**Created**: 2025-11-08
**Effort**: 1-2 weeks
**Prerequisites**: Complete gRPC migration (Phase 1)

---

## What is gRPC Gateway?

gRPC Gateway automatically generates a REST/JSON API from your gRPC services. You only implement gRPC, but clients can use either gRPC or REST.

**This is exactly what Memos does.**

```
┌─────────────┐
│ REST Client │ (curl, browser, scripts, third-party APIs)
└──────┬──────┘
       │ HTTP/JSON
       ▼
┌─────────────────┐
│  gRPC Gateway   │ (auto-generated reverse proxy)
└──────┬──────────┘
       │ gRPC (internal)
       ▼
┌─────────────────┐
│  gRPC Server    │ (your Go code - unchanged)
└─────────────────┘
       ▲
       │ gRPC (internal)
┌──────┴──────┐
│ Flutter App │
└─────────────┘
```

---

## Why Consider This?

### Use Cases

**gRPC for mobile apps:**
- Efficient binary protocol (Protobuf)
- Native streaming support
- Type-safe client generation

**REST for everything else:**
- Public API access (third-party integrations)
- Webhooks and automation
- CLI tools and scripts
- Web browser access (fetch API)
- curl commands for debugging

### Benefits

1. **Single Implementation**: Only write gRPC handlers once
2. **Zero Code Duplication**: REST is auto-generated from proto annotations
3. **Best of Both Worlds**: Efficient gRPC + accessible REST
4. **Production Pattern**: Used by Memos, Buf, Google Cloud APIs

---

## How It Works

### 1. Add HTTP Annotations to Proto Files

**Before** (gRPC only):
```protobuf
service NotesService {
  rpc PushNotes(PushNotesRequest) returns (PushNotesResponse);
}
```

**After** (gRPC + REST):
```protobuf
import "google/api/annotations.proto";

service NotesService {
  rpc PushNotes(PushNotesRequest) returns (PushNotesResponse) {
    option (google.api.http) = {
      post: "/v1/sync/notes/push"
      body: "*"
    };
  }

  rpc PullNotes(PullNotesRequest) returns (PullNotesResponse) {
    option (google.api.http) = {
      get: "/v1/sync/notes/pull"
    };
  }
}
```

### 2. Generate Gateway Code

```bash
# Install gateway plugin
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

# Update Makefile
.PHONY: proto
proto:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		--grpc-gateway_out=. \
		proto/sync/v1/*.proto
```

### 3. Deploy Gateway Proxy

**Option A: In-Process** (simpler, single binary)
```go
func main() {
    // gRPC server
    grpcServer := grpc.NewServer(...)

    // Gateway mux
    gwMux := runtime.NewServeMux()
    pb.RegisterNotesServiceHandlerServer(ctx, gwMux, notesServiceServer)
    pb.RegisterTasksServiceHandlerServer(ctx, gwMux, tasksServiceServer)
    // ... register all services

    // Serve both:
    go grpcServer.Serve(":9090")  // gRPC
    http.ListenAndServe(":8080", gwMux)  // REST gateway
}
```

**Option B: Sidecar** (more flexible, separate deployment)
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: grpc-server
        ports:
        - containerPort: 9090
      - name: grpc-gateway
        image: toolbridge-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: GRPC_BACKEND
          value: "localhost:9090"
```

---

## Implementation Timeline

### Phase 3: REST Gateway (Optional, 1-2 weeks)

**Prerequisites:**
- ✅ Phase 1 complete (gRPC migration done)
- ✅ All proto files defined
- ✅ gRPC server stable in production

**Week 1: Add Annotations**
- Day 1-2: Add `google/api/annotations.proto` imports
- Day 3-4: Add HTTP mappings to all RPC methods
- Day 5: Regenerate proto with gateway plugin

**Week 2: Deploy & Test**
- Day 1-2: Implement gateway in-process OR deploy sidecar
- Day 3: Add authentication (JWT passthrough)
- Day 4: Integration testing (curl, Postman)
- Day 5: Deploy to production, update docs

---

## Example: REST API After Gateway

**Before (gRPC only):**
```bash
# Must use grpcurl
grpcurl -d '{"session_id":"abc","notes":[...]}' \
  -H "authorization: Bearer $TOKEN" \
  api.toolbridge.io:9090 \
  sync.v1.NotesService/PushNotes
```

**After (gRPC + REST):**
```bash
# Can use curl
curl -X POST https://api.toolbridge.io/v1/sync/notes/push \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"abc","notes":[...]}'

# OR still use gRPC (unchanged)
grpcurl -d '{"session_id":"abc","notes":[...]}' \
  -H "authorization: Bearer $TOKEN" \
  api.toolbridge.io:9090 \
  sync.v1.NotesService/PushNotes
```

---

## Early Preparation (Recommended)

You can reduce future work by including gateway annotations **during Phase 1**:

### During Proto Definition (Week 1-2)

1. Install gateway plugin early:
   ```bash
   go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
   ```

2. Add HTTP annotations to proto files from the start
3. Include gateway generation in Makefile (even if not deployed)
4. **Time cost**: +4 hours (negligible)

**Benefits:**
- Gateway code is generated but not deployed yet
- Can enable REST anytime in 1-2 days
- No retroactive annotation work needed
- Matches Memos architecture from day 1

**Result:** Zero-cost insurance policy for future REST access.

---

## REST API Use Cases

Once gateway is deployed, you enable:

### 1. Public API Access
```bash
# Third-party integrations
curl -X POST https://api.toolbridge.io/v1/sync/notes/push \
  -H "Authorization: Bearer $API_KEY" \
  -d '{"notes":[{"content":"From external tool"}]}'
```

### 2. CLI Tools
```bash
# Bash scripts, automation
toolbridge notes create "My note" --api-token=$TOKEN
```

### 3. Webhooks
```javascript
// Zapier, IFTTT, n8n
fetch('https://api.toolbridge.io/v1/sync/notes/push', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: JSON.stringify({ notes: [...] })
});
```

### 4. Web Browsers
```javascript
// Browser-based integrations (no gRPC-web needed)
const response = await fetch('/v1/sync/notes/pull', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const notes = await response.json();
```

---

## Cost Analysis

### Development
- Week 1: Add annotations (if not done in Phase 1) - 40 hours
- Week 2: Deploy gateway and test - 40 hours
- **Total**: 80 hours (~2 weeks)

**If annotations added during Phase 1:**
- Week 1: Deploy gateway only - 40 hours
- **Total**: 40 hours (~1 week)

### Infrastructure
- **In-Process**: Zero additional cost (same pods)
- **Sidecar**: Minimal (1 additional container per pod)

### Maintenance
- Gateway code is auto-generated (no manual updates)
- HTTP mappings defined in proto files (version controlled)

---

## Decision Points

### When to Implement

**Implement Gateway If:**
- ✅ You want public API access for third parties
- ✅ You need webhook/automation support
- ✅ You want CLI tools to use REST
- ✅ Browser-based integrations are needed

**Skip Gateway If:**
- ❌ Only Flutter app needs API access (gRPC sufficient)
- ❌ No external integrations planned
- ❌ Want to minimize complexity

### Recommendation

**Include annotations during Phase 1** (costs <1 day) but **delay gateway deployment** until you have a concrete REST use case.

This gives you flexibility without commitment.

---

## Technical Notes

### Authentication

Gateway passes JWT tokens through:
```go
gwMux := runtime.NewServeMux(
    runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
        if key == "Authorization" {
            return key, true  // Pass through to gRPC metadata
        }
        return runtime.DefaultHeaderMatcher(key)
    }),
)
```

### Error Handling

Gateway automatically converts gRPC errors to HTTP:
- `codes.NotFound` → HTTP 404
- `codes.InvalidArgument` → HTTP 400
- `codes.Unauthenticated` → HTTP 401
- `codes.PermissionDenied` → HTTP 403
- `codes.Internal` → HTTP 500

### Streaming

Gateway supports gRPC server streaming via Server-Sent Events (SSE):
```go
// gRPC streaming RPC
rpc WatchChanges(WatchRequest) returns (stream ChangeEvent);

// Automatically exposed as:
// GET /v1/sync/watch?stream=true
// Content-Type: text/event-stream
```

---

## References

- [gRPC Gateway GitHub](https://github.com/grpc-ecosystem/grpc-gateway)
- [gRPC Gateway Documentation](https://grpc-ecosystem.github.io/grpc-gateway/)
- [Memos gRPC Implementation](https://github.com/usememos/memos/tree/main/proto)
- [Google API HTTP Annotations](https://cloud.google.com/apis/design/custom_methods)

---

## Next Steps

### After Phase 1 Completion

1. **Evaluate need**: Do you have REST API use cases?
2. **Review annotations**: Are HTTP mappings already in proto files?
3. **Choose deployment**: In-process or sidecar?
4. **Implement**: Follow 1-2 week timeline above
5. **Document**: Create public API docs for REST endpoints

### Questions to Answer

- [ ] Do we need public API access for third parties?
- [ ] Will we offer webhook/automation support?
- [ ] Do we want browser-based integrations?
- [ ] Should we expose REST from day 1 or wait for demand?

---

**Summary**: gRPC Gateway lets you expose REST API with minimal effort after your gRPC migration. Consider adding HTTP annotations during Phase 1 (costs <1 day), then deploy gateway later only if needed (1-2 weeks). This matches Memos' architecture and gives you maximum flexibility.
