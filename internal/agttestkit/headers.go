package agttestkit

import (
	"strings"
	"unicode"
)

// Canonical column roles after controlled header normalization.
const (
	colNIF     = "nif"
	colNome    = "nome"
	colPrivada = "chave_privada"
	colPublica = "chave_publica"
)

var expectedHeaderOrder = []string{colNIF, colNome, colPrivada, colPublica}

// normalizeHeader folds accents (controlled map), case and whitespace into a stable token.
func normalizeHeader(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á', 'à', 'ã', 'â', 'ä':
			r = 'a'
		case 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ó', 'ò', 'õ', 'ô', 'ö':
			r = 'o'
		case 'ú', 'ù', 'û', 'ü':
			r = 'u'
		case 'ç':
			r = 'c'
		}
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevUnderscore = false
		case unicode.IsSpace(r) || r == '_' || r == '-':
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func mapNormalizedHeader(token string) (string, bool) {
	switch token {
	case "nif":
		return colNIF, true
	case "nome", "name":
		return colNome, true
	case "chave_privada", "chave_priv", "private_key", "private":
		return colPrivada, true
	case "chave_publica", "chave_pub", "public_key", "public":
		return colPublica, true
	default:
		return "", false
	}
}
