package agent

import (
	"context"
	"errors"

	"gopen-manus/internal/llm"
	"gopen-manus/internal/schema"
)

// ReActAgent defines the think/act loop.
type ReActAgent struct {
	*BaseAgent

	ThinkFunc func(context.Context) (bool, error)
	ActFunc   func(context.Context) (string, error)
}

func NewReActAgent(name string) *ReActAgent {
	base := NewBaseAgent(name)
	ra := &ReActAgent{BaseAgent: base}
	base.SetStepFunc(ra.Step)
	return ra
}

func (r *ReActAgent) Initialize() {
	r.BaseAgent.Initialize()
	if r.LLM == nil {
		r.LLM = &llm.NoopLLM{}
	}
	if r.Memory == nil {
		r.Memory = schema.NewMemory()
	}
}

func (r *ReActAgent) Step(ctx context.Context) (string, error) {
	if r.ThinkFunc == nil || r.ActFunc == nil {
		return "", errors.New("think/act functions not set")
	}
	shouldAct, err := r.ThinkFunc(ctx)
	if err != nil {
		return "", err
	}
	if !shouldAct {
		return "Thinking complete - no action needed", nil
	}
	return r.ActFunc(ctx)
}
