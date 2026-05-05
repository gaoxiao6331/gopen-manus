package tool

import (
	"context"

	"gopen-manus/internal/llm"
)

// Tool represents an executable capability exposed to the LLM.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) (Result, error)
}

// Param converts a Tool into an llm.ToolParam definition.
func Param(t Tool) llm.ToolParam {
	params := t.Parameters()
	if params == nil {
		params = map[string]any{}
	}
	return llm.ToolParam{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  params,
	}
}
