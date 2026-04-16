package steps

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"webpage-analyzer/internal/pipeline"
)

func TestCountHeadings(t *testing.T) {
	tests := []struct {
		name    string
		htmlStr string
		want    map[string]int
	}{
		{
			"mixed levels",
			"<html><body><h1>A</h1><h2>B</h2><h2>C</h2><h3>D</h3></body></html>",
			map[string]int{"h1": 1, "h2": 2, "h3": 1, "h4": 0, "h5": 0, "h6": 0},
		},
		{
			"no headings",
			"<html><body><p>No headings here</p></body></html>",
			map[string]int{"h1": 0, "h2": 0, "h3": 0, "h4": 0, "h5": 0, "h6": 0},
		},
		{
			"all levels",
			"<html><body><h1>1</h1><h2>2</h2><h3>3</h3><h4>4</h4><h5>5</h5><h6>6</h6></body></html>",
			map[string]int{"h1": 1, "h2": 1, "h3": 1, "h4": 1, "h5": 1, "h6": 1},
		},
		{
			"nested headings",
			"<html><body><div><h1>A</h1><div><h1>B</h1></div></div></body></html>",
			map[string]int{"h1": 2, "h2": 0, "h3": 0, "h4": 0, "h5": 0, "h6": 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.htmlStr))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			got := countHeadings(doc)
			for level, wantCount := range tt.want {
				if got[level] != wantCount {
					t.Errorf("%s: got %d, want %d", level, got[level], wantCount)
				}
			}
		})
	}
}

func TestHeadingsRun(t *testing.T) {
	state := pipeline.NewState("example.com")
	state.SetRawHTML("<html><body><h1>Title</h1><h2>Sub</h2></body></html>")

	step := &Headings{}
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("headings")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "done" {
		t.Errorf("Status = %q, want done", result.Status)
	}

	headings, ok := result.Data.(map[string]int)
	if !ok {
		t.Fatalf("Data type = %T, want map[string]int", result.Data)
	}
	if headings["h1"] != 1 {
		t.Errorf("h1 = %d, want 1", headings["h1"])
	}
	if headings["h2"] != 1 {
		t.Errorf("h2 = %d, want 1", headings["h2"])
	}
}
