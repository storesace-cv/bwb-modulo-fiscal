package adminapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

func (h *Handler) getAuthorityProfileHistory(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("profile_id")
	action := "authority_profile.history"
	if h.Audit == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE", "Audit Unavailable")
		return
	}
	if _, err := h.Registry.GetAuthorityProfile(r.Context(), id); err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		h.audit(r, claims, action, "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		limit = n
	}
	events, err := h.Audit.ListByResource(r.Context(), "authority_profile", id, limit)
	if err != nil {
		h.audit(r, claims, action, "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	items := make([]auditEventResp, 0, len(events))
	for _, e := range events {
		items = append(items, auditEventResp{
			EventID: e.ID, OccurredAt: e.OccurredAt.UTC().Format(time.RFC3339Nano),
			ActorSubject: e.ActorSubject, ActorRoles: e.ActorRoles,
			Action: e.Action, ResourceType: e.ResourceType, ResourceID: e.ResourceID,
			Result: e.Result, RequestID: e.RequestID,
		})
	}
	_ = h.Audit.Record(r.Context(), claims, action, "authority_profile", id, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"profile_id":        id,
		"append_only":       true,
		"external_verified": false,
		"items":             items,
		"note":              "Histórico append-only de auditoria (perfil/material sync). Sem plaintext. ≠ AGT.",
	})
}
