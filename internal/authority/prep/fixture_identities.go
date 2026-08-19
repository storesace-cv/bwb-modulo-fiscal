package prep

import (
	"fmt"
	"os"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fehub"
)

// FixtureIdentityCatalog loads sanitized workbook identity refs when path is configured.
// Path must be operator-supplied (typically under local/); never versioned in Git.
func FixtureIdentityCatalog(workbookPath string) (configured bool, refs []agttestkit.SanitizedRef, err error) {
	path := strings.TrimSpace(workbookPath)
	if path == "" {
		return false, nil, nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return false, nil, fmt.Errorf("prep: workbook stat: %w", err)
	}
	if st.IsDir() {
		return false, nil, fmt.Errorf("prep: workbook path is directory")
	}
	inv, err := agttestkit.LoadAndValidate(path)
	if err != nil {
		return false, nil, fmt.Errorf("prep: workbook validate: %w", err)
	}
	out := make([]agttestkit.SanitizedRef, 0, len(inv.Identities))
	for _, id := range inv.Identities {
		out = append(out, agttestkit.SanitizedRef{
			Ref: id.OpaqueRef, Algorithm: id.Algorithm, RSABits: id.RSABits, Role: agttestkit.RoleTaxpayerTest,
		})
	}
	return true, out, nil
}

// FixtureHubView returns sanitized fixture hub metadata (transport allowed for BWB-MOCK only).
func FixtureHubView() fehub.PublicView {
	return fehub.NewFixture().View()
}
