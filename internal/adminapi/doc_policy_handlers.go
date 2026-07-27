package adminapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
)

type upsertDocGroupReq struct {
	Environment string `json:"environment"`
	Grupo       string `json:"grupo"`
	Active      bool   `json:"active"`
}

type upsertDocTypeReq struct {
	Environment    string `json:"environment"`
	CodigoCanonico string `json:"codigo_canonico"`
	Active         bool   `json:"active"`
}

func (h *Handler) getDocumentAvailability(w http.ResponseWriter, r *http.Request) {
	estID := r.PathValue("establishment_id")
	env := strings.TrimSpace(r.URL.Query().Get("environment"))
	if env == "" {
		env = adminregistry.EnvHomologation
	}
	est, err := h.Registry.GetEstablishment(r.Context(), estID)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		if errors.Is(err, adminregistry.ErrValidation) {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	cfg, err := h.Registry.LoadDocPolicyConfig(r.Context(), estID, env)
	if err != nil {
		if errors.Is(err, adminregistry.ErrValidation) {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	feStatus := adminregistry.FEEnrollmentNotEnrolled
	if enrollments, err := h.Registry.ListFEEnrollments(r.Context(), est.TaxpayerID); err == nil {
		feStatus = adminregistry.EffectiveFEStatus(enrollments, env)
	}
	reg, err := doctype.Default()
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	rep := reg.ComputeAvailability(doctype.AvailabilityInput{
		FEEnrollmentStatus: feStatus,
		Config:             cfg.AvailabilityConfig(),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"establishment_id":     estID,
		"taxpayer_id":          est.TaxpayerID,
		"environment":          env,
		"fe_enrollment_status": rep.FEEnrollmentStatus,
		"fe_aderiu":            rep.FEAderiu,
		"groups":               rep.Groups,
		"types":                rep.Types,
		"computed_at":          time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) putDocumentGroupConfig(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	estID := r.PathValue("establishment_id")
	var req upsertDocGroupReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "document_group.upsert", "establishment", estID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	err := h.Registry.UpsertDocGroupConfig(r.Context(), adminregistry.UpsertDocGroupInput{
		EstablishmentID: estID, Environment: req.Environment, Grupo: req.Grupo, Active: req.Active,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "document_group.upsert", "establishment", estID, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "document_group.upsert", "establishment", estID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"establishment_id": estID,
		"environment":      req.Environment,
		"grupo":            req.Grupo,
		"active":           req.Active,
	})
}

func (h *Handler) putDocumentTypeConfig(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	estID := r.PathValue("establishment_id")
	var req upsertDocTypeReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "document_type.upsert", "establishment", estID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	err := h.Registry.UpsertDocTypeConfig(r.Context(), adminregistry.UpsertDocTypeInput{
		EstablishmentID: estID, Environment: req.Environment,
		CodigoCanonico: req.CodigoCanonico, Active: req.Active,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "document_type.upsert", "establishment", estID, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "document_type.upsert", "establishment", estID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"establishment_id": estID,
		"environment":      req.Environment,
		"codigo_canonico":  req.CodigoCanonico,
		"active":           req.Active,
	})
}
