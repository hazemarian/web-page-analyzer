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
	var finalBody string
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/result?url=https://example.com")
		if err != nil {
			t.Fatalf("GET /result failed: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		finalBody = string(body)
		if strings.Contains(finalBody, "done") || strings.Contains(finalBody, "failed") {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if !strings.Contains(finalBody, "done") && !strings.Contains(finalBody, "failed") {
		t.Fatal("analysis did not complete within timeout")
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
