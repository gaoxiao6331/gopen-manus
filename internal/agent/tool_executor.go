package agent

import (
	"context"

	"gopen-manus/internal/llm"
	"gopen-manus/internal/tool"
)

// ToolExecutor provides tool schemas to the LLM and executes tool calls.
type ToolExecutor interface {
	ToParams() []llm.ToolParam
	HasTool(name string) bool
	Execute(ctx context.Context, name string, args map[string]any) (tool.Result, error)
}
