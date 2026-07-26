package adminui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

const (
	sessionCookieName = "fiscal_admin_session"
	sessionTTL        = 8 * time.Hour
	sessionIDBytes    = 32
)

// SessionStore maps opaque session IDs to Claims (server-side only; never JWT in cookie).
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]sessionEntry
	now      func() time.Time
	maxSize  int
	secure   bool
}

type sessionEntry struct {
	claims adminauth.Claims
	exp    time.Time
}

// NewSessionStore builds an in-memory session store.
func NewSessionStore(now func() time.Time, cookieSecure bool) *SessionStore {
	if now == nil {
		now = time.Now
	}
	return &SessionStore{
		sessions: make(map[string]sessionEntry),
		now:      now,
		maxSize:  8192,
		secure:   cookieSecure,
	}
}

// Create issues a new session id, stores claims, sets HttpOnly Secure SameSite cookie.
func (s *SessionStore) Create(w http.ResponseWriter, claims adminauth.Claims) (string, error) {
	if s == nil {
		return "", adminauth.ErrValidation
	}
	if strings.TrimSpace(claims.Subject) == "" || len(claims.Roles) == 0 {
		return "", adminauth.ErrValidation
	}
	var b [sessionIDBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b[:])
	s.mu.Lock()
	s.gcLocked()
	s.sessions[id] = sessionEntry{claims: claims, exp: s.now().UTC().Add(sessionTTL)}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/admin/ui",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.secure,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return id, nil
}

// Destroy clears the session cookie and removes the server entry.
func (s *SessionStore) Destroy(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		return
	}
	if c, err := r.Cookie(sessionCookieName); err == nil && c != nil && c.Value != "" {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/admin/ui",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.secure,
		MaxAge:   -1,
	})
}

// Lookup returns claims for a valid session id.
func (s *SessionStore) Lookup(id string) (adminauth.Claims, bool) {
	if s == nil || id == "" {
		return adminauth.Claims{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ent, ok := s.sessions[id]
	if !ok || s.now().UTC().After(ent.exp) {
		if ok {
			delete(s.sessions, id)
		}
		return adminauth.Claims{}, false
	}
	return ent.claims, true
}

func (s *SessionStore) gcLocked() {
	now := s.now().UTC()
	for k, ent := range s.sessions {
		if now.After(ent.exp) {
			delete(s.sessions, k)
		}
	}
	for len(s.sessions) >= s.maxSize {
		// drop an arbitrary expired-or-oldest: delete first key
		for k := range s.sessions {
			delete(s.sessions, k)
			break
		}
	}
}

// SessionAuthenticator authenticates via opaque server-side session cookie (no JWT in browser).
type SessionAuthenticator struct {
	Store *SessionStore
}

// Authenticate implements adminauth.Authenticator.
func (a SessionAuthenticator) Authenticate(_ context.Context, r *http.Request) (adminauth.Claims, error) {
	if a.Store == nil {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c == nil || c.Value == "" {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	claims, ok := a.Store.Lookup(c.Value)
	if !ok {
		return adminauth.Claims{}, adminauth.ErrUnauthorized
	}
	return claims, nil
}
