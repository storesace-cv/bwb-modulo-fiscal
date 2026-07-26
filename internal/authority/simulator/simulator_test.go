package simulator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaljws"
)

func TestSubmitAcceptRejectUnknown(t *testing.T) {
	c := simulator.New(simulator.OutcomeAccept)
	c.Script("s-rej", simulator.OutcomeReject)
	c.Script("s-unk", simulator.OutcomeUnknown)

	r, err := c.Submit(context.Background(), simulator.Request{SubmissionID: "s-ok", DocumentID: "d1"})
	if err != nil || r.Outcome != simulator.OutcomeAccept {
		t.Fatalf("accept: %+v %v", r, err)
	}
	r, err = c.Submit(context.Background(), simulator.Request{SubmissionID: "s-rej", DocumentID: "d1"})
	if err != nil || r.Outcome != simulator.OutcomeReject {
		t.Fatalf("reject: %+v %v", r, err)
	}
	r, err = c.Submit(context.Background(), simulator.Request{SubmissionID: "s-unk", DocumentID: "d1"})
	if err != nil || r.Outcome != simulator.OutcomeUnknown {
		t.Fatalf("unknown: %+v %v", r, err)
	}
}

func TestSubmitUnavailable(t *testing.T) {
	c := simulator.New(simulator.OutcomeUnavailable)
	_, err := c.Submit(context.Background(), simulator.Request{SubmissionID: "s1", DocumentID: "d1"})
	if !errors.Is(err, simulator.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestSubmitRequiresValidJWS(t *testing.T) {
	signer, err := fiscaljws.NewEphemeral(fiscaljws.DefaultRSABits)
	if err != nil {
		t.Fatal(err)
	}
	c := simulator.New(simulator.OutcomeAccept)
	c.VerifyPublic = signer.PublicKey()

	_, err = c.Submit(context.Background(), simulator.Request{SubmissionID: "s1", DocumentID: "d1"})
	if !errors.Is(err, simulator.ErrBadJWS) {
		t.Fatalf("want ErrBadJWS missing, got %v", err)
	}

	jws, err := signer.SignEnvelope("s1", "d1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Submit(context.Background(), simulator.Request{SubmissionID: "s1", DocumentID: "d1", JWS: jws})
	if err != nil || !r.JWSVerified || r.Outcome != simulator.OutcomeAccept {
		t.Fatalf("%+v %v", r, err)
	}
}
