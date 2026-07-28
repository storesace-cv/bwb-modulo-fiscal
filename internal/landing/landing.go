// Package landing serves a public, non-authoritative HTML page at GET /.
//
// UX only: a previous nginx catch-all 404 on / did not mean the service was down
// when /v1/health and /admin were healthy. This page must not expose secrets,
// NIF, tokens, or fiscal payloads, and must not claim AGT homologation.
package landing

import (
	"net/http"
	"strings"
)

const htmlPage = `<!DOCTYPE html>
<html lang="pt">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>BWB Módulo Fiscal — sandbox</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;max-width:40rem;line-height:1.45;color:#1a1a1a;background:#f7f7f5}
h1{font-size:1.35rem;margin:0 0 .5rem}
p,li{font-size:.95rem}
a{color:#0b5fff}
.note{padding:.75rem 1rem;background:#eee;border-left:4px solid #888;margin:1.25rem 0}
code{font-size:.85em}
</style>
</head>
<body>
<h1>BWB Módulo Fiscal (sandbox)</h1>
<p>Ambiente técnico de integração. A raiz HTTP é só uma página de orientação — <strong>não</strong> é um indicador de disponibilidade do serviço.</p>
<div class="note">
<p><strong>404 na raiz ≠ serviço em baixo.</strong> A disponibilidade verifica-se em <code>/v1/health</code> (e admin health/ready). Um 404 antigo na raiz era apenas UX de API-only.</p>
</div>
<ul>
<li><a href="/v1/health"><code>GET /v1/health</code></a> — liveness da API POS</li>
<li><a href="/admin/v1/health"><code>GET /admin/v1/health</code></a> — liveness admin</li>
<li><a href="/admin/ui/"><code>/admin/ui/</code></a> — backoffice (autenticação fail-closed)</li>
</ul>
<p><code>FISCAL_ENV=homologation</code> neste sandbox é designação técnica BWB — <strong>não</strong> é homologação oficial AGT.</p>
</body>
</html>
`

// Handler serves GET / only (exact root).
type Handler struct{}

// NewHandler returns the public root landing handler.
func NewHandler() *Handler {
	return &Handler{}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// Defence in depth if mounted under a catch-all by mistake.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(htmlPage))
}

// MentionsAvailabilityConfusion is used by tests to lock the UX disclaimer.
func MentionsAvailabilityConfusion() bool {
	return strings.Contains(htmlPage, "404 na raiz") && strings.Contains(htmlPage, "/v1/health")
}
