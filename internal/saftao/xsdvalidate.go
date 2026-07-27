package saftao

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ValidateXMLAgainstEmbeddedXSD validates XML bytes with xmllint --schema when available.
// Success means structural XSD acceptance of the fixture — not AGT certification or AO-*.
func ValidateXMLAgainstEmbeddedXSD(xmlBytes []byte) error {
	xsd, err := XSDBytes()
	if err != nil {
		return err
	}
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		return fmt.Errorf("saftao: xmllint não disponível: %w", err)
	}
	dir, err := os.MkdirTemp("", "saftao-xsd-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	xsdPath := filepath.Join(dir, "schema.xsd")
	xmlPath := filepath.Join(dir, "doc.xml")
	if err := os.WriteFile(xsdPath, xsd, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(xmlPath, xmlBytes, 0o600); err != nil {
		return err
	}
	cmd := exec.Command(xmllint, "--noout", "--schema", xsdPath, xmlPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("saftao: XSD inválido (estrutural): %w\n%s", err, out)
	}
	return nil
}

// XSDValidatorAvailable reports whether xmllint is on PATH.
func XSDValidatorAvailable() bool {
	_, err := exec.LookPath("xmllint")
	return err == nil
}
