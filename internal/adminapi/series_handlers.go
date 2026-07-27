package adminapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

type createSeriesReq struct {
	Environment    string `json:"environment"`
	CodigoCanonico string `json:"codigo_canonico"`
	Code           string `json:"code"`
	ValidFrom      string `json:"valid_from,omitempty"`
}

type transitionSeriesReq struct {
	ExpectedVersion int    `json:"expected_version"`
	Status          string `json:"status"`
}

type seriesResp struct {
	SeriesID        string  `json:"series_id"`
	EstablishmentID string  `json:"establishment_id"`
	Environment     string  `json:"environment"`
	CodigoCanonico  string  `json:"codigo_canonico"`
	Code            string  `json:"code"`
	Status          string  `json:"status"`
	ValidFrom       string  `json:"valid_from"`
	ValidTo         *string `json:"valid_to,omitempty"`
	Version         int     `json:"version"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func seriesJSON(s adminregistry.EstablishmentSeries) seriesResp {
	out := seriesResp{
		SeriesID: s.ID, EstablishmentID: s.EstablishmentID, Environment: s.Environment,
		CodigoCanonico: s.CodigoCanonico, Code: s.Code, Status: s.Status,
		ValidFrom: s.ValidFrom.UTC().Format(time.RFC3339Nano),
		Version:   s.Version,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if s.ValidTo != nil {
		v := s.ValidTo.UTC().Format(time.RFC3339Nano)
		out.ValidTo = &v
	}
	return out
}

func (h *Handler) createSeries(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	estID := r.PathValue("establishment_id")
	var req createSeriesReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "series.create", "series", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	var validFrom time.Time
	if req.ValidFrom != "" {
		t, err := time.Parse(time.RFC3339Nano, req.ValidFrom)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.ValidFrom)
		}
		if err != nil {
			h.audit(r, claims, "series.create", "series", "-", adminaudit.ResultError)
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		validFrom = t
	}
	out, err := h.Registry.CreateSeries(r.Context(), adminregistry.CreateSeriesInput{
		EstablishmentID: estID, Environment: req.Environment,
		CodigoCanonico: req.CodigoCanonico, Code: req.Code, ValidFrom: validFrom,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "series.create", "series", "-", err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "series.create", "series", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusCreated, seriesJSON(out))
}

func (h *Handler) listSeries(w http.ResponseWriter, r *http.Request) {
	estID := r.PathValue("establishment_id")
	limit := adminops.ClampLimit(r.URL.Query().Get("limit"))
	env := r.URL.Query().Get("environment")
	rows, err := h.Registry.ListSeries(r.Context(), estID, env, limit)
	if err != nil {
		if errors.Is(err, adminregistry.ErrValidation) {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	items := make([]seriesResp, 0, len(rows))
	for _, s := range rows {
		items = append(items, seriesJSON(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) getSeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("series_id")
	out, err := h.Registry.GetSeries(r.Context(), id)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, seriesJSON(out))
}

func (h *Handler) transitionSeries(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("series_id")
	var req transitionSeriesReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "series.transition", "series", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	out, err := h.Registry.TransitionSeries(r.Context(), adminregistry.TransitionSeriesInput{
		SeriesID: id, ExpectedVersion: req.ExpectedVersion, ToStatus: req.Status,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "series.transition", "series", id, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "series.transition", "series", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, seriesJSON(out))
}
