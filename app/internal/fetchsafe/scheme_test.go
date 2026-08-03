package fetchsafe

import "testing"

// T-20 — AC-08/EC-01/EC-10: scheme validation rejects everything except
// http/https before any network activity.

func TestValidateScheme_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, raw := range []string{"http://example.com/page", "https://example.com/page"} {
		if _, err := validateScheme(raw); err != nil {
			t.Errorf("validateScheme(%q) = %v, want nil", raw, err)
		}
	}
}

func TestValidateScheme_RejectsNonHTTPSchemes(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"ftp://example.com/file",
		"data:text/html,<script>alert(1)</script>",
		"",
		"   ",
		"not a url at all",
	}
	for _, raw := range cases {
		if _, err := validateScheme(raw); err != ErrSchemeRejected {
			t.Errorf("validateScheme(%q) = %v, want ErrSchemeRejected", raw, err)
		}
	}
}

func TestValidateScheme_RejectsMissingHost(t *testing.T) {
	if _, err := validateScheme("http://"); err != ErrSchemeRejected {
		t.Errorf("validateScheme(\"http://\") = %v, want ErrSchemeRejected", err)
	}
}
