-- Ops disposition + action idempotency for admin queue actions (RM-BO-016).
-- No secrets / payload / JWS storage.

ALTER TABLE outbox_messages ADD COLUMN ops_disposition TEXT NULL;

-- SQLite: CHECK added via table rebuild is heavy; enforce in app + nullable column.

CREATE TABLE admin_ops_action_idempotency (
  idempotency_key TEXT NOT NULL PRIMARY KEY,
  action TEXT NOT NULL,
  submission_id TEXT NOT NULL,
  result_queue_status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  CONSTRAINT admin_ops_action_idempotency_key_nonempty CHECK (length(trim(idempotency_key)) > 0),
  CONSTRAINT admin_ops_action_idempotency_action_check CHECK (
    action IN ('retry', 'cancel', 'manual_review')
  ),
  CONSTRAINT admin_ops_action_idempotency_submission_nonempty CHECK (length(trim(submission_id)) > 0)
);
