package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gopen-manus/internal/llm"
	"gopen-manus/internal/logger"
	"gopen-manus/internal/schema"
)

// BaseAgent provides the core execution loop and memory management.
type BaseAgent struct {
	Name               string
	Description        string
	SystemPrompt       string
	NextStepPrompt     string
	LLM                llm.Client
	Memory             *schema.Memory
	State              schema.AgentState
	MaxSteps           int
	CurrentStep        int
	DuplicateThreshold int

	stepFunc func(context.Context) (string, error)
}

func NewBaseAgent(name string) *BaseAgent {
	return &BaseAgent{
		Name:               name,
		LLM:                &llm.NoopLLM{},
		Memory:             schema.NewMemory(),
		State:              schema.AgentStateIdle,
		MaxSteps:           10,
		CurrentStep:        0,
		DuplicateThreshold: 2,
	}
}

func (a *BaseAgent) Initialize() {
	if a.LLM == nil {
		a.LLM = &llm.NoopLLM{}
	}
	if a.Memory == nil {
		a.Memory = schema.NewMemory()
	}
	if a.DuplicateThreshold <= 0 {
		a.DuplicateThreshold = 2
	}
	if a.MaxSteps <= 0 {
		a.MaxSteps = 10
	}
}

func (a *BaseAgent) WithState(newState schema.AgentState, fn func() error) (err error) {
	switch newState {
	case schema.AgentStateIdle, schema.AgentStateRunning, schema.AgentStateFinished, schema.AgentStateError:
		// ok
	default:
		return fmt.Errorf("invalid state: %s", newState)
	}

	previous := a.State
	a.State = newState
	defer func() {
		if r := recover(); r != nil {
			a.State = schema.AgentStateError
			panic(r)
		}
		if err != nil {
			a.State = schema.AgentStateError
		}
		a.State = previous
	}()

	err = fn()
	return err
}

func (a *BaseAgent) UpdateMemory(role schema.Role, content string, base64Image *string, extra map[string]string) error {
	if err := schema.ValidateRole(role); err != nil {
		return err
	}

	switch role {
	case schema.RoleUser:
		a.Memory.AddMessage(schema.UserMessage(content, base64Image))
	case schema.RoleSystem:
		a.Memory.AddMessage(schema.SystemMessage(content))
	case schema.RoleAssistant:
		a.Memory.AddMessage(schema.AssistantMessage(&content, base64Image))
	case schema.RoleTool:
		name := ""
		toolCallID := ""
		if extra != nil {
			name = extra["name"]
			toolCallID = extra["tool_call_id"]
		}
		a.Memory.AddMessage(schema.ToolMessage(content, name, toolCallID, base64Image))
	}
	return nil
}

func (a *BaseAgent) Messages() []schema.Message {
	return a.Memory.Messages
}

func (a *BaseAgent) SetStepFunc(step func(context.Context) (string, error)) {
	a.stepFunc = step
}

func (a *BaseAgent) Run(ctx context.Context, request string) (string, error) {
	a.Initialize()
	if a.State != schema.AgentStateIdle {
		return "", fmt.Errorf("cannot run agent from state: %s", a.State)
	}
	if request != "" {
		_ = a.UpdateMemory(schema.RoleUser, request, nil, nil)
	}

	results := []string{}
	err := a.WithState(schema.AgentStateRunning, func() error {
		for a.CurrentStep < a.MaxSteps && a.State != schema.AgentStateFinished {
			a.CurrentStep++
			logger.Info.Printf("Executing step %d/%d", a.CurrentStep, a.MaxSteps)
			if a.stepFunc == nil {
				return errors.New("no step function provided")
			}
			stepResult, err := a.stepFunc(ctx)
			if err != nil {
				return err
			}
			if a.IsStuck() {
				a.HandleStuckState()
			}
			results = append(results, fmt.Sprintf("Step %d: %s", a.CurrentStep, stepResult))
		}

		if a.CurrentStep >= a.MaxSteps {
			a.CurrentStep = 0
			a.State = schema.AgentStateIdle
			results = append(results, fmt.Sprintf("Terminated: Reached max steps (%d)", a.MaxSteps))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No steps executed", nil
	}
	return strings.Join(results, "\n"), nil
}

func (a *BaseAgent) HandleStuckState() {
	stuckPrompt := "Observed duplicate responses. Consider new strategies and avoid repeating ineffective paths already attempted."
	if a.NextStepPrompt != "" {
		a.NextStepPrompt = stuckPrompt + "\n" + a.NextStepPrompt
	} else {
		a.NextStepPrompt = stuckPrompt
	}
	logger.Warn.Printf("Agent detected stuck state. Added prompt: %s", stuckPrompt)
}

func (a *BaseAgent) IsStuck() bool {
	msgs := a.Memory.Messages
	if len(msgs) < 2 {
		return false
	}
	last := msgs[len(msgs)-1]
	if last.Content == nil || *last.Content == "" {
		return false
	}
	duplicateCount := 0
	for i := len(msgs) - 2; i >= 0; i-- {
		msg := msgs[i]
		if msg.Role == schema.RoleAssistant && msg.Content != nil && *msg.Content == *last.Content {
			duplicateCount++
		}
	}
	return duplicateCount >= a.DuplicateThreshold
}
