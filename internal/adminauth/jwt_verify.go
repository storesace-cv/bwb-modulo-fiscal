package adminauth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
)

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

func verifyAndParseClaims(token string, key cachedKey, allowed map[string]struct{}, now time.Time, issuer, audience string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: jwt partes", ErrUnauthorized)
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: jwt header", ErrUnauthorized)
	}
	var hdr jwtHeader
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return nil, fmt.Errorf("%w: jwt header json", ErrUnauthorized)
	}
	alg := strings.TrimSpace(hdr.Alg)
	if alg == "" || strings.EqualFold(alg, "none") {
		return nil, fmt.Errorf("%w: alg proibido", ErrUnauthorized)
	}
	if _, ok := allowed[alg]; !ok {
		return nil, fmt.Errorf("%w: alg não permitido", ErrUnauthorized)
	}
	// Key confusion: JWK alg (if set) must match header alg; key type must match alg family.
	if key.alg != "" && !strings.EqualFold(key.alg, alg) {
		return nil, fmt.Errorf("%w: alg ≠ jwk", ErrUnauthorized)
	}
	switch alg {
	case "RS256":
		if key.rsa == nil {
			return nil, fmt.Errorf("%w: chave incompatível RS256", ErrUnauthorized)
		}
	case "ES256":
		if key.ec == nil {
			return nil, fmt.Errorf("%w: chave incompatível ES256", ErrUnauthorized)
		}
	default:
		return nil, fmt.Errorf("%w: alg", ErrUnauthorized)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: jwt sig", ErrUnauthorized)
	}
	signingInput := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signingInput))
	switch alg {
	case "RS256":
		if err := rsa.VerifyPKCS1v15(key.rsa, crypto.SHA256, sum[:], sig); err != nil {
			return nil, fmt.Errorf("%w: assinatura", ErrUnauthorized)
		}
	case "ES256":
		if len(sig) != 64 {
			return nil, fmt.Errorf("%w: assinatura es", ErrUnauthorized)
		}
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		if !ecdsa.Verify(key.ec, sum[:], r, s) {
			return nil, fmt.Errorf("%w: assinatura", ErrUnauthorized)
		}
	}

	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: jwt payload", ErrUnauthorized)
	}
	var claims map[string]any
	dec := json.NewDecoder(strings.NewReader(string(pb)))
	dec.UseNumber()
	if err := dec.Decode(&claims); err != nil {
		return nil, fmt.Errorf("%w: jwt claims", ErrUnauthorized)
	}
	if err := validateRegisteredClaims(claims, now, issuer, audience); err != nil {
		return nil, err
	}
	return claims, nil
}

func validateRegisteredClaims(claims map[string]any, now time.Time, issuer, audience string) error {
	iss, _ := claims["iss"].(string)
	if iss != issuer {
		return fmt.Errorf("%w: iss", ErrUnauthorized)
	}
	sub, _ := claims["sub"].(string)
	if strings.TrimSpace(sub) == "" {
		return fmt.Errorf("%w: sub", ErrUnauthorized)
	}
	if !audienceMatches(claims["aud"], audience) {
		return fmt.Errorf("%w: aud", ErrUnauthorized)
	}
	exp, ok := claimTime(claims["exp"])
	if !ok {
		return fmt.Errorf("%w: exp", ErrUnauthorized)
	}
	if !now.Before(exp) {
		return fmt.Errorf("%w: expirado", ErrUnauthorized)
	}
	if nbf, ok := claimTime(claims["nbf"]); ok {
		if now.Before(nbf) {
			return fmt.Errorf("%w: nbf", ErrUnauthorized)
		}
	}
	iat, ok := claimTime(claims["iat"])
	if !ok {
		return fmt.Errorf("%w: iat", ErrUnauthorized)
	}
	if iat.After(now.Add(60 * time.Second)) {
		return fmt.Errorf("%w: iat futuro", ErrUnauthorized)
	}
	return nil
}

func audienceMatches(raw any, want string) bool {
	switch v := raw.(type) {
	case string:
		return v == want
	case []any:
		for _, item := range v {
			s, ok := item.(string)
			if ok && s == want {
				return true
			}
		}
	}
	return false
}

func claimTime(raw any) (time.Time, bool) {
	switch v := raw.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil || f < 0 || f > math.MaxInt64 {
			return time.Time{}, false
		}
		return time.Unix(int64(f), 0).UTC(), true
	case float64:
		if v < 0 || v > math.MaxInt64 {
			return time.Time{}, false
		}
		return time.Unix(int64(v), 0).UTC(), true
	default:
		return time.Time{}, false
	}
}
