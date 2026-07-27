DROP TABLE IF EXISTS fiscal.admin_ops_action_idempotency;
ALTER TABLE fiscal.outbox_messages DROP CONSTRAINT IF EXISTS outbox_messages_ops_disposition_check;
ALTER TABLE fiscal.outbox_messages DROP COLUMN IF EXISTS ops_disposition;
