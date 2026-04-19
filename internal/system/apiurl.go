package system

import (
	"net/url"
	"strings"
)

// IsAllowedAPIURL returns true if the URL uses HTTPS or targets loopback.
// Used to validate OPENBOOT_API_URL environment variable.
// Parsed hostname check prevents prefix-bypass attacks such as
// http://localhost.attacker.com passing a simple HasPrefix check.
func IsAllowedAPIURL(u string) bool {
	if strings.HasPrefix(u, "https://") {
		return true
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme != "http" {
		return false
	}
	host := parsed.Hostname() // strips port, handles [::1]
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
