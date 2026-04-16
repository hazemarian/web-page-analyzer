package pipeline

import "context"

// Step is the unit of work in the pipeline.
// Each step declares which stage it belongs to.
// Steps within the same stage run concurrently.
// Stages are executed sequentially — a stage only starts after the previous one succeeds.
type Step interface {
	// Name is a unique identifier used as the Redis hash field key.
	Name() string
	// Stage determines execution order. Lower numbers run first.
	Stage() int
	// Run executes the step. It should write its result into state via state.SetResult.
	// Returning a non-nil error stops the entire pipeline (use only for fatal failures).
	// Non-fatal step failures should be recorded in state and return nil.
	Run(ctx context.Context, state *State) error
}

// StepResult holds the outcome of a single step.
type StepResult struct {
	Status string // pending | done | failed
	Data   any
	Error  string
}
