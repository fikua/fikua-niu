package fetchsafe

import (
	"os"
	"strings"
)

// hardcodedDeniedHosts is the fixed set of hostnames that must never be
// contacted, regardless of what NIU_PUBLIC_HOST is configured to (T-03b,
// NFR-06 — F-03/F-04): niu.fikua.com itself (resolves to a PUBLIC
// Cloudflare-edge IP, so no IP-range check catches it, design.md ADR-02
// point 8) plus the known service names reachable over the traefik-public
// Docker network from inside the niu container (app/compose.yaml).
var hardcodedDeniedHosts = []string{
	"niu.fikua.com",
	"otel-collector",
	"dozzle",
	"openobserve",
	"traefik",
}

// isDeniedHost reports whether host matches the hostname denylist —
// checked case-insensitively, BEFORE and INDEPENDENTLY of any DNS
// resolution or IP validation (T-03c). This is deliberately a separate
// mechanism from IP classification: niu.fikua.com resolves to a public
// Cloudflare-edge IP that would pass any IP-range/allowlist check, so no
// amount of IP validation alone can catch this vector (design.md ADR-02
// point 8, R-03).
//
// host may carry a port (net/url.URL.Hostname() strips it before calling
// this) — callers MUST pass the hostname only, not host:port.
//
// host is also normalized to strip a trailing dot before comparison
// (F-12): "niu.fikua.com" and "niu.fikua.com." (the absolute/FQDN form)
// are the same DNS name and resolve identically, but without this
// normalization only the first would match the denylist, letting the
// trailing-dot form bypass it entirely.
func isDeniedHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, denied := range hardcodedDeniedHosts {
		if host == denied {
			return true
		}
	}
	if publicHost := strings.TrimSuffix(strings.ToLower(os.Getenv("NIU_PUBLIC_HOST")), "."); publicHost != "" && host == publicHost {
		return true
	}
	return false
}
