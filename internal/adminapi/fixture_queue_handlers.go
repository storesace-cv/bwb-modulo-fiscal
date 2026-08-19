package adminapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fefixqueue"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fixtruntime"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

type fixtureSubmissionReq struct {
	Operation      string             `json:"operation"`
	IdentityRef    string             `json:"identity_ref"`
	IdempotencyKey string             `json:"idempotency_key"`
	Payload        fefixqueue.Payload `json:"payload"`
}

func (h *Handler) getAuthorityFixtureRuntime(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	st := fixtruntime.Status{Configured: false, MockOnly: true, Note: "workbook path not configured"}
	if h.FixtureRuntime != nil {
		st = h.FixtureRuntime.Status()
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_runtime", "authority", "fixture", adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":        st.Configured,
		"mock_loopback":     st.MockLoopback,
		"identity_count":    st.IdentityCount,
		"worker_interval_s": st.WorkerInterval.Seconds(),
		"external_verified": st.ExternalVerified,
		"mock_only":         st.MockOnly,
		"note":              st.Note,
	})
}

func (h *Handler) listAuthorityFixtureSubmissions(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	rt, code, prob := h.requireFixtureRuntime()
	if prob != nil {
		h.auditFixtureErr(r, claims, "authority.fixture_submissions.list", code)
		writeProblem(w, r, code, "ADMIN_VALIDATION", prob.title)
		return
	}
	rows, err := rt.Queue.ListRecent(r.Context(), 50)
	if err != nil {
		h.auditFixtureErr(r, claims, "authority.fixture_submissions.list", http.StatusInternalServerError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_INTERNAL", "Internal Server Error")
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, fixtureRowView(row))
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_submissions.list", "authority", "fixture", adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(items), "submissions": items, "mock_only": true, "external_verified": false,
	})
}

func (h *Handler) getAuthorityFixtureSubmission(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	rt, code, prob := h.requireFixtureRuntime()
	if prob != nil {
		h.auditFixtureErr(r, claims, "authority.fixture_submissions.get", code)
		writeProblem(w, r, code, "ADMIN_VALIDATION", prob.title)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeProblem(w, r, http.StatusBadRequest, "ADMIN_VALIDATION", "Bad Request")
		return
	}
	row, err := rt.Queue.Get(r.Context(), id)
	if errors.Is(err, fefixqueue.ErrNotFound) {
		writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
		return
	}
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_INTERNAL", "Internal Server Error")
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_submissions.get", "authority", id, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, fixtureRowView(row))
}

func (h *Handler) postAuthorityFixtureSubmission(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	rt, code, prob := h.requireFixtureRuntime()
	if prob != nil {
		h.auditFixtureErr(r, claims, "authority.fixture_submissions.create", code)
		writeProblem(w, r, code, "ADMIN_VALIDATION", prob.title)
		return
	}
	var req fixtureSubmissionReq
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "ADMIN_VALIDATION", "Bad Request")
		return
	}
	if err := validateFixtureSubmission(req); err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	row, err := rt.Queue.Enqueue(r.Context(), fefixqueue.EnqueueInput{
		Operation: req.Operation, IdentityRef: req.IdentityRef,
		IdempotencyKey: req.IdempotencyKey, Payload: req.Payload,
	})
	if err != nil {
		if errors.Is(err, fefixqueue.ErrInvalidInput) {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_INTERNAL", "Internal Server Error")
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_submissions.create", "authority", row.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusAccepted, fixtureRowView(row))
}

func (h *Handler) postAuthorityFixtureProcessNext(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	rt, code, prob := h.requireFixtureRuntime()
	if prob != nil {
		h.auditFixtureErr(r, claims, "authority.fixture_submissions.process", code)
		writeProblem(w, r, code, "ADMIN_VALIDATION", prob.title)
		return
	}
	out, err := rt.ProcessOne(r.Context())
	if errors.Is(err, fefixqueue.ErrEmpty) {
		writeJSON(w, http.StatusOK, map[string]any{"processed": false, "note": "queue empty"})
		return
	}
	if err != nil {
		h.auditFixtureErr(r, claims, "authority.fixture_submissions.process", http.StatusInternalServerError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_INTERNAL", "Internal Server Error")
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_submissions.process", "authority", out.RowID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"processed": true, "row_id": out.RowID, "operation": out.Operation,
		"state": out.State, "attempts": out.Attempts, "retried": out.Retried,
		"mock_request_id": out.MockRequest, "mock_only": true, "external_verified": false,
	})
}

type fixtureProb struct{ title string }

func (h *Handler) requireFixtureRuntime() (*fixtruntime.Runtime, int, *fixtureProb) {
	if h.FixtureRuntime == nil {
		return nil, http.StatusServiceUnavailable, &fixtureProb{title: "Service Unavailable"}
	}
	return h.FixtureRuntime, 0, nil
}

func (h *Handler) auditFixtureErr(r *http.Request, claims adminauth.Claims, action string, status int) {
	res := adminaudit.ResultError
	if status == http.StatusNotFound {
		res = adminaudit.ResultSuccess
	}
	_ = h.Audit.Record(r.Context(), claims, action, "authority", "fixture", res, requestID(r))
}

func validateFixtureSubmission(req fixtureSubmissionReq) error {
	op := strings.TrimSpace(req.Operation)
	if op != feboundary.OpSoftwareInfo && op != feboundary.OpObterEstado && op != feboundary.OpConsultarFactura {
		return fefixqueue.ErrInvalidInput
	}
	if strings.TrimSpace(req.IdentityRef) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return fefixqueue.ErrInvalidInput
	}
	switch op {
	case feboundary.OpSoftwareInfo:
		if req.Payload.Software == nil {
			return fefixqueue.ErrInvalidInput
		}
		if _, err := feprofile.MarshalSoftwareInfoPayload(*req.Payload.Software); err != nil {
			return err
		}
	case feboundary.OpObterEstado:
		if req.Payload.ObterEstado == nil {
			return fefixqueue.ErrInvalidInput
		}
		if _, err := feprofile.MarshalObterEstadoRequestPayload(*req.Payload.ObterEstado); err != nil {
			return err
		}
	case feboundary.OpConsultarFactura:
		if req.Payload.Consultar == nil {
			return fefixqueue.ErrInvalidInput
		}
		if _, err := feprofile.MarshalConsultarFacturaRequestPayload(*req.Payload.Consultar); err != nil {
			return err
		}
	}
	return req.Payload.Validate(op)
}

func fixtureRowView(row fefixqueue.Row) map[string]any {
	return map[string]any{
		"id":              row.ID,
		"operation":       row.Operation,
		"state":           row.State,
		"identity_ref":    row.IdentityRef,
		"idempotency_key": row.IdempotencyKey,
		"attempts":        row.Attempts,
		"mock_request_id": row.MockRequestID,
		"mock_code":       row.MockCode,
		"source_id":       row.SourceID,
		"note":            row.Note,
		"agt_accepted":    false,
		"mock_only":       true,
	}
}
