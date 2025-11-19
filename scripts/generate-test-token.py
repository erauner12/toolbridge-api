#!/usr/bin/env python3
"""
Generate HS256 JWT token for testing ToolBridge API

This creates a token compatible with the production API's dual auth mode.
The token includes Auth0-compatible claims (iss, aud) but uses HS256 signing.
"""
import jwt
import time
import sys
from datetime import datetime, timedelta

# Production API configuration (from K8s)
JWT_SECRET = "ZQS+HjOePeZGMbK5VnbSSkc/s+lcT4NVVcNidbUBGEQ="
AUTH0_ISSUER = "https://dev-zysv6k3xo7pkwmcb.us.auth0.com/"
AUTH0_AUDIENCE = "https://toolbridgeapi.erauner.dev"

# Token claims
now = int(time.time())
exp = now + (30 * 24 * 60 * 60)  # 30 days from now

claims = {
    "sub": "claude-desktop-user",  # User identifier
    "iss": AUTH0_ISSUER,           # Issuer (must match Auth0 config)
    "aud": AUTH0_AUDIENCE,         # Audience (must match API config)
    "iat": now,                    # Issued at
    "exp": exp                     # Expiration (30 days)
}

# Generate token
token = jwt.encode(claims, JWT_SECRET, algorithm="HS256")

print("=" * 80)
print("ToolBridge Test Token (HS256)")
print("=" * 80)
print(f"\nClaims:")
print(f"  sub: {claims['sub']}")
print(f"  iss: {claims['iss']}")
print(f"  aud: {claims['aud']}")
print(f"  iat: {datetime.fromtimestamp(claims['iat']).isoformat()}")
print(f"  exp: {datetime.fromtimestamp(claims['exp']).isoformat()} (30 days)")
print(f"\nToken:")
print(token)
print(f"\nUsage:")
print(f'  export AUTH_TOKEN="{token}"')
print(f'  curl -H "Authorization: Bearer $AUTH_TOKEN" https://toolbridgeapi.erauner.dev/api/v1/tenants')
print("=" * 80)
