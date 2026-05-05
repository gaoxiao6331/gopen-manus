package llm

import (
	"context"
	"fmt"
	"strings"

	"gopen-manus/internal/schema"

	"github.com/cenkalti/backoff/v4"
	"github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the Client interface for OpenAI's API.
type OpenAIClient struct {
	client         *openai.Client
	config         openai.ClientConfig
	model          string
	maxTokens      int
	temperature    float32
	tokenCounter   *TokenCounter
	maxInputTokens int
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(config openai.ClientConfig, model string, maxTokens int, temperature float32, maxInputTokens int) (*OpenAIClient, error) {
	client := openai.NewClientWithConfig(config)

	tokenizer, err := NewTokenCounter(model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	return &OpenAIClient{
		client:         client,
		config:         config,
		model:          model,
		maxTokens:      maxTokens,
		temperature:    temperature,
		tokenCounter:   tokenizer,
		maxInputTokens: maxInputTokens,
	}, nil
}

func (o *OpenAIClient) Ask(ctx context.Context, messages []schema.Message, systemMsgs []schema.Message) (string, error) {
	// Format messages
	formattedMessages, err := o.formatMessages(messages, systemMsgs)
	if err != nil {
		return "", fmt.Errorf("failed to format messages: %w", err)
	}

	// Calculate input tokens
	inputTokens := o.countMessageTokens(formattedMessages)

	// Check token limit
	if o.maxInputTokens > 0 && inputTokens > o.maxInputTokens {
		return "", fmt.Errorf("input token limit exceeded: %d > %d", inputTokens, o.maxInputTokens)
	}

	req := openai.ChatCompletionRequest{
		Model:       o.model,
		Messages:    formattedMessages,
		MaxTokens:   o.maxTokens,
		Temperature: o.temperature,
	}

	var response string
	operation := func() error {
		resp, err := o.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return err
		}
		response = resp.Choices[0].Message.Content
		return nil
	}

	err = backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		return "", fmt.Errorf("failed to ask OpenAI: %w", err)
	}

	return response, nil
}

func (o *OpenAIClient) formatMessages(messages []schema.Message, systemMsgs []schema.Message) ([]openai.ChatCompletionMessage, error) {
	formattedMessages := make([]openai.ChatCompletionMessage, 0, len(systemMsgs)+len(messages))

	for _, msg := range systemMsgs {
		formattedMessages = append(formattedMessages, toOpenAIMessage(msg))
	}

	for _, msg := range messages {
		formattedMessages = append(formattedMessages, toOpenAIMessage(msg))
	}

	return formattedMessages, nil
}

func toOpenAIMessage(msg schema.Message) openai.ChatCompletionMessage {
	role := string(msg.Role)
	switch msg.Role {
	case schema.RoleUser:
		if msg.Base64Image != nil {
			return openai.ChatCompletionMessage{
				Role:         role,
				MultiContent: multimodalParts(msg.Content, *msg.Base64Image),
			}
		}
	case schema.RoleAssistant:
		if len(msg.ToolCalls) > 0 {
			return openai.ChatCompletionMessage{
				Role:      role,
				Content:   safeText(msg.Content),
				ToolCalls: toOpenAIToolCalls(msg.ToolCalls),
			}
		}
	case schema.RoleTool:
		return openai.ChatCompletionMessage{
			Role:       role,
			Content:    safeText(msg.Content),
			Name:       safePointer(msg.Name),
			ToolCallID: safePointer(msg.ToolCallID),
		}
	}

	return openai.ChatCompletionMessage{
		Role:    role,
		Content: safeText(msg.Content),
	}
}

func multimodalParts(content *string, base64 string) []openai.ChatMessagePart {
	parts := []openai.ChatMessagePart{}
	text := safeText(content)
	if strings.TrimSpace(text) != "" {
		parts = append(parts, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: text,
		})
	}
	parts = append(parts, openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeImageURL,
		ImageURL: &openai.ChatMessageImageURL{
			URL:    fmt.Sprintf("data:image/jpeg;base64,%s", base64),
			Detail: openai.ImageURLDetailAuto,
		},
	})
	return parts
}

func safeText(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func safePointer(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func toOpenAIToolCalls(calls []schema.ToolCall) []openai.ToolCall {
	result := make([]openai.ToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, openai.ToolCall{
			ID:   call.ID,
			Type: openai.ToolType(call.Type),
			Function: openai.FunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return result
}

func (o *OpenAIClient) countMessageTokens(messages []openai.ChatCompletionMessage) int {
	var totalTokens int
	for _, msg := range messages {
		if len(msg.MultiContent) > 0 {
			for _, part := range msg.MultiContent {
				if part.Type == openai.ChatMessagePartTypeText {
					totalTokens += o.tokenCounter.CountText(part.Text)
				}
			}
			continue
		}
		totalTokens += o.tokenCounter.CountText(msg.Content)
	}
	return totalTokens
}

func (o *OpenAIClient) AskTool(ctx context.Context, messages []schema.Message, systemMsgs []schema.Message, tools []ToolParam, toolChoice schema.ToolChoice) (*Response, error) {
	// Format messages
	formattedMessages, err := o.formatMessages(messages, systemMsgs)
	if err != nil {
		return nil, fmt.Errorf("failed to format messages: %w", err)
	}

	// Format tools
	formattedTools := o.formatTools(tools)

	// Calculate input tokens
	inputTokens := o.countMessageTokens(formattedMessages) // Simplified, does not count tool definition tokens

	// Check token limit
	if o.maxInputTokens > 0 && inputTokens > o.maxInputTokens {
		return nil, fmt.Errorf("input token limit exceeded: %d > %d", inputTokens, o.maxInputTokens)
	}

	req := openai.ChatCompletionRequest{
		Model:       o.model,
		Messages:    formattedMessages,
		Tools:       formattedTools,
		ToolChoice:  string(toolChoice),
		MaxTokens:   o.maxTokens,
		Temperature: o.temperature,
	}

	var response *Response
	operation := func() error {
		resp, err := o.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return err
		}

		response = &Response{
			Content:   resp.Choices[0].Message.Content,
			ToolCalls: o.toSchemaToolCalls(resp.Choices[0].Message.ToolCalls),
		}
		return nil
	}

	err = backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, fmt.Errorf("failed to ask OpenAI with tools: %w", err)
	}

	return response, nil
}

func (o *OpenAIClient) formatTools(tools []ToolParam) []openai.Tool {
	var formattedTools []openai.Tool
	for _, tool := range tools {
		formattedTools = append(formattedTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return formattedTools
}

func (o *OpenAIClient) toSchemaToolCalls(toolCalls []openai.ToolCall) []schema.ToolCall {
	var schemaToolCalls []schema.ToolCall
	for _, tc := range toolCalls {
		schemaToolCalls = append(schemaToolCalls, schema.ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: schema.Function{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return schemaToolCalls
}
