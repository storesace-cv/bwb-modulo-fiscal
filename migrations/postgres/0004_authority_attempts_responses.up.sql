-- Authority attempts/responses for outbox worker + AGT simulator (≠ HML oficial).
-- Does not authorize production AGT integration or credentials.

CREATE TABLE fiscal.authority_attempts (
  id TEXT NOT NULL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES fiscal.documents (id) ON DELETE RESTRICT,
  submission_id TEXT NOT NULL,
  attempt_no BIGINT NOT NULL,
  sent_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT authority_attempts_attempt_no_positive CHECK (attempt_no > 0),
  CONSTRAINT authority_attempts_submission_attempt_unique UNIQUE (submission_id, attempt_no),
  CONSTRAINT authority_attempts_submission_id_nonempty CHECK (length(trim(submission_id)) > 0)
);

CREATE TRIGGER authority_attempts_no_update
  BEFORE UPDATE ON fiscal.authority_attempts
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TRIGGER authority_attempts_no_delete
  BEFORE DELETE ON fiscal.authority_attempts
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TABLE fiscal.authority_responses (
  id TEXT NOT NULL PRIMARY KEY,
  attempt_id TEXT NOT NULL REFERENCES fiscal.authority_attempts (id) ON DELETE RESTRICT,
  authority_request_id TEXT NULL,
  outcome TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT authority_responses_attempt_unique UNIQUE (attempt_id),
  CONSTRAINT authority_responses_outcome_check CHECK (
    outcome IN (
      'authority_accepted',
      'authority_rejected',
      'authority_outcome_unknown'
    )
  )
);

CREATE TRIGGER authority_responses_no_update
  BEFORE UPDATE ON fiscal.authority_responses
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TRIGGER authority_responses_no_delete
  BEFORE DELETE ON fiscal.authority_responses
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE INDEX authority_attempts_document_id_idx ON fiscal.authority_attempts (document_id);
CREATE INDEX authority_attempts_submission_id_idx ON fiscal.authority_attempts (submission_id);
