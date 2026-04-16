package steps

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Private IPv4
		{"10.0.0.1", "10.0.0.1", true},
		{"10.255.255.255", "10.255.255.255", true},
		{"172.16.0.1", "172.16.0.1", true},
		{"172.31.255.255", "172.31.255.255", true},
		{"192.168.0.1", "192.168.0.1", true},
		{"192.168.255.255", "192.168.255.255", true},

		// Loopback
		{"127.0.0.1", "127.0.0.1", true},
		{"127.0.0.2", "127.0.0.2", true},

		// Link-local
		{"169.254.1.1", "169.254.1.1", true},

		// Zero network
		{"0.0.0.0", "0.0.0.0", true},

		// Public IPv4
		{"8.8.8.8", "8.8.8.8", false},
		{"1.1.1.1", "1.1.1.1", false},
		{"93.184.216.34", "93.184.216.34", false},

		// Private IPv6
		{"::1", "::1", true},
		{"fc00::1", "fc00::1", true},
		{"fe80::1", "fe80::1", true},

		// Public IPv6
		{"2001:db8::1", "2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestDomainRegex(t *testing.T) {
	tests := []struct {
		domain string
		valid  bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"a.b.c.d.com", true},
		{"example.co.uk", true},
		{"123.com", true},
		{"a-b.com", true},
		{"-example.com", false},
		{"example-.com", false},
		{"example", false},
		{".com", false},
		{"example..com", false},
		{"localhost", false},
		{"192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := domainRegex.MatchString(tt.domain)
			if got != tt.valid {
				t.Errorf("domainRegex.Match(%q) = %v, want %v", tt.domain, got, tt.valid)
			}
		})
	}
}

func TestURLValidationInvalidDomain(t *testing.T) {
	// We can't easily test the full Run method (it does DNS) but we can test
	// that invalid domains are caught by the regex
	tests := []string{
		"not-a-domain",
		"192.168.1.1",
		"-bad.com",
	}

	for _, domain := range tests {
		if domainRegex.MatchString(domain) {
			t.Errorf("domainRegex should reject %q", domain)
		}
	}
}
