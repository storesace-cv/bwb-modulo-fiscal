package femock

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

func checkBasicAuth(r *http.Request, wantUser, wantPass string) bool {
	h := r.Header.Get("Authorization")
	const prefix = "Basic "
	if !strings.HasPrefix(h, prefix) {
		// Still compare dummies to reduce timing skew on missing header path.
		_ = subtle.ConstantTimeCompare([]byte(wantUser), []byte(wantUser))
		_ = subtle.ConstantTimeCompare([]byte(wantPass), []byte(wantPass))
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(h[len(prefix):])
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		return false
	}
	// Constant-time compare; unequal lengths yield 0 (Go subtle).
	uOK := subtle.ConstantTimeCompare([]byte(user), []byte(wantUser))
	pOK := subtle.ConstantTimeCompare([]byte(pass), []byte(wantPass))
	return uOK&pOK == 1
}
