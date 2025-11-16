package server

import (
	"encoding/json"
	"testing"
)

func TestJSONRPCRequest_Parsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *JSONRPCRequest)
	}{
		{
			name:    "valid request with id",
			input:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			wantErr: false,
			check: func(t *testing.T, req *JSONRPCRequest) {
				if req.JSONRPC != "2.0" {
					t.Errorf("Expected jsonrpc 2.0, got %s", req.JSONRPC)
				}
				if req.Method != "initialize" {
					t.Errorf("Expected method initialize, got %s", req.Method)
				}
				if len(req.ID) == 0 {
					t.Error("Expected ID to be present")
				}
			},
		},
		{
			name:    "notification without id",
			input:   `{"jsonrpc":"2.0","method":"notification"}`,
			wantErr: false,
			check: func(t *testing.T, req *JSONRPCRequest) {
				if !req.IsNotification() {
					t.Error("Expected IsNotification to be true")
				}
			},
		},
		{
			name:    "request with string id",
			input:   `{"jsonrpc":"2.0","id":"abc123","method":"test"}`,
			wantErr: false,
			check: func(t *testing.T, req *JSONRPCRequest) {
				if req.IsNotification() {
					t.Error("Expected IsNotification to be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req JSONRPCRequest
			err := json.Unmarshal([]byte(tt.input), &req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, &req)
			}
		})
	}
}

func TestJSONRPCResponse_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		response JSONRPCResponse
		wantJSON string
	}{
		{
			name: "success response",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Result:  json.RawMessage(`{"status":"ok"}`),
			},
			wantJSON: `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`,
		},
		{
			name: "error response",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Error: &JSONRPCError{
					Code:    InvalidRequest,
					Message: "invalid request",
				},
			},
			wantJSON: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"invalid request"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// Compare as JSON to ignore whitespace
			var gotObj, wantObj interface{}
			if err := json.Unmarshal(got, &gotObj); err != nil {
				t.Fatalf("Failed to unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantObj); err != nil {
				t.Fatalf("Failed to unmarshal want: %v", err)
			}

			gotJSON, _ := json.Marshal(gotObj)
			wantJSON, _ := json.Marshal(wantObj)

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Marshal() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestJSONRPCErrorCodes(t *testing.T) {
	tests := []struct {
		code int
		name string
	}{
		{ParseError, "ParseError"},
		{InvalidRequest, "InvalidRequest"},
		{MethodNotFound, "MethodNotFound"},
		{InvalidParams, "InvalidParams"},
		{InternalError, "InternalError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code >= 0 {
				t.Errorf("Error code %d should be negative", tt.code)
			}
		})
	}
}

func TestJSONRPCRequest_IsNotification(t *testing.T) {
	tests := []struct {
		name string
		req  JSONRPCRequest
		want bool
	}{
		{
			name: "request with id is not notification",
			req:  JSONRPCRequest{ID: json.RawMessage(`1`)},
			want: false,
		},
		{
			name: "request without id is notification",
			req:  JSONRPCRequest{},
			want: true,
		},
		{
			name: "request with null id is notification",
			req:  JSONRPCRequest{ID: json.RawMessage(`null`)},
			want: false, // null is still a valid ID in JSON-RPC 2.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.IsNotification(); got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}
