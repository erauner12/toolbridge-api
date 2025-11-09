# Client-Side Sync Implementation Guide

This guide explains how clients should implement synchronization with the ToolBridge API, covering the architecture, sync patterns, and common operations.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Sync Protocol](#sync-protocol)
- [Core Concepts](#core-concepts)
- [Client Implementation](#client-implementation)
- [Sync Operations](#sync-operations)
- [Error Handling](#error-handling)
- [Common Patterns](#common-patterns)

## Architecture Overview

### Server-Side Design Principles

The ToolBridge sync API is built on these principles:

1. **Generic Entity Model**: All entities (notes, tasks, chats, comments, messages) follow the same sync pattern
2. **Cursor-Based Pagination**: Deterministic ordering using `(updated_at_ms, uid)` composite cursors
3. **Last-Write-Wins (LWW)**: Conflict resolution based on timestamps
4. **Epoch System**: Single integer per user for wipe/reset coordination
5. **Idempotent Operations**: Safe for retries and duplicate requests
6. **Stateless Sessions**: Short-lived sync sessions with 30-minute TTL

### Why This Design?

**Cursor-Based Pagination** over offset/limit:
- Deterministic: Same cursor always returns same results
- Efficient: No need to skip rows (better DB performance)
- Safe: Handles concurrent insertions/deletions gracefully

**Epoch System** for wipe coordination:
- Simple: Single integer comparison
- Reliable: No race conditions
- Automatic: Clients self-heal on 409 responses

**Generic `payload_json`** storage:
- Schema evolution without migrations
- Client controls data structure
- Server only needs sync metadata

## Sync Protocol

### Three-Phase Bidirectional Sync

Every sync operation follows this pattern:

```
┌─────────────────────────────────────────────────┐
│  Phase 1: DELETE_REMOTE                         │
│  Push tombstones (locally deleted items)        │
└─────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────┐
│  Phase 2: UPLOAD                                │
│  Push dirty items (new/modified)                │
└─────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────┐
│  Phase 3: DOWNLOAD                              │
│  Pull server changes using cursors              │
└─────────────────────────────────────────────────┘
```

**Why this order?**

1. **DELETE_REMOTE first**: Ensures deletions propagate before uploads (prevents resurrection)
2. **UPLOAD second**: Sends local changes before fetching server state
3. **DOWNLOAD last**: Pulls server changes and resolves conflicts

### Session Management

```
1. Client: POST /v1/sync/sessions
   Server: Returns { id, userId, epoch, expiresAt }

2. Client: Stores sessionId and epoch locally

3. Client: Includes headers in all sync requests:
   - X-Sync-Session: <sessionId>
   - X-Sync-Epoch: <epoch>

4. Server: Validates epoch matches current state
   - Match → Process request
   - Mismatch → Return 409 with new epoch

5. Client: On sync completion or error:
   DELETE /v1/sync/sessions/{id}
```

## Core Concepts

### Entity Structure

All entities must include sync metadata:

```json
{
  "uid": "550e8400-e29b-41d4-a716-446655440000",
  "title": "My Note",
  "content": "Note content...",
  "sync": {
    "version": 3,
    "isDeleted": false
  },
  "updatedTs": "2025-11-07T10:00:00.000Z"
}
```

**Required Fields:**
- `uid` (string): UUID v4, immutable, client-generated
- `sync.version` (int): Server-controlled version number (starts at 1)
- `sync.isDeleted` (bool): Tombstone marker
- `updatedTs` (string): RFC3339 timestamp with milliseconds

### Cursors

Cursors are opaque base64-encoded strings representing `(updated_at_ms, uid)`:

```
Format: base64("<unix_ms>|<uuid>")
Example: "MTczMDk3MjQwMDAwMHw1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDA="
```

**Client Rules:**
- Treat cursors as opaque (don't parse or construct them)
- Store cursors per entity type
- `null` cursor = fetch from beginning
- Use `nextCursor` from response for pagination

### Epoch System

The epoch is a monotonically increasing integer per user:

```
Initial state:  epoch = 1
After wipe:     epoch = 2
After reset:    epoch = 3
```

**Client Behavior:**

```
┌─────────────────────────────────────────────────┐
│  1. Store epoch from session creation           │
│  2. Send X-Sync-Epoch header in all requests    │
│  3. On 409 response:                            │
│     - Extract new epoch from response           │
│     - Clear local entity data                   │
│     - Reset cursors to null                     │
│     - Create new session with new epoch         │
│     - Retry sync (becomes download-only)        │
└─────────────────────────────────────────────────┘
```

**Why 409 triggers data clear:**
Server epoch bump means server data was wiped/reset. Client must discard stale local state to avoid re-uploading deleted data.

### Dirty Tracking

Clients must track which items have local changes:

```dart
// Example schema
class LocalNote {
  String uid;
  String title;
  String content;
  bool isDirty;        // Has unsynced local changes
  bool isDeleted;      // Tombstone (deleted locally)
  int version;         // Last known server version
  DateTime updatedAt;  // Last modification time
}
```

**Dirty States:**

| State | `isDirty` | `isDeleted` | Action |
|-------|-----------|-------------|--------|
| Clean | false | false | Skip (already synced) |
| Modified | true | false | Push in UPLOAD phase |
| Deleted | true | true | Push in DELETE_REMOTE phase |
| Tombstone from server | false | true | Keep for sync, hide in UI |

## Client Implementation

### 1. Initialize Sync State

```dart
class SyncState {
  String? sessionId;
  int? clientEpoch;
  Map<EntityType, String?> cursors; // Per-entity-type cursors

  // Load from persistent storage on app start
  SyncState.load() {
    sessionId = prefs.getString('sync_session_id');
    clientEpoch = prefs.getInt('sync_epoch');
    cursors = {
      EntityType.note: prefs.getString('cursor_notes'),
      EntityType.task: prefs.getString('cursor_tasks'),
      // ... other entity types
    };
  }

  void save() {
    prefs.setString('sync_session_id', sessionId ?? '');
    prefs.setInt('sync_epoch', clientEpoch ?? 0);
    prefs.setString('cursor_notes', cursors[EntityType.note] ?? '');
    // ... other cursors
  }
}
```

### 2. Session Creation

```dart
Future<void> ensureSession() async {
  // Check if existing session is still valid
  if (sessionId != null) {
    final session = await api.getSession(sessionId!);
    if (session != null && !session.isExpired) {
      return; // Session still valid
    }
  }

  // Create new session
  final session = await api.beginSession();

  // Check for epoch mismatch
  if (clientEpoch != null && session.epoch > clientEpoch!) {
    // Server was wiped/reset - adopt new epoch
    await handleEpochMismatch(session.epoch);
  }

  sessionId = session.id;
  clientEpoch = session.epoch;
  save();
}
```

### 3. Epoch Mismatch Handling

```dart
Future<void> handleEpochMismatch(int newEpoch) async {
  print('Epoch mismatch: client=$clientEpoch, server=$newEpoch');
  print('Server was wiped/reset - clearing local data');

  // Clear all local entity data
  await clearAllLocalData();

  // Reset all cursors
  cursors = {
    EntityType.note: null,
    EntityType.task: null,
    EntityType.chat: null,
    EntityType.comment: null,
    EntityType.message: null,
  };

  // Adopt new epoch
  clientEpoch = newEpoch;
  save();

  // Next sync will be download-only (no local data to push)
}
```

### 4. Push Implementation

```dart
Future<void> pushNotes() async {
  // Get dirty items from local DB
  final dirtyNotes = await noteRepo.getDirty();
  if (dirtyNotes.isEmpty) return;

  // Build push request
  final items = dirtyNotes.map((note) => {
    'uid': note.uid,
    'title': note.title,
    'content': note.content,
    'sync': {
      'version': note.version,
      'isDeleted': note.isDeleted,
    },
    'updatedTs': note.updatedAt.toIso8601String(),
  }).toList();

  // Push to server
  final response = await api.pushNotes(
    items: items,
    sessionId: sessionId!,
    epoch: clientEpoch!,
  );

  // Process acknowledgments
  for (final ack in response) {
    if (ack.error != null) {
      print('Push failed for ${ack.uid}: ${ack.error}');
      continue;
    }

    // Update local version and mark clean
    await noteRepo.updateAfterPush(
      uid: ack.uid,
      version: ack.version,
      updatedAt: DateTime.parse(ack.updatedAt),
      isDirty: false,
    );
  }
}
```

### 5. Pull Implementation

```dart
Future<void> pullNotes() async {
  String? cursor = cursors[EntityType.note];
  bool hasMore = true;

  while (hasMore) {
    final response = await api.pullNotes(
      cursor: cursor,
      limit: 500,
      sessionId: sessionId!,
      epoch: clientEpoch!,
    );

    // Process upserts (new or modified items)
    for (final item in response.upserts) {
      final localNote = await noteRepo.findByUid(item['uid']);

      if (localNote == null) {
        // New item from server - insert
        await noteRepo.insert(item);
      } else if (localNote.isDirty) {
        // Conflict: local has unsynced changes
        await resolveConflict(localNote, item);
      } else {
        // Update with server version
        await noteRepo.update(item);
      }
    }

    // Process deletes (tombstones)
    for (final delete in response.deletes) {
      await noteRepo.markDeleted(
        uid: delete['uid'],
        deletedAt: DateTime.parse(delete['deletedAt']),
      );
    }

    // Update cursor for next page
    cursor = response.nextCursor;
    hasMore = response.nextCursor != null;

    // Save cursor after each page
    cursors[EntityType.note] = cursor;
    save();
  }
}
```

### 6. Conflict Resolution (LWW)

```dart
Future<void> resolveConflict(LocalNote local, Map<String, dynamic> server) async {
  final localMs = local.updatedAt.millisecondsSinceEpoch;
  final serverMs = DateTime.parse(server['updatedTs']).millisecondsSinceEpoch;

  if (serverMs > localMs) {
    // Server wins - overwrite local
    print('Conflict: Server wins (${server['uid']})');
    await noteRepo.update(server, isDirty: false);
  } else if (localMs > serverMs) {
    // Local wins - keep dirty flag to re-upload
    print('Conflict: Local wins (${server['uid']}) - will re-push');
    // Keep isDirty=true so next UPLOAD phase sends it
  } else {
    // Same timestamp - mark clean (idempotent push)
    print('Conflict: Same timestamp (${server['uid']}) - marking clean');
    await noteRepo.markClean(server['uid']);
  }
}
```

## Sync Operations

### Full Sync (Bidirectional)

```dart
Future<bool> triggerSync() async {
  try {
    // Ensure valid session
    await ensureSession();

    // Phase 1: DELETE_REMOTE
    await pushDeletedNotes();
    await pushDeletedTasks();
    // ... other entity types

    // Phase 2: UPLOAD
    await pushNotes();
    await pushTasks();
    // ... other entity types

    // Phase 3: DOWNLOAD
    await pullNotes();
    await pullTasks();
    // ... other entity types

    return true;
  } catch (e) {
    if (e is EpochMismatchException) {
      // Handle 409 response
      await handleEpochMismatch(e.serverEpoch);
      // Retry sync (will be download-only)
      return triggerSync();
    }
    return false;
  }
}
```

### Download-Only Sync

This happens automatically when local DB is empty (e.g., after epoch mismatch):

```dart
// When local is empty:
// - Phase 1 (DELETE_REMOTE): No tombstones → skipped
// - Phase 2 (UPLOAD): No dirty items → skipped
// - Phase 3 (DOWNLOAD): Pure server import
```

### Selective Sync

Pull specific entity types:

```dart
Future<void> syncNotesOnly() async {
  await ensureSession();
  await pushDeletedNotes(); // Still push deletes
  await pushNotes();        // Still push changes
  await pullNotes();        // Pull updates
  // Other entities not synced
}
```

## Error Handling

### 409 Epoch Mismatch

```dart
try {
  await api.pushNotes(...);
} catch (e) {
  if (e is ApiException && e.statusCode == 409) {
    final newEpoch = e.body['epoch'];
    await handleEpochMismatch(newEpoch);

    // Retry will succeed (local now empty)
    await triggerSync();
  }
}
```

**When 409 occurs:**
- Server bumped epoch (wipe/reset happened)
- Client's local data is stale
- Client must clear local state and re-download

### Network Errors

```dart
try {
  await pullNotes();
} catch (e) {
  if (e is NetworkException) {
    // Cursor is already saved after last successful page
    // Next sync will resume from saved cursor
    print('Network error - will retry from cursor: ${cursors[EntityType.note]}');
  }
}
```

**Cursor safety:**
Always save cursor after processing each page, not at the end. This makes sync resumable on network failures.

### Session Expiration

```dart
try {
  await api.pullNotes(sessionId: sessionId!);
} catch (e) {
  if (e is ApiException && e.statusCode == 404) {
    // Session expired (30min TTL)
    sessionId = null;
    await ensureSession(); // Create new session
    await triggerSync();   // Retry
  }
}
```

## Common Patterns

### Pattern 1: Reset to Server State

**Goal**: Replace all local data with server copy (discard local changes)

```dart
Future<bool> resetToServerState() async {
  // Step 1: Clear local data
  await clearAllLocalData();

  // Step 2: Reset cursors
  cursors = {
    EntityType.note: null,
    EntityType.task: null,
    EntityType.chat: null,
    EntityType.comment: null,
    EntityType.message: null,
  };
  save();

  // Step 3: Trigger sync (will be download-only)
  // - DELETE_REMOTE: No tombstones (skipped)
  // - UPLOAD: No dirty items (skipped)
  // - DOWNLOAD: Pure server import
  return await triggerSync();
}
```

**Use Cases:**
- Recover from local corruption
- Resolve persistent conflicts
- Start fresh from server state
- Debug sync issues

### Pattern 2: Full Wipe (Client-Side)

**Goal**: Delete all data locally and on server

```dart
Future<void> fullWipe() async {
  await ensureSession();

  // Call server wipe endpoint
  final response = await api.wipeAccount(confirm: 'WIPE');

  // Server returns new epoch
  final newEpoch = response.epoch;

  // Clear local state
  await clearAllLocalData();
  cursors.clear();
  clientEpoch = newEpoch;
  save();

  // Server data is empty, local is empty
  // Next sync finds nothing
}
```

### Pattern 3: Full Reset (Client-Side)

**Goal**: Clear local data, reset epoch, re-sync from server

```dart
Future<void> fullReset() async {
  // Clear local data
  await clearAllLocalData();

  // Reset cursors
  cursors = {
    EntityType.note: null,
    EntityType.task: null,
    // ... other types
  };

  // Create new session (will detect if server epoch changed)
  sessionId = null;
  await ensureSession();

  // Trigger sync (download-only since local is empty)
  await triggerSync();
}
```

**Difference from Reset to Server State:**
- Full Reset may bump epoch on server (if implemented)
- Reset to Server State keeps current epoch

### Pattern 4: Check for Unsynced Changes

**Goal**: Warn user before destructive operations

```dart
Future<Map<String, int>> getUnsyncedChanges() async {
  final counts = <String, int>{};

  counts['note'] = await noteRepo.countDirty();
  counts['task'] = await taskRepo.countDirty();
  counts['chat'] = await chatRepo.countDirty();
  counts['comment'] = await commentRepo.countDirty();
  counts['message'] = await messageRepo.countDirty();

  // Filter out zeros
  return Map.fromEntries(
    counts.entries.where((e) => e.value > 0),
  );
}

// Usage:
final unsynced = await getUnsyncedChanges();
if (unsynced.isNotEmpty) {
  final total = unsynced.values.fold(0, (sum, count) => sum + count);
  showWarning('You have $total unsynced items that will be lost');
}
```

### Pattern 5: Background Sync

```dart
class SyncScheduler {
  Timer? _timer;

  void startPeriodicSync({Duration interval = const Duration(minutes: 15)}) {
    _timer?.cancel();
    _timer = Timer.periodic(interval, (_) async {
      if (!isOnline) return;
      if (isSyncing) return;

      await triggerSync();
    });
  }

  void stopPeriodicSync() {
    _timer?.cancel();
    _timer = null;
  }
}
```

### Pattern 6: Incremental Cursor Updates

**Goal**: Sync new data without re-downloading everything

```dart
// First sync: cursor = null (download all)
await pullNotes(); // Downloads 1000 items, cursor = "ABC..."

// Later sync: cursor = "ABC..." (download only new items)
await pullNotes(); // Downloads 5 new items, cursor = "XYZ..."

// Cursors are automatically updated and saved
```

**When to reset cursors:**
- Epoch mismatch (409 response)
- Full reset operation
- Local database corruption detected

## Summary

**Key Takeaways:**

1. **Always use cursors** - Don't try to track timestamps manually
2. **Save cursors incrementally** - After each page, not at the end
3. **Handle 409 gracefully** - Clear local state and re-download
4. **Trust server version** - Server controls version numbers
5. **DELETE_REMOTE first** - Order matters for correct sync
6. **Idempotent operations** - Safe to retry pushes with same data
7. **Session management** - 30min TTL, create new if expired
8. **Three-phase sync** - DELETE → UPLOAD → DOWNLOAD

**Common Mistakes to Avoid:**

- ❌ Parsing cursor strings (treat as opaque)
- ❌ Skipping DELETE_REMOTE phase (causes resurrection)
- ❌ Not saving cursors incrementally (loses progress)
- ❌ Ignoring 409 responses (causes infinite loops)
- ❌ Clearing local data without resetting cursors
- ❌ Assuming local version is authoritative (server controls versions)

**Architecture Benefits:**

- ✅ Resumable sync (cursor-based pagination)
- ✅ Automatic conflict resolution (LWW)
- ✅ Self-healing clients (409 → clear → re-download)
- ✅ Idempotent operations (safe retries)
- ✅ Schema evolution (payload_json)
- ✅ Simple mental model (three phases)

---

For server-side implementation details, see the main [README.md](../README.md).

For deployment instructions, see [DEPLOY.md](../DEPLOY.md).
