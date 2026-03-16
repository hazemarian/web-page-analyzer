package handler

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"webpage-analyzer/internal/pipeline"
	"webpage-analyzer/web"
)

// fakeStore implements store.Store for testing.
type fakeStore struct {
	mu   sync.RWMutex
	data map[string]map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: make(map[string]map[string]string)}
}

func (f *fakeStore) Ping(_ context.Context) error { return nil }

func (f *fakeStore) InitJob(_ context.Context, key string, stepNames []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := map[string]string{"overall_status": "pending", "overall_error": ""}
	for _, name := range stepNames {
		m["step:"+name+":status"] = "pending"
	}
	f.data[key] = m
	return nil
}

func (f *fakeStore) SetStep(_ context.Context, key, stepName, status, data, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.data[key] == nil {
		f.data[key] = make(map[string]string)
	}
	f.data[key]["step:"+stepName+":status"] = status
	if data != "" {
		f.data[key]["step:"+stepName+":data"] = data
	}
	if errMsg != "" {
		f.data[key]["step:"+stepName+":error"] = errMsg
	}
	return nil
}

func (f *fakeStore) SetOverallStatus(_ context.Context, key, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.data[key] == nil {
		f.data[key] = make(map[string]string)
	}
	f.data[key]["overall_status"] = status
	if errMsg != "" {
		f.data[key]["overall_error"] = errMsg
	}
	return nil
}

func (f *fakeStore) GetAll(_ context.Context, key string) (map[string]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	m, ok := f.data[key]
	if !ok {
		return map[string]string{}, nil
	}
	// Return a copy
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp, nil
}

// noopStep is a step that immediately succeeds.
type noopStep struct {
	name  string
	stage int
}

func (s *noopStep) Name() string { return s.name }
func (s *noopStep) Stage() int   { return s.stage }
func (s *noopStep) Run(_ context.Context, state *pipeline.State) error {
	state.SetResult(s.name, pipeline.StepResult{Status: "done"})
	return nil
}

func setupRouter(t *testing.T, store *fakeStore) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tmpl := template.Must(template.New("").ParseFS(web.Templates, "templates/*.html"))
	router.SetHTMLTemplate(tmpl)

	steps := []pipeline.Step{
		&noopStep{name: "url_validation", stage: 1},
		&noopStep{name: "fetch_html", stage: 2},
	}

	analyzeH := NewAnalyzeHandler(store, steps)
	resultH := NewResultHandler(store)

	router.POST("/analyze", analyzeH.Handle)
	router.GET("/result", resultH.Handle)

	return router
}

func TestAnalyzeMissingURL(t *testing.T) {
	store := newFakeStore()
	router := setupRouter(t, store)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/analyze", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAnalyzeNewJob(t *testing.T) {
	store := newFakeStore()
	router := setupRouter(t, store)

	form := url.Values{"url": {"https://example.com"}}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/analyze", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestAnalyzeExistingJob(t *testing.T) {
	s := newFakeStore()
	router := setupRouter(t, s)

	// Pre-populate a pending job directly in the store
	normalized := "example.com"
	jobKey := "job:" + "a379a6f6eeafb9a55e378c118034e2751e682fab9f2d30ab13d2125586ce1947"
	s.mu.Lock()
	s.data[jobKey] = map[string]string{"overall_status": "pending"}
	s.mu.Unlock()

	// Submit — should get 200 (existing pending job)
	form := url.Values{"url": {"https://" + normalized}}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/analyze", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (existing job)", w.Code, http.StatusOK)
	}
}

func TestResultNotFound(t *testing.T) {
	store := newFakeStore()
	router := setupRouter(t, store)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/result?url=https://nonexistent.com", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "not_found") {
		t.Error("response should indicate not_found")
	}
}

func TestResultSuccess(t *testing.T) {
	store := newFakeStore()
	router := setupRouter(t, store)

	// First create a job
	form := url.Values{"url": {"https://example.com"}}
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/analyze", strings.NewReader(form.Encode()))
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w1, req1)

	// Then fetch result
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/result?url=https://example.com", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusOK)
	}
}

func TestBuildViewModel(t *testing.T) {
	data := map[string]string{
		"overall_status":          "done",
		"overall_error":           "",
		"step:html_version:status": "done",
		"step:html_version:data":   `"HTML5"`,
		"step:title:status":        "done",
		"step:title:data":          `"My Page"`,
		"step:headings:status":     "done",
		"step:headings:data":       `{"h1":2,"h2":1,"h3":0,"h4":0,"h5":0,"h6":0}`,
		"step:login_form:status":   "done",
		"step:login_form:data":     "true",
		"step:links:status":        "done",
		"step:links:data":          "5",
		"step:link_checker:status": "done",
		"step:link_checker:data":   `{"internal":3,"external":2,"inaccessible":1}`,
	}

	vm := buildViewModel("example.com", data)

	if vm.URL != "example.com" {
		t.Errorf("URL = %q", vm.URL)
	}
	if vm.Polling {
		t.Error("Polling should be false for 'done' status")
	}
	if vm.OverallStatus != "done" {
		t.Errorf("OverallStatus = %q", vm.OverallStatus)
	}
	if vm.HTMLVersion != "HTML5" {
		t.Errorf("HTMLVersion = %q, want HTML5", vm.HTMLVersion)
	}
	if vm.Title != "My Page" {
		t.Errorf("Title = %q, want My Page", vm.Title)
	}
	if vm.Headings["h1"] != 2 {
		t.Errorf("Headings[h1] = %d, want 2", vm.Headings["h1"])
	}
	if !vm.HasLoginForm {
		t.Error("HasLoginForm should be true")
	}
	if vm.TotalLinks != 5 {
		t.Errorf("TotalLinks = %d, want 5", vm.TotalLinks)
	}
	if vm.LinkCounts.Internal != 3 {
		t.Errorf("LinkCounts.Internal = %d, want 3", vm.LinkCounts.Internal)
	}
	if vm.LinkCounts.Inaccessible != 1 {
		t.Errorf("LinkCounts.Inaccessible = %d, want 1", vm.LinkCounts.Inaccessible)
	}
}

func TestBuildViewModelPolling(t *testing.T) {
	for _, status := range []string{"pending", "processing"} {
		data := map[string]string{"overall_status": status}
		vm := buildViewModel("example.com", data)
		if !vm.Polling {
			t.Errorf("Polling should be true for status %q", status)
		}
	}
}

func TestParseStepKey(t *testing.T) {
	tests := []struct {
		key      string
		wantName string
		wantField string
	}{
		{"step:html_version:status", "html_version", "status"},
		{"step:link_checker:data", "link_checker", "data"},
		{"step:title:error", "title", "error"},
		{"overall_status", "", ""},
		{"step:", "", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			name, field := parseStepKey(tt.key)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if field != tt.wantField {
				t.Errorf("field = %q, want %q", field, tt.wantField)
			}
		})
	}
}

func TestJsonHelpers(t *testing.T) {
	t.Run("jsonString", func(t *testing.T) {
		if got := jsonString(`"hello"`); got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
		if got := jsonString(""); got != "" {
			t.Errorf("got %q, want empty", got)
		}
		if got := jsonString("not-json"); got != "not-json" {
			t.Errorf("got %q, want %q", got, "not-json")
		}
	})

	t.Run("jsonBool", func(t *testing.T) {
		if got := jsonBool("true"); !got {
			t.Error("expected true")
		}
		if got := jsonBool("false"); got {
			t.Error("expected false")
		}
		if got := jsonBool(""); got {
			t.Error("expected false for empty")
		}
	})

	t.Run("jsonInt", func(t *testing.T) {
		if got := jsonInt("42"); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
		if got := jsonInt(""); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("jsonHeadings", func(t *testing.T) {
		got := jsonHeadings(`{"h1":1,"h2":2}`)
		if got["h1"] != 1 || got["h2"] != 2 {
			t.Errorf("got %v", got)
		}
		if got := jsonHeadings(""); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("jsonLinkCounts", func(t *testing.T) {
		got := jsonLinkCounts(`{"internal":3,"external":2,"inaccessible":1}`)
		if got.Internal != 3 || got.External != 2 || got.Inaccessible != 1 {
			t.Errorf("got %+v", got)
		}
		empty := jsonLinkCounts("")
		if empty.Internal != 0 {
			t.Errorf("got %+v, want zeros", empty)
		}
	})
}
