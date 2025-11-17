package tools

import (
	"context"
	"encoding/json"
)

// ToolDefinition describes an MCP tool with its name, description, and JSON schemas
type ToolDefinition struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

// Handler processes a tool invocation with the given context and parameters
type Handler func(context.Context, *ToolContext, json.RawMessage) (interface{}, error)

// ToolDescriptor is returned by tools/list (MCP specification format)
type ToolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// CallRequest represents a tools/call JSON-RPC request
type CallRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallResult wraps successful tool execution results
type CallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a piece of tool output
type ContentBlock struct {
	Type string `json:"type"` // "text", "resource", etc.
	Text string `json:"text,omitempty"`
}
