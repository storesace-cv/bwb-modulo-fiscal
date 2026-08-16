package agttestkit

import (
	"bytes"
	"regexp"
)

// privatePEMBlockRE matches a full PEM private-key envelope (multiline), not the
// bare documentary phrase "BEGIN PRIVATE KEY" alone.
var privatePEMBlockRE = regexp.MustCompile(
	`(?ms)-----BEGIN (?:RSA )?PRIVATE KEY-----\r?\n` +
		`(?:[A-Za-z0-9+/=\r\n]+)\r?\n` +
		`-----END (?:RSA )?PRIVATE KEY-----`,
)

// ContainsPrivatePEMBlock reports whether b contains a complete private-key PEM block.
func ContainsPrivatePEMBlock(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	// Normalize CRLF for matching without allocating a huge copy when small.
	if bytes.Contains(b, []byte("-----BEGIN ")) && bytes.Contains(b, []byte("PRIVATE KEY-----")) {
		return privatePEMBlockRE.Match(b)
	}
	return false
}
