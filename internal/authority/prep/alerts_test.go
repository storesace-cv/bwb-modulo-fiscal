package prep_test

import (
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
)

func TestBuildReadinessAlerts(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	expSoon := now.Add(10 * 24 * time.Hour)
	p := adminregistry.AuthorityProfile{
		Status:    adminregistry.AuthorityStatusDraft,
		ExpiresAt: &expSoon,
	}
	alerts := prep.BuildReadinessAlerts(p, now)
	codes := map[string]bool{}
	for _, a := range alerts {
		codes[a.Code] = true
		if a.Message == "" || containsSecretLike(a.Message) {
			t.Fatalf("bad alert: %+v", a)
		}
	}
	for _, want := range []string{
		"config_not_ready", "secrets_not_ready", "offline_not_validated",
		"external_verified_false", "refs_incomplete", "certificate_expiring_soon",
	} {
		if !codes[want] {
			t.Fatalf("missing %s in %#v", want, codes)
		}
	}

	p2 := adminregistry.AuthorityProfile{
		ConfigReady: true, SecretsReady: true, OfflineValidated: true,
		CertificateRef: "c", ProducerKeyRef: "k",
		Status: adminregistry.AuthorityStatusActive,
	}
	alerts2 := prep.BuildReadinessAlerts(p2, now)
	for _, a := range alerts2 {
		if a.Severity == prep.SeverityBlocking {
			t.Fatalf("unexpected blocking: %+v", a)
		}
	}
}

func containsSecretLike(s string) bool {
	low := s
	for _, bad := range []string{"BEGIN ", "password", "private", "-----"} {
		if len(low) >= len(bad) {
			for i := 0; i+len(bad) <= len(low); i++ {
				if equalFoldASCII(low[i:i+len(bad)], bad) {
					return true
				}
			}
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
