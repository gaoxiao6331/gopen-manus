package flow

import (
	"fmt"

	"gopen-manus/internal/agent"
)

// FlowType defines available flow types.
type FlowType string

const (
	FlowTypePlanning FlowType = "planning"
)

// Factory creates flows by type.
type Factory struct{}

func (f *Factory) Create(flowType FlowType, agentsInput any, kwargs map[string]any) (Executor, error) {
	switch flowType {
	case FlowTypePlanning:
		executors := []string{}
		planID := ""
		if v, ok := kwargs["executors"]; ok {
			if list, ok := v.([]string); ok {
				executors = list
			}
		}
		if v, ok := kwargs["plan_id"]; ok {
			if s, ok := v.(string); ok {
				planID = s
			}
		}
		flow, err := NewPlanningFlow(agentsInput, executors, planID)
		if err != nil {
			return nil, err
		}
		return flow, nil
	default:
		return nil, fmt.Errorf("unsupported flow type: %s", flowType)
	}
}

// Helper to align with Python-style agent lists.
func AgentsToAny(agents []agent.Agent) any {
	return agents
}
