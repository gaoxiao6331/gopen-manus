package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopen-manus/internal/agent"
	"gopen-manus/internal/llm"
	"gopen-manus/internal/logger"
	"gopen-manus/internal/planning"
	"gopen-manus/internal/schema"
	"gopen-manus/internal/tool"
)

// PlanningFlow manages planning and execution using agents.
type PlanningFlow struct {
	*BaseFlow
	LLM              llm.Client
	PlanStore        *planning.Store
	PlanningTool     *tool.PlanningTool
	ExecutorKeys     []string
	ActivePlanID     string
	CurrentStepIndex *int
}

func NewPlanningFlow(agentsInput any, executorKeys []string, planID string) (*PlanningFlow, error) {
	base, err := NewBaseFlow(agentsInput, "")
	if err != nil {
		return nil, err
	}
	if planID == "" {
		planID = fmt.Sprintf("plan_%d", time.Now().Unix())
	}
	store := planning.NewStore()
	planningTool := tool.NewPlanningTool(store)
	pf := &PlanningFlow{
		BaseFlow:     base,
		LLM:          &llm.NoopLLM{},
		PlanStore:    store,
		PlanningTool: planningTool,
		ExecutorKeys: executorKeys,
		ActivePlanID: planID,
	}
	if len(pf.ExecutorKeys) == 0 {
		for key := range pf.Agents {
			pf.ExecutorKeys = append(pf.ExecutorKeys, key)
		}
	}
	return pf, nil
}

func (p *PlanningFlow) getExecutor(stepType string) agent.Agent {
	if stepType != "" {
		if ag, ok := p.Agents[stepType]; ok {
			return ag
		}
	}
	for _, key := range p.ExecutorKeys {
		if ag, ok := p.Agents[key]; ok {
			return ag
		}
	}
	return p.PrimaryAgent()
}

func (p *PlanningFlow) Execute(ctx context.Context, inputText string) (string, error) {
	if p.PrimaryAgent() == nil {
		return "", fmt.Errorf("no primary agent available")
	}
	if inputText != "" {
		if err := p.createInitialPlan(ctx, inputText); err != nil {
			logger.Error.Printf("Plan creation failed: %s", err)
			return fmt.Sprintf("Failed to create plan for: %s", inputText), nil
		}
	}

	var result strings.Builder
	for {
		idx, stepInfo := p.getCurrentStepInfo()
		p.CurrentStepIndex = idx
		if idx == nil {
			final, _ := p.finalizePlan(ctx)
			result.WriteString(final)
			break
		}
		stepType := ""
		if stepInfo != nil {
			stepType = stepInfo["type"]
		}
		executor := p.getExecutor(stepType)
		stepResult, _ := p.executeStep(ctx, executor, stepInfo)
		result.WriteString(stepResult)
		result.WriteString("\n")
		if executor != nil && executor.StateValue() == schema.AgentStateFinished {
			break
		}
	}
	return strings.TrimSpace(result.String()), nil
}

func (p *PlanningFlow) createInitialPlan(ctx context.Context, request string) error {
	if p.PlanningTool == nil || p.LLM == nil {
		return p.createDefaultPlan(request)
	}

	systemContent := "You are a planning assistant. Create a concise, actionable plan with clear steps. Focus on key milestones rather than detailed sub-steps. Optimize for clarity and efficiency."
	if len(p.ExecutorKeys) > 1 {
		agentsDescription := []map[string]string{}
		for _, key := range p.ExecutorKeys {
			if ag, ok := p.Agents[key]; ok {
				agentsDescription = append(agentsDescription, map[string]string{
					"name":        strings.ToUpper(key),
					"description": ag.DescriptionText(),
				})
			}
		}
		if len(agentsDescription) > 0 {
			agentInfo, _ := json.Marshal(agentsDescription)
			systemContent += "\nNow we have multiple agents available: " + string(agentInfo) + ". When creating steps, annotate the responsible agent using the format '[agent_name]'."
		}
	}

	systemMsg := schema.SystemMessage(systemContent)
	userMsg := schema.UserMessage("Create a reasonable plan with clear steps to accomplish the task: "+request, nil)
	tools := []llm.ToolParam{tool.Param(p.PlanningTool)}

	resp, err := p.LLM.AskTool(ctx, []schema.Message{userMsg}, []schema.Message{systemMsg}, tools, schema.ToolChoiceAuto)
	if err != nil {
		logger.Warn.Printf("Plan creation via LLM failed: %v", err)
		return p.createDefaultPlan(request)
	}
	if resp != nil {
		for _, call := range resp.ToolCalls {
			if !strings.EqualFold(call.Function.Name, p.PlanningTool.Name()) {
				continue
			}
			arguments := map[string]any{}
			if call.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil {
					logger.Error.Printf("Failed to parse planning tool arguments: %v", err)
					continue
				}
			}
			arguments["plan_id"] = p.ActivePlanID
			result, execErr := p.PlanningTool.Execute(ctx, arguments)
			if execErr != nil {
				logger.Error.Printf("Planning tool execution error: %v", execErr)
				continue
			}
			if result.Error == "" {
				logger.Info.Printf("Plan created via LLM for %s", p.ActivePlanID)
				return nil
			}
			logger.Warn.Printf("Planning tool reported error: %s", result.Error)
		}
	}

	logger.Warn.Println("Falling back to default plan creation")
	return p.createDefaultPlan(request)
}

func (p *PlanningFlow) createDefaultPlan(request string) error {
	title := fmt.Sprintf("Plan for: %s", truncate(request, 50))
	steps := []string{"Analyze request", "Execute task", "Verify results"}
	_, err := p.PlanStore.Create(p.ActivePlanID, title, steps)
	return err
}

func (p *PlanningFlow) getCurrentStepInfo() (*int, map[string]string) {
	if p.PlanStore == nil {
		return nil, nil
	}
	stored, ok := p.PlanStore.InternalPlan(p.ActivePlanID)
	if !ok {
		logger.Error.Printf("Plan with ID %s not found", p.ActivePlanID)
		return nil, nil
	}

	for i, step := range stored.Steps {
		status := planning.StatusNotStarted
		if i < len(stored.StepStatuses) {
			status = stored.StepStatuses[i]
		}
		if status == planning.StatusNotStarted || status == planning.StatusInProgress {
			stepInfo := map[string]string{"text": step}
			if m := regexp.MustCompile(`\[([A-Z_]+)\]`).FindStringSubmatch(step); len(m) > 1 {
				stepInfo["type"] = strings.ToLower(m[1])
			}
			_, _ = p.PlanStore.MarkStep(p.ActivePlanID, i, planning.StatusInProgress, nil)
			idx := i
			return &idx, stepInfo
		}
	}
	return nil, nil
}

func (p *PlanningFlow) executeStep(ctx context.Context, executor agent.Agent, stepInfo map[string]string) (string, error) {
	if executor == nil {
		return "", fmt.Errorf("no executor available")
	}
	planStatus, _ := p.getPlanText()
	stepText := fmt.Sprintf("Step %d", derefInt(p.CurrentStepIndex))
	if stepInfo != nil && stepInfo["text"] != "" {
		stepText = stepInfo["text"]
	}
	prompt := fmt.Sprintf("\nCURRENT PLAN STATUS:\n%s\n\nYOUR CURRENT TASK:\nYou are now working on step %d: \"%s\"\n\nPlease only execute this current step using the appropriate tools. When you're done, provide a summary of what you accomplished.\n",
		planStatus,
		derefInt(p.CurrentStepIndex),
		stepText,
	)

	result, err := executor.Run(ctx, prompt)
	if err != nil {
		logger.Error.Printf("Error executing step %d: %s", derefInt(p.CurrentStepIndex), err)
		return fmt.Sprintf("Error executing step %d: %s", derefInt(p.CurrentStepIndex), err), err
	}
	_ = p.markStepCompleted()
	return result, nil
}

func (p *PlanningFlow) markStepCompleted() error {
	if p.CurrentStepIndex == nil {
		return nil
	}
	_, err := p.PlanStore.MarkStep(p.ActivePlanID, *p.CurrentStepIndex, planning.StatusCompleted, nil)
	if err != nil {
		logger.Warn.Printf("Failed to update plan status: %s", err)
	}
	return err
}

func (p *PlanningFlow) getPlanText() (string, error) {
	return p.PlanStore.Get(p.ActivePlanID)
}

func (p *PlanningFlow) finalizePlan(ctx context.Context) (string, error) {
	planText, _ := p.getPlanText()
	system := schema.SystemMessage("You are a planning assistant. Your task is to summarize the completed plan.")
	user := schema.UserMessage("The plan has been completed. Here is the final plan status:\n\n"+planText+"\n\nPlease provide a summary of what was accomplished and any final thoughts.", nil)
	resp, err := p.LLM.Ask(ctx, []schema.Message{user}, []schema.Message{system})
	if err == nil && resp != "" {
		return "Plan completed:\n\n" + resp, nil
	}

	primary := p.PrimaryAgent()
	if primary != nil {
		summary, err2 := primary.Run(ctx, "The plan has been completed. Here is the final plan status:\n\n"+planText+"\n\nPlease provide a summary of what was accomplished and any final thoughts.")
		if err2 == nil {
			return "Plan completed:\n\n" + summary, nil
		}
	}
	return "Plan completed. Error generating summary.", nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
