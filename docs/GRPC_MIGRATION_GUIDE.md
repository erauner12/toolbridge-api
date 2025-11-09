# gRPC Migration Implementation Guide

**Status**: Phase 2 Complete ✅
**Last Updated**: 2025-11-09

This document tracks the gRPC migration implementation, establishing gRPC as a parallel transport alongside the existing REST API.

---

## ✅ Phase 1: Foundation (Complete)

### 1. Protocol Buffers Definition
- **File**: `proto/sync/v1/sync.proto`
- **What**: Defined all sync service interfaces using unary (batch-oriented) RPCs
- **Services**:
  - `SyncService` - Core service (sessions, info, wipe, state)
  - `NoteSyncService`, `TaskSyncService`, `CommentSyncService`, `ChatSyncService`, `ChatMessageSyncService`
- **Messages**: Request/Response types using `google.protobuf.Struct` for flexibility
- **Generated Code**:
  - `gen/go/sync/v1/sync.pb.go` - Message types
  - `gen/go/sync/v1/sync_grpc.pb.go` - Service interfaces

### 2. Shared Session Store
- **Files**:
  - `internal/session/store.go` - Shared in-memory session store
  - `internal/httpapi/sessions.go` - Updated to use shared store
  - `internal/httpapi/session_required.go` - Updated to use shared store
- **What**: Refactored session management out of HTTP package so both HTTP and gRPC can share it
- **Interface**: `CreateSession`, `GetSession`, `DeleteSession`, `DeleteUserSessions`
- **Benefit**: Both transports share session state

### 3. Build Infrastructure
- **Files**:
  - `scripts/generate_proto.sh` - Script to generate Go protobuf stubs
  - `proto/README.md` - Documentation for installing protoc and generating code
- **Commands**:
  ```bash
  brew install protobuf
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ./scripts/generate_proto.sh
  ```

---

## ✅ Phase 2: Complete Implementation (Complete)

**Delivered in PR #20**: Complete gRPC server with service layer refactoring

### 4. Service Layer (Business Logic Extraction)
- **Package**: `internal/service/syncservice/`
- **Implemented**:
  - `notes_service.go` - Notes push/pull logic
  - `tasks_service.go` - Tasks push/pull logic
  - `comments_service.go` - Comments push/pull logic (with parent validation)
  - `chats_service.go` - Chats push/pull logic
  - `chat_messages_service.go` - Chat messages push/pull logic (with parent validation)
- **Pattern**:
  - `PushXItem(ctx, tx, userID, item)` - LWW upsert for single item
  - `PullXs(ctx, userID, cursor, limit)` - Cursor-based pagination
- **Benefit**: Transport-agnostic business logic, returns simple structs (`PushAck`, `PullResponse`)

### 5. HTTP Handlers Refactored
**All HTTP handlers now use service layer:**
- `internal/httpapi/sync_notes.go` - Calls `NoteSvc.PushNoteItem()`, `NoteSvc.PullNotes()`
- `internal/httpapi/sync_tasks.go` - Calls `TaskSvc.PushTaskItem()`, `TaskSvc.PullTasks()`
- `internal/httpapi/sync_comments.go` - Calls `CommentSvc.PushCommentItem()`, `CommentSvc.PullComments()`
- `internal/httpapi/sync_chats.go` - Calls `ChatSvc.PushChatItem()`, `ChatSvc.PullChats()`
- `internal/httpapi/sync_chat_messages.go` - Calls `ChatMessageSvc.PushChatMessageItem()`, `ChatMessageSvc.PullChatMessages()`

**Result**: Zero business logic duplication between HTTP and gRPC

### 6. gRPC Interceptors
**File**: `internal/grpcapi/interceptors.go`

**Implemented interceptors** (mirror HTTP middleware):
- `CorrelationIDInterceptor()` - Generates/reads correlation ID from metadata
- `AuthInterceptor()` - Validates JWT tokens (supports both RS256 and HS256)
  - DevMode support via `x-debug-sub` header
  - Creates/finds `app_user` record and sets `userID` in context
- `SessionInterceptor()` - Validates `X-Sync-Session` header
  - Checks session exists and belongs to authenticated user
  - Exempt methods: `GetServerInfo`, `BeginSession`
- `EpochInterceptor()` - Validates `X-Sync-Epoch` header
  - Detects epoch mismatches and triggers client reset
  - Exempt methods: `GetServerInfo`, `BeginSession`, `EndSession`, `GetSyncState`, `WipeAccount`
- `RecoveryInterceptor()` - Panic recovery
- `LoggingInterceptor()` - Request logging
- `ChainUnaryServer()` - Helper to chain multiple interceptors

**Key Fix**: DevMode auth goes through same `app_user` lookup as JWT to ensure proper UUID conversion

### 7. gRPC Server Implementation
**File**: `internal/grpcapi/server.go`

**Core SyncService RPCs** (all implemented):
- `GetServerInfo` - Returns server capabilities, rate limits, entity configs
- `BeginSession` - Creates session with epoch coordination
- `EndSession` - Terminates session
- `WipeAccount` - Deletes all user data and increments epoch
- `GetSyncState` - Returns current epoch and sync metadata

**Entity Service RPCs** (all implemented):
- `NoteSyncService.Push/Pull` - Push/pull notes
- `TaskSyncService.Push/Pull` - Push/pull tasks
- `CommentSyncService.Push/Pull` - Push/pull comments
- `ChatSyncService.Push/Pull` - Push/pull chats
- `ChatMessageSyncService.Push/Pull` - Push/pull chat messages

**Pattern**:
```go
1. Extract userID from context (set by auth interceptor)
2. Begin transaction (for push operations)
3. Loop through items, call service layer
4. Convert service response to proto
5. Commit transaction
6. Return proto response
```

### 8. Server Setup and Wiring
**File**: `cmd/server/main.go`

**Build tags**: gRPC server conditionally compiled with `-tags grpc`
- `cmd/server/grpc_setup.go` - gRPC server setup (when tag present)
- `cmd/server/grpc_noop.go` - No-op implementation (when tag absent)

**Service initialization**:
```go
srv := &httpapi.Server{
    DB:              pool,
    RateLimitConfig: httpapi.DefaultRateLimitConfig,
    NoteSvc:         syncservice.NewNoteService(pool),
    TaskSvc:         syncservice.NewTaskService(pool),
    CommentSvc:      syncservice.NewCommentService(pool),
    ChatSvc:         syncservice.NewChatService(pool),
    ChatMessageSvc:  syncservice.NewChatMessageService(pool),
}
```

**gRPC server** (runs on `:8082` when `-tags grpc` enabled):
```go
grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpcapi.RecoveryInterceptor(),
        grpcapi.CorrelationIDInterceptor(),
        grpcapi.AuthInterceptor(pool, jwtCfg),
        grpcapi.SessionInterceptor(),
        grpcapi.EpochInterceptor(pool),
        grpcapi.LoggingInterceptor(),
    ),
)
```

**Makefile targets**:
- `make dev` - HTTP only (port 8081)
- `make dev-grpc` - HTTP + gRPC (ports 8081 + 8082)

### 9. Test Infrastructure Updates
**Fixed all HTTP integration tests** to use service layer:
- `internal/httpapi/sync_tasks_test.go` - Added `TaskSvc` initialization
- `internal/httpapi/sync_comments_test.go` - Added `NoteSvc`, `TaskSvc`, `CommentSvc`
- `internal/httpapi/sync_chats_test.go` - Added `ChatSvc`
- `internal/httpapi/sync_chat_messages_test.go` - Added `ChatSvc`, `ChatMessageSvc`

**Result**: All 31 integration tests passing

---

## ✅ Flutter Client (Complete)

**Delivered in PR #149**: Complete Flutter gRPC client implementation

### Flutter gRPC Client
**File**: `lib/services/sync/grpc_sync_api.dart`

**Implemented**:
- Full `SyncApi` interface implementation using gRPC
- All core operations: `getServerInfo`, `beginSession`, `endSession`, `wipeAccount`, `getSyncState`
- All entity operations: Push/Pull for notes, tasks, comments, chats, chat_messages
- Proper metadata handling (`x-correlation-id`, `x-sync-session`, `x-sync-epoch`)
- DevMode authentication support (`x-debug-sub` header)
- Connection management and error handling

### Settings Integration
**File**: `lib/page/setting/remote_sync_setting.dart`

**Added**:
- Transport selection toggle (HTTP/gRPC)
- Persisted user preference in settings
- UI for switching between transports

### Manual Testing
**File**: `test/manual/test_grpc_connectivity.dart`

**Test coverage**:
- Session management (begin/end)
- Push operations (all entities)
- Pull operations (all entities)
- Epoch validation
- Error handling

**Verification**: All tests passing with proper UUID handling

---

## 🚧 Phase 3: Integration Testing (Next)

**Goal**: Automated gRPC integration tests for CI/CD

### Scope
- **File**: `internal/grpcapi/server_test.go`
- **Test coverage**:
  - ✅ Auth interceptor (JWT + DevMode)
  - ✅ Session interceptor (validation + exemptions)
  - ✅ Epoch interceptor (mismatch detection + exemptions)
  - ✅ Core RPCs (GetServerInfo, BeginSession, EndSession, WipeAccount, GetSyncState)
  - ✅ Entity RPCs (Push/Pull for all 5 entities)
  - ✅ Error scenarios (auth failures, session errors, epoch mismatches)
  - ✅ Edge cases (concurrent requests, transaction rollbacks)

### Test Pattern
Follow existing HTTP test patterns:
```go
func TestGrpcNotePush(t *testing.T) {
    pool := setupTestDB(t)
    defer pool.Close()

    // Create gRPC server with test DB
    grpcServer := setupTestGrpcServer(pool)

    // Create gRPC client
    conn, client := setupTestGrpcClient(grpcServer)
    defer conn.Close()

    // Test push operation
    resp, err := client.Push(ctx, &syncv1.PushRequest{...})
    assert.NoError(t, err)
    assert.Len(t, resp.Acks, 1)
}
```

### CI Integration
- Add gRPC tests to `make test-all`
- Run gRPC tests in GitHub Actions
- Ensure tests run with `-tags grpc` build flag

---

## 🎯 Current Status

### What's Working
✅ HTTP API (port 8081)
✅ gRPC API (port 8082)
✅ Shared session store
✅ Shared service layer
✅ DevMode auth for both transports
✅ All 31 HTTP integration tests passing
✅ Flutter client with gRPC support
✅ Manual testing validated

### What's Next
🚧 Automated gRPC integration tests
⏸️ Production deployment (after tests)
⏸️ Performance benchmarking
⏸️ gRPC Gateway (optional, for REST exposure)

---

## 🔑 Key Design Decisions

### 1. Service Layer Pattern
**Decision**: Extract business logic to `syncservice` package
**Benefit**:
- HTTP and gRPC handlers stay thin (transport concerns only)
- Zero business logic duplication
- Easier testing (test service layer directly)
- Enables future transports (WebSocket, etc.)

### 2. Shared Session Store
**Decision**: In-memory store accessible to both transports
**Benefit**:
- Session continuity across transports
- Simpler than external store (Redis) for single-instance deployment
**Future**: Could migrate to Redis for multi-instance scalability

### 3. Proto `Struct` Usage
**Decision**: Use `google.protobuf.Struct` for entity payloads
**Benefit**:
- Matches existing `List<Map<String, dynamic>>` pattern
- Flexibility for schema evolution
- Faster initial implementation
**Future**: Could migrate to typed messages for better performance/validation

### 4. Interceptor Chain
**Decision**: gRPC interceptors mirror HTTP middleware stack
**Benefit**:
- Consistent auth/session/epoch behavior across transports
- Reuses validation logic from `auth` and `session` packages
- Same security guarantees for both APIs

### 5. DevMode Authentication
**Decision**: Both HTTP and gRPC support `x-debug-sub` header in dev mode
**Implementation**: Both paths go through `app_user` table lookup for UUID conversion
**Benefit**: Consistent testing experience across transports

### 6. Build Tags
**Decision**: gRPC server conditionally compiled with `-tags grpc`
**Benefit**:
- Can run HTTP-only or HTTP+gRPC
- Reduces binary size when gRPC not needed
- Easier local development

---

## 📚 Additional Resources

- **Proto files**: `proto/sync/v1/sync.proto`
- **Generated stubs**: `gen/go/sync/v1/`
- **Service layer**: `internal/service/syncservice/`
- **gRPC server**: `internal/grpcapi/server.go`
- **gRPC interceptors**: `internal/grpcapi/interceptors.go`
- **Flutter client**: `lib/services/sync/grpc_sync_api.dart`
- **Flutter tests**: `test/manual/test_grpc_connectivity.dart`

---

## 🧪 Testing Commands

### Backend Tests
```bash
# Run all tests (HTTP integration tests)
make test-all

# Run unit tests only
make test-unit

# Start dev server (HTTP only)
make dev

# Start dev server (HTTP + gRPC)
make dev-grpc
```

### Manual gRPC Testing (grpcurl)
```bash
# List services
grpcurl -plaintext localhost:8082 list

# Get server info
grpcurl -plaintext -d '{}' localhost:8082 toolbridge.sync.v1.SyncService/GetServerInfo

# Begin session (with dev mode auth)
grpcurl -plaintext \
  -H "x-debug-sub: test-user" \
  -d '{}' \
  localhost:8082 toolbridge.sync.v1.SyncService/BeginSession
```

### Flutter Manual Testing
```bash
cd /Users/erauner/git/side/ToolBridge
dart test/manual/test_grpc_connectivity.dart
```

---

**Next Step**: Implement Phase 3 integration tests to enable confident production deployment
