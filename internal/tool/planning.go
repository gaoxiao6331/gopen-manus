package tool

import (
	"context"
	"errors"
	"fmt"

	"gopen-manus/internal/planning"
)

// PlanningTool exposes plan management commands similar to the Python implementation.
type PlanningTool struct {
	store *planning.Store
}

// NewPlanningTool constructs a planning tool backed by the provided store.
func NewPlanningTool(store *planning.Store) *PlanningTool {
	if store == nil {
		store = planning.NewStore()
	}
	return &PlanningTool{store: store}
}

func (p *PlanningTool) Name() string {
	return "planning"
}

func (p *PlanningTool) Description() string {
	return "A planning tool that allows the agent to create and manage plans for solving complex tasks."
}

func (p *PlanningTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to execute. Available commands: create, update, list, get, set_active, mark_step, delete.",
				"enum":        []string{"create", "update", "list", "get", "set_active", "mark_step", "delete"},
			},
			"plan_id": map[string]any{
				"type":        "string",
				"description": "Unique identifier for the plan. Required for create, update, set_active, and delete commands. Optional for get and mark_step (uses active plan if not specified).",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Title for the plan. Required for create command, optional for update command.",
			},
			"steps": map[string]any{
				"type":        "array",
				"description": "List of plan steps. Required for create command, optional for update command.",
				"items":       map[string]any{"type": "string"},
			},
			"step_index": map[string]any{
				"type":        "integer",
				"description": "Index of the step to update (0-based). Required for mark_step command.",
			},
			"step_status": map[string]any{
				"type":        "string",
				"description": "Status to set for a step. Used with mark_step command.",
				"enum": []string{
					string(planning.StatusNotStarted),
					string(planning.StatusInProgress),
					string(planning.StatusCompleted),
					string(planning.StatusBlocked),
				},
			},
			"step_notes": map[string]any{
				"type":        "string",
				"description": "Additional notes for a step. Optional for mark_step command.",
			},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

func (p *PlanningTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	_ = ctx
	command, err := readString(args, "command")
	if err != nil {
		return Result{Error: err.Error()}, nil
	}

	switch command {
	case "create":
		return p.handleCreate(args)
	case "update":
		return p.handleUpdate(args)
	case "list":
		return p.handleList()
	case "get":
		return p.handleGet(args)
	case "set_active":
		return p.handleSetActive(args)
	case "mark_step":
		return p.handleMarkStep(args)
	case "delete":
		return p.handleDelete(args)
	default:
		return Result{Error: fmt.Sprintf("unrecognized command: %s", command)}, nil
	}
}

func (p *PlanningTool) handleCreate(args map[string]any) (Result, error) {
	planID, err := readString(args, "plan_id")
	if err != nil || planID == "" {
		return Result{Error: "parameter `plan_id` is required for create"}, nil
	}
	title, err := readString(args, "title")
	if err != nil || title == "" {
		return Result{Error: "parameter `title` is required for create"}, nil
	}
	steps, err := readStringSlice(args, "steps")
	if err != nil || len(steps) == 0 {
		return Result{Error: "parameter `steps` must be a non-empty list for create"}, nil
	}

	output, err := p.store.Create(planID, title, steps)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func (p *PlanningTool) handleUpdate(args map[string]any) (Result, error) {
	planID, err := readString(args, "plan_id")
	if err != nil || planID == "" {
		return Result{Error: "parameter `plan_id` is required for update"}, nil
	}
	title, _ := readOptionalString(args, "title")
	steps, _ := readStringSlice(args, "steps")

	output, err := p.store.Update(planID, title, steps)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func (p *PlanningTool) handleList() (Result, error) {
	return Result{Output: p.store.List()}, nil
}

func (p *PlanningTool) handleGet(args map[string]any) (Result, error) {
	planID, _ := readOptionalString(args, "plan_id")
	output, err := p.store.Get(valueOrEmpty(planID))
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func (p *PlanningTool) handleSetActive(args map[string]any) (Result, error) {
	planID, err := readString(args, "plan_id")
	if err != nil || planID == "" {
		return Result{Error: "parameter `plan_id` is required for set_active"}, nil
	}
	output, err := p.store.SetActive(planID)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func (p *PlanningTool) handleMarkStep(args map[string]any) (Result, error) {
	planID, _ := readOptionalString(args, "plan_id")
	index, err := readInt(args, "step_index")
	if err != nil {
		return Result{Error: "parameter `step_index` is required for mark_step"}, nil
	}
	status, statusErr := readOptionalString(args, "step_status")
	if statusErr != nil {
		return Result{Error: statusErr.Error()}, nil
	}
	notes, _ := readOptionalString(args, "step_notes")

	var stepStatus planning.StepStatus
	if status != nil {
		stepStatus = planning.StepStatus(*status)
		if !validStepStatus(stepStatus) {
			return Result{Error: fmt.Sprintf("invalid step_status: %s", *status)}, nil
		}
	}

	output, err := p.store.MarkStep(valueOrEmpty(planID), index, stepStatus, notes)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func (p *PlanningTool) handleDelete(args map[string]any) (Result, error) {
	planID, err := readString(args, "plan_id")
	if err != nil || planID == "" {
		return Result{Error: "parameter `plan_id` is required for delete"}, nil
	}
	output, err := p.store.Delete(planID)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func readString(args map[string]any, key string) (string, error) {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
		return "", fmt.Errorf("parameter `%s` must be a string", key)
	}
	return "", errors.New("missing required parameter")
}

func readOptionalString(args map[string]any, key string) (*string, error) {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return &s, nil
		}
		return nil, fmt.Errorf("parameter `%s` must be a string", key)
	}
	return nil, nil
}

func readStringSlice(args map[string]any, key string) ([]string, error) {
	v, ok := args[key]
	if !ok {
		return nil, errors.New("missing required parameter")
	}
	switch items := v.(type) {
	case []string:
		return items, nil
	case []any:
		result := make([]string, 0, len(items))
		for _, item := range items {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				return nil, fmt.Errorf("parameter `%s` must be a list of strings", key)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("parameter `%s` must be a list of strings", key)
	}
}

func readInt(args map[string]any, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, errors.New("missing required parameter")
	}
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("parameter `%s` must be an integer", key)
	}
}

func validStepStatus(status planning.StepStatus) bool {
	switch status {
	case planning.StatusNotStarted, planning.StatusInProgress, planning.StatusCompleted, planning.StatusBlocked:
		return true
	default:
		return false
	}
}

func valueOrEmpty(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
