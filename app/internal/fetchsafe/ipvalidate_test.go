package fetchsafe

import (
	"net/netip"
	"testing"
)

// T-27a/T-27d — EC-02/EC-07 (private/loopback/link-local rejected) and
// the F-02 regression (IPv4-mapped-to-IPv6 addresses must be rejected
// exactly like their IPv4 literal equivalent, which requires Unmap()
// before any Is*() call).

func TestValidateIPForConnect_RejectsPrivateLoopbackLinkLocal(t *testing.T) {
	cases := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.169.254", // link-local — classic cloud metadata endpoint
		"::1",
		"fc00::1", // unique local
		"fe80::1", // link-local
	}
	for _, raw := range cases {
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatalf("ParseAddr(%q): %v", raw, err)
		}
		if err := validateIPForConnect(addr); err != ErrDestinationForbidden {
			t.Errorf("validateIPForConnect(%q) = %v, want ErrDestinationForbidden", raw, err)
		}
	}
}

func TestValidateIPForConnect_AcceptsPublicUnicast(t *testing.T) {
	cases := []string{
		"8.8.8.8",
		"1.1.1.1",
		"2606:4700:4700::1111", // Cloudflare public IPv6
	}
	for _, raw := range cases {
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatalf("ParseAddr(%q): %v", raw, err)
		}
		if err := validateIPForConnect(addr); err != nil {
			t.Errorf("validateIPForConnect(%q) = %v, want nil", raw, err)
		}
	}
}

// TestValidateIPForConnect_RejectsIPv4MappedIPv6 is the F-02 regression
// test at the unit level (T-27d covers it again at integration level
// against a real dial): an IPv4-mapped IPv6 form of a loopback/link-local
// address MUST be rejected exactly like its literal IPv4 form. Without
// Unmap() before Is*() calls, IsLoopback()/IsPrivate()/
// IsLinkLocalUnicast() all silently return false on the mapped form.
func TestValidateIPForConnect_RejectsIPv4MappedIPv6(t *testing.T) {
	cases := []string{
		"::ffff:127.0.0.1",
		"::ffff:169.254.169.254",
		"::ffff:10.0.0.1",
		"::ffff:192.168.1.1",
	}
	for _, raw := range cases {
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatalf("ParseAddr(%q): %v", raw, err)
		}
		if !addr.Is4In6() {
			t.Fatalf("test setup: %q is not recognized as an IPv4-mapped IPv6 address", raw)
		}
		if err := validateIPForConnect(addr); err != ErrDestinationForbidden {
			t.Errorf("validateIPForConnect(%q) = %v, want ErrDestinationForbidden (F-02 regression)", raw, err)
		}
	}
}

// TestValidateIPForConnect_RejectsNAT64And6to4 is the F-07 regression:
// NAT64 and 6to4 prefixes can re-encode a private/loopback IPv4 address
// inside a technically-global-looking IPv6 address.
func TestValidateIPForConnect_RejectsNAT64And6to4(t *testing.T) {
	cases := []string{
		"64:ff9b::7f00:1",    // NAT64-encoded 127.0.0.1
		"64:ff9b::a9fe:a9fe", // NAT64-encoded 169.254.169.254
		"2002:7f00:0001::",   // 6to4-encoded 127.0.0.1
	}
	for _, raw := range cases {
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatalf("ParseAddr(%q): %v", raw, err)
		}
		if err := validateIPForConnect(addr); err != ErrDestinationForbidden {
			t.Errorf("validateIPForConnect(%q) = %v, want ErrDestinationForbidden", raw, err)
		}
	}
}

func TestValidateIPForConnect_RejectsMulticast(t *testing.T) {
	addr := netip.MustParseAddr("224.0.0.1")
	if err := validateIPForConnect(addr); err != ErrDestinationForbidden {
		t.Errorf("validateIPForConnect(multicast) = %v, want ErrDestinationForbidden", err)
	}
}
