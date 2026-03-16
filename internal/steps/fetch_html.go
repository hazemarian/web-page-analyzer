package steps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"webpage-analyzer/internal/pipeline"
	"webpage-analyzer/internal/urlutil"
)

const maxBodySize = 10 * 1024 * 1024 // 10 MB

// FetchHTML is Stage 2. Fetches the raw HTML of the target URL.
// Tries HTTPS first and falls back to HTTP.
type FetchHTML struct {
	client *http.Client
}

func NewFetchHTML(client *http.Client) *FetchHTML {
	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Second,
		}
	}
	return &FetchHTML{client: client}
}

func (s *FetchHTML) Name() string { return "fetch_html" }
func (s *FetchHTML) Stage() int   { return 2 }

func (s *FetchHTML) Run(ctx context.Context, state *pipeline.State) error {
	body, err := s.fetch(ctx, urlutil.ToHTTPS(state.URL))
	if err != nil {
		// Fallback to HTTP
		body, err = s.fetch(ctx, urlutil.ToHTTP(state.URL))
		if err != nil {
			e := fmt.Errorf("URL unreachable: %w", err)
			state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: e.Error()})
			return e
		}
	}

	state.SetRawHTML(body)
	state.SetResult(s.Name(), pipeline.StepResult{Status: "done"})
	return nil
}

func (s *FetchHTML) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "webpage-analyzer/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(raw), nil
}
