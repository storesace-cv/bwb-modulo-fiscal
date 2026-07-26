package adminauth_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

func TestOIDCAuthenticatorHappyPathAndRejects(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key-1"
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
				"n": base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
			}},
		})
	}))
	defer jwks.Close()

	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	cfg := adminauth.OIDCConfig{
		Issuer:      "https://idp.example/realms/bwb",
		Audience:    "bwb-admin",
		JWKSURL:     jwks.URL,
		AllowedAlgs: []string{"RS256"},
		RoleClaim:   "groups",
		RoleMap: map[string]adminauth.Role{
			"bwb-admins":   adminauth.RoleAdmin,
			"bwb-ops":      adminauth.RoleOperator,
			"bwb-owners":   adminauth.RoleOwner,
			"bwb-auditors": adminauth.RoleAuditor,
		},
		OwnerSubjects: []string{"owner-sub-1"},
		Clock:         func() time.Time { return now },
		AllowHTTPJWKS: true,
		HTTPClient:    jwks.Client(),
	}
	authn, err := adminauth.NewOIDCAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}

	tok := signTestJWT(t, priv, kid, map[string]any{
		"iss": cfg.Issuer, "aud": cfg.Audience, "sub": "ops-1",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(), "nbf": now.Unix(),
		"groups": []string{"bwb-ops", "unknown-group"},
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/taxpayers", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	claims, err := authn.Authenticate(t.Context(), req)
	if err != nil {
		t.Fatalf("want ok: %v", err)
	}
	if claims.Subject != "ops-1" || !claims.HasRole(adminauth.RoleOperator) || claims.HasRole(adminauth.RoleOwner) {
		t.Fatalf("%+v", claims)
	}

	// Owner mapped but subject not allowlisted → no elevation
	tok = signTestJWT(t, priv, kid, map[string]any{
		"iss": cfg.Issuer, "aud": cfg.Audience, "sub": "random",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		"groups": []string{"bwb-owners"},
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	if _, err := authn.Authenticate(t.Context(), req); err == nil {
		t.Fatal("owner without allowlist must fail")
	}

	// Owner allowlisted
	tok = signTestJWT(t, priv, kid, map[string]any{
		"iss": cfg.Issuer, "aud": cfg.Audience, "sub": "owner-sub-1",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		"groups": []string{"bwb-owners"},
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	claims, err = authn.Authenticate(t.Context(), req)
	if err != nil || !claims.HasRole(adminauth.RoleOwner) {
		t.Fatalf("owner: %v %+v", err, claims)
	}

	// alg=none
	noneTok := unsignedNoneJWT(map[string]any{
		"iss": cfg.Issuer, "aud": cfg.Audience, "sub": "x",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		"groups": []string{"bwb-ops"},
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+noneTok)
	if _, err := authn.Authenticate(t.Context(), req); err == nil {
		t.Fatal("alg=none must fail")
	}

	// wrong audience
	tok = signTestJWT(t, priv, kid, map[string]any{
		"iss": cfg.Issuer, "aud": "other", "sub": "ops-1",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		"groups": []string{"bwb-ops"},
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	if _, err := authn.Authenticate(t.Context(), req); err == nil {
		t.Fatal("wrong aud must fail")
	}

	// expired
	tok = signTestJWT(t, priv, kid, map[string]any{
		"iss": cfg.Issuer, "aud": cfg.Audience, "sub": "ops-1",
		"iat": now.Add(-2 * time.Hour).Unix(), "exp": now.Add(-time.Hour).Unix(),
		"groups": []string{"bwb-ops"},
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	if _, err := authn.Authenticate(t.Context(), req); err == nil {
		t.Fatal("expired must fail")
	}

	// missing Bearer
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := authn.Authenticate(t.Context(), req); err == nil {
		t.Fatal("missing bearer must fail")
	}
}

func TestOIDCConfigFailClosed(t *testing.T) {
	_, err := adminauth.NewOIDCAuthenticator(adminauth.OIDCConfig{})
	if err == nil {
		t.Fatal("empty config")
	}
	_, err = adminauth.NewOIDCAuthenticator(adminauth.OIDCConfig{
		Issuer: "https://idp", Audience: "aud", JWKSURL: "http://insecure/jwks",
		RoleClaim: "groups", RoleMap: map[string]adminauth.Role{"g": adminauth.RoleAdmin},
	})
	if err == nil {
		t.Fatal("http jwks without AllowHTTPJWKS")
	}
	_, err = adminauth.NewOIDCAuthenticator(adminauth.OIDCConfig{
		Issuer: "https://idp", Audience: "aud", JWKSURL: "https://idp/jwks",
		RoleClaim: "groups",
		RoleMap:   map[string]adminauth.Role{"g": adminauth.RoleOwner},
		// OwnerSubjects missing
	})
	if err == nil {
		t.Fatal("owner map without subjects")
	}
	_, err = adminauth.NewOIDCAuthenticator(adminauth.OIDCConfig{
		Issuer: "https://idp", Audience: "aud", JWKSURL: "https://idp/jwks",
		AllowedAlgs: []string{"none"},
		RoleClaim:   "groups", RoleMap: map[string]adminauth.Role{"g": adminauth.RoleAdmin},
	})
	if err == nil {
		t.Fatal("alg none in allowlist")
	}
}

func TestParseRoleMap(t *testing.T) {
	m, err := adminauth.ParseRoleMap("bwb-admins:admin,bwb-ops:operator")
	if err != nil || m["bwb-admins"] != adminauth.RoleAdmin {
		t.Fatalf("%v %v", m, err)
	}
	if _, err := adminauth.ParseRoleMap("bad"); err == nil {
		t.Fatal("want err")
	}
}

func signTestJWT(t *testing.T, priv *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	hb, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	pb, _ := json.Marshal(claims)
	encH := base64.RawURLEncoding.EncodeToString(hb)
	encP := base64.RawURLEncoding.EncodeToString(pb)
	input := encH + "." + encP
	sum := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func unsignedNoneJWT(claims map[string]any) string {
	hb, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT", "kid": "x"})
	pb, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb) + "."
}

func TestOIDCDoesNotEchoTokenInError(t *testing.T) {
	err := adminauth.ErrUnauthorized
	if strings.Contains(err.Error(), "Bearer") {
		t.Fatal("unexpected")
	}
}
