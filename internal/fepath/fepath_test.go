package fepath_test

import (
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fepath"
)

func TestCFE001PackageInvariants(t *testing.T) {
	v := fepath.CheckInvariants()
	if len(v) != 0 {
		t.Fatalf("C-FE-001 invariants: %v", v)
	}
}

func TestCFE001PrefixesDistinct(t *testing.T) {
	if fepath.PrefixV1 == fepath.PrefixWSV1 {
		t.Fatal("prefixes must differ")
	}
	if !strings.Contains(fepath.PrefixWSV1, "/ws/") {
		t.Fatal("ws prefix must contain /ws/")
	}
	if strings.Contains(fepath.PrefixV1, "/ws/") {
		t.Fatal("v1 prefix must not contain /ws/")
	}
	if !fepath.ConflictOpen {
		t.Fatal("ConflictOpen must stay true until C-FE-001 closed")
	}
}

func TestServicePathConflict(t *testing.T) {
	if !fepath.ServiceHasPathConflict(fepath.ServiceSolicitarSerie) {
		t.Fatal("solicitarSerie must be conflicted")
	}
	if !fepath.ServiceHasPathConflict(fepath.ServiceListarFacturas) {
		t.Fatal("listarFacturas must be conflicted")
	}
	if fepath.ServiceHasPathConflict("registarFactura") {
		t.Fatal("registarFactura is aligned in inventory")
	}
	if !fepath.ServiceIsAligned("consultarFactura") || !fepath.ServiceIsAligned("validarDocumento") {
		t.Fatal("consultarFactura/validarDocumento must be aligned")
	}
}

func TestBuildAlignedURL(t *testing.T) {
	got, err := fepath.BuildAlignedURL(fepath.HostHML, "registarFactura")
	if err != nil {
		t.Fatal(err)
	}
	want := fepath.HostHML + fepath.PrefixV1 + "/registarFactura"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	if _, err := fepath.BuildAlignedURL(fepath.HostPRD, fepath.ServiceListarFacturas); err == nil {
		t.Fatal("must refuse listarFacturas while C-FE-001 open")
	}
	if _, err := fepath.BuildAlignedURL(fepath.HostHML, fepath.ServiceSolicitarSerie); err == nil {
		t.Fatal("must refuse solicitarSerie while C-FE-001 open")
	}
	if _, err := fepath.BuildAlignedURL("https://example.invalid", "registarFactura"); err == nil {
		t.Fatal("must refuse unknown host")
	}
}

func TestRejectAmbiguousURL(t *testing.T) {
	ok := fepath.HostHML + fepath.PrefixV1 + "/obterEstado"
	if err := fepath.RejectAmbiguousURL(ok, "obterEstado"); err != nil {
		t.Fatal(err)
	}
	ws := fepath.HostHML + fepath.PrefixWSV1 + "/listarFacturas"
	if err := fepath.RejectAmbiguousURL(ws, fepath.ServiceListarFacturas); err == nil {
		t.Fatal("must reject conflicted service URL while open")
	}
	if err := fepath.RejectAmbiguousURL(ok, fepath.ServiceSolicitarSerie); err == nil {
		t.Fatal("must reject any URL for solicitarSerie while open")
	}
}
