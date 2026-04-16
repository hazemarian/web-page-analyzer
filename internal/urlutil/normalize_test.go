package urlutil

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips https", "https://example.com", "example.com"},
		{"strips http", "http://example.com", "example.com"},
		{"strips www", "https://www.example.com", "example.com"},
		{"lowercases", "https://EXAMPLE.COM", "example.com"},
		{"trims trailing slash", "https://example.com/", "example.com"},
		{"preserves path", "https://example.com/path/to", "example.com/path/to"},
		{"preserves query", "https://example.com/path?q=1", "example.com/path?q=1"},
		{"strips scheme and www together", "http://www.example.com/", "example.com"},
		{"no scheme", "example.com", "example.com"},
		{"multiple trailing slashes", "https://example.com///", "example.com"},
		{"preserves path percent-encoding case", "https://example.com/%D8%B9%D8%B1%D8%A8%D9%8A", "example.com/%D8%B9%D8%B1%D8%A8%D9%8A"},
		{"lowercases host but not path", "https://EXAMPLE.COM/Path/To", "example.com/Path/To"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestJobKey(t *testing.T) {
	key1 := JobKey("example.com")
	key2 := JobKey("example.com")
	key3 := JobKey("other.com")

	if key1 != key2 {
		t.Error("same input should produce same key")
	}
	if key1 == key3 {
		t.Error("different input should produce different key")
	}
	if len(key1) < 10 {
		t.Error("key should be a reasonably long hash")
	}
	// Should start with "job:" prefix
	if key1[:4] != "job:" {
		t.Errorf("key should start with 'job:', got %q", key1[:4])
	}
}

func TestToHTTPS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "https://example.com"},
		{"example.com/path", "https://example.com/path"},
	}
	for _, tt := range tests {
		got := ToHTTPS(tt.input)
		if got != tt.want {
			t.Errorf("ToHTTPS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToHTTP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "http://example.com"},
		{"example.com/path", "http://example.com/path"},
	}
	for _, tt := range tests {
		got := ToHTTP(tt.input)
		if got != tt.want {
			t.Errorf("ToHTTP(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDomain(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bare domain", "example.com", "example.com"},
		{"with path", "example.com/path/to", "example.com"},
		{"with query", "example.com/path?q=1", "example.com"},
		{"subdomain", "sub.example.com/page", "sub.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Domain(tt.input)
			if got != tt.want {
				t.Errorf("Domain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
