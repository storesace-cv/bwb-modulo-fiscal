package feqr_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/feqr"
)

func TestCFEQR001Invariants(t *testing.T) {
	if v := feqr.CheckInvariants(); len(v) != 0 {
		t.Fatalf("%v", v)
	}
	if !feqr.ConflictOpen {
		t.Fatal("ConflictOpen must stay true")
	}
}

func TestRejectAndBuild(t *testing.T) {
	u := "https://" + feqr.HostQuiosqueAGT + "/facturacao-eletronica/consultar-fe?document=FT%201"
	if err := feqr.RejectAmbiguousQRURL(u); err == nil {
		t.Fatal("must reject while conflict open")
	}
	if _, err := feqr.BuildPrintedQRURL(feqr.HostQuiosqueAGT, "FT 1", "5000000000"); err == nil {
		t.Fatal("must not build URL while open")
	}
	if feqr.EncodeDocumentNoSpaces("FT 1/2025") != "FT%201/2025" {
		t.Fatal("space encoding")
	}
}
