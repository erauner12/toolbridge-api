package server

import (
	"context"
	"time"

	"github.com/erauner12/toolbridge-api/internal/mcpserver/auth"
)

// PassthroughTokenProvider provides the JWT from the incoming HTTP request
// This is used in Streamable HTTP mode where Claude provides the JWT
type PassthroughTokenProvider struct {
	jwt       string
	expiresAt time.Time
}

// NewPassthroughTokenProvider creates a token provider from a JWT
func NewPassthroughTokenProvider(jwt string, expiresAt time.Time) *PassthroughTokenProvider {
	return &PassthroughTokenProvider{
		jwt:       jwt,
		expiresAt: expiresAt,
	}
}

// GetToken returns the JWT from the request
func (p *PassthroughTokenProvider) GetToken(ctx context.Context, audience, scope string, interactive bool) (*auth.TokenResult, error) {
	return &auth.TokenResult{
		AccessToken: p.jwt,
		ExpiresAt:   p.expiresAt,
		TokenType:   "Bearer",
	}, nil
}

// InvalidateToken is a no-op for passthrough provider
func (p *PassthroughTokenProvider) InvalidateToken(audience, scope string) {
	// No-op: token comes from Claude, we can't invalidate it
}
