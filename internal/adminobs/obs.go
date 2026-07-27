// Package adminobs provides backoffice observability (RM-BO-007):
// request_id correlation, sanitized structured logs, low-cardinality metrics,
// and health/readiness — without tokens, cookies, DSN, keys, NIF or bodies.
package adminobs

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	headerRequestID = "X-Request-Id"
	maxSeries       = 256
)

type ctxKey int

const requestIDKey ctxKey = 1

// Outcome is a low-cardinality result class for metrics.
type Outcome string

const (
	OutcomeOK           Outcome = "ok"
	OutcomeUnauthorized Outcome = "unauthorized"
	OutcomeForbidden    Outcome = "forbidden"
	OutcomeValidation   Outcome = "validation"
	OutcomeError        Outcome = "error"
)

// RouteClass groups paths without IDs (bounded cardinality).
type RouteClass string

const (
	RouteHealth         RouteClass = "health"
	RouteReady          RouteClass = "ready"
	RouteMetrics        RouteClass = "metrics"
	RouteTaxpayers      RouteClass = "taxpayers"
	RouteEstablishments RouteClass = "establishments"
	RouteBindings       RouteClass = "scope_bindings"
	RouteAudit          RouteClass = "audit"
	RouteOps            RouteClass = "ops"
	RouteSecretMeta     RouteClass = "secret_meta"
	RouteSecAdm         RouteClass = "secadm"
	RouteUI             RouteClass = "ui"
	RouteOther          RouteClass = "other"
)

// Observer is fail-safe: metric/log failures never break requests.
type Observer struct {
	Log      *slog.Logger
	AuthMode string // fail_closed|injected|oidc_jwt — process config only

	mu      sync.Mutex
	series  map[seriesKey]*atomic.Uint64
	dropped atomic.Uint64

	authOK   atomic.Uint64
	authFail atomic.Uint64
}

type seriesKey struct {
	Route   RouteClass
	Method  string
	Outcome Outcome
}

// New builds an Observer. log may be nil (no-op logs).
func New(log *slog.Logger, authMode string) *Observer {
	if log == nil {
		log = slog.Default()
	}
	mode := strings.TrimSpace(authMode)
	if mode == "" {
		mode = "fail_closed"
	}
	return &Observer{
		Log:      log,
		AuthMode: mode,
		series:   make(map[seriesKey]*atomic.Uint64),
	}
}

// RequestIDFromContext returns the correlation id set by middleware.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// ClassifyPath maps URL path → bounded route class (no IDs in labels).
func ClassifyPath(path string) RouteClass {
	p := strings.TrimSpace(path)
	switch {
	case p == "/admin/v1/health":
		return RouteHealth
	case p == "/admin/v1/ready":
		return RouteReady
	case p == "/admin/v1/ops/metrics":
		return RouteMetrics
	case strings.HasPrefix(p, "/admin/v1/secadm/"):
		return RouteSecAdm
	case strings.HasPrefix(p, "/admin/v1/secret-refs/"):
		return RouteSecretMeta
	case strings.HasPrefix(p, "/admin/v1/ops/"):
		return RouteOps
	case strings.HasPrefix(p, "/admin/v1/audit-events"):
		return RouteAudit
	case strings.HasPrefix(p, "/admin/v1/taxpayers"):
		return RouteTaxpayers
	case strings.HasPrefix(p, "/admin/v1/establishments"):
		return RouteEstablishments
	case strings.HasPrefix(p, "/admin/v1/scope-bindings"):
		return RouteBindings
	case strings.HasPrefix(p, "/admin/ui"):
		return RouteUI
	default:
		return RouteOther
	}
}

// Inc increments a counter; drops new series beyond maxSeries (fail-safe).
func (o *Observer) Inc(route RouteClass, method string, outcome Outcome) {
	if o == nil {
		return
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "UNKNOWN"
	}
	key := seriesKey{Route: route, Method: method, Outcome: outcome}
	o.mu.Lock()
	c, ok := o.series[key]
	if !ok {
		if len(o.series) >= maxSeries {
			o.mu.Unlock()
			o.dropped.Add(1)
			return
		}
		c = &atomic.Uint64{}
		o.series[key] = c
	}
	o.mu.Unlock()
	c.Add(1)
}

// RecordAuth counts authenticator outcomes (no token/subject logged here).
func (o *Observer) RecordAuth(ok bool) {
	if o == nil {
		return
	}
	if ok {
		o.authOK.Add(1)
	} else {
		o.authFail.Add(1)
	}
}

// Snapshot returns a copy of counters for /ops/metrics (sanitized labels only).
func (o *Observer) Snapshot() MetricsSnapshot {
	if o == nil {
		return MetricsSnapshot{}
	}
	out := MetricsSnapshot{
		AuthMode: o.AuthMode,
		AuthOK:   o.authOK.Load(),
		AuthFail: o.authFail.Load(),
		Dropped:  o.dropped.Load(),
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	out.Series = make([]MetricSeries, 0, len(o.series))
	for k, c := range o.series {
		out.Series = append(out.Series, MetricSeries{
			RouteClass: string(k.Route),
			Method:     k.Method,
			Outcome:    string(k.Outcome),
			Count:      c.Load(),
		})
	}
	return out
}

// MetricsSnapshot is a sanitized metrics export.
type MetricsSnapshot struct {
	AuthMode string         `json:"auth_mode"`
	AuthOK   uint64         `json:"auth_ok"`
	AuthFail uint64         `json:"auth_fail"`
	Dropped  uint64         `json:"series_dropped"`
	Series   []MetricSeries `json:"series"`
}

// MetricSeries is one low-cardinality counter.
type MetricSeries struct {
	RouteClass string `json:"route_class"`
	Method     string `json:"method"`
	Outcome    string `json:"outcome"`
	Count      uint64 `json:"count"`
}

// LogRequest emits a sanitized access log line (fail-safe).
func (o *Observer) LogRequest(ctx context.Context, route RouteClass, method string, status int, dur time.Duration, roles []string) {
	if o == nil || o.Log == nil {
		return
	}
	// Bound role label cardinality: join known roles only, sorted unique.
	roleLabel := sanitizeRoles(roles)
	o.Log.Info("admin_request",
		"request_id", RequestIDFromContext(ctx),
		"route_class", string(route),
		"method", strings.ToUpper(method),
		"status", status,
		"duration_ms", dur.Milliseconds(),
		"roles", roleLabel,
		"auth_mode", o.AuthMode,
		// intentionally absent: subject, Authorization, Cookie, path with ids, body, NIF
	)
}

func sanitizeRoles(roles []string) string {
	seen := map[string]struct{}{}
	var parts []string
	for _, r := range roles {
		r = strings.TrimSpace(r)
		switch r {
		case "owner", "admin", "operator", "auditor":
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			parts = append(parts, r)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	// stable order
	order := []string{"owner", "admin", "operator", "auditor"}
	out := make([]string, 0, len(parts))
	for _, o := range order {
		if _, ok := seen[o]; ok {
			out = append(out, o)
		}
	}
	return strings.Join(out, ",")
}

// OutcomeFromStatus maps HTTP status → Outcome.
func OutcomeFromStatus(status int) Outcome {
	switch {
	case status >= 200 && status < 400:
		return OutcomeOK
	case status == http.StatusUnauthorized:
		return OutcomeUnauthorized
	case status == http.StatusForbidden:
		return OutcomeForbidden
	case status == http.StatusUnprocessableEntity || status == http.StatusBadRequest:
		return OutcomeValidation
	default:
		return OutcomeError
	}
}
