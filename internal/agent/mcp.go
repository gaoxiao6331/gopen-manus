package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gopen-manus/internal/logger"
	"gopen-manus/internal/mcp"
	"gopen-manus/internal/prompt"
	"gopen-manus/internal/schema"
)

// MCPAgent connects to an MCP server and uses its tools.
type MCPAgent struct {
	*ToolCallAgent

	MCPClients      *mcp.Clients
	ConnectionType  string
	ToolSchemas     map[string]map[string]any
	RefreshInterval int
}

func NewMCPAgent() *MCPAgent {
	base := NewToolCallAgent("mcp_agent")
	base.SystemPrompt = prompt.MCPSystemPrompt
	base.NextStepPrompt = prompt.MCPNextStepPrompt
	agent := &MCPAgent{
		ToolCallAgent:   base,
		MCPClients:      mcp.NewClients(),
		ConnectionType:  "stdio",
		ToolSchemas:     map[string]map[string]any{},
		RefreshInterval: 5,
	}
	return agent
}

func (m *MCPAgent) Initialize(ctx context.Context, connectionType string, serverURL string, command string, args []string) error {
	if connectionType != "" {
		m.ConnectionType = connectionType
	}

	switch m.ConnectionType {
	case "sse":
		if serverURL == "" {
			return fmt.Errorf("server URL is required for SSE connection")
		}
		if err := m.MCPClients.ConnectSSE(ctx, serverURL, serverURL); err != nil {
			return err
		}
	case "stdio":
		if command == "" {
			return fmt.Errorf("command is required for stdio connection")
		}
		if err := m.MCPClients.ConnectStdio(ctx, command, args, command); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported connection type: %s", m.ConnectionType)
	}

	m.AvailableTools = m.MCPClients
	_, _ = m.RefreshTools(ctx)

	toolNames := []string{}
	for name := range m.MCPClients.ToolMap {
		toolNames = append(toolNames, name)
	}
	toolsInfo := strings.Join(toolNames, ", ")
	m.Memory.AddMessage(schema.SystemMessage(fmt.Sprintf("%s\n\nAvailable MCP tools: %s", m.SystemPrompt, toolsInfo)))
	return nil
}

func (m *MCPAgent) RefreshTools(ctx context.Context) ([]string, []string) {
	if len(m.MCPClients.Sessions) == 0 {
		return nil, nil
	}
	response, err := m.MCPClients.ListTools(ctx)
	if err != nil {
		return nil, nil
	}
	currentTools := map[string]map[string]any{}
	for _, tool := range response.Tools {
		currentTools[tool.Name] = schemaToMap(tool.InputSchema)
	}

	currentNames := map[string]struct{}{}
	for name := range currentTools {
		currentNames[name] = struct{}{}
	}
	prevNames := map[string]struct{}{}
	for name := range m.ToolSchemas {
		prevNames[name] = struct{}{}
	}

	added := []string{}
	removed := []string{}
	for name := range currentNames {
		if _, ok := prevNames[name]; !ok {
			added = append(added, name)
		}
	}
	for name := range prevNames {
		if _, ok := currentNames[name]; !ok {
			removed = append(removed, name)
		}
	}

	m.ToolSchemas = currentTools
	if len(added) > 0 {
		logger.Info.Printf("Added MCP tools: %v", added)
		m.Memory.AddMessage(schema.SystemMessage("New tools available: " + strings.Join(added, ", ")))
	}
	if len(removed) > 0 {
		logger.Info.Printf("Removed MCP tools: %v", removed)
		m.Memory.AddMessage(schema.SystemMessage("Tools no longer available: " + strings.Join(removed, ", ")))
	}
	return added, removed
}

func (m *MCPAgent) Think(ctx context.Context) (bool, error) {
	if len(m.MCPClients.Sessions) == 0 || len(m.MCPClients.ToolMap) == 0 {
		logger.Info.Println("MCP service is no longer available, ending interaction")
		m.State = schema.AgentStateFinished
		return false, nil
	}
	if m.CurrentStep%m.RefreshInterval == 0 {
		m.RefreshTools(ctx)
		if len(m.MCPClients.ToolMap) == 0 {
			logger.Info.Println("MCP service has shut down, ending interaction")
			m.State = schema.AgentStateFinished
			return false, nil
		}
	}
	return m.ToolCallAgent.Think(ctx)
}

func (m *MCPAgent) Cleanup(ctx context.Context) {
	if len(m.MCPClients.Sessions) > 0 {
		_ = m.MCPClients.Disconnect("")
		logger.Info.Println("MCP connection closed")
	}
}

func schemaToMap(schema any) map[string]any {
	if schema == nil {
		return nil
	}
	switch v := schema.(type) {
	case map[string]any:
		return v
	default:
		buf, err := json.Marshal(schema)
		if err != nil {
			return nil
		}
		var out map[string]any
		if err := json.Unmarshal(buf, &out); err != nil {
			return nil
		}
		return out
	}
}
