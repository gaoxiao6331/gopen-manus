package flow

import (
	"context"
	"errors"

	"gopen-manus/internal/agent"
)

// BaseFlow provides a shared structure for execution flows.
type BaseFlow struct {
	Agents           map[string]agent.Agent
	PrimaryAgentKey  string
	Tools            []any
	primaryAgentInit bool
}

func NewBaseFlow(agentsInput any, primaryKey string) (*BaseFlow, error) {
	agents := map[string]agent.Agent{}
	switch v := agentsInput.(type) {
	case agent.Agent:
		agents["default"] = v
	case []agent.Agent:
		for i, ag := range v {
			key := "agent_" + itoa(i)
			agents[key] = ag
		}
	case map[string]agent.Agent:
		agents = v
	default:
		return nil, errors.New("unsupported agents input")
	}

	if primaryKey == "" {
		for key := range agents {
			primaryKey = key
			break
		}
	}

	return &BaseFlow{
		Agents:          agents,
		PrimaryAgentKey: primaryKey,
	}, nil
}

func (b *BaseFlow) PrimaryAgent() agent.Agent {
	if b == nil {
		return nil
	}
	return b.Agents[b.PrimaryAgentKey]
}

func (b *BaseFlow) GetAgent(key string) agent.Agent {
	return b.Agents[key]
}

func (b *BaseFlow) AddAgent(key string, ag agent.Agent) {
	b.Agents[key] = ag
}

// Execute should be implemented by concrete flows.
type Executor interface {
	Execute(ctx context.Context, inputText string) (string, error)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
