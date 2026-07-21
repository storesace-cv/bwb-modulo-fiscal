// Package health expõe o endpoint de prontidão alinhado com o contrato OpenAPI.
package health

import (
	"encoding/json"
	"net/http"
)

// Status valores permitidos pelo schema HealthResponse.
const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
)

// Response é o corpo JSON de GET /v1/health.
type Response struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	FiscalPackage string `json:"fiscalPackage"`
}

// Handler serve GET /v1/health sem lógica fiscal.
type Handler struct {
	Version       string
	FiscalPackage string
	Status        string
}

// NewHandler cria um handler de health com estado ok por omissão.
func NewHandler(version, fiscalPackage string) *Handler {
	return &Handler{
		Version:       version,
		FiscalPackage: fiscalPackage,
		Status:        StatusOK,
	}
}

// ServeHTTP implementa http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	status := h.Status
	if status == "" {
		status = StatusOK
	}

	body := Response{
		Status:        status,
		Version:       h.Version,
		FiscalPackage: h.FiscalPackage,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
