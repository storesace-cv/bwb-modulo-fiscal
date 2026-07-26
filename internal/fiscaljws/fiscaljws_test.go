package fiscaljws_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaljws"
)

func TestEphemeralSignVerifyRoundTrip(t *testing.T) {
	s, err := fiscaljws.NewEphemeral(fiscaljws.DefaultRSABits)
	if err != nil {
		t.Fatal(err)
	}
	fp, err := s.PublicFingerprintSHA256()
	if err != nil || len(fp) != 64 {
		t.Fatalf("fingerprint=%q err=%v", fp, err)
	}
	compact, err := s.SignEnvelope("sub-1", "doc-1", time.Unix(1_700_000_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	env, err := fiscaljws.ParseEnvelope(s.PublicKey(), compact)
	if err != nil {
		t.Fatal(err)
	}
	if env.SubmissionID != "sub-1" || env.DocumentID != "doc-1" || env.Certified {
		t.Fatalf("%+v", env)
	}
}

func TestTamperedJWSRejected(t *testing.T) {
	s, err := fiscaljws.NewEphemeral(fiscaljws.DefaultRSABits)
	if err != nil {
		t.Fatal(err)
	}
	compact, err := s.SignEnvelope("sub-1", "doc-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		t.Fatalf("parts=%d", len(parts))
	}
	pay, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	pay[0] ^= 0xff
	bad := parts[0] + "." + base64.RawURLEncoding.EncodeToString(pay) + "." + parts[2]
	_, err = fiscaljws.VerifyCompact(s.PublicKey(), bad)
	if !errors.Is(err, fiscaljws.ErrInvalidJWS) {
		t.Fatalf("want ErrInvalidJWS, got %v", err)
	}
}

func TestWrongKeyRejected(t *testing.T) {
	a, err := fiscaljws.NewEphemeral(fiscaljws.DefaultRSABits)
	if err != nil {
		t.Fatal(err)
	}
	b, err := fiscaljws.NewEphemeral(fiscaljws.DefaultRSABits)
	if err != nil {
		t.Fatal(err)
	}
	compact, err := a.SignEnvelope("sub-1", "doc-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_, err = fiscaljws.ParseEnvelope(b.PublicKey(), compact)
	if !errors.Is(err, fiscaljws.ErrInvalidJWS) {
		t.Fatalf("want ErrInvalidJWS, got %v", err)
	}
}
