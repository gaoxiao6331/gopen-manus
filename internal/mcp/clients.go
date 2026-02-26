package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"gopen-manus/internal/llm"
	"gopen-manus/internal/logger"
	"gopen-manus/internal/tool"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClientTool proxies a remote MCP tool.
type MCPClientTool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Session     *mcp.ClientSession
	ServerID    string
	Original    string
}

// Clients manages MCP sessions and tools.
type Clients struct {
	Sessions map[string]*mcp.ClientSession
	ToolMap  map[string]*MCPClientTool
}

func NewClients() *Clients {
	return &Clients{
		Sessions: map[string]*mcp.ClientSession{},
		ToolMap:  map[string]*MCPClientTool{},
	}
}

func (c *Clients) ConnectStdio(ctx context.Context, command string, args []string, serverID string) error {
	if command == "" {
		return errors.New("server command is required")
	}
	if serverID == "" {
		serverID = command
	}
	if existing := c.Sessions[serverID]; existing != nil {
		_ = existing.Close()
		delete(c.Sessions, serverID)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "gopen-manus", Version: "0.1.0"}, nil)
	cmd := exec.CommandContext(ctx, command, args...)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}
	c.Sessions[serverID] = session
	return c.refreshTools(ctx, serverID)
}

func (c *Clients) ConnectSSE(ctx context.Context, serverURL string, serverID string) error {
	if serverURL == "" {
		return errors.New("server URL is required")
	}
	if serverID == "" {
		serverID = serverURL
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "gopen-manus", Version: "0.1.0"}, nil)
	transport := &mcp.SSEClientTransport{Endpoint: serverURL}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}
	c.Sessions[serverID] = session
	return c.refreshTools(ctx, serverID)
}

func (c *Clients) refreshTools(ctx context.Context, serverID string) error {
	session := c.Sessions[serverID]
	if session == nil {
		return fmt.Errorf("session not initialized for server %s", serverID)
	}
	response, err := session.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	for _, toolInfo := range response.Tools {
		original := toolInfo.Name
		toolName := sanitizeToolName(fmt.Sprintf("mcp_%s_%s", serverID, original))
		params := schemaToMap(toolInfo.InputSchema)
		c.ToolMap[toolName] = &MCPClientTool{
			Name:        toolName,
			Description: toolInfo.Description,
			Parameters:  params,
			Session:     session,
			ServerID:    serverID,
			Original:    original,
		}
	}
	logger.Info.Printf("Connected to server %s with tools: %v", serverID, toolNames(response.Tools))
	return nil
}

func (c *Clients) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	result := &mcp.ListToolsResult{Tools: []*mcp.Tool{}}
	for _, session := range c.Sessions {
		resp, err := session.ListTools(ctx, nil)
		if err != nil {
			return nil, err
		}
		result.Tools = append(result.Tools, resp.Tools...)
	}
	return result, nil
}

func (c *Clients) Disconnect(serverID string) error {
	if serverID != "" {
		session := c.Sessions[serverID]
		if session != nil {
			_ = session.Close()
			delete(c.Sessions, serverID)
		}
		for name, toolRef := range c.ToolMap {
			if toolRef.ServerID == serverID {
				delete(c.ToolMap, name)
			}
		}
		return nil
	}
	for sid, session := range c.Sessions {
		_ = session.Close()
		delete(c.Sessions, sid)
	}
	c.ToolMap = map[string]*MCPClientTool{}
	return nil
}

func (c *Clients) ToParams() []llm.ToolParam {
	params := make([]llm.ToolParam, 0, len(c.ToolMap))
	for _, toolRef := range c.ToolMap {
		params = append(params, llm.ToolParam{
			Name:        toolRef.Name,
			Description: toolRef.Description,
			Parameters:  toolRef.Parameters,
		})
	}
	return params
}

func (c *Clients) HasTool(name string) bool {
	_, ok := c.ToolMap[name]
	return ok
}

func (c *Clients) Execute(ctx context.Context, name string, args map[string]any) (tool.Result, error) {
	toolRef := c.ToolMap[name]
	if toolRef == nil {
		return tool.Result{Error: fmt.Sprintf("unknown tool '%s'", name)}, nil
	}
	params := &mcp.CallToolParams{Name: toolRef.Original, Arguments: args}
	result, err := toolRef.Session.CallTool(ctx, params)
	if err != nil {
		return tool.Result{Error: err.Error()}, nil
	}
	if result == nil {
		return tool.Result{Output: "No output returned."}, nil
	}
	if result.IsError {
		return tool.Result{Error: "tool error"}, nil
	}

	parts := []string{}
	var imageData *string
	for _, item := range result.Content {
		switch v := item.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		case *mcp.ImageContent:
			if len(v.Data) > 0 {
				encoded := string(v.Data)
				imageData = &encoded
			}
		}
	}
	output := strings.Join(parts, ", ")
	if output == "" {
		output = "No output returned."
	}
	return tool.Result{Output: output, Base64Image: imageData}, nil
}

func sanitizeToolName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := re.ReplaceAllString(name, "_")
	re2 := regexp.MustCompile(`_+`)
	sanitized = re2.ReplaceAllString(sanitized, "_")
	sanitized = strings.Trim(sanitized, "_")
	if len(sanitized) > 64 {
		sanitized = sanitized[:64]
	}
	return sanitized
}

func toolNames(tools []*mcp.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name)
	}
	return out
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
