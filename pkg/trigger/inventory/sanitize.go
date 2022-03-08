package inventory

import (
	"fmt"
	"net/url"
	"strings"
)

// SanitizeURL takes a raw repository URL and returns only the hostname and path, removing possible
// prefix protocol, and extension suffixes.
func SanitizeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	urlPath := strings.TrimSuffix(u.EscapedPath(), ".git")
	return fmt.Sprintf("%s%s", u.Hostname(), urlPath), nil
}

// CompareURLs compare the informed URLs.
func CompareURLs(a, b string) bool {
	if a == b {
		return true
	}
	aSanitized, err := SanitizeURL(a)
	if err != nil {
		return false
	}
	bSanitized, err := SanitizeURL(b)
	if err != nil {
		return false
	}
	return aSanitized == bSanitized
}
