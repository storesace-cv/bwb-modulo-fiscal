// Package feprofile builds FE JWS payloads only for profiles marked eligible
// in the RM-FEFIX-003 matrix. Blocked profiles return ErrProfileBlocked.
//
// Sources: AO-FE-SNAP-HML-2026-07-25-* (pending_validation). ≠ SAF-T (C-SIGN-001).
package feprofile

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fejws"
)

// Signature kinds (documentation / guards — not wire field names alone).
const (
	KindSoftware = "jwsSoftwareSignature"
	KindDocument = "jwsDocumentSignature"
	KindRequest  = "jwsSignature"
)

var (
	ErrProfileBlocked = errors.New("feprofile: profile blocked_conflict")
	ErrValidation     = errors.New("feprofile: validation")
)

// ProfileID identifies a matrix row.
type ProfileID string

const (
	ProfileSoftwareInfo             ProfileID = "software_info"
	ProfileObterEstadoRequest       ProfileID = "obter_estado_request"
	ProfileConsultarFacturaRequest  ProfileID = "consultar_factura_request"
	ProfileRegistarDocument         ProfileID = "registar_document"
	ProfileRegistarRequestSignature ProfileID = "registar_request_jwsSignature"
	ProfileSolicitarSerieRequest    ProfileID = "solicitar_serie_request"
	ProfileListarSeriesRequest      ProfileID = "listar_series_request"
	ProfileValidarDocumentoRequest  ProfileID = "validar_documento_request"
	ProfileListarFacturasRequest    ProfileID = "listar_facturas_request"
)

// SoftwareInfoClaims is the eligible jwsSoftwareSignature payload
// (AO-FE-SNAP-HML-2026-07-25-ESTRUTURA / REGISTAR samples; pending_validation).
// Protected header: alg=RS256 only — typ omitted (C-FE-JWS-TYP-001).
type SoftwareInfoClaims struct {
	ProductID                string `json:"productId"`
	ProductVersion           string `json:"productVersion"`
	SoftwareValidationNumber string `json:"softwareValidationNumber"`
}

// ObterEstadoRequestClaims — table and Payload assinatura agree
// (AO-FE-SNAP-HML-2026-07-25-CONSULTAR; pending_validation).
type ObterEstadoRequestClaims struct {
	TaxRegistrationNumber string `json:"taxRegistrationNumber"`
	RequestID             string `json:"requestID"`
}

// ConsultarFacturaRequestClaims — table and Payload assinatura agree
// (AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA; pending_validation).
type ConsultarFacturaRequestClaims struct {
	TaxRegistrationNumber string `json:"taxRegistrationNumber"`
	DocumentNo            string `json:"documentNo"`
}

// SignSoftwareInfo signs eligible software claims with a producer identity.
// softwareValidationNumber must be a synthetic placeholder in tests — never a real certificate id.
func SignSoftwareInfo(provider agttestkit.IdentityProvider, ref string, claims SoftwareInfoClaims) (string, error) {
	if err := provider.RequireRole(ref, agttestkit.RoleProducerEphemeral); err != nil {
		return "", err
	}
	if claims.ProductID == "" || claims.ProductVersion == "" || claims.SoftwareValidationNumber == "" {
		return "", ErrValidation
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signer, err := provider.Signer(ref)
	if err != nil {
		return "", err
	}
	return fejws.SignCompact(signer, payload, fejws.ProtectedHeader{Alg: fejws.Algorithm})
}

// SignObterEstadoRequest signs eligible request claims with a taxpayer identity.
func SignObterEstadoRequest(provider agttestkit.IdentityProvider, ref string, claims ObterEstadoRequestClaims) (string, error) {
	if err := provider.RequireRole(ref, agttestkit.RoleTaxpayerTest); err != nil {
		return "", err
	}
	if err := provider.ValidateTaxpayerBinding(ref, claims.TaxRegistrationNumber); err != nil {
		return "", err
	}
	if claims.RequestID == "" {
		return "", ErrValidation
	}
	return signTyped(provider, ref, claims)
}

// SignConsultarFacturaRequest signs eligible request claims with a taxpayer identity.
func SignConsultarFacturaRequest(provider agttestkit.IdentityProvider, ref string, claims ConsultarFacturaRequestClaims) (string, error) {
	if err := provider.RequireRole(ref, agttestkit.RoleTaxpayerTest); err != nil {
		return "", err
	}
	if err := provider.ValidateTaxpayerBinding(ref, claims.TaxRegistrationNumber); err != nil {
		return "", err
	}
	if claims.DocumentNo == "" {
		return "", ErrValidation
	}
	return signTyped(provider, ref, claims)
}

func signTyped(provider agttestkit.IdentityProvider, ref string, claims any) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signer, err := provider.Signer(ref)
	if err != nil {
		return "", err
	}
	return fejws.SignCompact(signer, payload, fejws.ProtectedHeader{Alg: fejws.Algorithm})
}

// VerifyWithSignerPublic verifies compact JWS using the provider's signer public key.
func VerifyWithSignerPublic(provider agttestkit.IdentityProvider, ref, compact string) ([]byte, error) {
	signer, err := provider.Signer(ref)
	if err != nil {
		return nil, err
	}
	pub, err := fejws.PublicRSA(signer)
	if err != nil {
		return nil, err
	}
	return fejws.VerifyCompact(pub, compact)
}

// Blocked constructors — prove profiles cannot be built while conflicts remain open.

func SignRegistarDocumentBlocked(_ agttestkit.IdentityProvider, _ string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("%w: %s (%s)", ErrProfileBlocked, ProfileRegistarDocument, "C-FE-JWS-DOC-001")
}

func SignRegistarRequestSignatureBlocked(_ agttestkit.IdentityProvider, _ string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("%w: %s (FE-RNG-031 vs schema; typ conflict C-FE-JWS-TYP-001)", ErrProfileBlocked, ProfileRegistarRequestSignature)
}

func SignSolicitarSerieRequestBlocked(_ agttestkit.IdentityProvider, _ string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("%w: %s (%s)", ErrProfileBlocked, ProfileSolicitarSerieRequest, "C-FE-JWS-REQ-001")
}

func SignListarSeriesRequestBlocked(_ agttestkit.IdentityProvider, _ string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("%w: %s (%s)", ErrProfileBlocked, ProfileListarSeriesRequest, "C-FE-JWS-REQ-002")
}

func SignValidarDocumentoRequestBlocked(_ agttestkit.IdentityProvider, _ string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("%w: %s (%s)", ErrProfileBlocked, ProfileValidarDocumentoRequest, "C-FE-JWS-REQ-003")
}

func SignListarFacturasRequestBlocked(_ agttestkit.IdentityProvider, _ string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("%w: %s (%s)", ErrProfileBlocked, ProfileListarFacturasRequest, "C-FE-JWS-REQ-004")
}
