package femock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fejws"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

// SignSoftwareMock signs software claims with typ=BWB-MOCK (test wire only).
func SignSoftwareMock(provider agttestkit.IdentityProvider, ref string, claims feprofile.SoftwareInfoClaims) (string, error) {
	if err := provider.RequireRole(ref, agttestkit.RoleProducerEphemeral); err != nil {
		return "", err
	}
	payload, err := feprofile.MarshalSoftwareInfoPayload(claims)
	if err != nil {
		return "", err
	}
	return signMock(provider, ref, payload)
}

// SignObterEstadoMock signs obterEstado claims with typ=BWB-MOCK.
func SignObterEstadoMock(provider agttestkit.IdentityProvider, ref string, claims feprofile.ObterEstadoRequestClaims) (string, error) {
	if err := provider.RequireRole(ref, agttestkit.RoleTaxpayerTest); err != nil {
		return "", err
	}
	if err := provider.ValidateTaxpayerBinding(ref, claims.TaxRegistrationNumber); err != nil {
		return "", err
	}
	payload, err := feprofile.MarshalObterEstadoRequestPayload(claims)
	if err != nil {
		return "", err
	}
	return signMock(provider, ref, payload)
}

// SignConsultarFacturaMock signs consultarFactura claims with typ=BWB-MOCK.
func SignConsultarFacturaMock(provider agttestkit.IdentityProvider, ref string, claims feprofile.ConsultarFacturaRequestClaims) (string, error) {
	if err := provider.RequireRole(ref, agttestkit.RoleTaxpayerTest); err != nil {
		return "", err
	}
	if err := provider.ValidateTaxpayerBinding(ref, claims.TaxRegistrationNumber); err != nil {
		return "", err
	}
	payload, err := feprofile.MarshalConsultarFacturaRequestPayload(claims)
	if err != nil {
		return "", err
	}
	return signMock(provider, ref, payload)
}

func signMock(provider agttestkit.IdentityProvider, ref string, payload []byte) (string, error) {
	signer, err := provider.Signer(ref)
	if err != nil {
		return "", err
	}
	return fejws.SignCompact(signer, payload, fejws.ProtectedHeader{Alg: fejws.Algorithm, Typ: TypMock})
}

func verifyMockJWS(provider agttestkit.IdentityProvider, ref, compact string) (payload []byte, hdr fejws.ProtectedHeader, err error) {
	hdr, err = fejws.InspectCompactHeader(compact)
	if err != nil {
		return nil, hdr, fmt.Errorf("%s", CodeJWSInvalid)
	}
	if hdr.Typ != TypMock {
		return nil, hdr, fmt.Errorf("%s", CodeJWSTypRejected)
	}
	signer, err := provider.Signer(ref)
	if err != nil {
		return nil, hdr, fmt.Errorf("%s", CodeJWSInvalid)
	}
	pub, err := fejws.PublicRSA(signer)
	if err != nil {
		return nil, hdr, fmt.Errorf("%s", CodeJWSInvalid)
	}
	payload, err = fejws.VerifyCompact(pub, compact)
	if err != nil {
		return nil, hdr, fmt.Errorf("%s", CodeJWSInvalid)
	}
	return payload, hdr, nil
}

func decodeStrictJSON(raw []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Reject trailing junk.
	if dec.More() {
		return errors.New("trailing json")
	}
	return nil
}
