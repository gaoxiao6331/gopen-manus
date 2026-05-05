package agent

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopen-manus/internal/config"
	"gopen-manus/internal/llm"
	"gopen-manus/internal/logger"
	"gopen-manus/internal/planning"
	"gopen-manus/internal/prompt"
	"gopen-manus/internal/tool"

	openai "github.com/sashabaranov/go-openai"
)

type ManusAgent struct {
	*ToolCallAgent

	Tools     *tool.Collection
	PlanStore *planning.Store
}

// NewManusAgent constructs a Manus agent with core local tools configured.
func NewManusAgent() *ManusAgent {
	base := NewToolCallAgent("manus")
	workspace := discoverWorkspace()
	base.SystemPrompt = strings.ReplaceAll(prompt.ManusSystemPrompt, "{directory}", workspace)
	base.NextStepPrompt = prompt.ManusNextStepPrompt

	store := planning.NewStore()
	collection := tool.NewCollection(&tool.TerminateTool{}, tool.NewPlanningTool(store))
	base.AvailableTools = collection
	configureDefaultLLM(base)

	return &ManusAgent{
		ToolCallAgent: base,
		Tools:         collection,
		PlanStore:     store,
	}
}

// AddTools registers additional tools on top of the default set.
func (m *ManusAgent) AddTools(tools ...tool.Tool) {
	if m == nil || m.Tools == nil {
		return
	}
	m.Tools.AddTools(tools...)
}

func discoverWorkspace() string {
	if dir := os.Getenv("OPENMANUS_WORKSPACE"); dir != "" {
		return dir
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func configureDefaultLLM(agent *ToolCallAgent) {
	if agent == nil {
		return
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return
	}

	cfg := openai.DefaultConfig(apiKey)
	if baseURL := selectOpenAIBaseURL(); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	if timeout := config.Settings.LLM.RequestTimeout; timeout > 0 {
		cfg.HTTPClient = &http.Client{Timeout: timeout}
	}

	model := getenvDefault("OPENAI_MODEL", "gpt-4o")
	maxTokens := getenvInt("OPENAI_MAX_TOKENS", 2048)
	maxInputTokens := getenvInt("OPENAI_MAX_INPUT_TOKENS", 12000)
	temperature := getenvFloat("OPENAI_TEMPERATURE", 0.7)

	client, err := llm.NewOpenAIClient(cfg, model, maxTokens, float32(temperature), maxInputTokens)
	if err != nil {
		logger.Warn.Printf("Failed to initialize OpenAI client: %v", err)
		return
	}

	agent.ReActAgent.LLM = client
	if agent.ReActAgent.BaseAgent != nil {
		agent.ReActAgent.BaseAgent.LLM = client
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func selectOpenAIBaseURL() string {
	keys := []string{"OPENAI_API_BASE_URL", "OPENAI_BASE_URL"}
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return sanitizeBaseURL(value)
		}
	}
	return ""
}

func sanitizeBaseURL(base string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		return ""
	}
	for strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimSuffix(trimmed, "/")
	}
	return trimmed
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
		logger.Warn.Printf("Invalid value for %s: %s", key, v)
	}
	return fallback
}

func getenvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseFloat(v, 32); err == nil {
			return parsed
		}
		logger.Warn.Printf("Invalid value for %s: %s", key, v)
	}
	return fallback
}
