package adminobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

type requestBag struct {
	Roles []string
}

const bagKey ctxKey = 2

// Middleware adds request_id, metrics and sanitized access logs around next.
// Place outermost so 401/403 from auth still get correlation + metrics.
func (o *Observer) Middleware(next http.Handler) http.Handler {
	if o == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := newRequestID()
		bag := &requestBag{}
		w.Header().Set(headerRequestID, reqID)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		ctx = context.WithValue(ctx, bagKey, bag)
		r2 := r.Clone(ctx)
		r2.Header = r.Header.Clone()
		r2.Header.Set(headerRequestID, reqID)

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r2)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		route := ClassifyPath(r2.URL.Path)
		outcome := OutcomeFromStatus(status)
		o.Inc(route, r2.Method, outcome)
		o.LogRequest(r2.Context(), route, r2.Method, status, time.Since(start), bag.Roles)
	})
}

// CaptureClaims copies RBAC roles into the observer bag (call inside auth chain).
func CaptureClaims(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bag, ok := r.Context().Value(bagKey).(*requestBag); ok {
			if claims, ok := adminauth.ClaimsFromContext(r.Context()); ok {
				for _, role := range claims.Roles {
					bag.Roles = append(bag.Roles, string(role))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "areq_" + hex.EncodeToString(b[:])
}

// ObservingAuthenticator wraps an Authenticator to count success/failure without logging secrets.
type ObservingAuthenticator struct {
	Inner adminauth.Authenticator
	Obs   *Observer
}

// Authenticate implements adminauth.Authenticator.
func (a ObservingAuthenticator) Authenticate(ctx context.Context, r *http.Request) (adminauth.Claims, error) {
	if a.Inner == nil {
		if a.Obs != nil {
			a.Obs.RecordAuth(false)
		}
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	claims, err := a.Inner.Authenticate(ctx, r)
	if a.Obs != nil {
		a.Obs.RecordAuth(err == nil)
	}
	return claims, err
}
