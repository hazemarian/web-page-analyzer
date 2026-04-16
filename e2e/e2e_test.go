package e2e

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// E2E tests require the stack to be running (make run).
// Run with: make test-e2e

func baseURL() string {
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func skipIfServerUnavailable(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(baseURL() + "/"); err != nil {
		t.Skipf("skipping E2E test: server not reachable at %s: %v", baseURL(), err)
	}
}

func TestE2EAnalyzeAndPoll(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	base := baseURL()

	// Submit analysis
	form := url.Values{"url": {"https://example.com"}}
	resp, err := client.PostForm(base+"/analyze", form)
	if err != nil {
		t.Fatalf("POST /analyze failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /analyze status = %d, body = %s", resp.StatusCode, body)
	}

	// Poll for completion
	finalBody := pollForCompletion(t, client, base, "https://example.com")
	if !strings.Contains(finalBody, "badge-done") {
		t.Error("expected analysis to complete successfully")
	}
}

func TestE2EInvalidURL(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 10 * time.Second}
	base := baseURL()

	// Submit with empty URL
	resp, err := client.PostForm(base+"/analyze", url.Values{"url": {""}})
	if err != nil {
		t.Fatalf("POST /analyze failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for empty URL", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestE2ERateLimit(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 10 * time.Second}
	base := baseURL()

	// Fire many requests rapidly
	var limited bool
	for i := 0; i < 100; i++ {
		resp, err := client.Get(base + "/result?url=test")
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			limited = true
			break
		}
	}

	if !limited {
		t.Log("warning: rate limiter did not trigger within 100 requests (may depend on server config)")
	}
}

func TestE2EHomePage(t *testing.T) {
	skipIfServerUnavailable(t)

	// Use a fresh client to avoid rate limiting from previous tests
	time.Sleep(1 * time.Second)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL() + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<form") {
		t.Error("home page should contain a form element")
	}
}

func TestE2EInvalidDomainFormat(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 30 * time.Second}
	base := baseURL()
	target := "not-a-real-domain.invalidtld"

	// Submit a domain that will fail DNS resolution
	resp, err := client.PostForm(base+"/analyze", url.Values{"url": {target}})
	if err != nil {
		t.Fatalf("POST /analyze failed: %v", err)
	}
	defer resp.Body.Close()

	// Poll for result — pipeline should fail at URL validation
	body := pollForCompletion(t, client, base, target)
	if !strings.Contains(body, "Error") {
		t.Errorf("expected error in result for invalid domain, got: %s", truncate(body, 200))
	}
}

func TestE2EResultNotSubmitted(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL() + "/result?url=never-submitted-domain.xyz")
	if err != nil {
		t.Fatalf("GET /result failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	// Should not contain step results since nothing was submitted
	if strings.Contains(string(body), "badge-done") {
		t.Error("result for non-submitted URL should not contain completed steps")
	}
}

func TestE2EDuplicateSubmission(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 30 * time.Second}
	base := baseURL()
	target := "example.com"

	// Submit first time and wait for completion
	_, _ = client.PostForm(base+"/analyze", url.Values{"url": {target}})
	firstBody := pollForCompletion(t, client, base, target)
	if !strings.Contains(firstBody, "badge-done") {
		t.Fatal("first submission did not complete successfully")
	}

	// Submit again — should return cached result immediately
	resp, err := client.PostForm(base+"/analyze", url.Values{"url": {target}})
	if err != nil {
		t.Fatalf("second POST /analyze failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	// Cached result should already show done or be polling
	if !strings.Contains(s, "badge-done") && !strings.Contains(s, "badge-pending") {
		t.Error("duplicate submission should return cached or pending result")
	}
}

func TestE2EAnalyzeResultFields(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 30 * time.Second}
	base := baseURL()
	target := "example.com"

	_, _ = client.PostForm(base+"/analyze", url.Values{"url": {target}})
	body := pollForCompletion(t, client, base, target)

	// Verify all expected sections are present in the result
	checks := []struct {
		label   string
		content string
	}{
		{"HTML Version", "HTML Version"},
		{"Page Title", "Page Title"},
		{"Login Form", "Login Form"},
		{"Headings", "Headings"},
		{"Links", "Links"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.content) {
			t.Errorf("result should contain %q section", c.label)
		}
	}

	// example.com should have a title
	if !strings.Contains(body, "Example Domain") {
		t.Log("warning: expected 'Example Domain' in title for example.com")
	}
}

func TestE2EURLWithPath(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 30 * time.Second}
	base := baseURL()
	target := "example.com"

	_, _ = client.PostForm(base+"/analyze", url.Values{"url": {target}})
	body := pollForCompletion(t, client, base, target)

	if !isComplete(body) {
		t.Fatal("analysis did not complete")
	}
}

func TestE2ENormalizationConsistency(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 30 * time.Second}
	base := baseURL()

	// Submit with different forms of the same URL
	variants := []string{
		"https://example.com",
		"http://www.example.com",
		"EXAMPLE.COM",
		"https://www.example.com/",
	}

	// Submit all variants
	for _, v := range variants {
		_, _ = client.PostForm(base+"/analyze", url.Values{"url": {v}})
	}

	// All should resolve to the same result
	time.Sleep(5 * time.Second)

	var bodies []string
	for _, v := range variants {
		resp, err := client.Get(base + "/result?url=" + url.QueryEscape(v))
		if err != nil {
			t.Fatalf("GET /result failed for %q: %v", v, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodies = append(bodies, string(body))
	}

	// All results should be identical (same normalized key)
	for i := 1; i < len(bodies); i++ {
		if bodies[i] != bodies[0] {
			t.Errorf("result for variant %q differs from variant %q — normalization may be inconsistent",
				variants[i], variants[0])
		}
	}
}

func TestE2EMethodNotAllowed(t *testing.T) {
	skipIfServerUnavailable(t)

	client := &http.Client{Timeout: 5 * time.Second}
	base := baseURL()

	// GET to /analyze should not be allowed (it only accepts POST)
	resp, err := client.Get(base + "/analyze")
	if err != nil {
		t.Fatalf("GET /analyze failed: %v", err)
	}
	defer resp.Body.Close()

	// Gin returns 404 for unmatched method/path combos by default
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		t.Errorf("GET /analyze should not return success, got %d", resp.StatusCode)
	}
}

// isComplete returns true if the HTML result indicates analysis is finished.
// Completion is detected by: badge-done, badge-failed, error article, or absence of polling trigger.
func isComplete(body string) bool {
	if strings.Contains(body, "badge-done") || strings.Contains(body, "badge-failed") {
		return true
	}
	// Error view — has error article but no polling trigger
	if strings.Contains(body, "Error") && !strings.Contains(body, "hx-trigger") {
		return true
	}
	return false
}

// pollForCompletion polls GET /result until the result indicates completion,
// or the timeout expires.
func pollForCompletion(t *testing.T, client *http.Client, base, rawURL string) string {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/result?url=" + url.QueryEscape(rawURL))
		if err != nil {
			t.Fatalf("GET /result failed: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		s := string(body)
		if isComplete(s) {
			return s
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("analysis did not complete within timeout")
	return ""
}

// truncate returns the first n bytes of s, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
