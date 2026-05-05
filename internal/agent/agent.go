package agent

import (
	"context"

	"gopen-manus/internal/schema"
)

// Agent defines the minimal behavior needed by flows.
type Agent interface {
	Run(ctx context.Context, request string) (string, error)
	DescriptionText() string
	StateValue() schema.AgentState
}

// Helpers to satisfy Agent interface on BaseAgent.
func (a *BaseAgent) DescriptionText() string {
	return a.Description
}

func (a *BaseAgent) StateValue() schema.AgentState {
	return a.State
}
