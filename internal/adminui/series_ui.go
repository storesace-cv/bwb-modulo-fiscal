package adminui

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

type seriesPage struct {
	pageBase
	Establishment *adminregistry.Establishment
	Series        []adminregistry.EstablishmentSeries
	Environment   string
}

func (h *Handler) establishmentSeries(w http.ResponseWriter, r *http.Request) {
	estID := r.PathValue("establishment_id")
	est, err := h.Registry.GetEstablishment(r.Context(), estID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	env := strings.TrimSpace(r.URL.Query().Get("environment"))
	rows, err := h.Registry.ListSeries(r.Context(), estID, env, listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "series.html", seriesPage{
		pageBase:      h.baseWithCSRF(w, r, "Séries", "Séries do estabelecimento", "establishments"),
		Establishment: &est,
		Series:        rows,
		Environment:   env,
	})
}

func (h *Handler) newSeriesForm(w http.ResponseWriter, r *http.Request) {
	estID := r.PathValue("establishment_id")
	est, err := h.Registry.GetEstablishment(r.Context(), estID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tok, err := h.CSRF.Issue(w)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "form_series.html", formPage{
		pageBase:      h.base(r, "Nova série", "Nova série (draft)", "establishments"),
		CSRFToken:     tok,
		Establishment: &est,
	})
}

func (h *Handler) createSeriesForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	estID := r.PathValue("establishment_id")
	_, err := h.Registry.CreateSeries(r.Context(), adminregistry.CreateSeriesInput{
		EstablishmentID: estID,
		Environment:     r.FormValue("environment"),
		CodigoCanonico:  r.FormValue("codigo_canonico"),
		Code:            r.FormValue("code"),
	})
	if err != nil {
		http.Error(w, "falha ao criar série", http.StatusUnprocessableEntity)
		return
	}
	http.Redirect(w, r, "/admin/ui/establishments/"+estID+"/series", http.StatusSeeOther)
}

func (h *Handler) transitionSeriesForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	seriesID := r.PathValue("series_id")
	ver, _ := strconv.Atoi(r.FormValue("expected_version"))
	out, err := h.Registry.TransitionSeries(r.Context(), adminregistry.TransitionSeriesInput{
		SeriesID: seriesID, ExpectedVersion: ver, ToStatus: r.FormValue("status"),
	})
	if err != nil {
		http.Error(w, "falha na transição", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/admin/ui/establishments/"+out.EstablishmentID+"/series", http.StatusSeeOther)
}
