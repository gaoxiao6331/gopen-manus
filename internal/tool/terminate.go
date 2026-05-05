package tool

import "context"

// TerminateTool ends the agent interaction when invoked.
type TerminateTool struct{}

func (t *TerminateTool) Name() string {
	return "terminate"
}

func (t *TerminateTool) Description() string {
	return "Terminate the interaction when the request is met OR if the assistant cannot proceed further with the task."
}

func (t *TerminateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type":        "string",
				"description": "The finish status of the interaction.",
				"enum":        []string{"success", "failure"},
			},
		},
		"required": []string{"status"},
	}
}

func (t *TerminateTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	_ = ctx
	status, ok := args["status"].(string)
	if !ok || status == "" {
		return Result{Error: "parameter `status` is required"}, nil
	}
	return Result{Output: "The interaction has been completed with status: " + status}, nil
}
