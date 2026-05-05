package schema

import "errors"

// Role represents message roles.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolChoice represents tool choice options.
type ToolChoice string

const (
	ToolChoiceNone     ToolChoice = "none"
	ToolChoiceAuto     ToolChoice = "auto"
	ToolChoiceRequired ToolChoice = "required"
)

// AgentState represents the execution state of an agent.
type AgentState string

const (
	AgentStateIdle     AgentState = "IDLE"
	AgentStateRunning  AgentState = "RUNNING"
	AgentStateFinished AgentState = "FINISHED"
	AgentStateError    AgentState = "ERROR"
)

// Function represents a tool/function call.
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall represents a tool/function call in a message.
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Message represents a chat message in the conversation.
type Message struct {
	Role        Role       `json:"role"`
	Content     *string    `json:"content,omitempty"`
	ToolCalls   []ToolCall `json:"tool_calls,omitempty"`
	Name        *string    `json:"name,omitempty"`
	ToolCallID  *string    `json:"tool_call_id,omitempty"`
	Base64Image *string    `json:"base64_image,omitempty"`
}

func (m Message) ToMap() map[string]any {
	msg := map[string]any{"role": m.Role}
	if m.Content != nil {
		msg["content"] = *m.Content
	}
	if len(m.ToolCalls) > 0 {
		msg["tool_calls"] = m.ToolCalls
	}
	if m.Name != nil {
		msg["name"] = *m.Name
	}
	if m.ToolCallID != nil {
		msg["tool_call_id"] = *m.ToolCallID
	}
	if m.Base64Image != nil {
		msg["base64_image"] = *m.Base64Image
	}
	return msg
}

func UserMessage(content string, base64Image *string) Message {
	return Message{Role: RoleUser, Content: &content, Base64Image: base64Image}
}

func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: &content}
}

func AssistantMessage(content *string, base64Image *string) Message {
	return Message{Role: RoleAssistant, Content: content, Base64Image: base64Image}
}

func ToolMessage(content string, name string, toolCallID string, base64Image *string) Message {
	return Message{
		Role:        RoleTool,
		Content:     &content,
		Name:        &name,
		ToolCallID:  &toolCallID,
		Base64Image: base64Image,
	}
}

func FromToolCalls(toolCalls []ToolCall, content *string, base64Image *string) Message {
	return Message{
		Role:        RoleAssistant,
		Content:     content,
		ToolCalls:   toolCalls,
		Base64Image: base64Image,
	}
}

// Memory stores messages for an agent.
type Memory struct {
	Messages    []Message
	MaxMessages int
}

func NewMemory() *Memory {
	return &Memory{Messages: []Message{}, MaxMessages: 100}
}

func (m *Memory) AddMessage(message Message) {
	m.Messages = append(m.Messages, message)
	if m.MaxMessages > 0 && len(m.Messages) > m.MaxMessages {
		m.Messages = m.Messages[len(m.Messages)-m.MaxMessages:]
	}
}

func (m *Memory) AddMessages(messages []Message) {
	m.Messages = append(m.Messages, messages...)
	if m.MaxMessages > 0 && len(m.Messages) > m.MaxMessages {
		m.Messages = m.Messages[len(m.Messages)-m.MaxMessages:]
	}
}

func (m *Memory) Clear() {
	m.Messages = nil
}

func (m *Memory) GetRecentMessages(n int) []Message {
	if n <= 0 || n >= len(m.Messages) {
		return append([]Message{}, m.Messages...)
	}
	return append([]Message{}, m.Messages[len(m.Messages)-n:]...)
}

func (m *Memory) ToMapList() []map[string]any {
	out := make([]map[string]any, 0, len(m.Messages))
	for _, msg := range m.Messages {
		out = append(out, msg.ToMap())
	}
	return out
}

func ValidateRole(role Role) error {
	switch role {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return nil
	default:
		return errors.New("unsupported message role")
	}
}
