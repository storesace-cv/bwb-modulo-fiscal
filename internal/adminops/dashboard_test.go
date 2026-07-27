package adminops_test

import (
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
)

func TestBuildOpsAlerts(t *testing.T) {
	empty := adminops.BuildOpsAlerts(adminops.QueueCounts{})
	if len(empty) != 1 || empty[0].Code != "ops_queue_empty" {
		t.Fatalf("%+v", empty)
	}
	alerts := adminops.BuildOpsAlerts(adminops.QueueCounts{
		ManualReview: 12, Retry: 25, Processing: 1, Total: 40,
	})
	codes := map[string]adminops.AlertSeverity{}
	for _, a := range alerts {
		codes[a.Code] = a.Severity
		low := strings.ToLower(a.Message)
		for _, ban := range []string{"begin ", "eyj", "nif=", "password", "jws"} {
			if strings.Contains(low, ban) {
				t.Fatalf("secret-like alert %+v", a)
			}
		}
	}
	if codes["ops_manual_review_backlog"] != adminops.SeverityBlocking {
		t.Fatal(codes)
	}
	if codes["ops_retry_backlog"] != adminops.SeverityWarning {
		t.Fatal(codes)
	}
	if _, ok := codes["ops_processing_inflight"]; !ok {
		t.Fatal(codes)
	}
}

func TestClampPage(t *testing.T) {
	if adminops.ClampPage("") != 1 || adminops.ClampPage("0") != 1 || adminops.ClampPage("3") != 3 {
		t.Fatal("clamp page")
	}
}
