// Package adminapi serves /admin/v1 cadastros (RM-BO-001). No secrets in requests/responses.
package adminapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

const (
	headerRequestID = "X-Request-Id"
	maxBodyBytes    = 1 << 20
)

// Handler serves admin cadastro and ops visibility endpoints.
type Handler struct {
	Registry    *adminregistry.Registry
	Audit       *adminaudit.Store
	Ops         *adminops.Store
	SecretsMeta secretstore.AdminView // optional; Metadata only — never Reveal
}

type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
}

type createTaxpayerReq struct {
	NIF       string `json:"nif"`
	LegalName string `json:"legal_name"`
	Status    string `json:"status"`
}

type taxpayerResp struct {
	TaxpayerID string `json:"taxpayer_id"`
	NIF        string `json:"nif"`
	LegalName  string `json:"legal_name"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
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

// Mount registers /admin/v1 routes on mux with auth middleware.
func Mount(mux *http.ServeMux, authn adminauth.Authenticator, h *Handler) {
	authMW := adminauth.Middleware(authn)
	writeCadastro := adminauth.RequirePermission(adminauth.PermCadastroWrite)
	readCadastro := adminauth.RequirePermission(adminauth.PermCadastroRead)
	readAudit := adminauth.RequirePermission(adminauth.PermAuditRead)
	readOps := adminauth.RequirePermission(adminauth.PermOpsRead)
	readSecretMeta := adminauth.RequirePermission(adminauth.PermSecretMetaRead)

	mux.Handle("POST /admin/v1/taxpayers", authMW(writeCadastro(http.HandlerFunc(h.createTaxpayer))))
	mux.Handle("GET /admin/v1/taxpayers/{taxpayer_id}", authMW(readCadastro(http.HandlerFunc(h.getTaxpayer))))
	mux.Handle("POST /admin/v1/establishments", authMW(writeCadastro(http.HandlerFunc(h.createEstablishment))))
	mux.Handle("GET /admin/v1/establishments/{establishment_id}", authMW(readCadastro(http.HandlerFunc(h.getEstablishment))))
	mux.Handle("POST /admin/v1/scope-bindings", authMW(writeCadastro(http.HandlerFunc(h.createScopeBinding))))
	mux.Handle("GET /admin/v1/scope-bindings/{scope_id}", authMW(readCadastro(http.HandlerFunc(h.getScopeBinding))))
	mux.Handle("PATCH /admin/v1/scope-bindings/{scope_id}", authMW(writeCadastro(http.HandlerFunc(h.patchScopeBinding))))
	mux.Handle("GET /admin/v1/audit-events", authMW(readAudit(http.HandlerFunc(h.listAuditEvents))))
	mux.Handle("GET /admin/v1/ops/submissions", authMW(readOps(http.HandlerFunc(h.listOpsSubmissions))))
	mux.Handle("GET /admin/v1/secret-refs/metadata", authMW(readSecretMeta(http.HandlerFunc(h.getSecretRefMetadata))))
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
		h.writeRegistryErr(w, r, claims, "taxpayer.create", "taxpayer", req.NIF, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "taxpayer.create", "taxpayer", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusCreated, taxpayerResp{
		TaxpayerID: out.ID, NIF: out.NIF, LegalName: out.LegalName, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
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
	writeJSON(w, http.StatusOK, taxpayerResp{
		TaxpayerID: out.ID, NIF: out.NIF, LegalName: out.LegalName, Status: out.Status,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
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
	return strings.TrimSpace(r.Header.Get(headerRequestID))
}
