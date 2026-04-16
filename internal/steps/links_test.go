package steps

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name       string
		htmlStr    string
		baseDomain string
		wantCount  int
	}{
		{
			"relative and absolute",
			`<html><body>
				<a href="/about">About</a>
				<a href="https://example.com/contact">Contact</a>
				<a href="https://other.com/page">Other</a>
			</body></html>`,
			"example.com",
			3,
		},
		{
			"dedup same link",
			`<html><body>
				<a href="https://example.com/page">First</a>
				<a href="https://example.com/page">Duplicate</a>
			</body></html>`,
			"example.com",
			1,
		},
		{
			"filtered schemes",
			`<html><body>
				<a href="https://example.com/ok">OK</a>
				<a href="mailto:test@test.com">Mail</a>
				<a href="javascript:void(0)">JS</a>
				<a href="tel:+123">Phone</a>
				<a href="#">Hash</a>
			</body></html>`,
			"example.com",
			1,
		},
		{
			"no links",
			`<html><body><p>No links</p></body></html>`,
			"example.com",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.htmlStr))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			base, _ := url.Parse("https://example.com")
			links := extractLinks(doc, base, tt.baseDomain)
			if len(links) != tt.wantCount {
				t.Errorf("got %d links, want %d", len(links), tt.wantCount)
			}
		})
	}
}

func TestIsInternalLink(t *testing.T) {
	tests := []struct {
		name       string
		href       string
		baseDomain string
		want       bool
	}{
		{"relative URL", "/about", "example.com", true},
		{"same domain", "https://example.com/page", "example.com", true},
		{"different domain", "https://other.com/page", "example.com", false},
		{"www prefix", "https://www.example.com/page", "example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, _ := url.Parse(tt.href)
			got := isInternalLink(parsed, tt.baseDomain)
			if got != tt.want {
				t.Errorf("isInternalLink(%q, %q) = %v, want %v", tt.href, tt.baseDomain, got, tt.want)
			}
		})
	}
}

func TestLinksRun(t *testing.T) {
	state := pipeline.NewState("example.com")
	state.SetRawHTML(`<html><body>
		<a href="/page1">P1</a>
		<a href="https://other.com/page2">P2</a>
	</body></html>`)

	step := &Links{}
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("links")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "done" {
		t.Errorf("Status = %q, want done", result.Status)
	}

	links := state.GetLinks()
	if len(links) != 2 {
		t.Fatalf("len(links) = %d, want 2", len(links))
	}
}
