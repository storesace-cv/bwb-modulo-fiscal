package feprofile_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

func TestSoftwareInfoEligible(t *testing.T) {
	p, err := agttestkit.OpenEphemeralProducerProvider(agttestkit.MinRSABits)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	ref := p.List()[0].Ref
	claims := feprofile.SoftwareInfoClaims{
		ProductID:                "SYNTHETIC_PRODUCT",
		ProductVersion:           "0.0.1-test",
		SoftwareValidationNumber: "SYNTHETIC_SVN_PLACEHOLDER",
	}
	compact, err := feprofile.SignSoftwareInfo(p, ref, claims)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(compact, ".")
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(hb), `"typ"`) {
		t.Fatalf("typ must not be defaulted: %s", hb)
	}
	if !strings.Contains(string(hb), `"alg":"RS256"`) {
		t.Fatalf("header %s", hb)
	}
	payload, err := feprofile.VerifyWithSignerPublic(p, ref, compact)
	if err != nil {
		t.Fatal(err)
	}
	var got feprofile.SoftwareInfoClaims
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got != claims {
		t.Fatalf("%+v", got)
	}
}

func TestSoftwareRejectsTaxpayerRole(t *testing.T) {
	path, cleanup, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	_, err = feprofile.SignSoftwareInfo(p, p.List()[0].Ref, feprofile.SoftwareInfoClaims{
		ProductID: "X", ProductVersion: "1", SoftwareValidationNumber: "SYNTHETIC_SVN_PLACEHOLDER",
	})
	if !errors.Is(err, agttestkit.ErrRoleMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestObterEstadoEligibleAndBinding(t *testing.T) {
	path, cleanup, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	ref := p.List()[0].Ref
	nif := "9100000000"
	compact, err := feprofile.SignObterEstadoRequest(p, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: nif,
		RequestID:             "SYNTHETIC_REQ_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := feprofile.VerifyWithSignerPublic(p, ref, compact); err != nil {
		t.Fatal(err)
	}
	_, err = feprofile.SignObterEstadoRequest(p, ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000999",
		RequestID:             "SYNTHETIC_REQ_1",
	})
	if !errors.Is(err, agttestkit.ErrBindingMismatch) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "9100000999") || strings.Contains(err.Error(), nif) {
		t.Fatalf("nif leaked: %v", err)
	}
}

func TestConsultarFacturaEligible(t *testing.T) {
	path, cleanup, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	p, err := agttestkit.OpenWorkbookProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	ref := p.List()[0].Ref
	compact, err := feprofile.SignConsultarFacturaRequest(p, ref, feprofile.ConsultarFacturaRequestClaims{
		TaxRegistrationNumber: "9100000000",
		DocumentNo:            "FT SYNTHETIC/1",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := feprofile.VerifyWithSignerPublic(p, ref, compact)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "BEGIN") {
		t.Fatal("pem in payload")
	}
}

func TestTaxpayerCannotUseProducerEphemeralForRequest(t *testing.T) {
	p, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	_, err = feprofile.SignObterEstadoRequest(p, p.List()[0].Ref, feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "1", RequestID: "x",
	})
	if !errors.Is(err, agttestkit.ErrRoleMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestBlockedProfiles(t *testing.T) {
	p, err := agttestkit.OpenEphemeralProducerProvider(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	ref := p.List()[0].Ref
	raw := json.RawMessage(`{}`)
	checks := []struct {
		name string
		fn   func() (string, error)
	}{
		{"doc", func() (string, error) { return feprofile.SignRegistarDocumentBlocked(p, ref, raw) }},
		{"regreq", func() (string, error) { return feprofile.SignRegistarRequestSignatureBlocked(p, ref, raw) }},
		{"solicitar", func() (string, error) { return feprofile.SignSolicitarSerieRequestBlocked(p, ref, raw) }},
		{"listar", func() (string, error) { return feprofile.SignListarSeriesRequestBlocked(p, ref, raw) }},
		{"validar", func() (string, error) { return feprofile.SignValidarDocumentoRequestBlocked(p, ref, raw) }},
		{"listarfat", func() (string, error) { return feprofile.SignListarFacturasRequestBlocked(p, ref, raw) }},
	}
	for _, c := range checks {
		_, err := c.fn()
		if !errors.Is(err, feprofile.ErrProfileBlocked) {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
}
