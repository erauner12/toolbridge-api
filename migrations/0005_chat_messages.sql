-- Chat Messages table for delta sync
-- Messages belong to a single parent: chat
-- Simpler than comments (no polymorphic parent)

CREATE TABLE chat_message (
  uid            UUID NOT NULL,
  owner_id       UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  updated_at_ms  BIGINT NOT NULL,            -- Unix milliseconds for cursor-based pagination
  deleted_at_ms  BIGINT,                     -- NULL = alive, non-NULL = tombstone
  version        INT NOT NULL DEFAULT 1,     -- Server-controlled version for conflict detection
  payload_json   JSONB NOT NULL,             -- Original client JSON (preserved as-is)
  chat_uid       UUID NOT NULL,              -- Parent chat UID (validated at application level)
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (owner_id, uid)                -- Composite key for tenant isolation
);

-- Indexes for efficient delta sync queries
CREATE INDEX chat_message_owner_updated_idx ON chat_message (owner_id, updated_at_ms);
CREATE INDEX chat_message_owner_deleted_idx ON chat_message (owner_id, deleted_at_ms) WHERE deleted_at_ms IS NOT NULL;

-- Composite index for cursor-based pagination (updated_at_ms, uid)
CREATE INDEX chat_message_cursor_idx ON chat_message (updated_at_ms, uid);

-- Index for querying messages by parent chat (e.g., "get all messages for chat X")
CREATE INDEX chat_message_parent_idx ON chat_message (owner_id, chat_uid);

COMMENT ON TABLE chat_message IS 'Chat messages with delta sync support - uses LWW conflict resolution. Belongs to chats.';
COMMENT ON COLUMN chat_message.updated_at_ms IS 'Unix milliseconds timestamp for cursor pagination and LWW conflict resolution';
COMMENT ON COLUMN chat_message.deleted_at_ms IS 'Tombstone timestamp - NULL means active record';
COMMENT ON COLUMN chat_message.version IS 'Server-controlled version number - increments on each update';
COMMENT ON COLUMN chat_message.payload_json IS 'Full client JSON preserved as-is - allows flexible schema evolution';
COMMENT ON COLUMN chat_message.chat_uid IS 'UID of parent chat - chat existence validated at application level';
