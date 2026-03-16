package steps

import (
	"webpage-analyzer/config"
	"webpage-analyzer/internal/pipeline"
)

// All returns the ordered list of steps that form the analysis pipeline.
// Adding a new analysis step requires only implementing pipeline.Step and registering it here.
func All(cfg *config.Config) []pipeline.Step {
	return []pipeline.Step{
		// Stage 1 — fail-fast validation
		&URLValidation{},
		// Stage 2 — fetch raw HTML
		NewFetchHTML(nil),
		// Stage 3 — concurrent analysis (all read from state.RawHTML)
		&HTMLVersion{},
		&Title{},
		&Headings{},
		&LoginForm{},
		&Links{},
		// Stage 4 — link accessibility (reads state.Links populated by Stage 3)
		NewLinkChecker(cfg.LinkCheckConcurrency, nil),
	}
}
