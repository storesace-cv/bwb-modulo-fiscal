// Package adminapi serves /admin/v1 cadastros (RM-BO-001). No secrets in requests/responses.
package adminapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminobs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

const (
	headerRequestID = "X-Request-Id"
	maxBodyBytes    = 1 << 20
)

// Handler serves admin cadastro, ops visibility and SecAdm write-only endpoints.
type Handler struct {
	Registry      *adminregistry.Registry
	Audit         *adminaudit.Store
	Ops           *adminops.Store
	SecretsMeta   secretstore.AdminView // optional; Metadata only — never Reveal
	SecAdm        *secadm.Gate          // optional; owner-only Put/Rotate/Revoke
	Obs           *adminobs.Observer    // optional; RM-BO-007
	DB            *sql.DB               // readiness ping only
	AuthMode      string                // fail_closed|injected|oidc_jwt
	Version       string
	Revision      string
	AuthorityMode string // FISCAL_AUTHORITY (simulator|…)
	FiscalEnv     string // FISCAL_ENV
}

type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
}

type createTaxpayerReq struct {
	NIF          string           `json:"nif"`
	LegalName    string           `json:"legal_name"`
	Status       string           `json:"status"`
	FEEnrollment *feEnrollmentReq `json:"fe_enrollment,omitempty"`
}

type feEnrollmentReq struct {
	Environment string `json:"environment"`
	Status      string `json:"status"`
}

type feEnrollmentResp struct {
	Environment string `json:"environment"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at"`
}

type taxpayerResp struct {
	TaxpayerID    string             `json:"taxpayer_id"`
	NIF           string             `json:"nif"`
	LegalName     string             `json:"legal_name"`
	Status        string             `json:"status"`
	CreatedAt     string             `json:"created_at"`
	FEEnrollments []feEnrollmentResp `json:"fe_enrollments"`
}

type createEstablishmentReq struct {
	TaxpayerID string `json:"taxpayer_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Status     string `json:"status"`
}

type establishmentResp struct {
	EstablishmentID string `json:"establishment_id"`
	TaxpayerID      string `json:"taxpayer_id"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
}

type createScopeBindingReq struct {
	ScopeID             string `json:"scope_id"`
	TaxpayerID          string `json:"taxpayer_id"`
	EstablishmentID     string `json:"establishment_id"`
	Environment         string `json:"environment"`
	IANATimezone        string `json:"iana_timezone"`
	SeriesEffectiveCode string `json:"series_effective_code"`
	Status              string `json:"status"`
}

type updateScopeConfigReq struct {
	Environment         string `json:"environment"`
	IANATimezone        string `json:"iana_timezone"`
	SeriesEffectiveCode string `json:"series_effective_code"`
	Status              string `json:"status"`
}

type scopeBindingResp struct {
	ScopeID             string `json:"scope_id"`
	TaxpayerID          string `json:"taxpayer_id"`
	EstablishmentID     string `json:"establishment_id"`
	Environment         string `json:"environment"`
	IANATimezone        string `json:"iana_timezone"`
	SeriesEffectiveCode string `json:"series_effective_code"`
	Status              string `json:"status"`
	CreatedAt           string `json:"created_at"`
}

// Mount registers /admin/v1 routes on mux with auth + observability middleware.
func Mount(mux *http.ServeMux, authn adminauth.Authenticator, h *Handler) {
	if h == nil {
		return
	}
	obs := h.Obs
	if obs == nil {
		obs = adminobs.New(nil, h.AuthMode)
	}
	authn = adminobs.ObservingAuthenticator{Inner: authn, Obs: obs}
	authMW := adminauth.Middleware(authn)
	writeCadastro := adminauth.RequirePermission(adminauth.PermCadastroWrite)
	readCadastro := adminauth.RequirePermission(adminauth.PermCadastroRead)
	readAudit := adminauth.RequirePermission(adminauth.PermAuditRead)
	readOps := adminauth.RequirePermission(adminauth.PermOpsRead)
	readSecretMeta := adminauth.RequirePermission(adminauth.PermSecretMetaRead)

	wrap := func(next http.Handler) http.Handler {
		return obs.Middleware(authMW(adminobs.CaptureClaims(next)))
	}
	wrapPublic := func(next http.Handler) http.Handler {
		return obs.Middleware(next)
	}

	mux.Handle("GET /admin/v1/health", wrapPublic(adminobs.HealthHandler(h.Version, h.Revision)))
	mux.Handle("GET /admin/v1/ready", wrapPublic(adminobs.ReadyHandler(adminobs.ReadyDeps{
		DB: h.DB, AuthMode: h.AuthMode, SecAdmReady: h.SecAdm != nil,
		Version: h.Version, Revision: h.Revision,
	})))
	mux.Handle("GET /admin/v1/ops/metrics", wrap(readOps(adminobs.MetricsHandler(obs))))

	mux.Handle("POST /admin/v1/taxpayers", wrap(writeCadastro(http.HandlerFunc(h.createTaxpayer))))
	mux.Handle("GET /admin/v1/taxpayers", wrap(readCadastro(http.HandlerFunc(h.listTaxpayers))))
	mux.Handle("GET /admin/v1/taxpayers/{taxpayer_id}", wrap(readCadastro(http.HandlerFunc(h.getTaxpayer))))
	mux.Handle("PATCH /admin/v1/taxpayers/{taxpayer_id}", wrap(writeCadastro(http.HandlerFunc(h.patchTaxpayer))))
	mux.Handle("PUT /admin/v1/taxpayers/{taxpayer_id}/fe-enrollments", wrap(writeCadastro(http.HandlerFunc(h.putTaxpayerFEEnrollment))))
	mux.Handle("POST /admin/v1/establishments", wrap(writeCadastro(http.HandlerFunc(h.createEstablishment))))
	mux.Handle("GET /admin/v1/establishments", wrap(readCadastro(http.HandlerFunc(h.listEstablishments))))
	mux.Handle("GET /admin/v1/establishments/{establishment_id}", wrap(readCadastro(http.HandlerFunc(h.getEstablishment))))
	mux.Handle("PATCH /admin/v1/establishments/{establishment_id}", wrap(writeCadastro(http.HandlerFunc(h.patchEstablishment))))
	mux.Handle("POST /admin/v1/scope-bindings", wrap(writeCadastro(http.HandlerFunc(h.createScopeBinding))))
	mux.Handle("GET /admin/v1/scope-bindings", wrap(readCadastro(http.HandlerFunc(h.listScopeBindings))))
	mux.Handle("GET /admin/v1/scope-bindings/{scope_id}", wrap(readCadastro(http.HandlerFunc(h.getScopeBinding))))
	mux.Handle("PATCH /admin/v1/scope-bindings/{scope_id}", wrap(writeCadastro(http.HandlerFunc(h.patchScopeBinding))))
	mux.Handle("GET /admin/v1/audit-events", wrap(readAudit(http.HandlerFunc(h.listAuditEvents))))
	mux.Handle("GET /admin/v1/ops/submissions", wrap(readOps(http.HandlerFunc(h.listOpsSubmissions))))
	mux.Handle("GET /admin/v1/secret-refs/metadata", wrap(readSecretMeta(http.HandlerFunc(h.getSecretRefMetadata))))

	secadmWrite := adminauth.RequirePermission(adminauth.PermSecAdmWrite)
	mux.Handle("POST /admin/v1/authority-profiles", wrap(secadmWrite(http.HandlerFunc(h.createAuthorityProfile))))
	mux.Handle("GET /admin/v1/authority-profiles", wrap(readOps(http.HandlerFunc(h.listAuthorityProfiles))))
	mux.Handle("GET /admin/v1/authority-profiles/{profile_id}", wrap(readOps(http.HandlerFunc(h.getAuthorityProfile))))
	mux.Handle("GET /admin/v1/authority-profiles/{profile_id}/readiness", wrap(readOps(http.HandlerFunc(h.getAuthorityReadiness))))
	mux.Handle("GET /admin/v1/authority-profiles/{profile_id}/history", wrap(readAudit(http.HandlerFunc(h.getAuthorityProfileHistory))))
	mux.Handle("PATCH /admin/v1/authority-profiles/{profile_id}", wrap(secadmWrite(http.HandlerFunc(h.patchAuthorityProfile))))

	mux.Handle("PUT /admin/v1/secadm/secret-refs", wrap(secadmWrite(http.HandlerFunc(h.secadmPut))))
	mux.Handle("POST /admin/v1/secadm/secret-refs/rotate", wrap(secadmWrite(http.HandlerFunc(h.secadmRotate))))
	mux.Handle("POST /admin/v1/secadm/secret-refs/revoke", wrap(secadmWrite(http.HandlerFunc(h.secadmRevoke))))
	mux.Handle("POST /admin/v1/secadm/material", wrap(secadmWrite(http.HandlerFunc(h.secadmPutMaterial))))
	mux.Handle("POST /admin/v1/secadm/material/rotate", wrap(secadmWrite(http.HandlerFunc(h.secadmRotateMaterial))))
	mux.Handle("POST /admin/v1/secadm/material/validate-offline", wrap(secadmWrite(http.HandlerFunc(h.secadmValidateOffline))))
	mux.Handle("GET /admin/v1/authority-profiles/{profile_id}/material-lifecycle", wrap(secadmWrite(http.HandlerFunc(h.getAuthorityMaterialLifecycle))))
	mux.Handle("POST /admin/v1/authority/probe-config", wrap(secadmWrite(http.HandlerFunc(h.probeAuthorityConfig))))
}

func (h *Handler) createTaxpayer(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	var req createTaxpayerReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "taxpayer.create", "taxpayer", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.CreateTaxpayer(r.Context(), adminregistry.CreateTaxpayerInput{
		NIF: req.NIF, LegalName: req.LegalName, Status: req.Status,
	})
	if err != nil {
		// Never put NIF in audit resource_id (RM-BO-007 / RM-BO-012).
		h.writeRegistryErr(w, r, claims, "taxpayer.create", "taxpayer", "-", err)
		return
	}
	if req.FEEnrollment != nil {
		_, err = h.Registry.UpsertFEEnrollment(r.Context(), adminregistry.UpsertFEEnrollmentInput{
			TaxpayerID: out.ID, Environment: req.FEEnrollment.Environment, Status: req.FEEnrollment.Status,
		})
		if err != nil {
			h.writeRegistryErr(w, r, claims, "taxpayer.fe_enrollment.upsert", "taxpayer", out.ID, err)
			return
		}
		_ = h.Audit.Record(r.Context(), claims, "taxpayer.fe_enrollment.upsert", "taxpayer", out.ID, adminaudit.ResultSuccess, requestID(r))
	}
	_ = h.Audit.Record(r.Context(), claims, "taxpayer.create", "taxpayer", out.ID, adminaudit.ResultSuccess, requestID(r))
	resp, err := h.taxpayerJSON(r.Context(), out)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) getTaxpayer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("taxpayer_id")
	out, err := h.Registry.GetTaxpayer(r.Context(), id)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	resp, err := h.taxpayerJSON(r.Context(), out)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) listTaxpayers(w http.ResponseWriter, r *http.Request) {
	limit := adminops.ClampLimit(r.URL.Query().Get("limit"))
	rows, err := h.Registry.ListTaxpayers(r.Context(), limit)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	items := make([]taxpayerResp, 0, len(rows))
	for _, out := range rows {
		resp, err := h.taxpayerJSON(r.Context(), out)
		if err != nil {
			writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
			return
		}
		items = append(items, resp)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type patchStatusReq struct {
	Status string `json:"status"`
}

func (h *Handler) patchTaxpayer(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("taxpayer_id")
	var req patchStatusReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "taxpayer.update_status", "taxpayer", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.UpdateTaxpayerStatus(r.Context(), id, req.Status)
	if err != nil {
		h.writeRegistryErr(w, r, claims, "taxpayer.update_status", "taxpayer", id, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "taxpayer.update_status", "taxpayer", out.ID, adminaudit.ResultSuccess, requestID(r))
	resp, err := h.taxpayerJSON(r.Context(), out)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) putTaxpayerFEEnrollment(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("taxpayer_id")
	var req feEnrollmentReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "taxpayer.fe_enrollment.upsert", "taxpayer", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.UpsertFEEnrollment(r.Context(), adminregistry.UpsertFEEnrollmentInput{
		TaxpayerID: id, Environment: req.Environment, Status: req.Status,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "taxpayer.fe_enrollment.upsert", "taxpayer", id, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "taxpayer.fe_enrollment.upsert", "taxpayer", out.TaxpayerID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, feEnrollmentResp{
		Environment: out.Environment,
		Status:      out.Status,
		UpdatedAt:   out.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) taxpayerJSON(ctx context.Context, out adminregistry.Taxpayer) (taxpayerResp, error) {
	enrollments, err := h.Registry.ListFEEnrollments(ctx, out.ID)
	if err != nil {
		return taxpayerResp{}, err
	}
	fe := make([]feEnrollmentResp, 0, len(enrollments))
	for _, e := range enrollments {
		fe = append(fe, feEnrollmentResp{
			Environment: e.Environment,
			Status:      e.Status,
			UpdatedAt:   e.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return taxpayerResp{
		TaxpayerID: out.ID, NIF: out.NIF, LegalName: out.LegalName, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano), FEEnrollments: fe,
	}, nil
}

func (h *Handler) createEstablishment(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	var req createEstablishmentReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "establishment.create", "establishment", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.CreateEstablishment(r.Context(), adminregistry.CreateEstablishmentInput{
		TaxpayerID: req.TaxpayerID, Code: req.Code, Name: req.Name, Status: req.Status,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "establishment.create", "establishment", req.Code, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "establishment.create", "establishment", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusCreated, establishmentResp{
		EstablishmentID: out.ID, TaxpayerID: out.TaxpayerID, Code: out.Code, Name: out.Name, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) getEstablishment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("establishment_id")
	out, err := h.Registry.GetEstablishment(r.Context(), id)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, establishmentResp{
		EstablishmentID: out.ID, TaxpayerID: out.TaxpayerID, Code: out.Code, Name: out.Name, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) listEstablishments(w http.ResponseWriter, r *http.Request) {
	limit := adminops.ClampLimit(r.URL.Query().Get("limit"))
	tp := r.URL.Query().Get("taxpayer_id")
	rows, err := h.Registry.ListEstablishments(r.Context(), tp, limit)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	items := make([]establishmentResp, 0, len(rows))
	for _, out := range rows {
		items = append(items, establishmentResp{
			EstablishmentID: out.ID, TaxpayerID: out.TaxpayerID, Code: out.Code, Name: out.Name, Status: out.Status,
			CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) patchEstablishment(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("establishment_id")
	var req patchStatusReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "establishment.update_status", "establishment", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.UpdateEstablishmentStatus(r.Context(), id, req.Status)
	if err != nil {
		h.writeRegistryErr(w, r, claims, "establishment.update_status", "establishment", id, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "establishment.update_status", "establishment", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, establishmentResp{
		EstablishmentID: out.ID, TaxpayerID: out.TaxpayerID, Code: out.Code, Name: out.Name, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) listScopeBindings(w http.ResponseWriter, r *http.Request) {
	limit := adminops.ClampLimit(r.URL.Query().Get("limit"))
	rows, err := h.Registry.ListScopeBindings(r.Context(), limit)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	items := make([]scopeBindingResp, 0, len(rows))
	for _, out := range rows {
		items = append(items, scopeBindingResp{
			ScopeID: out.ScopeID, TaxpayerID: out.TaxpayerID, EstablishmentID: out.EstablishmentID,
			Environment: out.Environment, IANATimezone: out.IANATimezone,
			SeriesEffectiveCode: out.SeriesEffectiveCode, Status: out.Status,
			CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createScopeBinding(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	var req createScopeBindingReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "scope_binding.create", "scope_binding", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.CreateScopeBinding(r.Context(), adminregistry.CreateScopeBindingInput{
		ScopeID: req.ScopeID, TaxpayerID: req.TaxpayerID, EstablishmentID: req.EstablishmentID,
		Environment: req.Environment, IANATimezone: req.IANATimezone,
		SeriesEffectiveCode: req.SeriesEffectiveCode, Status: req.Status,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "scope_binding.create", "scope_binding", req.ScopeID, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "scope_binding.create", "scope_binding", out.ScopeID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusCreated, scopeBindingResp{
		ScopeID: out.ScopeID, TaxpayerID: out.TaxpayerID, EstablishmentID: out.EstablishmentID,
		Environment: out.Environment, IANATimezone: out.IANATimezone,
		SeriesEffectiveCode: out.SeriesEffectiveCode, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) getScopeBinding(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("scope_id")
	out, err := h.Registry.GetScopeBinding(r.Context(), id)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, scopeBindingResp{
		ScopeID: out.ScopeID, TaxpayerID: out.TaxpayerID, EstablishmentID: out.EstablishmentID,
		Environment: out.Environment, IANATimezone: out.IANATimezone,
		SeriesEffectiveCode: out.SeriesEffectiveCode, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) patchScopeBinding(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	scopeID := r.PathValue("scope_id")
	var req updateScopeConfigReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "scope_binding.update_config", "scope_binding", scopeID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.UpdateScopeConfig(r.Context(), adminregistry.UpdateScopeConfigInput{
		ScopeID: scopeID, Environment: req.Environment, IANATimezone: req.IANATimezone,
		SeriesEffectiveCode: req.SeriesEffectiveCode, Status: req.Status,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "scope_binding.update_config", "scope_binding", scopeID, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "scope_binding.update_config", "scope_binding", out.ScopeID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, scopeBindingResp{
		ScopeID: out.ScopeID, TaxpayerID: out.TaxpayerID, EstablishmentID: out.EstablishmentID,
		Environment: out.Environment, IANATimezone: out.IANATimezone,
		SeriesEffectiveCode: out.SeriesEffectiveCode, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) writeRegistryErr(w http.ResponseWriter, r *http.Request, claims adminauth.Claims, action, resType, resID string, err error) {
	switch {
	case errors.Is(err, adminregistry.ErrValidation):
		h.audit(r, claims, action, resType, resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
	case errors.Is(err, adminregistry.ErrConflict):
		h.audit(r, claims, action, resType, resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusConflict, "ADMIN_CONFLICT", "Conflict")
	case errors.Is(err, adminregistry.ErrNotFound):
		h.audit(r, claims, action, resType, resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
	default:
		h.audit(r, claims, action, resType, resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
	}
}

func (h *Handler) audit(r *http.Request, claims adminauth.Claims, action, resType, resID, result string) {
	if h.Audit == nil {
		return
	}
	_ = h.Audit.Record(r.Context(), claims, action, resType, resID, result, requestID(r))
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title string) {
	writeJSON(w, status, problem{
		Type: "about:blank", Title: title, Status: status, Code: code, RequestID: requestID(r),
	})
}

func requestID(r *http.Request) string {
	if id := adminobs.RequestIDFromContext(r.Context()); id != "" {
		return id
	}
	return strings.TrimSpace(r.Header.Get(headerRequestID))
}
