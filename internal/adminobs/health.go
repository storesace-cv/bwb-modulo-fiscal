package adminobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

const readyPingTimeout = 2 * time.Second

// ReadyDeps are non-secret readiness inputs (fail-safe timeouts).
type ReadyDeps struct {
	DB          *sql.DB
	AuthMode    string
	SecAdmReady bool // gate configured (not secrets)
	Version     string
	Revision    string
	// Auth diagnostics (RM-BO-018); optional — empty means legacy behaviour.
	OIDCConfigured   string // ok|not_configured|incomplete
	InteractiveLogin string // unavailable|ready
}

// HealthHandler serves GET /admin/v1/health (liveness; no auth; no secrets).
func HealthHandler(version, revision string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"component":  "admin",
			"version":    version,
			"revision":   revision,
			"request_id": RequestIDFromContext(r.Context()),
		})
	})
}

// ReadyHandler serves GET /admin/v1/ready (readiness; no auth; sanitized checks only).
func ReadyHandler(deps ReadyDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{}
		ready := true

		checks["admin_auth_mode"] = deps.AuthMode
		if deps.AuthMode == "" {
			checks["admin_auth_mode"] = "fail_closed"
		}
		// fail_closed is a valid configured mode (explicit deny); injected/oidc_jwt also ok.
		checks["admin_auth_configured"] = "ok"

		if deps.OIDCConfigured != "" {
			checks["admin_oidc"] = deps.OIDCConfigured
		}
		if deps.InteractiveLogin != "" {
			checks["admin_interactive_login"] = deps.InteractiveLogin
		} else {
			checks["admin_interactive_login"] = "unavailable"
		}

		if deps.SecAdmReady {
			checks["secadm_gate"] = "configured"
		} else {
			checks["secadm_gate"] = "absent"
		}

		if deps.DB == nil {
			checks["database"] = "absent"
			ready = false
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), readyPingTimeout)
			err := deps.DB.PingContext(ctx)
			cancel()
			if err != nil {
				checks["database"] = "fail"
				ready = false
			} else {
				checks["database"] = "ok"
			}
		}

		status := "ok"
		code := http.StatusOK
		if !ready {
			status = "not_ready"
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, map[string]any{
			"status":     status,
			"component":  "admin",
			"version":    deps.Version,
			"revision":   deps.Revision,
			"checks":     checks,
			"request_id": RequestIDFromContext(r.Context()),
		})
	})
}

// MetricsHandler serves GET /admin/v1/ops/metrics (auth required by Mount).
func MetricsHandler(obs *Observer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := MetricsSnapshot{}
		if obs != nil {
			snap = obs.Snapshot()
		}
		writeJSON(w, http.StatusOK, snap)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
