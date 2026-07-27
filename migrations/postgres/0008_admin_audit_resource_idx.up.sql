-- Index for AuthorityProfile history filter (RM-AGTPREP-011).

CREATE INDEX admin_audit_events_resource_occurred_idx
  ON fiscal.admin_audit_events (resource_type, resource_id, occurred_at DESC);
