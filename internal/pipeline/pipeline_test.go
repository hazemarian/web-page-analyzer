package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// mockStep is a test helper implementing Step.
type mockStep struct {
	name  string
	stage int
	run   func(ctx context.Context, state *State) error
}

func (m *mockStep) Name() string { return m.name }
func (m *mockStep) Stage() int   { return m.stage }
func (m *mockStep) Run(ctx context.Context, state *State) error {
	return m.run(ctx, state)
}

func TestPipelineStagesInOrder(t *testing.T) {
	var order []int

	s1 := &mockStep{name: "s1", stage: 1, run: func(_ context.Context, _ *State) error {
		order = append(order, 1)
		return nil
	}}
	s2 := &mockStep{name: "s2", stage: 2, run: func(_ context.Context, _ *State) error {
		order = append(order, 2)
		return nil
	}}
	s3 := &mockStep{name: "s3", stage: 3, run: func(_ context.Context, _ *State) error {
		order = append(order, 3)
		return nil
	}}

	p := New(s3, s1, s2) // deliberately out of order
	state := NewState("example.com")

	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(order))
	}
	for i, want := range []int{1, 2, 3} {
		if order[i] != want {
			t.Errorf("order[%d] = %d, want %d", i, order[i], want)
		}
	}
}

func TestPipelineConcurrentWithinStage(t *testing.T) {
	var count atomic.Int32

	makeStep := func(name string) *mockStep {
		return &mockStep{name: name, stage: 1, run: func(_ context.Context, _ *State) error {
			count.Add(1)
			return nil
		}}
	}

	p := New(makeStep("a"), makeStep("b"), makeStep("c"))
	state := NewState("example.com")

	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := count.Load(); got != 3 {
		t.Errorf("count = %d, want 3", got)
	}
}

func TestPipelineStopsOnError(t *testing.T) {
	stage2Ran := false
	errBoom := errors.New("boom")

	s1 := &mockStep{name: "s1", stage: 1, run: func(_ context.Context, _ *State) error {
		return errBoom
	}}
	s2 := &mockStep{name: "s2", stage: 2, run: func(_ context.Context, _ *State) error {
		stage2Ran = true
		return nil
	}}

	p := New(s1, s2)
	state := NewState("example.com")

	err := p.Run(context.Background(), state)
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom, got %v", err)
	}
	if stage2Ran {
		t.Error("stage 2 should not run when stage 1 fails")
	}
}

func TestPipelineCallback(t *testing.T) {
	var called []string

	s1 := &mockStep{name: "step1", stage: 1, run: func(_ context.Context, state *State) error {
		state.SetResult("step1", StepResult{Status: "done", Data: "ok"})
		return nil
	}}

	p := New(s1).WithCallback(func(_ context.Context, name string, result StepResult) {
		called = append(called, name)
	})
	state := NewState("example.com")

	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(called) != 1 || called[0] != "step1" {
		t.Errorf("callback called = %v, want [step1]", called)
	}
}

func TestPipelineEmpty(t *testing.T) {
	p := New()
	state := NewState("example.com")

	if err := p.Run(context.Background(), state); err != nil {
		t.Fatalf("empty pipeline should not error, got %v", err)
	}
}
