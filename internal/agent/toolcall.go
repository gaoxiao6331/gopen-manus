package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopen-manus/internal/llm"
	"gopen-manus/internal/logger"
	"gopen-manus/internal/schema"
	"gopen-manus/internal/tool"
)

const toolCallRequired = "Tool calls required but none provided"

// ToolCallAgent handles tool/function calls.
type ToolCallAgent struct {
	*ReActAgent

	AvailableTools   ToolExecutor
	ToolChoice       schema.ToolChoice
	SpecialToolNames []string
	ToolCalls        []schema.ToolCall
	CurrentImage     *string
	MaxObserve       int
}

func NewToolCallAgent(name string) *ToolCallAgent {
	ra := NewReActAgent(name)
	ta := &ToolCallAgent{
		ReActAgent:       ra,
		ToolChoice:       schema.ToolChoiceAuto,
		SpecialToolNames: []string{"terminate"},
		MaxObserve:       0,
	}
	ta.ThinkFunc = ta.Think
	ta.ActFunc = ta.Act
	return ta
}

func (t *ToolCallAgent) Think(ctx context.Context) (bool, error) {
	if t.NextStepPrompt != "" {
		msg := schema.UserMessage(t.NextStepPrompt, nil)
		t.Memory.AddMessage(msg)
	}

	var toolParams []llm.ToolParam
	if t.AvailableTools != nil {
		toolParams = t.AvailableTools.ToParams()
	}

	response, err := t.LLM.AskTool(ctx, t.Messages(), systemMessages(t.SystemPrompt), toolParams, t.ToolChoice)
	if err != nil {
		return false, err
	}

	if response == nil {
		return false, errors.New("no response received from the LLM")
	}

	content := response.Content
	t.ToolCalls = response.ToolCalls

	logger.Info.Printf("%s's thoughts: %s", t.Name, content)
	logger.Info.Printf("%s selected %d tools", t.Name, len(t.ToolCalls))

	if t.ToolChoice == schema.ToolChoiceNone {
		if len(t.ToolCalls) > 0 {
			logger.Warn.Printf("%s tried to use tools when unavailable", t.Name)
		}
		if content != "" {
			t.Memory.AddMessage(schema.AssistantMessage(&content, nil))
			return true, nil
		}
		return false, nil
	}

	assistantMsg := schema.AssistantMessage(nil, nil)
	if len(t.ToolCalls) > 0 {
		assistantMsg = schema.FromToolCalls(t.ToolCalls, strPtr(content), nil)
	} else if content != "" {
		assistantMsg = schema.AssistantMessage(&content, nil)
	}
	if assistantMsg.Content != nil || len(assistantMsg.ToolCalls) > 0 {
		t.Memory.AddMessage(assistantMsg)
	}

	if t.ToolChoice == schema.ToolChoiceRequired && len(t.ToolCalls) == 0 {
		return true, nil
	}
	if t.ToolChoice == schema.ToolChoiceAuto && len(t.ToolCalls) == 0 {
		return content != "", nil
	}

	return len(t.ToolCalls) > 0, nil
}

func (t *ToolCallAgent) Act(ctx context.Context) (string, error) {
	if len(t.ToolCalls) == 0 {
		if t.ToolChoice == schema.ToolChoiceRequired {
			return "", errors.New(toolCallRequired)
		}
		msgs := t.Messages()
		if len(msgs) == 0 || msgs[len(msgs)-1].Content == nil {
			return "No content or commands to execute", nil
		}
		return *msgs[len(msgs)-1].Content, nil
	}

	results := []string{}
	for _, command := range t.ToolCalls {
		t.CurrentImage = nil
		result, err := t.ExecuteTool(ctx, command)
		if err != nil {
			result = fmt.Sprintf("Error: %s", err)
		}
		if t.MaxObserve > 0 && len(result) > t.MaxObserve {
			result = result[:t.MaxObserve]
		}

		logger.Info.Printf("Tool '%s' result: %s", command.Function.Name, result)
		toolMsg := schema.ToolMessage(result, command.Function.Name, command.ID, t.CurrentImage)
		t.Memory.AddMessage(toolMsg)
		results = append(results, result)
	}

	return strings.Join(results, "\n\n"), nil
}

func (t *ToolCallAgent) ExecuteTool(ctx context.Context, command schema.ToolCall) (string, error) {
	if command.Function.Name == "" {
		return "", errors.New("invalid command format")
	}
	if t.AvailableTools == nil {
		return "", errors.New("no tool executor configured")
	}
	if !t.AvailableTools.HasTool(command.Function.Name) {
		return "", fmt.Errorf("unknown tool '%s'", command.Function.Name)
	}

	args := map[string]any{}
	if command.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(command.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error parsing arguments for %s: %v", command.Function.Name, err)
		}
	}

	result, err := t.AvailableTools.Execute(ctx, command.Function.Name, args)
	if err != nil {
		return "", err
	}
	if result.Base64Image != nil {
		t.CurrentImage = result.Base64Image
	}

	if t.isSpecialTool(command.Function.Name) {
		t.State = schema.AgentStateFinished
	}

	observation := ""
	if result.Error != "" {
		return "", errors.New(result.Error)
	}
	if result.Output != "" {
		observation = fmt.Sprintf("Observed output of cmd `%s` executed:\n%s", command.Function.Name, result.Output)
	} else {
		observation = fmt.Sprintf("Cmd `%s` completed with no output", command.Function.Name)
	}

	return observation, nil
}

func (t *ToolCallAgent) isSpecialTool(name string) bool {
	for _, n := range t.SpecialToolNames {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

func systemMessages(systemPrompt string) []schema.Message {
	if systemPrompt == "" {
		return nil
	}
	return []schema.Message{schema.SystemMessage(systemPrompt)}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ = tool.Result{}
