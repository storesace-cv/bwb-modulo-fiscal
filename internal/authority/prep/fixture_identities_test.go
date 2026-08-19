package prep_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
)

func TestFixtureIdentityCatalogEmpty(t *testing.T) {
	ok, refs, err := prep.FixtureIdentityCatalog("")
	if err != nil || ok || len(refs) != 0 {
		t.Fatalf("ok=%v refs=%d err=%v", ok, len(refs), err)
	}
}

func TestFixtureIdentityCatalogSynthetic(t *testing.T) {
	path, cleanup, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	ok, refs, err := prep.FixtureIdentityCatalog(path)
	if err != nil || !ok || len(refs) != 3 {
		t.Fatalf("ok=%v len=%d err=%v", ok, len(refs), err)
	}
	for _, r := range refs {
		if r.Ref == "" || r.Algorithm == "" || r.RSABits < agttestkit.MinRSABits {
			t.Fatalf("%+v", r)
		}
	}
}

func TestFixtureHubViewFixtureOnly(t *testing.T) {
	v := prep.FixtureHubView()
	if !v.TransportAllowed || v.ExternalVerified {
		t.Fatalf("%+v", v)
	}
}
