package agttestkit

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestLoadAndValidateSyntheticOK(t *testing.T) {
	dir := t.TempDir()
	path, cleanup, err := WriteSyntheticWorkbook(dir, SyntheticOptions{Count: 3, IncludeBlankLeadRow: true})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	inv, err := LoadAndValidate(path)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if inv.IdentityCount != 3 || !inv.AllPairsValid {
		t.Fatalf("count/valid: %+v", inv)
	}
	if inv.Algorithm != "RSA" || inv.MinRSABits != MinRSABits {
		t.Fatalf("alg: %+v", inv)
	}
	for _, id := range inv.Identities {
		if id.OpaqueRef == "" || !strings.HasPrefix(id.OpaqueRef, "agt-test-") {
			t.Fatalf("opaque: %+v", id)
		}
		if id.RSABits < MinRSABits || !id.PairMatch || id.PublicFingerprint == "" {
			t.Fatalf("id: %+v", id)
		}
		assertNoSecretsInString(t, id.OpaqueRef)
		assertNoSecretsInString(t, id.PublicFingerprint)
		assertNoSecretsInString(t, id.NIFHashTruncated)
		assertNoSecretsInString(t, id.NomeHashTruncated)
	}
	for _, n := range inv.Notes {
		assertNoSecretsInString(t, n)
	}
}

func TestHeaderNormalizationAccented(t *testing.T) {
	dir := t.TempDir()
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		synthRow(t, "9100000001", "A"),
	}, false)
	if _, err := LoadAndValidate(path); err != nil {
		t.Fatal(err)
	}
}

func TestRejectUnexpectedColumns(t *testing.T) {
	dir := t.TempDir()
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA", "EXTRA"},
		append(synthRow(t, "9100000001", "A"), "x"),
	}, false)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrUnexpectedColumns.Error()) {
		t.Fatalf("want unexpected columns, got %v", err)
	}
	assertNoSecretsInString(t, err.Error())
}

func TestRejectEmptyRequiredCell(t *testing.T) {
	dir := t.TempDir()
	row := synthRow(t, "9100000001", "A")
	row[1] = "" // missing nome
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		row,
	}, true)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrMissingRequiredCell.Error()) {
		t.Fatalf("want missing cell, got %v", err)
	}
	assertNoSecretsInString(t, err.Error())
}

func TestRejectDuplicateNIF(t *testing.T) {
	dir := t.TempDir()
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		synthRow(t, "9100000001", "A"),
		synthRow(t, "9100000001", "B"),
	}, false)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrDuplicateNIF.Error()) {
		t.Fatalf("want duplicate nif, got %v", err)
	}
	assertNoSecretsInString(t, err.Error())
	if strings.Contains(err.Error(), "9100000001") {
		t.Fatal("error leaked nif value")
	}
}

func TestRejectDuplicateNome(t *testing.T) {
	dir := t.TempDir()
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		synthRow(t, "9100000001", "SAME"),
		synthRow(t, "9100000002", "SAME"),
	}, false)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrDuplicateNome.Error()) {
		t.Fatalf("want duplicate nome, got %v", err)
	}
	assertNoSecretsInString(t, err.Error())
	if strings.Contains(err.Error(), "SAME") {
		t.Fatal("error leaked nome value")
	}
}

func TestRejectInvalidPrivatePEM(t *testing.T) {
	dir := t.TempDir()
	row := synthRow(t, "9100000001", "A")
	row[2] = "not-a-pem"
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		row,
	}, false)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrInvalidPrivatePEM.Error()) {
		t.Fatalf("want invalid private, got %v", err)
	}
}

func TestRejectInvalidPublicPEM(t *testing.T) {
	dir := t.TempDir()
	row := synthRow(t, "9100000001", "A")
	row[3] = "not-a-pem"
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		row,
	}, false)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrInvalidPublicPEM.Error()) {
		t.Fatalf("want invalid public, got %v", err)
	}
}

func TestRejectSwappedPair(t *testing.T) {
	dir := t.TempDir()
	a := synthRow(t, "9100000001", "A")
	b := synthRow(t, "9100000002", "B")
	// Use A's private with B's public.
	swapped := []string{a[0], a[1], a[2], b[3]}
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		swapped,
	}, false)
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrKeyPairMismatch.Error()) {
		t.Fatalf("want mismatch, got %v", err)
	}
	assertNoSecretsInString(t, err.Error())
}

func TestRejectNonRSA(t *testing.T) {
	dir := t.TempDir()
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalECPrivateKey(ec)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	pubDER, err := x509.MarshalPKIXPublicKey(&ec.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		{"9100000001", "EC", string(privPEM), string(pubPEM)},
	}, false)
	_, err = LoadAndValidate(path)
	if err == nil {
		t.Fatal("want error for EC keys")
	}
	if !strings.Contains(err.Error(), ErrUnsupportedKey.Error()) &&
		!strings.Contains(err.Error(), ErrInvalidPrivatePEM.Error()) {
		t.Fatalf("want unsupported/invalid, got %v", err)
	}
}

func TestRejectRSATooSmall(t *testing.T) {
	dir := t.TempDir()
	path, cleanup, err := WriteSyntheticWorkbook(dir, SyntheticOptions{Count: 1, Bits: 1024})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	_, err = LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrRSATooSmall.Error()) {
		t.Fatalf("want rsa too small, got %v", err)
	}
}

func TestRejectAmbiguousSheets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.xlsx")
	f := excelize.NewFile()
	_ = f.SetSheetName(f.GetSheetName(0), "A")
	_, _ = f.NewSheet("B")
	for _, sheet := range []string{"A", "B"} {
		_ = f.SetCellValue(sheet, "A1", "NIF")
		_ = f.SetCellValue(sheet, "B1", "NOME")
		_ = f.SetCellValue(sheet, "C1", "CHAVE PRÍVADA")
		_ = f.SetCellValue(sheet, "D1", "CHAVE PÚBLICA")
		row := synthRow(t, "9100000001", sheet)
		_ = f.SetCellValue(sheet, "A2", row[0])
		_ = f.SetCellValue(sheet, "B2", row[1])
		_ = f.SetCellValue(sheet, "C2", row[2])
		_ = f.SetCellValue(sheet, "D2", row[3])
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	_, err := LoadAndValidate(path)
	if err == nil || !strings.Contains(err.Error(), ErrAmbiguousSheets.Error()) {
		t.Fatalf("want ambiguous sheets, got %v", err)
	}
}

func TestQuotedPEMCells(t *testing.T) {
	dir := t.TempDir()
	row := synthRow(t, "9100000001", "A")
	row[2] = `"` + row[2] + `"`
	row[3] = `"` + row[3] + `"`
	path := writeCustomWorkbook(t, dir, [][]string{
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		row,
	}, false)
	inv, err := LoadAndValidate(path)
	if err != nil {
		t.Fatal(err)
	}
	if inv.IdentityCount != 1 || !inv.AllPairsValid {
		t.Fatalf("%+v", inv)
	}
}

func TestSkipEmptyRows(t *testing.T) {
	dir := t.TempDir()
	path := writeCustomWorkbook(t, dir, [][]string{
		{},
		{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"},
		{},
		synthRow(t, "9100000001", "A"),
		{},
	}, false)
	inv, err := LoadAndValidate(path)
	if err != nil {
		t.Fatal(err)
	}
	if inv.IdentityCount != 1 {
		t.Fatalf("count=%d", inv.IdentityCount)
	}
}

func synthRow(t *testing.T, nif, nome string) []string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, MinRSABits)
	if err != nil {
		t.Fatal(err)
	}
	privPEM, err := marshalPKCS8Private(priv)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM, err := marshalPKIXPublic(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return []string{nif, nome, string(privPEM), string(pubPEM)}
}

func writeCustomWorkbook(t *testing.T, dir string, rows [][]string, _ bool) string {
	t.Helper()
	path := filepath.Join(dir, "custom.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetSheetName(sheet, "Folha1")
	sheet = "Folha1"
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return path
}

func assertNoSecretsInString(t *testing.T, s string) {
	t.Helper()
	if ContainsPrivatePEMBlock([]byte(s)) {
		t.Fatal("string contains private PEM block")
	}
	low := strings.ToLower(s)
	if strings.Contains(low, "begin private") && strings.Contains(low, "end private") {
		t.Fatal("string looks like private pem fencing")
	}
}
