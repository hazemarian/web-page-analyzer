package steps

import (
	"context"
	"strings"

	"webpage-analyzer/internal/pipeline"
)

// HTMLVersion is Stage 3. Detects the HTML version from the DOCTYPE declaration.
type HTMLVersion struct{}

func (s *HTMLVersion) Name() string { return "html_version" }
func (s *HTMLVersion) Stage() int   { return 3 }

func (s *HTMLVersion) Run(ctx context.Context, state *pipeline.State) error {
	version := detectVersion(state.GetRawHTML())
	state.SetResult(s.Name(), pipeline.StepResult{Status: "done", Data: version})
	return nil
}

func detectVersion(html string) string {
	sample := html
	if len(sample) > 512 {
		sample = sample[:512]
	}
	u := strings.ToUpper(sample)

	switch {
	case strings.Contains(u, "<!DOCTYPE HTML>"):
		return "HTML5"
	case strings.Contains(u, "HTML 4.01") && strings.Contains(u, "STRICT"):
		return "HTML 4.01 Strict"
	case strings.Contains(u, "HTML 4.01") && strings.Contains(u, "TRANSITIONAL"):
		return "HTML 4.01 Transitional"
	case strings.Contains(u, "HTML 4.01") && strings.Contains(u, "FRAMESET"):
		return "HTML 4.01 Frameset"
	case strings.Contains(u, "XHTML 1.0") && strings.Contains(u, "STRICT"):
		return "XHTML 1.0 Strict"
	case strings.Contains(u, "XHTML 1.0") && strings.Contains(u, "TRANSITIONAL"):
		return "XHTML 1.0 Transitional"
	case strings.Contains(u, "XHTML 1.1"):
		return "XHTML 1.1"
	case strings.Contains(u, "HTML 3.2"):
		return "HTML 3.2"
	case strings.Contains(u, "HTML 2.0"):
		return "HTML 2.0"
	case strings.Contains(u, "<!DOCTYPE"):
		return "HTML (unknown version)"
	default:
		return "Unknown"
	}
}
