# REST API Testing

This document describes how to test the ToolBridge REST API endpoints.

## Manual Testing Script

The `test_rest_api.sh` script provides a comprehensive test of all REST API endpoints.

### Prerequisites

- Dev server running (`make dev`)
- `jq` installed for JSON parsing
- `curl` for making HTTP requests

### Running the Test Script

```bash
# Start the dev server (in a separate terminal)
make dev

# Run the test script
./test_rest_api.sh
```

### What the Script Tests

The script demonstrates and tests the following features:

1. **Session Creation** - Creating a sync session with `POST /v1/sync/sessions`
2. **Creating Notes** - Creating notes via `POST /v1/notes`
3. **Retrieving Notes** - Getting notes via `GET /v1/notes/{uid}`
4. **Optimistic Locking** - Testing `If-Match` header with quoted ETags (RFC 7232 compliant)
5. **Conflict Detection** - Testing 412 Precondition Failed for stale versions
6. **Soft Deletes** - Deleting notes via `DELETE /v1/notes/{uid}`
7. **Tombstone Handling** - 410 Gone responses for deleted items
8. **includeDeleted Parameter** - Retrieving tombstones with `?includeDeleted=true`
9. **Mutation Prevention** - 410 responses when trying to mutate deleted items

### Expected Output

The script will output detailed test results for each step, including:
- Session ID and epoch
- Note creation with version tracking
- Successful updates with correct ETags
- Rejection of stale updates (412)
- Soft delete confirmation
- Proper 410 Gone responses
- Tombstone retrieval with `includeDeleted=true`
- Prevention of mutations on deleted items

## Integration Tests

The project includes comprehensive integration tests that cover the same functionality:

```bash
# Run all tests (requires database)
make test-all

# Run only integration tests
make test-integration
```

### Key Integration Test Files

- `internal/httpapi/rest_items_test.go` - Tests for REST item endpoints (notes, tasks)
- `internal/httpapi/sync_notes_test.go` - Sync protocol tests for notes
- `internal/httpapi/sync_tasks_test.go` - Sync protocol tests for tasks
- `internal/httpapi/session_required_test.go` - Session requirement tests
- `internal/httpapi/ratelimit_test.go` - Rate limiting tests

### Test Features

All integration tests properly:
- Create test users with valid UUIDs
- Create sync sessions before making requests
- Include required headers (`X-Debug-Sub`, `X-Sync-Session`, `X-Sync-Epoch`)
- Test optimistic locking with quoted ETags
- Verify soft delete behavior
- Test tombstone handling
- Validate error responses

## Development Notes

The REST API requires:
- Valid sync session created via `POST /v1/sync/sessions`
- Session headers on all sync-related endpoints:
  - `X-Sync-Session`: Session ID
  - `X-Sync-Epoch`: Current epoch number
- In dev mode, `X-Debug-Sub` header can be used instead of JWT tokens

For production use, replace `X-Debug-Sub` with proper JWT authentication.
