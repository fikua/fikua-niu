package fetchsafe

import "testing"

// T-27b — NFR-06: the hostname denylist is a mechanism separate from IP
// validation (T-03c) — verified here at the unit level with the exact
// hardcoded names design.md ADR-02 point 8 lists, plus NIU_PUBLIC_HOST.

func TestIsDeniedHost_HardcodedNames(t *testing.T) {
	cases := []string{
		"niu.fikua.com",
		"NIU.FIKUA.COM", // case-insensitive
		"otel-collector",
		"dozzle",
		"openobserve",
		"traefik",
	}
	for _, host := range cases {
		if !isDeniedHost(host) {
			t.Errorf("isDeniedHost(%q) = false, want true", host)
		}
	}
}

func TestIsDeniedHost_NIUPublicHostEnv(t *testing.T) {
	t.Setenv("NIU_PUBLIC_HOST", "staging.example.com")
	if !isDeniedHost("staging.example.com") {
		t.Error("isDeniedHost with NIU_PUBLIC_HOST set = false, want true")
	}
	if !isDeniedHost("STAGING.EXAMPLE.COM") {
		t.Error("isDeniedHost with NIU_PUBLIC_HOST set (case-insensitive) = false, want true")
	}
}

func TestIsDeniedHost_AllowsUnrelatedHosts(t *testing.T) {
	for _, host := range []string{"example.com", "instagram.com", "www.google.com"} {
		if isDeniedHost(host) {
			t.Errorf("isDeniedHost(%q) = true, want false", host)
		}
	}
}
