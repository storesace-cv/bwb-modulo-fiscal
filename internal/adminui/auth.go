package adminui

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

// DevCookieAuthenticator accepts an HttpOnly session cookie in development only.
// It never invents a login form; the cookie value must match FISCAL_ADMIN_UI_DEV_COOKIE
// (≥32 bytes) and FISCAL_ENV=development + injected admin auth. Production: unused.
type DevCookieAuthenticator struct {
	Claims    adminauth.Claims
	CookieVal string
	AllowDev  bool
}

// Authenticate implements adminauth.Authenticator.
func (d DevCookieAuthenticator) Authenticate(_ context.Context, r *http.Request) (adminauth.Claims, error) {
	if !d.AllowDev || len(d.CookieVal) < 32 {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c == nil {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	got := sha256.Sum256([]byte(c.Value))
	want := sha256.Sum256([]byte(d.CookieVal))
	if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	if strings.TrimSpace(d.Claims.Subject) == "" || len(d.Claims.Roles) == 0 {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	return d.Claims, nil
}

// ChainAuthenticator tries authenticators in order (first success wins).
type ChainAuthenticator []adminauth.Authenticator

// Authenticate implements adminauth.Authenticator.
func (c ChainAuthenticator) Authenticate(ctx context.Context, r *http.Request) (adminauth.Claims, error) {
	var last error = adminauth.ErrUnauthorized
	for _, a := range c {
		if a == nil {
			continue
		}
		claims, err := a.Authenticate(ctx, r)
		if err == nil {
			return claims, nil
		}
		last = err
	}
	return adminauth.Claims{}, last
}

// BuildUIAuthenticator wraps API admin authenticator with optional dev cookie bridge.
func BuildUIAuthenticator(apiAuth adminauth.Authenticator, env string, injectSubject string, injectRoles []adminauth.Role) adminauth.Authenticator {
	if apiAuth == nil {
		apiAuth = adminauth.FailClosedAuthenticator{}
	}
	devCookie := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_UI_DEV_COOKIE"))
	if env == "development" && len(devCookie) >= 32 && injectSubject != "" && len(injectRoles) > 0 {
		return ChainAuthenticator{
			apiAuth,
			DevCookieAuthenticator{
				AllowDev:  true,
				CookieVal: devCookie,
				Claims:    adminauth.Claims{Subject: injectSubject, Roles: injectRoles},
			},
		}
	}
	return apiAuth
}

func htmlAuthMiddleware(authn adminauth.Authenticator) func(http.Handler) http.Handler {
	if authn == nil {
		authn = adminauth.FailClosedAuthenticator{}
	}
	unauthTmpl, _ := embedded.ReadFile("templates/unauthorized.html")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := authn.Authenticate(r.Context(), r)
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write(unauthTmpl)
				return
			}
			next.ServeHTTP(w, r.WithContext(adminauth.ContextWithClaims(r.Context(), claims)))
		})
	}
}
