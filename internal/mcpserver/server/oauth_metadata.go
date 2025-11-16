package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleOAuthMetadata serves the OAuth authorization server metadata
// Reference: RFC 8414 (OAuth 2.0 Authorization Server Metadata)
func (s *MCPServer) handleOAuthMetadata(w http.ResponseWriter, r *http.Request) {
	domain := s.config.Auth0.Domain
	issuer := fmt.Sprintf("https://%s/", domain)

	metadata := map[string]interface{}{
		"issuer":                issuer,
		"authorization_endpoint": fmt.Sprintf("https://%s/authorize", domain),
		"token_endpoint":         fmt.Sprintf("https://%s/oauth/token", domain),
		"jwks_uri":               fmt.Sprintf("https://%s/.well-known/jwks.json", domain),
		"response_types_supported": []string{"code"},
		"grant_types_supported":    []string{"authorization_code", "refresh_token"},
		"subject_types_supported":  []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported": []string{"openid", "profile", "email", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}
