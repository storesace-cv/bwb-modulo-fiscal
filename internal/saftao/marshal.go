package saftao

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// MarshalAuditFile serializes AuditFile as deterministic UTF-8 XML with XML declaration.
// Structural only — does not claim AGT/schema acceptance beyond local XSD checks.
func MarshalAuditFile(doc AuditFile) ([]byte, error) {
	if doc.XMLName.Local == "" {
		doc.XMLName = xml.Name{Space: Namespace, Local: "AuditFile"}
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("saftao: marshal: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("saftao: flush: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
