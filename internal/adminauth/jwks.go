package adminauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksDoc struct {
	Keys []jwkKey `json:"keys"`
}

type cachedKey struct {
	kid string
	alg string // from JWK if present; may be empty
	rsa *rsa.PublicKey
	ec  *ecdsa.PublicKey
}

type jwksCache struct {
	mu         sync.Mutex
	url        string
	client     *http.Client
	maxBytes   int
	ttl        time.Duration
	minRefresh time.Duration
	fetchedAt  time.Time
	lastForced time.Time
	keys       map[string]cachedKey
}

func newJWKSCache(url string, client *http.Client, maxBytes int, ttl, minRefresh time.Duration) *jwksCache {
	return &jwksCache{
		url:        url,
		client:     client,
		maxBytes:   maxBytes,
		ttl:        ttl,
		minRefresh: minRefresh,
		keys:       make(map[string]cachedKey),
	}
}

func (c *jwksCache) get(ctx context.Context, kid string) (cachedKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UTC()
	stale := c.fetchedAt.IsZero() || now.Sub(c.fetchedAt) > c.ttl
	if key, ok := c.keys[kid]; ok && !stale {
		return key, nil
	}
	if stale {
		if err := c.refreshLocked(ctx); err != nil {
			return cachedKey{}, err
		}
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
	}
	// Unknown kid: forced refresh with rate limit (rotation).
	if now.Sub(c.lastForced) >= c.minRefresh {
		if err := c.refreshLocked(ctx); err != nil {
			return cachedKey{}, err
		}
		c.lastForced = now
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
	}
	return cachedKey{}, fmt.Errorf("%w: kid JWKS desconhecido", ErrUnauthorized)
}

func (c *jwksCache) refreshLocked(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("%w: pedido JWKS", ErrUnauthorized)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: fetch JWKS", ErrUnauthorized)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w: JWKS status", ErrUnauthorized)
	}
	limited := io.LimitReader(resp.Body, int64(c.maxBytes)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("%w: ler JWKS", ErrUnauthorized)
	}
	if len(raw) > c.maxBytes {
		return fmt.Errorf("%w: JWKS demasiado grande", ErrUnauthorized)
	}
	var doc jwksDoc
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc.Keys) == 0 {
		return fmt.Errorf("%w: JWKS inválido", ErrUnauthorized)
	}
	next := make(map[string]cachedKey, len(doc.Keys))
	for _, k := range doc.Keys {
		ck, err := parseJWK(k)
		if err != nil {
			continue // skip unusable keys; fail later if needed kid missing
		}
		if ck.kid == "" {
			continue
		}
		next[ck.kid] = ck
	}
	if len(next) == 0 {
		return fmt.Errorf("%w: JWKS sem chaves usáveis", ErrUnauthorized)
	}
	c.keys = next
	c.fetchedAt = time.Now().UTC()
	return nil
}

func parseJWK(k jwkKey) (cachedKey, error) {
	kid := strings.TrimSpace(k.Kid)
	alg := strings.TrimSpace(k.Alg)
	use := strings.TrimSpace(k.Use)
	if use != "" && use != "sig" {
		return cachedKey{}, fmt.Errorf("use")
	}
	switch strings.ToUpper(k.Kty) {
	case "RSA":
		pub, err := rsaPublicFromJWK(k.N, k.E)
		if err != nil {
			return cachedKey{}, err
		}
		if pub.N.BitLen() < defaultMinRSABits {
			return cachedKey{}, fmt.Errorf("rsa bits")
		}
		return cachedKey{kid: kid, alg: alg, rsa: pub}, nil
	case "EC":
		pub, err := ecdsaPublicFromJWK(k.Crv, k.X, k.Y)
		if err != nil {
			return cachedKey{}, err
		}
		return cachedKey{kid: kid, alg: alg, ec: pub}, nil
	default:
		return cachedKey{}, fmt.Errorf("kty")
	}
}

func rsaPublicFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	if len(nb) == 0 || len(eb) == 0 {
		return nil, fmt.Errorf("rsa params")
	}
	n := new(big.Int).SetBytes(nb)
	e := 0
	for _, b := range eb {
		e = e<<8 + int(b)
	}
	if e < 3 {
		return nil, fmt.Errorf("rsa e")
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

func ecdsaPublicFromJWK(crv, xB64, yB64 string) (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	default:
		return nil, fmt.Errorf("crv")
	}
	xb, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, err
	}
	yb, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, err
	}
	x := new(big.Int).SetBytes(xb)
	y := new(big.Int).SetBytes(yb)
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("point")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}
