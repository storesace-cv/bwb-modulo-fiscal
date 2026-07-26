package simulator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
)

func TestSubmitAcceptRejectUnknown(t *testing.T) {
	c := simulator.New(simulator.OutcomeAccept)
	c.Script("s-rej", simulator.OutcomeReject)
	c.Script("s-unk", simulator.OutcomeUnknown)

	r, err := c.Submit(context.Background(), "s-ok")
	if err != nil || r.Outcome != simulator.OutcomeAccept {
		t.Fatalf("accept: %+v %v", r, err)
	}
	r, err = c.Submit(context.Background(), "s-rej")
	if err != nil || r.Outcome != simulator.OutcomeReject {
		t.Fatalf("reject: %+v %v", r, err)
	}
	r, err = c.Submit(context.Background(), "s-unk")
	if err != nil || r.Outcome != simulator.OutcomeUnknown {
		t.Fatalf("unknown: %+v %v", r, err)
	}
}

func TestSubmitUnavailable(t *testing.T) {
	c := simulator.New(simulator.OutcomeUnavailable)
	_, err := c.Submit(context.Background(), "s1")
	if !errors.Is(err, simulator.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}
