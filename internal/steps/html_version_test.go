package steps

import (
	"context"
	"testing"

	"webpage-analyzer/internal/pipeline"
)

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"HTML5", "<!DOCTYPE html><html><head></head><body></body></html>", "HTML5"},
		{"HTML5 uppercase", "<!DOCTYPE HTML><html></html>", "HTML5"},
		{
			"HTML 4.01 Strict",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`,
			"HTML 4.01 Strict",
		},
		{
			"HTML 4.01 Transitional",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">`,
			"HTML 4.01 Transitional",
		},
		{
			"HTML 4.01 Frameset",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN">`,
			"HTML 4.01 Frameset",
		},
		{
			"XHTML 1.0 Strict",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN">`,
			"XHTML 1.0 Strict",
		},
		{
			"XHTML 1.0 Transitional",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN">`,
			"XHTML 1.0 Transitional",
		},
		{
			"XHTML 1.1",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN">`,
			"XHTML 1.1",
		},
		{
			"HTML 3.2",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">`,
			"HTML 3.2",
		},
		{
			"HTML 2.0",
			`<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">`,
			"HTML 2.0",
		},
		{
			"Unknown DOCTYPE",
			`<!DOCTYPE something-else>`,
			"HTML (unknown version)",
		},
		{
			"No DOCTYPE",
			`<html><head><title>test</title></head></html>`,
			"Unknown",
		},
		{
			"Empty",
			"",
			"Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectVersion(tt.html)
			if got != tt.want {
				t.Errorf("detectVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHTMLVersionRun(t *testing.T) {
	state := pipeline.NewState("example.com")
	state.SetRawHTML("<!DOCTYPE html><html></html>")

	step := &HTMLVersion{}
	if err := step.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := state.GetResult("html_version")
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "done" {
		t.Errorf("Status = %q, want done", result.Status)
	}
	if result.Data != "HTML5" {
		t.Errorf("Data = %v, want HTML5", result.Data)
	}
}
