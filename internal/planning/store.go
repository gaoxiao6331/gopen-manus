package planning

import (
	"errors"
	"fmt"
	"strings"
)

// StepStatus represents a plan step status.
type StepStatus string

const (
	StatusNotStarted StepStatus = "not_started"
	StatusInProgress StepStatus = "in_progress"
	StatusCompleted  StepStatus = "completed"
	StatusBlocked    StepStatus = "blocked"
)

// Plan represents a plan.
type Plan struct {
	PlanID       string
	Title        string
	Steps        []string
	StepStatuses []StepStatus
	StepNotes    []string
}

// Store manages plans in memory.
type Store struct {
	plans         map[string]*Plan
	currentPlanID string
}

func NewStore() *Store {
	return &Store{plans: map[string]*Plan{}}
}

func (s *Store) Create(planID, title string, steps []string) (string, error) {
	if planID == "" {
		return "", errors.New("parameter `plan_id` is required for create")
	}
	if _, exists := s.plans[planID]; exists {
		return "", fmt.Errorf("plan '%s' already exists", planID)
	}
	if title == "" {
		return "", errors.New("parameter `title` is required for create")
	}
	if len(steps) == 0 {
		return "", errors.New("parameter `steps` must be a non-empty list for create")
	}

	plan := &Plan{
		PlanID:       planID,
		Title:        title,
		Steps:        steps,
		StepStatuses: make([]StepStatus, len(steps)),
		StepNotes:    make([]string, len(steps)),
	}
	for i := range plan.StepStatuses {
		plan.StepStatuses[i] = StatusNotStarted
	}

	s.plans[planID] = plan
	s.currentPlanID = planID
	return s.format(plan), nil
}

func (s *Store) Update(planID string, title *string, steps []string) (string, error) {
	if planID == "" {
		return "", errors.New("parameter `plan_id` is required for update")
	}
	plan, ok := s.plans[planID]
	if !ok {
		return "", fmt.Errorf("no plan found with ID: %s", planID)
	}
	if title != nil && *title != "" {
		plan.Title = *title
	}
	if len(steps) > 0 {
		oldSteps := plan.Steps
		oldStatuses := plan.StepStatuses
		oldNotes := plan.StepNotes

		newStatuses := make([]StepStatus, 0, len(steps))
		newNotes := make([]string, 0, len(steps))
		for i, step := range steps {
			if i < len(oldSteps) && step == oldSteps[i] {
				newStatuses = append(newStatuses, oldStatuses[i])
				newNotes = append(newNotes, oldNotes[i])
			} else {
				newStatuses = append(newStatuses, StatusNotStarted)
				newNotes = append(newNotes, "")
			}
		}
		plan.Steps = steps
		plan.StepStatuses = newStatuses
		plan.StepNotes = newNotes
	}

	return s.format(plan), nil
}

func (s *Store) List() string {
	if len(s.plans) == 0 {
		return "No plans available. Create a plan with the 'create' command."
	}
	var b strings.Builder
	b.WriteString("Available plans:\n")
	for planID, plan := range s.plans {
		current := ""
		if planID == s.currentPlanID {
			current = " (active)"
		}
		completed := 0
		for _, st := range plan.StepStatuses {
			if st == StatusCompleted {
				completed++
			}
		}
		fmt.Fprintf(&b, "• %s%s: %s - %d/%d steps completed\n", planID, current, plan.Title, completed, len(plan.Steps))
	}
	return b.String()
}

func (s *Store) Get(planID string) (string, error) {
	if planID == "" {
		if s.currentPlanID == "" {
			return "", errors.New("no active plan")
		}
		planID = s.currentPlanID
	}
	plan, ok := s.plans[planID]
	if !ok {
		return "", fmt.Errorf("no plan found with ID: %s", planID)
	}
	return s.format(plan), nil
}

func (s *Store) SetActive(planID string) (string, error) {
	if planID == "" {
		return "", errors.New("parameter `plan_id` is required for set_active")
	}
	plan, ok := s.plans[planID]
	if !ok {
		return "", fmt.Errorf("no plan found with ID: %s", planID)
	}
	s.currentPlanID = planID
	return fmt.Sprintf("Plan '%s' is now the active plan.\n\n%s", planID, s.format(plan)), nil
}

func (s *Store) MarkStep(planID string, stepIndex int, stepStatus StepStatus, stepNotes *string) (string, error) {
	if planID == "" {
		if s.currentPlanID == "" {
			return "", errors.New("no active plan")
		}
		planID = s.currentPlanID
	}
	plan, ok := s.plans[planID]
	if !ok {
		return "", fmt.Errorf("no plan found with ID: %s", planID)
	}
	if stepIndex < 0 || stepIndex >= len(plan.Steps) {
		return "", fmt.Errorf("invalid step_index: %d", stepIndex)
	}
	if stepStatus != "" {
		plan.StepStatuses[stepIndex] = stepStatus
	}
	if stepNotes != nil {
		plan.StepNotes[stepIndex] = *stepNotes
	}
	return fmt.Sprintf("Step %d updated in plan '%s'.\n\n%s", stepIndex, planID, s.format(plan)), nil
}

func (s *Store) Delete(planID string) (string, error) {
	if planID == "" {
		return "", errors.New("parameter `plan_id` is required for delete")
	}
	if _, ok := s.plans[planID]; !ok {
		return "", fmt.Errorf("no plan found with ID: %s", planID)
	}
	delete(s.plans, planID)
	if s.currentPlanID == planID {
		s.currentPlanID = ""
	}
	return fmt.Sprintf("Plan '%s' has been deleted.", planID), nil
}

func (s *Store) format(plan *Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Plan: %s (ID: %s)\n", plan.Title, plan.PlanID)
	b.WriteString(strings.Repeat("=", b.Len()))
	b.WriteString("\n\n")

	total := len(plan.Steps)
	completed := 0
	inProgress := 0
	blocked := 0
	notStarted := 0
	for _, st := range plan.StepStatuses {
		switch st {
		case StatusCompleted:
			completed++
		case StatusInProgress:
			inProgress++
		case StatusBlocked:
			blocked++
		default:
			notStarted++
		}
	}
	percentage := 0.0
	if total > 0 {
		percentage = float64(completed) / float64(total) * 100
	}
	fmt.Fprintf(&b, "Progress: %d/%d steps completed (%.1f%%)\n", completed, total, percentage)
	fmt.Fprintf(&b, "Status: %d completed, %d in progress, %d blocked, %d not started\n\n", completed, inProgress, blocked, notStarted)
	b.WriteString("Steps:\n")

	for i, step := range plan.Steps {
		status := StatusNotStarted
		if i < len(plan.StepStatuses) {
			status = plan.StepStatuses[i]
		}
		mark := map[StepStatus]string{
			StatusNotStarted: "[ ]",
			StatusInProgress: "[→]",
			StatusCompleted:  "[✓]",
			StatusBlocked:    "[!]",
		}[status]
		fmt.Fprintf(&b, "%d. %s %s\n", i, mark, step)
		if i < len(plan.StepNotes) && plan.StepNotes[i] != "" {
			fmt.Fprintf(&b, "   Notes: %s\n", plan.StepNotes[i])
		}
	}
	return b.String()
}
