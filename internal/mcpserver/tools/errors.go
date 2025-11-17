package tools

import (
	"encoding/json"
	"fmt"

	"github.com/erauner12/toolbridge-api/internal/mcpserver/client"
)

// ToolError represents a structured error from tool execution
type ToolError struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ErrorCode categorizes tool errors for JSON-RPC translation
type ErrorCode string

const (
	ErrCodeInvalidParams  ErrorCode = "INVALID_PARAMS"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeDeleted        ErrorCode = "DELETED"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeRateLimit      ErrorCode = "RATE_LIMIT"
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeMethodNotFound ErrorCode = "METHOD_NOT_FOUND"
)

// NewToolError creates a tool error with optional data
func NewToolError(code ErrorCode, message string, data map[string]any) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// WrapClientError converts REST client errors into ToolErrors
func WrapClientError(err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case client.ErrNotFound:
		return NewToolError(ErrCodeNotFound, fmt.Sprintf("Entity %s not found", e.UID), nil)

	case client.ErrDeleted:
		return NewToolError(ErrCodeDeleted, fmt.Sprintf("Entity %s has been deleted", e.UID), nil)

	case client.ErrVersionMismatch:
		return NewToolError(ErrCodeConflict, "Version mismatch - entity was modified", map[string]any{
			"expected": e.Expected,
			"actual":   e.Actual,
		})

	case client.ErrEpochMismatch:
		return NewToolError(ErrCodeConflict, "Epoch mismatch - session out of sync", map[string]any{
			"clientEpoch": e.ClientEpoch,
			"serverEpoch": e.ServerEpoch,
		})

	case client.ErrRateLimited:
		return NewToolError(ErrCodeRateLimit, "Rate limit exceeded", map[string]any{
			"retryAfter": e.RetryAfter,
		})

	default:
		return NewToolError(ErrCodeInternal, err.Error(), nil)
	}
}

// ToJSONRPCError converts ToolError to JSON-RPC error code
func (e *ToolError) ToJSONRPCError() (int, string, json.RawMessage) {
	var code int
	switch e.Code {
	case ErrCodeInvalidParams, ErrCodeNotFound, ErrCodeDeleted:
		code = -32602 // InvalidParams
	case ErrCodeMethodNotFound:
		code = -32601 // MethodNotFound
	case ErrCodeConflict, ErrCodeRateLimit:
		code = -32603 // InternalError (retriable)
	default:
		code = -32603 // InternalError
	}

	var data json.RawMessage
	if e.Data != nil {
		dataBytes, _ := json.Marshal(e.Data)
		data = dataBytes
	}

	return code, e.Message, data
}
