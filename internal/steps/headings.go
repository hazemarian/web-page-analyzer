package steps

import (
	"context"
	"strings"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

// Headings is Stage 3. Counts heading elements (h1–h6) in the document.
type Headings struct{}

func (s *Headings) Name() string { return "headings" }
func (s *Headings) Stage() int   { return 3 }

func (s *Headings) Run(ctx context.Context, state *pipeline.State) error {
	doc, err := html.Parse(strings.NewReader(state.GetRawHTML()))
	if err != nil {
		state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: "failed to parse HTML"})
		return nil
	}

	state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: countHeadings(doc)})
	return nil
}

func countHeadings(n *html.Node) map[string]int {
	counts := map[string]int{"h1": 0, "h2": 0, "h3": 0, "h4": 0, "h5": 0, "h6": 0}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if _, ok := counts[node.Data]; ok {
				counts[node.Data]++
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return counts
}
