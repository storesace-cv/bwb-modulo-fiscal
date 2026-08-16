package fejws_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fejws"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	signer, pub := ephemeralSigner(t)
	payload := []byte(`{"a":1,"b":"x"}`)
	compact, err := fejws.SignCompact(signer, payload, fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := fejws.VerifyCompact(pub, compact)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch")
	}
}

func TestSignDeterministicRS256(t *testing.T) {
	signer, _ := ephemeralSigner(t)
	payload := []byte(`{"k":"v"}`)
	a, err := fejws.SignCompact(signer, payload, fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := fejws.SignCompact(signer, payload, fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("RS256 PKCS1v15 must be deterministic for same key+input")
	}
}

func TestVerifyWrongKeyFails(t *testing.T) {
	s1, _ := ephemeralSigner(t)
	_, pub2 := ephemeralSigner(t)
	compact, err := fejws.SignCompact(s1, []byte(`{}`), fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fejws.VerifyCompact(pub2, compact); !errors.Is(err, fejws.ErrVerifyFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestTamperPayloadHeaderSignature(t *testing.T) {
	signer, pub := ephemeralSigner(t)
	compact, err := fejws.SignCompact(signer, []byte(`{"n":1}`), fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(compact, ".")
	// tamper payload
	badPayload := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"n":2}`)) + "." + parts[2]
	if _, err := fejws.VerifyCompact(pub, badPayload); !errors.Is(err, fejws.ErrVerifyFailed) {
		t.Fatalf("payload tamper: %v", err)
	}
	// tamper header alg via rebuilt header (still RS256 but different bytes) — change typ
	hb, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "X"})
	badHeader := base64.RawURLEncoding.EncodeToString(hb) + "." + parts[1] + "." + parts[2]
	if _, err := fejws.VerifyCompact(pub, badHeader); !errors.Is(err, fejws.ErrVerifyFailed) {
		t.Fatalf("header tamper: %v", err)
	}
	// tamper signature
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sig[0] ^= 0xff
	badSig := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(sig)
	if _, err := fejws.VerifyCompact(pub, badSig); !errors.Is(err, fejws.ErrVerifyFailed) {
		t.Fatalf("sig tamper: %v", err)
	}
}

func TestRejectNoneAndPS256(t *testing.T) {
	_, pub := ephemeralSigner(t)
	for _, alg := range []string{"none", "PS256", "ES256"} {
		hb, _ := json.Marshal(map[string]string{"alg": alg})
		compact := base64.RawURLEncoding.EncodeToString(hb) + "." +
			base64.RawURLEncoding.EncodeToString([]byte(`{}`)) + "." +
			base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3})
		_, err := fejws.VerifyCompact(pub, compact)
		if !errors.Is(err, fejws.ErrAlgorithm) {
			t.Fatalf("alg %s: %v", alg, err)
		}
	}
}

func TestRejectBase64PaddingAndNonCanonical(t *testing.T) {
	signer, pub := ephemeralSigner(t)
	compact, err := fejws.SignCompact(signer, []byte(`{}`), fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(compact, ".")
	padded := parts[0] + "=" + "." + parts[1] + "." + parts[2]
	if _, err := fejws.VerifyCompact(pub, padded); !errors.Is(err, fejws.ErrInvalidCompact) {
		t.Fatalf("padding: %v", err)
	}
}

func TestRejectDuplicateHeaderKeys(t *testing.T) {
	_, pub := ephemeralSigner(t)
	// Hand-craft duplicate alg keys
	raw := []byte(`{"alg":"RS256","alg":"none"}`)
	compact := base64.RawURLEncoding.EncodeToString(raw) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(`{}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte{1})
	_, err := fejws.VerifyCompact(pub, compact)
	if !errors.Is(err, fejws.ErrDuplicateHeaderKey) {
		t.Fatalf("got %v", err)
	}
}

func TestRejectUnknownCrit(t *testing.T) {
	_, pub := ephemeralSigner(t)
	raw := []byte(`{"alg":"RS256","crit":["bork"]}`)
	compact := base64.RawURLEncoding.EncodeToString(raw) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(`{}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte{1})
	_, err := fejws.VerifyCompact(pub, compact)
	if !errors.Is(err, fejws.ErrCritical) {
		t.Fatalf("got %v", err)
	}
}

func TestRejectSegmentCountAndEmptySig(t *testing.T) {
	_, pub := ephemeralSigner(t)
	if _, err := fejws.VerifyCompact(pub, "a.b"); !errors.Is(err, fejws.ErrInvalidCompact) {
		t.Fatalf("got %v", err)
	}
	if _, err := fejws.VerifyCompact(pub, "a.b.c.d"); !errors.Is(err, fejws.ErrInvalidCompact) {
		t.Fatalf("got %v", err)
	}
	hb := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	pb := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	if _, err := fejws.VerifyCompact(pub, hb+"."+pb+"."); !errors.Is(err, fejws.ErrInvalidCompact) && !errors.Is(err, fejws.ErrEmptySignature) {
		t.Fatalf("got %v", err)
	}
}

func TestClosedOpaqueSignerFails(t *testing.T) {
	p, err := agttestkit.OpenEphemeralProducerProvider(agttestkit.MinRSABits)
	if err != nil {
		t.Fatal(err)
	}
	ref := p.List()[0].Ref
	s, err := p.Signer(ref)
	if err != nil {
		t.Fatal(err)
	}
	_ = p.Close()
	_, err = fejws.SignCompact(s, []byte(`{}`), fejws.ProtectedHeader{})
	if err == nil {
		t.Fatal("expected closed signer failure")
	}
	if !errors.Is(err, fejws.ErrSigner) && !errors.Is(err, agttestkit.ErrProviderClosed) {
		// SignCompact wraps as ErrSigner
		if !errors.Is(err, fejws.ErrSigner) {
			t.Fatalf("got %v", err)
		}
	}
}

func TestErrorsDoNotEmbedPayload(t *testing.T) {
	_, pub := ephemeralSigner(t)
	secret := "SECRET_PAYLOAD_VALUE_XYZ"
	hb := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	pb := base64.RawURLEncoding.EncodeToString([]byte(secret))
	_, err := fejws.VerifyCompact(pub, hb+"."+pb+".abc")
	if err == nil {
		t.Fatal("expected err")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked payload: %v", err)
	}
}

func TestExplicitTypAllowedWithoutDeclaringRule(t *testing.T) {
	signer, pub := ephemeralSigner(t)
	compact, err := fejws.SignCompact(signer, []byte(`{}`), fejws.ProtectedHeader{Typ: "JOSE"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(compact, ".")
	hb, _ := base64.RawURLEncoding.DecodeString(parts[0])
	if !strings.Contains(string(hb), `"typ":"JOSE"`) {
		t.Fatalf("header %s", hb)
	}
	if _, err := fejws.VerifyCompact(pub, compact); err != nil {
		t.Fatal(err)
	}
}

func ephemeralSigner(t *testing.T) (crypto.Signer, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return priv, &priv.PublicKey
}

// fixedDigestSigner is only for illustrating Sign path with crypto.Signer; not used for FE claim rules.
type fixedDigestSigner struct {
	pub  *rsa.PublicKey
	priv *rsa.PrivateKey
}

func (f *fixedDigestSigner) Public() crypto.PublicKey { return f.pub }
func (f *fixedDigestSigner) Sign(r io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return f.priv.Sign(r, digest, opts)
}

func TestFixedSignerStillHashesInput(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s := &fixedDigestSigner{pub: &priv.PublicKey, priv: priv}
	payload := []byte(`{"z":true}`)
	c1, err := fejws.SignCompact(s, payload, fejws.ProtectedHeader{})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("not-the-signing-input"))
	_ = sum
	if _, err := fejws.VerifyCompact(&priv.PublicKey, c1); err != nil {
		t.Fatal(err)
	}
}
