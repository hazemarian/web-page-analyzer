package steps

import (
	"testing"

	"webpage-analyzer/config"
)

func TestAllReturnsCorrectSteps(t *testing.T) {
	cfg := &config.Config{LinkCheckConcurrency: 10}
	steps := All(cfg)

	expectedNames := []string{
		"url_validation", "fetch_html", "html_version",
		"title", "headings", "login_form", "links", "link_checker",
	}

	if len(steps) != len(expectedNames) {
		t.Fatalf("got %d steps, want %d", len(steps), len(expectedNames))
	}

	for i, name := range expectedNames {
		if steps[i].Name() != name {
			t.Errorf("steps[%d].Name() = %q, want %q", i, steps[i].Name(), name)
		}
	}
}

func TestAllStageAssignment(t *testing.T) {
	cfg := &config.Config{LinkCheckConcurrency: 10}
	steps := All(cfg)

	expectedStages := map[string]int{
		"url_validation": 1,
		"fetch_html":     2,
		"html_version":   3,
		"title":          3,
		"headings":       3,
		"login_form":     3,
		"links":          3,
		"link_checker":   4,
	}

	for _, step := range steps {
		wantStage, ok := expectedStages[step.Name()]
		if !ok {
			t.Errorf("unexpected step %q", step.Name())
			continue
		}
		if step.Stage() != wantStage {
			t.Errorf("step %q: Stage() = %d, want %d", step.Name(), step.Stage(), wantStage)
		}
	}
}
