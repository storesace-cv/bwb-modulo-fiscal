package fefixqueue

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feprofile"
)

// StateDead marks exhausted retries (≠ AGT rejection semantics).
const StateDead = "fixture_boundary_dead"

var (
	ErrInvalidInput   = errors.New("fefixqueue: invalid input")
	ErrNotFound       = errors.New("fefixqueue: not found")
	ErrEmpty          = errors.New("fefixqueue: empty")
	ErrIdempotencyHit = errors.New("fefixqueue: idempotency conflict")
)

// Payload holds operation-specific claims serialized in payload_json.
type Payload struct {
	Software    *feprofile.SoftwareInfoClaims            `json:"software,omitempty"`
	ObterEstado *feprofile.ObterEstadoRequestClaims      `json:"obter_estado,omitempty"`
	Consultar   *feprofile.ConsultarFacturaRequestClaims `json:"consultar,omitempty"`
}

// Validate checks payload matches operation.
func (p Payload) Validate(op string) error {
	switch op {
	case feboundary.OpSoftwareInfo:
		if p.Software == nil {
			return fmt.Errorf("%w: software claims required", ErrInvalidInput)
		}
	case feboundary.OpObterEstado:
		if p.ObterEstado == nil {
			return fmt.Errorf("%w: obterEstado claims required", ErrInvalidInput)
		}
	case feboundary.OpConsultarFactura:
		if p.Consultar == nil {
			return fmt.Errorf("%w: consultarFactura claims required", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown operation %q", ErrInvalidInput, op)
	}
	return nil
}

func (p Payload) marshal() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	if len(b) < 3 {
		return "", fmt.Errorf("%w: empty payload", ErrInvalidInput)
	}
	return string(b), nil
}

func parsePayload(raw string) (Payload, error) {
	var p Payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return Payload{}, fmt.Errorf("%w: payload json", ErrInvalidInput)
	}
	return p, nil
}

// Row is a persisted fixture submission (sanitized; no PEM/NIF/JWS).
type Row struct {
	ID             string
	Operation      string
	State          string
	IdentityRef    string
	IdempotencyKey string
	PayloadJSON    string
	Attempts       int
	MockRequestID  string
	MockCode       string
	SourceID       string
	Note           string
}

// IsAGTAccepted is always false — fixture success ≠ AGT acceptance.
func (r Row) IsAGTAccepted() bool { return false }

func isTerminal(state string) bool {
	switch state {
	case feboundary.StateOK, feboundary.StateReject, feboundary.StateBlocked, StateDead:
		return true
	default:
		return false
	}
}

func isRetryableEngineState(state string) bool {
	return state == feboundary.StateFailed
}
