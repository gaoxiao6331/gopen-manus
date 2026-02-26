package llm

import (
	"context"

	"gopen-manus/internal/schema"
)

// ToolParam is a minimal tool schema placeholder for future integration.
type ToolParam struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Response represents an LLM response for tool-enabled calls.
type Response struct {
	Content   string
	ToolCalls []schema.ToolCall
}

// Client defines the interface an LLM must implement.
type Client interface {
	Ask(ctx context.Context, messages []schema.Message, systemMsgs []schema.Message) (string, error)
	AskTool(ctx context.Context, messages []schema.Message, systemMsgs []schema.Message, tools []ToolParam, toolChoice schema.ToolChoice) (*Response, error)
}

// NoopLLM is a default client that returns empty responses.
type NoopLLM struct{}

func (n *NoopLLM) Ask(ctx context.Context, messages []schema.Message, systemMsgs []schema.Message) (string, error) {
	return "", nil
}

func (n *NoopLLM) AskTool(ctx context.Context, messages []schema.Message, systemMsgs []schema.Message, tools []ToolParam, toolChoice schema.ToolChoice) (*Response, error) {
	return &Response{Content: "", ToolCalls: nil}, nil
}
