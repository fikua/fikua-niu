// ipvalidate.go implements T-03c (design.md ADR-02, mechanism 2 and 7):
// the single point of IP validation for every outgoing fetchsafe
// connection, wired via net.Dialer.ControlContext (never net.Dialer.
// Control — Go silently ignores Control when ControlContext is also set,
// and ControlContext gives access to the fetch's context for future use).
//
// Deliberately NOT accompanied by any separate call to
// net.DefaultResolver.LookupIPAddr before or alongside dialing — ADR-02
// explicitly rejects that as the TOCTOU pair identified as F-06: two
// independent DNS resolutions of the same hostname can disagree (a
// hostile DNS server can answer differently milliseconds apart),
// reopening the DNS-rebinding hole EC-07 is meant to close.
// ControlContext already receives the fully-resolved address at dial
// time — validating only here is sufficient.
package fetchsafe

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"syscall"
)

// nat64Prefix and sixToFourPrefix can re-encode a private/loopback IPv4
// address inside an IPv6 address that, once "unwrapped" by the resolver,
// resolves back to that private/loopback IPv4 (F-07). Rejected explicitly
// before the general allowlist criterion, because IsGlobalUnicast() alone
// does not guarantee the encapsulated IPv4 is also global.
var (
	nat64Prefix     = netip.MustParsePrefix("64:ff9b::/96")
	sixToFourPrefix = netip.MustParsePrefix("2002::/16")
)

// validateIPForConnect classifies a single resolved IP using an allowlist
// criterion, not a denylist enumeration (F-07): after Unmap(), the address
// is only acceptable if it is a global unicast address and not classified
// as private. This deliberately replaces enumerating RFC1918/link-local/
// loopback/etc. one by one — an enumeration is necessarily incomplete
// (security-engineer identified 0.0.0.0/8, 255.255.255.255,
// 240.0.0.0/4, 198.18.0.0/15 as gaps in the prior enumeration-based
// version of this ADR), whereas IsGlobalUnicast() excludes all of them by
// construction.
func validateIPForConnect(addr netip.Addr) error {
	// Pas 0, obligatori (F-02): Unmap() BEFORE any Is*() call. An
	// IPv4-mapped IPv6 address (::ffff:127.0.0.1, ::ffff:169.254.169.254)
	// returns false from IsLoopback()/IsPrivate()/IsLinkLocalUnicast() if
	// Unmap() is not called first — the range checks silently skip over
	// the mapped form otherwise.
	addr = addr.Unmap()

	// F-07: reject NAT64/6to4 prefixes explicitly before the general
	// allowlist criterion below.
	if addr.Is6() {
		if nat64Prefix.Contains(addr) || sixToFourPrefix.Contains(addr) {
			return ErrDestinationForbidden
		}
	}

	// Multicast is not excluded from IsGlobalUnicast() by definition in
	// all implementations' documentation clarity — reject explicitly for
	// clarity even though it is not a typical SSRF vector.
	if addr.IsMulticast() {
		return ErrDestinationForbidden
	}

	// Allowlist criterion (F-07): only a global unicast address that is
	// not classified as private is acceptable. This excludes loopback,
	// link-local, private (RFC1918), unspecified, and any future reserved
	// block IsGlobalUnicast() does not cover, without enumerating them.
	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return ErrDestinationForbidden
	}

	return nil
}

// newDialer builds the net.Dialer used by fetchsafe's dedicated
// http.Transport. ControlContext is the ONLY point of IP validation
// (T-03c) — every IP the dialer's internal resolution attempts to connect
// to is classified here, and DialContext aborts before connect() if this
// returns an error. No byte ever leaves toward a forbidden destination.
func newDialer() *net.Dialer {
	return &net.Dialer{
		ControlContext: func(_ context.Context, network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}

			addr, err := netip.ParseAddr(host)
			if err != nil {
				// ControlContext is only ever invoked by the dialer with an
				// already-resolved numeric address — a parse failure here
				// means something unexpected about the network stack, not a
				// validation case. Fail closed.
				return errors.New("fetchsafe: could not parse resolved address")
			}

			switch network {
			case "tcp4", "tcp6", "tcp", "udp4", "udp6", "udp":
			default:
				return errors.New("fetchsafe: unsupported network")
			}

			return validateIPForConnect(addr)
		},
	}
}
