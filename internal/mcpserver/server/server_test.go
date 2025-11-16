package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erauner12/toolbridge-api/internal/mcpserver/config"
	"github.com/golang-jwt/jwt/v5"
)

// mockJWKSServer creates a mock JWKS server for testing
func mockJWKSServer(t *testing.T, privateKey *rsa.PrivateKey) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return JWKS with public key
		n := privateKey.PublicKey.N
		e := privateKey.PublicKey.E

		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"use": "sig",
					"kid": "test-key-1",
					"n":   base64urlEncode(n.Bytes()),
					"e":   base64urlEncode(intToBytes(e)),
					"alg": "RS256",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
}

// Helper functions for JWKS encoding
func base64urlEncode(data []byte) string {
	// Simple base64url encoding without padding
	const encoding = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, 0, len(data)*4/3+1)

	for i := 0; i < len(data); i += 3 {
		remaining := len(data) - i
		chunk := uint(data[i]) << 16
		if remaining > 1 {
			chunk |= uint(data[i+1]) << 8
		}
		if remaining > 2 {
			chunk |= uint(data[i+2])
		}

		result = append(result, encoding[(chunk>>18)&63])
		result = append(result, encoding[(chunk>>12)&63])
		if remaining > 1 {
			result = append(result, encoding[(chunk>>6)&63])
		}
		if remaining > 2 {
			result = append(result, encoding[chunk&63])
		}
	}

	return string(result)
}

func intToBytes(n int) []byte {
	bytes := make([]byte, 0)
	for n > 0 {
		bytes = append([]byte{byte(n & 0xFF)}, bytes...)
		n >>= 8
	}
	if len(bytes) == 0 {
		bytes = []byte{0}
	}
	return bytes
}

// createTestJWT creates a valid JWT for testing
func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, audience, issuer, subject string) string {
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": subject,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-1"

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	return tokenString
}

func TestMCPServer_OAuthMetadata(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: "test.auth0.com",
			SyncAPI: &config.SyncAPIConfig{
				Audience: "https://api.test.com",
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	server := NewMCPServer(cfg)

	// Create test request
	req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	server.handleOAuthMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&metadata); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri"}
	for _, field := range requiredFields {
		if _, ok := metadata[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

func TestMCPServer_Initialize(t *testing.T) {
	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create mock JWKS server
	jwksServer := mockJWKSServer(t, privateKey)
	defer jwksServer.Close()

	// Extract domain from JWKS server URL
	issuer := jwksServer.URL + "/"
	audience := "https://api.test.com"

	// Create test config
	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: jwksServer.URL[7:], // Remove "http://" prefix
			SyncAPI: &config.SyncAPIConfig{
				Audience: audience,
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	// Override JWKS URL in validator for testing
	mcpServer := NewMCPServer(cfg)
	mcpServer.jwtValidator.jwksURL = jwksServer.URL + "/.well-known/jwks.json"
	mcpServer.jwtValidator.issuer = issuer

	// Create test JWT
	token := createTestJWT(t, privateKey, audience, issuer, "test-user-123")

	// Create initialize request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")

	w := httptest.NewRecorder()
	mcpServer.handleMCPPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check session ID in header
	sessionID := w.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("Expected Mcp-Session-Id header, got empty")
	}

	// Decode response
	var response JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %s", response.Error.Message)
	}

	// Verify response contains capabilities
	var result map[string]interface{}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if _, ok := result["capabilities"]; !ok {
		t.Error("Response missing capabilities")
	}
}

func TestMCPServer_MissingSessionID(t *testing.T) {
	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create mock JWKS server
	jwksServer := mockJWKSServer(t, privateKey)
	defer jwksServer.Close()

	issuer := jwksServer.URL + "/"
	audience := "https://api.test.com"

	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: jwksServer.URL[7:],
			SyncAPI: &config.SyncAPIConfig{
				Audience: audience,
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	mcpServer := NewMCPServer(cfg)
	mcpServer.jwtValidator.jwksURL = jwksServer.URL + "/.well-known/jwks.json"
	mcpServer.jwtValidator.issuer = issuer

	token := createTestJWT(t, privateKey, audience, issuer, "test-user-123")

	// Create tools/list request without session ID
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")
	// Note: No Mcp-Session-Id header

	w := httptest.NewRecorder()
	mcpServer.handleMCPPost(w, req)

	// Should return error
	var response JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil {
		t.Error("Expected error for missing session ID")
	}

	if response.Error.Code != InvalidRequest {
		t.Errorf("Expected error code %d, got %d", InvalidRequest, response.Error.Code)
	}
}

func TestMCPServer_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: "test.auth0.com",
			SyncAPI: &config.SyncAPIConfig{
				Audience: "https://api.test.com",
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	mcpServer := NewMCPServer(cfg)

	// Create request with invalid token
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")

	w := httptest.NewRecorder()
	mcpServer.handleMCPPost(w, req)

	// Should return error
	var response JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestMCPServer_DeleteSession(t *testing.T) {
	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: "test.auth0.com",
			SyncAPI: &config.SyncAPIConfig{
				Audience: "https://api.test.com",
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	mcpServer := NewMCPServer(cfg)

	// Create a session directly
	session := mcpServer.sessionMgr.CreateSession("test-user")

	// Delete the session
	req := httptest.NewRequest("DELETE", "/mcp", nil)
	req.Header.Set("Mcp-Session-Id", session.ID)

	w := httptest.NewRecorder()
	mcpServer.handleMCPDelete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify session is deleted
	_, err := mcpServer.sessionMgr.GetSession(session.ID)
	if err == nil {
		t.Error("Expected session to be deleted")
	}
}

func TestMCPServer_UnsupportedProtocolVersion(t *testing.T) {
	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: "test.auth0.com",
			SyncAPI: &config.SyncAPIConfig{
				Audience: "https://api.test.com",
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	mcpServer := NewMCPServer(cfg)

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer fake-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "1.0.0") // Unsupported version

	w := httptest.NewRecorder()
	mcpServer.handleMCPPost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create mock JWKS server
	jwksServer := mockJWKSServer(t, privateKey)
	defer jwksServer.Close()

	issuer := jwksServer.URL + "/"
	audience := "https://api.test.com"

	cfg := &config.Config{
		Auth0: config.Auth0Config{
			Domain: jwksServer.URL[7:],
			SyncAPI: &config.SyncAPIConfig{
				Audience: audience,
			},
		},
		APIBaseURL: "http://localhost:8081",
	}

	mcpServer := NewMCPServer(cfg)
	mcpServer.jwtValidator.jwksURL = jwksServer.URL + "/.well-known/jwks.json"
	mcpServer.jwtValidator.issuer = issuer

	token := createTestJWT(t, privateKey, audience, issuer, "test-user-123")

	// First initialize to get session
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}

	body, _ := json.Marshal(initReq)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")

	w := httptest.NewRecorder()
	mcpServer.handleMCPPost(w, req)

	sessionID := w.Header().Get("Mcp-Session-Id")

	// Now call tools/list
	toolsReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	body, _ = json.Marshal(toolsReq)
	req = httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "2025-03-26")
	req.Header.Set("Mcp-Session-Id", sessionID)

	w = httptest.NewRecorder()
	mcpServer.handleMCPPost(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %s", response.Error.Message)
	}

	// For Phase 4, tools list should be empty
	var result map[string]interface{}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Error("Expected tools to be an array")
	}

	if len(tools) != 0 {
		t.Errorf("Expected empty tools list, got %d tools", len(tools))
	}
}
