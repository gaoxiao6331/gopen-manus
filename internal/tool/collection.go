package tool

import (
	"context"
	"sync"

	"gopen-manus/internal/llm"
)

// Collection manages a set of tools and satisfies agent.ToolExecutor.
type Collection struct {
	mu    sync.RWMutex
	tools map[string]Tool
	order []string
}

// NewCollection constructs a collection with optional tools.
func NewCollection(tools ...Tool) *Collection {
	c := &Collection{tools: map[string]Tool{}, order: []string{}}
	c.AddTools(tools...)
	return c
}

// AddTools registers tools, skipping duplicates by name.
func (c *Collection) AddTools(tools ...Tool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, t := range tools {
		if t == nil {
			continue
		}
		name := t.Name()
		if name == "" {
			continue
		}
		if _, exists := c.tools[name]; exists {
			continue
		}
		c.tools[name] = t
		c.order = append(c.order, name)
	}
}

// Get retrieves a tool by name.
func (c *Collection) Get(name string) Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools[name]
}

// ToParams exposes tool schemas for LLM consumption.
func (c *Collection) ToParams() []llm.ToolParam {
	c.mu.RLock()
	defer c.mu.RUnlock()
	params := make([]llm.ToolParam, 0, len(c.order))
	for _, name := range c.order {
		params = append(params, Param(c.tools[name]))
	}
	return params
}

// HasTool reports whether a tool exists.
func (c *Collection) HasTool(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.tools[name]
	return ok
}

// Execute runs a tool by name with the given arguments.
func (c *Collection) Execute(ctx context.Context, name string, args map[string]any) (Result, error) {
	tool := c.Get(name)
	if tool == nil {
		return Result{Error: "unknown tool"}, nil
	}
	if args == nil {
		args = map[string]any{}
	}
	return tool.Execute(ctx, args)
}

// Ensure Collection satisfies the ToolExecutor contract expected by agents.
var _ interface {
	ToParams() []llm.ToolParam
	HasTool(string) bool
	Execute(context.Context, string, map[string]any) (Result, error)
} = (*Collection)(nil)
