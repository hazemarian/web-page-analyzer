package steps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"webpage-analyzer/internal/pipeline"
)

func TestFetchHTMLSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>Hello</body></html>"))
	}))
	defer ts.Close()

	step := NewFetchHTML(ts.Client())
	// Use the test server's URL (strip scheme for normalized form)
	normalized := strings.TrimPrefix(ts.URL, "http://")
	state := pipeline.NewState(normalized)

	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := state.GetRawHTML()
	if html != "<html><body>Hello</body></html>" {
		t.Errorf("RawHTML = %q, unexpected", html)
	}

	result, ok := state.GetResult("fetch_html")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "done" {
		t.Errorf("Status = %q, want done", result.Status)
	}
}

func TestFetchHTMLNon2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	step := NewFetchHTML(ts.Client())
	normalized := strings.TrimPrefix(ts.URL, "http://")
	state := pipeline.NewState(normalized)

	err := step.Run(context.Background(), state)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("error = %q, want it to contain 'unreachable'", err.Error())
	}
}

func TestFetchHTMLBothFail(t *testing.T) {
	// No server running — both HTTPS and HTTP will fail
	step := NewFetchHTML(&http.Client{})
	state := pipeline.NewState("127.0.0.1:1") // Nothing listens here

	err := step.Run(context.Background(), state)
	if err == nil {
		t.Fatal("expected error when both schemes fail")
	}

	result, ok := state.GetResult("fetch_html")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "failed" {
		t.Errorf("Status = %q, want failed", result.Status)
	}
}

func TestFetchHTMLUserAgent(t *testing.T) {
	var receivedUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Write([]byte("<html></html>"))
	}))
	defer ts.Close()

	step := NewFetchHTML(ts.Client())
	normalized := strings.TrimPrefix(ts.URL, "http://")
	state := pipeline.NewState(normalized)

	_ = step.Run(context.Background(), state)

	if receivedUA != "webpage-analyzer/1.0" {
		t.Errorf("User-Agent = %q, want %q", receivedUA, "webpage-analyzer/1.0")
	}
}

func TestFetchHTMLAcceptHeaders(t *testing.T) {
	var receivedAccept, receivedAcceptLang string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		receivedAcceptLang = r.Header.Get("Accept-Language")
		w.Write([]byte("<html></html>"))
	}))
	defer ts.Close()

	step := NewFetchHTML(ts.Client())
	normalized := strings.TrimPrefix(ts.URL, "http://")
	state := pipeline.NewState(normalized)

	_ = step.Run(context.Background(), state)

	wantAccept := "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	if receivedAccept != wantAccept {
		t.Errorf("Accept = %q, want %q", receivedAccept, wantAccept)
	}
	wantLang := "en-US,en;q=0.5"
	if receivedAcceptLang != wantLang {
		t.Errorf("Accept-Language = %q, want %q", receivedAcceptLang, wantLang)
	}
}
