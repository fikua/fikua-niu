package fetchsafe

import "net/url"

// validateScheme parses rawURL and rejects anything other than http/https
// (T-03a, NFR-05/EC-01) BEFORE any network activity — file://,
// javascript:, ftp://, data:, and an empty scheme are all rejected with
// ErrSchemeRejected, zero network requests issued.
func validateScheme(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrSchemeRejected
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrSchemeRejected
	}
	if parsed.Host == "" {
		return nil, ErrSchemeRejected
	}
	return parsed, nil
}
