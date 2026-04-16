package pipeline

import (
	"context"
	"sort"

	"golang.org/x/sync/errgroup"
)

// StepCallback is called after each step completes (success or failure).
// It receives the step name and the result written into state by that step.
// Callbacks from steps within the same stage may be called concurrently.
type StepCallback func(ctx context.Context, stepName string, result StepResult)

// Pipeline runs steps in stage order. Steps within a stage run concurrently.
type Pipeline struct {
	steps    []Step
	callback StepCallback
}

func New(steps ...Step) *Pipeline {
	return &Pipeline{steps: steps}
}

// WithCallback attaches a hook called after every step completes.
func (p *Pipeline) WithCallback(cb StepCallback) *Pipeline {
	p.callback = cb
	return p
}

// Run executes all stages in order. Returns on the first stage that fails.
func (p *Pipeline) Run(ctx context.Context, state *State) error {
	for _, stageSteps := range p.groupByStage() {
		if err := p.runStage(ctx, stageSteps, state); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) runStage(ctx context.Context, steps []Step, state *State) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, step := range steps {
		step := step
		g.Go(func() error {
			err := step.Run(gCtx, state)

			if p.callback != nil {
				result, _ := state.GetResult(step.Name())
				p.callback(ctx, step.Name(), result)
			}

			return err
		})
	}

	return g.Wait()
}

func (p *Pipeline) groupByStage() [][]Step {
	stageMap := make(map[int][]Step)
	for _, s := range p.steps {
		stageMap[s.Stage()] = append(stageMap[s.Stage()], s)
	}

	keys := make([]int, 0, len(stageMap))
	for k := range stageMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	result := make([][]Step, 0, len(keys))
	for _, k := range keys {
		result = append(result, stageMap[k])
	}
	return result
}
