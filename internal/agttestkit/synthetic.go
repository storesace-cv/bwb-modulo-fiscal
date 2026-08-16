package agttestkit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// SyntheticOptions controls ephemeral workbook generation for CI/tests.
type SyntheticOptions struct {
	// Count of identities (default 2).
	Count int
	// Bits for RSA keys (default MinRSABits).
	Bits int
	// IncludeBlankLeadRow adds an empty first row like the AGT sample layout.
	IncludeBlankLeadRow bool
	Headers             [4]string // default NIF/NOME/CHAVE PRÍVADA/CHAVE PÚBLICA
}

// WriteSyntheticWorkbook creates a temporary .xlsx with ephemeral RSA pairs.
// The file must not be committed; callers should remove the directory after tests.
func WriteSyntheticWorkbook(dir string, opt SyntheticOptions) (path string, cleanup func(), err error) {
	if opt.Count <= 0 {
		opt.Count = 2
	}
	if opt.Bits <= 0 {
		opt.Bits = MinRSABits
	}
	headers := opt.Headers
	if headers[0] == "" {
		headers = [4]string{"NIF", "NOME", "CHAVE PRÍVADA", "CHAVE PÚBLICA"}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, err
	}
	path = filepath.Join(dir, "synthetic-agt-test-identities.xlsx")
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	_ = f.SetSheetName(sheet, "Folha1")
	sheet = "Folha1"

	row := 1
	if opt.IncludeBlankLeadRow {
		row = 2
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return "", nil, err
		}
	}
	row++
	for i := 0; i < opt.Count; i++ {
		priv, err := rsa.GenerateKey(rand.Reader, opt.Bits)
		if err != nil {
			return "", nil, err
		}
		privPEM, err := marshalPKCS8Private(priv)
		if err != nil {
			return "", nil, err
		}
		pubPEM, err := marshalPKIXPublic(&priv.PublicKey)
		if err != nil {
			return "", nil, err
		}
		nif := fmt.Sprintf("9%09d", 100000000+i) // synthetic digits only; not a real NIF
		nome := fmt.Sprintf("SYNTHETIC_IDENTITY_%02d", i+1)
		values := []string{nif, nome, string(privPEM), string(pubPEM)}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				zeroBytes(privPEM)
				zeroBytes(pubPEM)
				return "", nil, err
			}
		}
		zeroBytes(privPEM)
		zeroBytes(pubPEM)
		row++
	}
	if err := f.SaveAs(path); err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	return path, cleanup, nil
}

func marshalPKCS8Private(priv *rsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func marshalPKIXPublic(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}
