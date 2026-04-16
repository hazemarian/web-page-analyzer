package steps

import (
	"context"
	"strings"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

// Title is Stage 3. Extracts the page title from the <title> element.
type Title struct{}

func (s *Title) Name() string { return "title" }
func (s *Title) Stage() int   { return 3 }

func (s *Title) Run(ctx context.Context, state *pipeline.State) error {
	doc, err := html.Parse(strings.NewReader(state.GetRawHTML()))
	if err != nil {
		state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: "failed to parse HTML"})
		return nil
	}

	state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: extractTitle(doc)})
	return nil
}

func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
		return strings.TrimSpace(n.FirstChild.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := extractTitle(c); t != "" {
			return t
		}
	}
	return ""
}
