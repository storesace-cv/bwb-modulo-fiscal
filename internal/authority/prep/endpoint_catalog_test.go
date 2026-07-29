package prep_test

import (
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fepath"
)

func TestEndpointCatalogHomologationFailClosed(t *testing.T) {
	rows, err := prep.EndpointCatalog("homologation")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 5 {
		t.Fatalf("want matrix ops, got %d", len(rows))
	}
	byOp := map[string]prep.EndpointCatalogEntry{}
	for _, r := range rows {
		byOp[r.Operation] = r
		if r.Environment != "homologation" {
			t.Fatalf("%+v", r)
		}
		if strings.Contains(strings.ToLower(r.DeclaredURL), "password") {
			t.Fatal("secret-like url")
		}
	}
	reg := byOp["registarFactura"]
	if reg.PathStatus != prep.PathStatusAligned || reg.DeclaredURL == "" {
		t.Fatalf("%+v", reg)
	}
	if !strings.HasPrefix(reg.DeclaredURL, fepath.HostHML) {
		t.Fatalf("host %s", reg.DeclaredURL)
	}
	sol := byOp["solicitarSerie"]
	if sol.PathStatus != prep.PathStatusConflictOpen || sol.DeclaredURL != "" || sol.ConflictID != "C-FE-001" {
		t.Fatalf("%+v", sol)
	}
	if _, err := prep.EndpointCatalog("development"); err == nil {
		t.Fatal("expected env validation")
	}
}

func TestJWSProfileScaffoldNoInventedClaims(t *testing.T) {
	s := prep.JWSProfileScaffoldDefault()
	if s.ExternalVerified {
		t.Fatal("external_verified must stay false")
	}
	if !s.InventedClaimsForbidden || s.ClaimsStatus != prep.ClaimsStatusPendingExternal {
		t.Fatalf("%+v", s)
	}
	if s.AlgorithmDeclared != "RS256" {
		t.Fatalf("alg=%s", s.AlgorithmDeclared)
	}
}
