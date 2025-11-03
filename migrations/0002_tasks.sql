-- Tasks table for delta sync
-- Follows same pattern as Notes with composite PK for tenant isolation

CREATE TABLE task (
  uid            UUID NOT NULL,
  owner_id       UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
  updated_at_ms  BIGINT NOT NULL,            -- Unix milliseconds for cursor-based pagination
  deleted_at_ms  BIGINT,                     -- NULL = alive, non-NULL = tombstone
  version        INT NOT NULL DEFAULT 1,     -- Server-controlled version for conflict detection
  payload_json   JSONB NOT NULL,             -- Original client JSON (preserved as-is)
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (owner_id, uid)                -- Composite key for tenant isolation
);

-- Indexes for efficient delta sync queries
CREATE INDEX task_owner_updated_idx ON task (owner_id, updated_at_ms);
CREATE INDEX task_owner_deleted_idx ON task (owner_id, deleted_at_ms) WHERE deleted_at_ms IS NOT NULL;

-- Composite index for cursor-based pagination (updated_at_ms, uid)
CREATE INDEX task_cursor_idx ON task (updated_at_ms, uid);

COMMENT ON TABLE task IS 'Tasks with delta sync support - uses LWW conflict resolution';
COMMENT ON COLUMN task.updated_at_ms IS 'Unix milliseconds timestamp for cursor pagination and LWW conflict resolution';
COMMENT ON COLUMN task.deleted_at_ms IS 'Tombstone timestamp - NULL means active record';
COMMENT ON COLUMN task.version IS 'Server-controlled version number - increments on each update';
COMMENT ON COLUMN task.payload_json IS 'Full client JSON preserved as-is - allows flexible schema evolution';
