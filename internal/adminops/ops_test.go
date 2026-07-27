package adminops_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
)

func TestDeriveQueueStatus(t *testing.T) {
	cases := []struct {
		state, outcome string
		attempts       int64
		want           string
	}{
		{"pending", "", 0, adminops.QueuePending},
		{"pending", "retried_unavailable", 2, adminops.QueueRetry},
		{"in_flight", "", 1, adminops.QueueProcessing},
		{"succeeded", "authority_accepted", 1, adminops.QueueAccepted},
		{"succeeded", "authority_rejected", 1, adminops.QueueRejected},
		{"succeeded", "authority_outcome_unknown", 1, adminops.QueueManualReview},
		{"dead", "", 3, adminops.QueueManualReview},
	}
	for _, tc := range cases {
		got := adminops.DeriveQueueStatus(tc.state, tc.outcome, tc.attempts)
		if got != tc.want {
			t.Fatalf("%s/%s/%d → %s want %s", tc.state, tc.outcome, tc.attempts, got, tc.want)
		}
	}
}

func TestSanitizeOpsError(t *testing.T) {
	if got := adminops.SanitizeOpsError("authority_rejected", "succeeded"); got != "authority_rejected" {
		t.Fatalf("got %q", got)
	}
	if got := adminops.SanitizeOpsError("-----BEGIN PRIVATE KEY-----", "pending"); got != "" {
		t.Fatalf("must drop free text: %q", got)
	}
	if got := adminops.SanitizeOpsError("", "dead"); got != "outbox_dead" {
		t.Fatalf("dead: %q", got)
	}
}
