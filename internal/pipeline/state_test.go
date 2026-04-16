package pipeline

import (
	"sync"
	"testing"
)

func TestNewState(t *testing.T) {
	s := NewState("example.com")
	if s.URL != "example.com" {
		t.Errorf("URL = %q, want %q", s.URL, "example.com")
	}
	if s.results == nil {
		t.Error("results map should be initialized")
	}
}

func TestSetGetResult(t *testing.T) {
	s := NewState("example.com")
	r := StepResult{Status: "done", Data: "test", Error: ""}

	s.SetResult("step1", r)
	got, ok := s.GetResult("step1")
	if !ok {
		t.Fatal("expected result to exist")
	}
	if got.Status != "done" {
		t.Errorf("Status = %q, want %q", got.Status, "done")
	}
	if got.Data != "test" {
		t.Errorf("Data = %v, want %q", got.Data, "test")
	}

	_, ok = s.GetResult("nonexistent")
	if ok {
		t.Error("expected nonexistent key to return false")
	}
}

func TestSetGetRawHTML(t *testing.T) {
	s := NewState("example.com")

	if got := s.GetRawHTML(); got != "" {
		t.Errorf("initial RawHTML = %q, want empty", got)
	}

	s.SetRawHTML("<html></html>")
	if got := s.GetRawHTML(); got != "<html></html>" {
		t.Errorf("RawHTML = %q, want %q", got, "<html></html>")
	}
}

func TestSetGetLinks(t *testing.T) {
	s := NewState("example.com")

	if got := s.GetLinks(); got != nil {
		t.Errorf("initial Links = %v, want nil", got)
	}

	links := []Link{
		{URL: "https://example.com/a", Internal: true},
		{URL: "https://other.com/b", Internal: false},
	}
	s.SetLinks(links)

	got := s.GetLinks()
	if len(got) != 2 {
		t.Fatalf("len(Links) = %d, want 2", len(got))
	}
	if got[0].URL != "https://example.com/a" || !got[0].Internal {
		t.Errorf("Link[0] = %+v, unexpected", got[0])
	}
}

func TestStateConcurrency(t *testing.T) {
	s := NewState("example.com")
	var wg sync.WaitGroup

	// Concurrent writes to different keys
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.SetResult("step", StepResult{Status: "done", Data: i})
			s.SetRawHTML("<html>test</html>")
			s.SetLinks([]Link{{URL: "https://example.com"}})
			s.GetResult("step")
			s.GetRawHTML()
			s.GetLinks()
		}(i)
	}

	wg.Wait()

	// Should not panic or race — just verify state is accessible
	if _, ok := s.GetResult("step"); !ok {
		t.Error("expected result to exist after concurrent writes")
	}
}
