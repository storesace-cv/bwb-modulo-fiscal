package adminops_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
)

func TestAlertDigestLinesAndCodes(t *testing.T) {
	t.Parallel()
	alerts := []adminops.OpsAlert{
		{Code: "ops_retry_backlog", Severity: adminops.SeverityInfo, Message: "fila retry=2"},
	}
	lines := adminops.AlertDigestLines(alerts)
	if len(lines) != 1 || lines[0].Code != "ops_retry_backlog" || lines[0].Severity != "info" {
		t.Fatalf("lines=%+v", lines)
	}
	codes := adminops.AlertCodes(alerts)
	if len(codes) != 1 || codes[0] != "ops_retry_backlog" {
		t.Fatalf("codes=%v", codes)
	}
	if len(adminops.AlertDigestLines(nil)) != 0 {
		t.Fatal("nil alerts should yield empty lines")
	}
}
