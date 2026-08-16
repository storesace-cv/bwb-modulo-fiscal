// Package feprofile builds typed FE JWS payloads for snapshot-confirmed claim
// sets and fail-closes full AGT wire signing while header/payload conflicts remain open.
//
// Sources: AO-FE-SNAP-HML-2026-07-25-* (pending_validation). ≠ SAF-T (C-SIGN-001).
// payload_confirmed_from_snapshot ≠ wire AGT profile aceite (C-FE-JWS-TYP-001).
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

// ConflictTyp is the open protected-header typ conflict (JWT vs JOSE).
const ConflictTyp = "C-FE-JWS-TYP-001"

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

// SoftwareInfoClaims — payload_confirmed_from_snapshot for jwsSoftwareSignature
// (AO-FE-SNAP-HML-2026-07-25-ESTRUTURA / REGISTAR; pending_validation).
// Wire signing remains blocked while C-FE-JWS-TYP-001 is open.
type SoftwareInfoClaims struct {
	ProductID                string `json:"productId"`
	ProductVersion           string `json:"productVersion"`
	SoftwareValidationNumber string `json:"softwareValidationNumber"`
}

// ObterEstadoRequestClaims — payload_confirmed_from_snapshot
// (AO-FE-SNAP-HML-2026-07-25-CONSULTAR; pending_validation).
type ObterEstadoRequestClaims struct {
	TaxRegistrationNumber string `json:"taxRegistrationNumber"`
	RequestID             string `json:"requestID"`
}

// ConsultarFacturaRequestClaims — payload_confirmed_from_snapshot
// (AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA; pending_validation).
type ConsultarFacturaRequestClaims struct {
	TaxRegistrationNumber string `json:"taxRegistrationNumber"`
	DocumentNo            string `json:"documentNo"`
}

// MarshalSoftwareInfoPayload returns deterministic JSON bytes for confirmed claims.
// softwareValidationNumber must be a synthetic placeholder in tests — never a real certificate id.
func MarshalSoftwareInfoPayload(claims SoftwareInfoClaims) ([]byte, error) {
	if claims.ProductID == "" || claims.ProductVersion == "" || claims.SoftwareValidationNumber == "" {
		return nil, ErrValidation
	}
	return json.Marshal(claims)
}

// MarshalObterEstadoRequestPayload returns deterministic JSON bytes for confirmed claims.
func MarshalObterEstadoRequestPayload(claims ObterEstadoRequestClaims) ([]byte, error) {
	if claims.TaxRegistrationNumber == "" || claims.RequestID == "" {
		return nil, ErrValidation
	}
	return json.Marshal(claims)
}

// MarshalConsultarFacturaRequestPayload returns deterministic JSON bytes for confirmed claims.
func MarshalConsultarFacturaRequestPayload(claims ConsultarFacturaRequestClaims) ([]byte, error) {
	if claims.TaxRegistrationNumber == "" || claims.DocumentNo == "" {
		return nil, ErrValidation
	}
	return json.Marshal(claims)
}

func blockedTyp(id ProfileID) error {
	return fmt.Errorf("%w: %s (%s)", ErrProfileBlocked, id, ConflictTyp)
}

// SignSoftwareInfo is blocked while C-FE-JWS-TYP-001 is open.
// Omitting typ is not an approved AGT wire profile.
func SignSoftwareInfo(_ agttestkit.IdentityProvider, _ string, _ SoftwareInfoClaims) (string, error) {
	return "", blockedTyp(ProfileSoftwareInfo)
}

// SignObterEstadoRequest is blocked while C-FE-JWS-TYP-001 is open.
func SignObterEstadoRequest(_ agttestkit.IdentityProvider, _ string, _ ObterEstadoRequestClaims) (string, error) {
	return "", blockedTyp(ProfileObterEstadoRequest)
}

// SignConsultarFacturaRequest is blocked while C-FE-JWS-TYP-001 is open.
func SignConsultarFacturaRequest(_ agttestkit.IdentityProvider, _ string, _ ConsultarFacturaRequestClaims) (string, error) {
	return "", blockedTyp(ProfileConsultarFacturaRequest)
}

// VerifyWithSignerPublic verifies compact JWS using the provider's signer public key.
// Technical primitive only — does not assert AGT acceptance.
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
	return "", fmt.Errorf("%w: %s (FE-RNG-031 vs schema; typ conflict %s)", ErrProfileBlocked, ProfileRegistarRequestSignature, ConflictTyp)
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
