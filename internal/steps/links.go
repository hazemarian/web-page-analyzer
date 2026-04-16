package steps

import (
	"context"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
	"webpage-analyzer/internal/urlutil"
)

// Links is Stage 3. Extracts all hyperlinks, classifies them as internal or external,
// and resolves relative URLs to absolute. Populates state.Links for Stage 4.
type Links struct{}

func (s *Links) Name() string { return "links" }
func (s *Links) Stage() int   { return 3 }

func (s *Links) Run(ctx context.Context, state *pipeline.State) error {
	doc, err := html.Parse(strings.NewReader(state.GetRawHTML()))
	if err != nil {
		state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: "failed to parse HTML"})
		return nil
	}

	base, _ := url.Parse(urlutil.ToHTTPS(state.URL))
	baseDomain := urlutil.Domain(state.URL)

	links := extractLinks(doc, base, baseDomain)
	state.SetLinks(links)

	state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: len(links)})
	return nil
}

func extractLinks(doc *html.Node, base *url.URL, baseDomain string) []pipeline.Link {
	var links []pipeline.Link
	seen := make(map[string]bool)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				href := strings.TrimSpace(attr.Val)
				if href == "" || strings.HasPrefix(href, "#") ||
					strings.HasPrefix(href, "javascript:") ||
					strings.HasPrefix(href, "mailto:") ||
					strings.HasPrefix(href, "tel:") {
					break
				}

				parsed, err := url.Parse(href)
				if err != nil {
					break
				}
				abs := base.ResolveReference(parsed).String()

				if seen[abs] {
					break
				}
				seen[abs] = true

				links = append(links, pipeline.Link{
					URL:      abs,
					Internal: isInternalLink(parsed, baseDomain),
				})
				break
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links
}

func isInternalLink(parsed *url.URL, baseDomain string) bool {
	if parsed.Host == "" {
		return true // relative URL
	}
	host := strings.TrimPrefix(parsed.Host, "www.")
	base := strings.TrimPrefix(baseDomain, "www.")
	return strings.EqualFold(host, base)
}
