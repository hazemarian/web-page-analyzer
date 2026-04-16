package steps

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		htmlStr  string
		want     string
	}{
		{
			"normal title",
			"<html><head><title>My Page</title></head><body></body></html>",
			"My Page",
		},
		{
			"empty title",
			"<html><head><title></title></head><body></body></html>",
			"",
		},
		{
			"no title tag",
			"<html><head></head><body></body></html>",
			"",
		},
		{
			"title with whitespace",
			"<html><head><title>  Hello World  </title></head></html>",
			"Hello World",
		},
		{
			"title with special chars",
			"<html><head><title>Hello &amp; World</title></head></html>",
			"Hello & World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.htmlStr))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}
			got := extractTitle(doc)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTitleRun(t *testing.T) {
	state := pipeline.NewState("example.com")
	state.SetRawHTML("<html><head><title>Test Page</title></head></html>")

	step := &Title{}
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("title")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "done" {
		t.Errorf("Status = %q, want done", result.Status)
	}
	if result.Data != "Test Page" {
		t.Errorf("Data = %v, want %q", result.Data, "Test Page")
	}
}
