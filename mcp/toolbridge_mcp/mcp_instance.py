"""
MCP server instance with OAuth 2.1 authentication.

This module creates the MCP server instance configured with Auth0Provider
for per-user authentication via browser-based OAuth 2.1 + PKCE flow.
"""

from fastmcp import FastMCP
from fastmcp.server.auth.providers.auth0 import Auth0Provider
from loguru import logger

from toolbridge_mcp.config import settings

# Validate OAuth configuration at module load
settings.validate_oauth_config()

# Create OAuth provider for per-user authentication
# Users authenticate via claude.ai web UI → browser → Auth0 login
auth_provider = Auth0Provider(
    domain=settings.oauth_domain,
    client_id=settings.oauth_client_id,
    client_secret=settings.oauth_client_secret or "",  # Empty string for public clients
    base_url=settings.oauth_base_url,
    # Scopes that users will consent to during OAuth flow
    required_scopes=[
        "openid",
        "profile",
        "email",
        "read:notes",
        "write:notes",
        "read:tasks",
        "write:tasks",
        "read:comments",
        "write:comments",
        "read:chats",
        "write:chats",
    ],
    # MCP server audience (this server's URL)
    audience=settings.oauth_audience,
)

logger.info(f"✓ Auth0Provider configured: domain={settings.oauth_domain}, audience={settings.oauth_audience}")

# Create MCP server instance with OAuth authentication
mcp = FastMCP(
    name="ToolBridge",
    auth=auth_provider,
)
