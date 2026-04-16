package steps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"webpage-analyzer/internal/pipeline"
)

func TestLinkCheckerAccessible(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	state := pipeline.NewState("example.com")
	state.SetLinks([]pipeline.Link{
		{URL: ts.URL + "/page1", Internal: true},
		{URL: ts.URL + "/page2", Internal: false},
	})

	step := NewLinkChecker(2, ts.Client())
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("link_checker")
	if !ok {
		t.Fatal("expected result")
	}
	counts := result.Data.(LinkCounts)
	if counts.Internal != 1 {
		t.Errorf("Internal = %d, want 1", counts.Internal)
	}
	if counts.External != 1 {
		t.Errorf("External = %d, want 1", counts.External)
	}
	if counts.Inaccessible != 0 {
		t.Errorf("Inaccessible = %d, want 0", counts.Inaccessible)
	}
}

func TestLinkCheckerInaccessible(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	state := pipeline.NewState("example.com")
	state.SetLinks([]pipeline.Link{
		{URL: ts.URL + "/bad", Internal: true},
	})

	step := NewLinkChecker(2, ts.Client())
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := state.GetResult("link_checker")
	counts := result.Data.(LinkCounts)
	if counts.Inaccessible != 1 {
		t.Errorf("Inaccessible = %d, want 1", counts.Inaccessible)
	}
}

func TestLinkCheckerHEADFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	state := pipeline.NewState("example.com")
	state.SetLinks([]pipeline.Link{
		{URL: ts.URL + "/page", Internal: true},
	})

	step := NewLinkChecker(2, ts.Client())
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := state.GetResult("link_checker")
	counts := result.Data.(LinkCounts)
	if counts.Inaccessible != 0 {
		t.Errorf("Inaccessible = %d, want 0 (GET fallback should succeed)", counts.Inaccessible)
	}
}

func TestLinkCheckerEmpty(t *testing.T) {
	state := pipeline.NewState("example.com")
	// No links set

	step := NewLinkChecker(2, nil)
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("link_checker")
	if !ok {
		t.Fatal("expected result")
	}
	counts := result.Data.(LinkCounts)
	if counts.Internal != 0 || counts.External != 0 || counts.Inaccessible != 0 {
		t.Errorf("counts = %+v, want all zeros", counts)
	}
}

func TestLinkCheckerConcurrency(t *testing.T) {
	var requestCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	state := pipeline.NewState("example.com")
	links := make([]pipeline.Link, 10)
	for i := range links {
		links[i] = pipeline.Link{URL: ts.URL + "/page", Internal: true}
	}
	state.SetLinks(links)

	step := NewLinkChecker(3, ts.Client())
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 10 links should have been checked
	if got := requestCount.Load(); got < 10 {
		t.Errorf("requestCount = %d, want >= 10", got)
	}
}

func TestLinkCheckerAcceptHeaders(t *testing.T) {
	var receivedAccept, receivedAcceptLang string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		receivedAcceptLang = r.Header.Get("Accept-Language")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	state := pipeline.NewState("example.com")
	state.SetLinks([]pipeline.Link{
		{URL: ts.URL + "/page", Internal: true},
	})

	step := NewLinkChecker(1, ts.Client())
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantAccept := "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	if receivedAccept != wantAccept {
		t.Errorf("Accept = %q, want %q", receivedAccept, wantAccept)
	}
	wantLang := "en-US,en;q=0.5"
	if receivedAcceptLang != wantLang {
		t.Errorf("Accept-Language = %q, want %q", receivedAcceptLang, wantLang)
	}
}
