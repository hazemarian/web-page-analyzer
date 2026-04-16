package pipeline

import "sync"

// State is the shared data bag that flows through the pipeline.
// All exported methods are safe for concurrent use.
type State struct {
	mu      sync.RWMutex
	URL     string
	rawHTML string
	links   []Link
	results map[string]StepResult
}

// Link represents a hyperlink found in the analyzed page.
type Link struct {
	URL      string
	Internal bool
}

func NewState(url string) *State {
	return &State{
		URL:     url,
		results: make(map[string]StepResult),
	}
}

func (s *State) SetResult(name string, result StepResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[name] = result
}

func (s *State) GetResult(name string) (StepResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.results[name]
	return r, ok
}

func (s *State) SetRawHTML(html string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rawHTML = html
}

func (s *State) GetRawHTML() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rawHTML
}

func (s *State) SetLinks(links []Link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links = links
}

func (s *State) GetLinks() []Link {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.links
}
