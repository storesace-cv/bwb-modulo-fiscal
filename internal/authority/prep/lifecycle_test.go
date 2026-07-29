package prep_test

import (
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestBuildMaterialLifecycleAndPatch(t *testing.T) {
	p := adminregistry.AuthorityProfile{
		ID: "p1", Environment: adminregistry.EnvHomologation,
		CertificateRef: "agt-cert", ProducerKeyRef: "agt-key",
		OfflineValidated: true, SecretsReady: true,
		Status: adminregistry.AuthorityStatusValidated,
	}
	exp := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	lookup := func(ref secretstore.Ref) (secretstore.Metadata, error) {
		switch ref.Name {
		case "agt-cert":
			return secretstore.Metadata{
				Ref: ref, Status: secretstore.StatusPresent, Version: 2,
				Fingerprint: "deadbeef", ExpiresAt: &exp,
			}, nil
		case "agt-key":
			return secretstore.Metadata{Ref: ref, Status: secretstore.StatusPresent, Version: 1, Fingerprint: "aabb"}, nil
		default:
			return secretstore.Metadata{Ref: ref, Status: secretstore.StatusAbsent}, nil
		}
	}
	lc, err := prep.BuildMaterialLifecycle(p, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if lc.ExternalVerified || lc.ProfileID != "p1" || len(lc.Refs) != 3 {
		t.Fatalf("%+v", lc)
	}
	var cert prep.MaterialRefView
	for _, r := range lc.Refs {
		if r.Role == prep.RoleCert {
			cert = r
		}
	}
	if cert.Status != secretstore.StatusPresent || cert.Version != 2 || cert.Fingerprint != "deadbeef" {
		t.Fatalf("cert view: %+v", cert)
	}

	patch := prep.ProfilePatchAfterMaterialChange(p, secretstore.KindCertificate, "agt-cert", secretstore.Metadata{
		Status: secretstore.StatusPresent, Fingerprint: "cafebabe", ExpiresAt: &exp, Version: 3,
	}, false)
	if patch.OfflineValidated == nil || *patch.OfflineValidated {
		t.Fatal("rotate must invalidate offline_validated")
	}
	if patch.SecretsReady == nil || *patch.SecretsReady {
		t.Fatal("material change must clear secrets_ready until re-validated")
	}
	if patch.FingerprintSanitized != "sha256:cafebabe" {
		t.Fatalf("fp=%q", patch.FingerprintSanitized)
	}

	rev := prep.ProfilePatchAfterMaterialChange(p, secretstore.KindCertificate, "agt-cert", secretstore.Metadata{}, true)
	if rev.SecretsReady == nil || *rev.SecretsReady || rev.OfflineValidated == nil || *rev.OfflineValidated {
		t.Fatalf("revoke patch: %+v", rev)
	}
}
