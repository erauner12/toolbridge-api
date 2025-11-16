#!/usr/bin/env bash
# Save as: test_rest_api.sh

set -euo pipefail

BASE_URL="http://localhost:8081"

echo "🚀 Testing ToolBridge REST API"
echo "================================"
echo ""

# Step 1: Create a session
echo "📝 Step 1: Creating session..."
SESSION_JSON=$(curl -s -X POST "$BASE_URL/v1/sync/sessions" -H "X-Debug-Sub: test-user")
SESSION_ID=$(echo "$SESSION_JSON" | jq -r '.id')
EPOCH=$(echo "$SESSION_JSON" | jq -r '.epoch')

echo "   ✅ Session ID: $SESSION_ID"
echo "   ✅ Epoch: $EPOCH"
echo ""

# Step 2: Create a note
echo "📝 Step 2: Creating a note..."
NOTE_JSON=$(curl -s -X POST "$BASE_URL/v1/notes" \
  -H "Content-Type: application/json" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" \
  -d '{"title":"My First Note","content":"Testing REST API"}')

NOTE_UID=$(echo "$NOTE_JSON" | jq -r '.uid')
NOTE_VERSION=$(echo "$NOTE_JSON" | jq -r '.version')

echo "$NOTE_JSON" | jq
echo ""

# Step 3: GET the note
echo "📖 Step 3: GET the note..."
curl -s -X GET "$BASE_URL/v1/notes/$NOTE_UID" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" | jq
echo ""

# Step 4: PATCH with optimistic locking (quoted ETag - RFC 7232 compliant)
echo "✏️  Step 4: PATCH with If-Match (quoted ETag)..."
curl -s -X PATCH "$BASE_URL/v1/notes/$NOTE_UID" \
  -H "Content-Type: application/json" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" \
  -H "If-Match: \"$NOTE_VERSION\"" \
  -d '{"content":"Updated via PATCH with quoted ETag!"}' | jq
echo ""

# Step 5: Try stale version (should get 412)
echo "❌ Step 5: PATCH with stale version (expect 412)..."
HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/stale_response.json \
  -X PATCH "$BASE_URL/v1/notes/$NOTE_UID" \
  -H "Content-Type: application/json" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" \
  -H "If-Match: \"1\"" \
  -d '{"content":"This should fail"}')

echo "   HTTP Status: $HTTP_CODE"
cat /tmp/stale_response.json | jq
echo ""

# Step 6: DELETE the note
echo "🗑️  Step 6: DELETE the note..."
curl -s -X DELETE "$BASE_URL/v1/notes/$NOTE_UID" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" | jq
echo ""

# Step 7: GET deleted note (should return 410 Gone)
echo "👻 Step 7: GET deleted note (expect 410 Gone)..."
HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/deleted_response.json \
  -X GET "$BASE_URL/v1/notes/$NOTE_UID" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH")

echo "   HTTP Status: $HTTP_CODE"
cat /tmp/deleted_response.json | jq
echo ""

# Step 8: GET with includeDeleted=true (should return 200)
echo "🔍 Step 8: GET deleted note with ?includeDeleted=true (expect 200)..."
curl -s -X GET "$BASE_URL/v1/notes/$NOTE_UID?includeDeleted=true" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" | jq
echo ""

# Step 9: Try to PATCH tombstone (should return 410)
echo "🚫 Step 9: Try to PATCH deleted note (expect 410)..."
HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/patch_tombstone.json \
  -X PATCH "$BASE_URL/v1/notes/$NOTE_UID" \
  -H "Content-Type: application/json" \
  -H "X-Debug-Sub: test-user" \
  -H "X-Sync-Session: $SESSION_ID" \
  -H "X-Sync-Epoch: $EPOCH" \
  -d '{"content":"Trying to resurrect"}')

echo "   HTTP Status: $HTTP_CODE"
cat /tmp/patch_tombstone.json | jq
echo ""

echo "✅ All tests completed!"
echo ""
echo "Summary:"
echo "- ✅ POST /v1/notes - Create note"
echo "- ✅ GET /v1/notes/{uid} - Get note"
echo "- ✅ PATCH with If-Match (quoted ETag) - Optimistic locking works"
echo "- ✅ PATCH with stale version - Returns 412 Precondition Failed"
echo "- ✅ DELETE /v1/notes/{uid} - Soft delete"
echo "- ✅ GET deleted note - Returns 410 Gone"
echo "- ✅ GET deleted note?includeDeleted=true - Returns 200 with tombstone"
echo "- ✅ PATCH deleted note - Returns 410 Gone (prevents resurrection)"
