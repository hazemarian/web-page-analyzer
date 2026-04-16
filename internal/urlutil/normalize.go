package urlutil

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Normalize strips the scheme and www prefix, lowercases the host,
// and trims trailing slashes. Path and query string are preserved.
func Normalize(rawURL string) string {
	u := rawURL
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "www.")
	u = strings.TrimRight(u, "/")
	if idx := strings.IndexByte(u, '/'); idx != -1 {
		return strings.ToLower(u[:idx]) + u[idx:]
	}
	return strings.ToLower(u)
}

// JobKey returns a Redis-safe key derived from a normalized URL.
func JobKey(normalizedURL string) string {
	hash := sha256.Sum256([]byte(normalizedURL))
	return fmt.Sprintf("job:%x", hash)
}

// ToHTTPS builds an absolute HTTPS URL from a normalized URL.
func ToHTTPS(normalizedURL string) string {
	return "https://" + normalizedURL
}

// ToHTTP builds an absolute HTTP URL from a normalized URL.
func ToHTTP(normalizedURL string) string {
	return "http://" + normalizedURL
}

// Domain extracts the host portion from a normalized URL.
func Domain(normalizedURL string) string {
	if idx := strings.IndexByte(normalizedURL, '/'); idx != -1 {
		return normalizedURL[:idx]
	}
	return normalizedURL
}
