package steps

import (
	"context"
	"net/http"
	"sync"
	"time"

	"webpage-analyzer/internal/pipeline"
)

// LinkCounts is the result of the link accessibility check.
type LinkCounts struct {
	Internal     int `json:"internal"`
	External     int `json:"external"`
	Inaccessible int `json:"inaccessible"`
}

// LinkChecker is Stage 4. Checks each link found by the Links step for accessibility.
// Concurrency is bounded by a semaphore to avoid exhausting file descriptors.
type LinkChecker struct {
	concurrency int
	client      *http.Client
}

func NewLinkChecker(concurrency int, client *http.Client) *LinkChecker {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
	}
	return &LinkChecker{concurrency: concurrency, client: client}
}

func (s *LinkChecker) Name() string { return "link_checker" }
func (s *LinkChecker) Stage() int   { return 4 }

func (s *LinkChecker) Run(ctx context.Context, state *pipeline.State) error {
	links := state.GetLinks()
	if len(links) == 0 {
		state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: LinkCounts{}})
		return nil
	}

	sem := make(chan struct{}, s.concurrency)
	var mu sync.Mutex
	counts := LinkCounts{}
	var wg sync.WaitGroup

	for _, link := range links {
		link := link
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			accessible := s.isAccessible(ctx, link.URL)

			mu.Lock()
			defer mu.Unlock()
			if link.Internal {
				counts.Internal++
			} else {
				counts.External++
			}
			if !accessible {
				counts.Inaccessible++
			}
		}()
	}

	wg.Wait()

	state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: counts})
	return nil
}

func (s *LinkChecker) isAccessible(ctx context.Context, rawURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "webpage-analyzer/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Some servers reject HEAD; fall back to GET
	if resp.StatusCode == http.StatusMethodNotAllowed {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return false
		}
		req.Header.Set("User-Agent", "webpage-analyzer/1.0")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		resp, err = s.client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
	}

	return resp.StatusCode < 400
}
