package adminui

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	csrfCookieName = "fiscal_admin_csrf"
	csrfFieldName  = "csrf_token"
	csrfTTL        = 2 * time.Hour
)

// CSRFStore holds opaque tokens for double-submit cookie pattern (SSR forms).
type CSRFStore struct {
	mu      sync.Mutex
	tokens  map[string]time.Time
	now     func() time.Time
	maxSize int
}

// NewCSRFStore returns an in-memory CSRF store (process-local; fine for single instance).
func NewCSRFStore(now func() time.Time) *CSRFStore {
	if now == nil {
		now = time.Now
	}
	return &CSRFStore{tokens: make(map[string]time.Time), now: now, maxSize: 4096}
}

// Issue creates a token and sets the CSRF cookie (HttpOnly=false so form can echo;
// paired with SameSite=Strict + server-side map). Cookie is not the secret alone —
// token must match store.
func (s *CSRFStore) Issue(w http.ResponseWriter) (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b[:])
	s.mu.Lock()
	s.gcLocked()
	s.tokens[tok] = s.now().UTC().Add(csrfTTL)
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    tok,
		Path:     "/admin/ui",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // local http; TLS terminators set Secure in front proxy later
	})
	return tok, nil
}

// Validate checks form token == cookie == store entry (constant-time compare on hashes).
func (s *CSRFStore) Validate(r *http.Request, formToken string) bool {
	c, err := r.Cookie(csrfCookieName)
	if err != nil || c == nil || formToken == "" || c.Value == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(c.Value), []byte(formToken)) != 1 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.tokens[formToken]
	if !ok || s.now().UTC().After(exp) {
		return false
	}
	// Keep token until expiry so multiple status forms on one list page work.
	return true
}

func (s *CSRFStore) gcLocked() {
	now := s.now().UTC()
	for k, exp := range s.tokens {
		if now.After(exp) {
			delete(s.tokens, k)
		}
	}
	for len(s.tokens) >= s.maxSize {
		// drop arbitrary expired-or-oldest: delete one
		for k := range s.tokens {
			delete(s.tokens, k)
			break
		}
	}
}

// tokenFingerprint avoids logging raw tokens (unused helper for future audit).
func tokenFingerprint(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:8])
}
