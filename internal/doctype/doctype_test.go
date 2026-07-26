package doctype_test

import (
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
)

func TestDefaultRegistryLoads(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	ft, ok := reg.Lookup(doctype.CanonicalFT)
	if !ok || ft.Activo != doctype.ActiveOn {
		t.Fatalf("FT: ok=%v activo=%q", ok, ft.Activo)
	}
	nc, ok := reg.Lookup(doctype.CanonicalNC)
	if !ok || nc.Activo != doctype.ActiveOn {
		t.Fatalf("NC: ok=%v activo=%q", ok, nc.Activo)
	}
	fr, ok := reg.Lookup("bwb.ao.vendas.fr")
	if !ok || fr.Activo != doctype.ActiveOff {
		t.Fatalf("FR should be off: ok=%v activo=%q", ok, fr.Activo)
	}
}

func TestResolveAPIActive(t *testing.T) {
	r, err := doctype.ResolveAPI(doctype.APIInvoice)
	if err != nil {
		t.Fatal(err)
	}
	if r.Canonical != doctype.CanonicalFT || r.APIType != doctype.APIInvoice {
		t.Fatalf("unexpected: %+v", r)
	}
	if r.Entry.ChannelAdapters.FECode != "FT" {
		t.Fatalf("FE adapter: %q", r.Entry.ChannelAdapters.FECode)
	}
	if r.Entry.ChannelAdapters.SAFTType != "InvoiceType=FT" {
		t.Fatalf("SAFT adapter: %q", r.Entry.ChannelAdapters.SAFTType)
	}
	if r.Entry.ChannelAdapters.SAFTStructure != "SalesInvoices" {
		t.Fatalf("L3: %q", r.Entry.ChannelAdapters.SAFTStructure)
	}

	r, err = doctype.ResolveAPI(doctype.APICreditNote)
	if err != nil {
		t.Fatal(err)
	}
	if r.Canonical != doctype.CanonicalNC {
		t.Fatalf("NC canonical: %s", r.Canonical)
	}
	if r.Entry.ChannelAdapters.FECode != "NC" {
		t.Fatalf("FE NC: %q", r.Entry.ChannelAdapters.FECode)
	}
}

func TestResolveAPIUnknown(t *testing.T) {
	_, err := doctype.ResolveAPI("receipt")
	if !errors.Is(err, doctype.ErrUnknownAPI) {
		t.Fatalf("want ErrUnknownAPI, got %v", err)
	}
}

func TestResolveAPIInactiveRejected(t *testing.T) {
	// Build a registry where FT is off to prove fail-closed path.
	md := []byte(`# test
| grupo | codigo_canonico | designacao | codigos_canal | estrutura_saft | elegibilidade | natureza_juridica | restricao_sectorial | serie_necessaria | requisitos | regras_rectificacao_anulacao | estado_normativo | activo |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| vendas | ` + "`bwb.ao.vendas.ft`" + ` | Factura | FE:` + "`FT`" + `; SAFT:` + "`InvoiceType=FT`" + ` | ` + "`SalesInvoices`" + ` | ambos | pending | nenhuma | pending | pending | pending | hipotese | off |
| vendas | ` + "`bwb.ao.vendas.nc`" + ` | Nota de Crédito | FE:` + "`NC`" + `; SAFT:` + "`InvoiceType=NC`" + ` | ` + "`SalesInvoices`" + ` | ambos | pending | nenhuma | pending | pending | pending | hipotese | on |
`)
	_, err := doctype.ParseCatalog(md)
	if !errors.Is(err, doctype.ErrCatalog) {
		t.Fatalf("want ErrCatalog when FT off, got %v", err)
	}
}

func TestHomonymsDistinctCanonicals(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	vendasAR, ok1 := reg.Lookup("bwb.ao.vendas.ar")
	pagAR, ok2 := reg.Lookup("bwb.ao.pagamentos.ar")
	if !ok1 || !ok2 {
		t.Fatal("homonym rows missing")
	}
	if vendasAR.CodigoCanonico == pagAR.CodigoCanonico {
		t.Fatal("homonyms must keep distinct canonical ids")
	}
	if vendasAR.ChannelAdapters.SAFTType == pagAR.ChannelAdapters.SAFTType {
		t.Fatalf("homonyms share SAFT type unexpectedly: %q", vendasAR.ChannelAdapters.SAFTType)
	}
	if vendasAR.ChannelAdapters.FECode != "AR" || pagAR.ChannelAdapters.FECode != "AR" {
		t.Fatalf("FE codes: %q / %q", vendasAR.ChannelAdapters.FECode, pagAR.ChannelAdapters.FECode)
	}
}

func TestFEOnlyFAAdapters(t *testing.T) {
	reg, err := doctype.Default()
	if err != nil {
		t.Fatal(err)
	}
	fa, ok := reg.Lookup("bwb.ao.vendas.fa")
	if !ok {
		t.Fatal("FA missing")
	}
	if fa.Activo != doctype.ActiveOff {
		t.Fatalf("FA should be off in slice: %q", fa.Activo)
	}
	if fa.ChannelAdapters.FECode != "FA" {
		t.Fatalf("FE FA: %q", fa.ChannelAdapters.FECode)
	}
	if fa.ChannelAdapters.SAFTType != "" || fa.ChannelAdapters.SAFTStructure != "" {
		t.Fatalf("FA must be FE-only: %+v", fa.ChannelAdapters)
	}
}
