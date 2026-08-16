package feprofile_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fejws"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

func TestMarshalSoftwareInfoDeterministic(t *testing.T) {
	claims := feprofile.SoftwareInfoClaims{
		ProductID:                "SYNTHETIC_PRODUCT",
		ProductVersion:           "0.0.1-test",
		SoftwareValidationNumber: "SYNTHETIC_SVN_PLACEHOLDER",
	}
	a, err := feprofile.MarshalSoftwareInfoPayload(claims)
	if err != nil {
		t.Fatal(err)
	}
	b, err := feprofile.MarshalSoftwareInfoPayload(claims)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("non-deterministic: %q vs %q", a, b)
	}
	want := `{"productId":"SYNTHETIC_PRODUCT","productVersion":"0.0.1-test","softwareValidationNumber":"SYNTHETIC_SVN_PLACEHOLDER"}`
	if string(a) != want {
		t.Fatalf("got %s", a)
	}
}

func TestMarshalRequestPayloadsExact(t *testing.T) {
	oe, err := feprofile.MarshalObterEstadoRequestPayload(feprofile.ObterEstadoRequestClaims{
		TaxRegistrationNumber: "9100000000",
		RequestID:             "SYNTHETIC_REQ_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(oe) != `{"taxRegistrationNumber":"9100000000","requestID":"SYNTHETIC_REQ_1"}` {
		t.Fatalf("%s", oe)
	}
	cf, err := feprofile.MarshalConsultarFacturaRequestPayload(feprofile.ConsultarFacturaRequestClaims{
		TaxRegistrationNumber: "9100000000",
		DocumentNo:            "FT SYNTHETIC/1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(cf) != `{"taxRegistrationNumber":"9100000000","documentNo":"FT SYNTHETIC/1"}` {
		t.Fatalf("%s", cf)
	}
}

func TestOperationalSignBlockedByTypConflict(t *testing.T) {
	checks := []struct {
		id feprofile.ProfileID
		fn func() (string, error)
	}{
		{feprofile.ProfileSoftwareInfo, func() (string, error) {
			return feprofile.SignSoftwareInfo(nil, "", feprofile.SoftwareInfoClaims{
				ProductID: "X", ProductVersion: "1", SoftwareValidationNumber: "SYNTHETIC_SVN_PLACEHOLDER",
			})
		}},
		{feprofile.ProfileObterEstadoRequest, func() (string, error) {
			return feprofile.SignObterEstadoRequest(nil, "", feprofile.ObterEstadoRequestClaims{
				TaxRegistrationNumber: "9100000000", RequestID: "r",
			})
		}},
		{feprofile.ProfileConsultarFacturaRequest, func() (string, error) {
			return feprofile.SignConsultarFacturaRequest(nil, "", feprofile.ConsultarFacturaRequestClaims{
				TaxRegistrationNumber: "9100000000", DocumentNo: "FT 1",
			})
		}},
	}
	for _, c := range checks {
		_, err := c.fn()
		if !errors.Is(err, feprofile.ErrProfileBlocked) {
			t.Fatalf("%s: want ErrProfileBlocked, got %v", c.id, err)
		}
		msg := err.Error()
		if !strings.Contains(msg, string(c.id)) || !strings.Contains(msg, feprofile.ConflictTyp) {
			t.Fatalf("%s: error must identify profile and conflict: %v", c.id, err)
		}
		if strings.Contains(msg, "9100000000") || strings.Contains(msg, "productId") || strings.Contains(msg, "SYNTHETIC") {
			t.Fatalf("%s: error leaked payload/NIF: %v", c.id, err)
		}
		if strings.Contains(strings.ToLower(msg), "agt valid") || strings.Contains(msg, "aceite AGT") {
			t.Fatalf("%s: must not claim AGT validity: %v", c.id, err)
		}
	}
}

func TestGenericEngineMaySignWithExplicitTyp(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"k":"v"}`)
	compact, err := fejws.SignCompact(key, payload, fejws.ProtectedHeader{Alg: fejws.Algorithm, Typ: "JOSE"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := fejws.VerifyCompact(&key.PublicKey, compact)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("%s", got)
	}
}

func TestGenericEngineMaySignWithoutTypAsTechnicalPrimitive(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"k":"v"}`)
	compact, err := fejws.SignCompact(key, payload, fejws.ProtectedHeader{Alg: fejws.Algorithm})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fejws.VerifyCompact(&key.PublicKey, compact); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(compact, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"typ"`) {
		t.Fatalf("typ must remain absent when omitted: %s", raw)
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

func TestBindingStillPrivateOnMismatch(t *testing.T) {
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
	err = p.ValidateTaxpayerBinding(ref, "9100000999")
	if !errors.Is(err, agttestkit.ErrBindingMismatch) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "9100000999") || strings.Contains(err.Error(), "9100000000") {
		t.Fatalf("nif leaked: %v", err)
	}
}
