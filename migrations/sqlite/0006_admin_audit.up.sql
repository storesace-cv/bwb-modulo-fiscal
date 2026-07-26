-- Admin audit events (DEC-BO-002): append-only mutations for /admin/v1.
-- Never stores secrets, tokens, passwords, or private keys.

CREATE TABLE admin_audit_events (
  event_id TEXT NOT NULL PRIMARY KEY,
  occurred_at TEXT NOT NULL,
  actor_subject TEXT NOT NULL,
  actor_roles TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  result TEXT NOT NULL,
  request_id TEXT NULL,
  CONSTRAINT admin_audit_events_id_nonempty CHECK (length(trim(event_id)) > 0),
  CONSTRAINT admin_audit_events_actor_nonempty CHECK (length(trim(actor_subject)) > 0),
  CONSTRAINT admin_audit_events_action_nonempty CHECK (length(trim(action)) > 0),
  CONSTRAINT admin_audit_events_resource_type_nonempty CHECK (length(trim(resource_type)) > 0),
  CONSTRAINT admin_audit_events_resource_id_nonempty CHECK (length(trim(resource_id)) > 0),
  CONSTRAINT admin_audit_events_result_check CHECK (result IN ('success', 'denied', 'error'))
);

CREATE TRIGGER admin_audit_events_no_update
BEFORE UPDATE ON admin_audit_events
BEGIN
  SELECT RAISE(ABORT, 'admin_audit_events is append-only');
END;

CREATE TRIGGER admin_audit_events_no_delete
BEFORE DELETE ON admin_audit_events
BEGIN
  SELECT RAISE(ABORT, 'admin_audit_events is append-only');
END;

CREATE INDEX admin_audit_events_occurred_at_idx ON admin_audit_events (occurred_at);
CREATE INDEX admin_audit_events_actor_subject_idx ON admin_audit_events (actor_subject);
