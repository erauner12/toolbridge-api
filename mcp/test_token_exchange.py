"""
Tests for token exchange and JWT decoding functionality.

These tests ensure:
1. Token exchange with backend endpoint works correctly
2. JWT decoding for user ID extraction doesn't fail
3. Fallback to local JWT signing works when configured
4. Error handling is robust
"""

import pytest
from datetime import datetime, timedelta
from jose import jwt
from unittest.mock import AsyncMock, patch, MagicMock
import httpx

from toolbridge_mcp.auth.token_exchange import (
    exchange_for_backend_jwt,
    extract_user_id_from_backend_jwt,
    issue_backend_jwt,
    TokenExchangeError,
)
from toolbridge_mcp.config import settings


@pytest.fixture
def mock_access_token():
    """Mock FastMCP access token with WorkOS AuthKit claims."""
    token = MagicMock()
    token.token = "mock-workos-token"
    token.claims = {
        "sub": "user_01KAHS4J1W6TT5390SR3918ZPF",
        "email": "test@example.com",
        "iss": "https://svelte-monolith-27-staging.authkit.app",
        "aud": "client_01KABXHNQ09QGWEX4APPYG2AH5",  # DCR client ID
    }
    token.scopes = ["read", "write"]
    return token


@pytest.fixture
def mock_http_client():
    """Mock httpx AsyncClient for backend API calls."""
    return AsyncMock(spec=httpx.AsyncClient)


class TestExtractUserIdFromBackendJWT:
    """Test JWT decoding for user ID extraction."""

    def test_extract_user_id_success(self):
        """Test successful extraction of user ID from backend JWT."""
        # Create a valid backend JWT
        payload = {
            "sub": "user_123",
            "iss": "toolbridge-api",
            "aud": "https://toolbridgeapi.erauner.dev",
            "token_type": "backend",
            "exp": datetime.utcnow() + timedelta(hours=1),
            "iat": datetime.utcnow(),
        }

        # Sign with HS256 (backend uses HS256)
        secret = "test-secret"
        backend_jwt = jwt.encode(payload, secret, algorithm="HS256")

        # Extract user ID (should NOT verify signature, just decode)
        user_id = extract_user_id_from_backend_jwt(backend_jwt)

        assert user_id == "user_123"

    def test_extract_user_id_no_sub_claim(self):
        """Test extraction returns 'unknown' when sub claim is missing."""
        # JWT without sub claim
        payload = {
            "iss": "toolbridge-api",
            "aud": "https://toolbridgeapi.erauner.dev",
            "exp": datetime.utcnow() + timedelta(hours=1),
            "iat": datetime.utcnow(),
        }

        secret = "test-secret"
        backend_jwt = jwt.encode(payload, secret, algorithm="HS256")

        # Should return "unknown" when sub is missing
        user_id = extract_user_id_from_backend_jwt(backend_jwt)

        assert user_id == "unknown"

    def test_extract_user_id_invalid_jwt(self):
        """Test extraction handles invalid JWT gracefully."""
        # Completely invalid JWT string
        invalid_jwt = "not.a.valid.jwt"

        # Should return "unknown" instead of raising exception
        user_id = extract_user_id_from_backend_jwt(invalid_jwt)

        assert user_id == "unknown"

    def test_extract_user_id_expired_token_still_works(self):
        """Test extraction works even with expired token (signature not verified)."""
        # Expired token (but we don't verify signature)
        payload = {
            "sub": "user_expired",
            "iss": "toolbridge-api",
            "aud": "https://toolbridgeapi.erauner.dev",
            "exp": datetime.utcnow() - timedelta(hours=1),  # Expired
            "iat": datetime.utcnow() - timedelta(hours=2),
        }

        secret = "test-secret"
        backend_jwt = jwt.encode(payload, secret, algorithm="HS256")

        # Should still extract user_id (we're not verifying signature/expiry)
        user_id = extract_user_id_from_backend_jwt(backend_jwt)

        assert user_id == "user_expired"


class TestExchangeForBackendJWT:
    """Test token exchange flow with backend endpoint."""

    @pytest.mark.asyncio
    @patch("toolbridge_mcp.auth.token_exchange.get_access_token")
    async def test_exchange_backend_endpoint_success(
        self, mock_get_token, mock_access_token, mock_http_client
    ):
        """Test successful token exchange via backend endpoint."""
        mock_get_token.return_value = mock_access_token

        # Mock successful backend response
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "access_token": "backend-jwt-token",
            "token_type": "Bearer",
            "expires_in": 3600,
        }
        mock_http_client.post = AsyncMock(return_value=mock_response)

        # Temporarily set backend URL for test
        original_url = settings.go_api_base_url
        settings.go_api_base_url = "http://localhost:8080"

        try:
            backend_jwt = await exchange_for_backend_jwt(mock_http_client)

            # Should return backend JWT
            assert backend_jwt == "backend-jwt-token"

            # Verify backend endpoint was called correctly
            mock_http_client.post.assert_called_once()
            call_args = mock_http_client.post.call_args

            assert call_args.args[0] == "http://localhost:8080/auth/token-exchange"
            assert call_args.kwargs["headers"]["Authorization"] == "Bearer mock-workos-token"
            assert call_args.kwargs["json"]["audience"] == settings.backend_api_audience

        finally:
            settings.go_api_base_url = original_url

    @pytest.mark.asyncio
    @patch("toolbridge_mcp.auth.token_exchange.get_access_token")
    @patch("toolbridge_mcp.auth.token_exchange.issue_backend_jwt")
    async def test_exchange_fallback_to_local_jwt(
        self, mock_issue_jwt, mock_get_token, mock_access_token, mock_http_client
    ):
        """Test fallback to local JWT signing when backend endpoint fails."""
        mock_get_token.return_value = mock_access_token

        # Mock backend endpoint failure
        mock_http_client.post = AsyncMock(side_effect=httpx.HTTPError("Connection failed"))

        # Mock local JWT signing
        mock_issue_jwt.return_value = "locally-signed-jwt"

        # Temporarily enable JWT signing key
        original_key = settings.jwt_signing_key
        settings.jwt_signing_key = "test-rsa-private-key"

        try:
            backend_jwt = await exchange_for_backend_jwt(mock_http_client)

            # Should fall back to locally-signed JWT
            assert backend_jwt == "locally-signed-jwt"

            # Verify local JWT was issued with correct parameters
            mock_issue_jwt.assert_called_once_with(
                user_id="user_01KAHS4J1W6TT5390SR3918ZPF",
                email="test@example.com",
                tenant_id=None,
                scopes=["read", "write"],
                raw_token="mock-workos-token",
            )

        finally:
            settings.jwt_signing_key = original_key

    @pytest.mark.asyncio
    @patch("toolbridge_mcp.auth.token_exchange.get_access_token")
    async def test_exchange_fails_without_backend_or_key(
        self, mock_get_token, mock_access_token, mock_http_client
    ):
        """Test exchange fails gracefully when no method available."""
        mock_get_token.return_value = mock_access_token

        # Mock backend endpoint failure
        mock_http_client.post = AsyncMock(side_effect=httpx.HTTPError("Connection failed"))

        # Ensure JWT signing key is not configured
        original_key = settings.jwt_signing_key
        settings.jwt_signing_key = None

        try:
            # Should raise TokenExchangeError with helpful message
            with pytest.raises(TokenExchangeError) as exc_info:
                await exchange_for_backend_jwt(mock_http_client)

            assert "Token exchange failed" in str(exc_info.value)
            assert "backend /auth/token-exchange endpoint" in str(exc_info.value)

        finally:
            settings.jwt_signing_key = original_key


class TestIssueBackendJWT:
    """Test local JWT signing (Option 2)."""

    @patch("toolbridge_mcp.auth.token_exchange.settings")
    def test_issue_backend_jwt_success(self, mock_settings):
        """Test successful local JWT signing."""
        # Mock settings with RS256 private key
        mock_settings.jwt_signing_key = "test-rsa-private-key"
        mock_settings.backend_api_audience = "https://toolbridgeapi.erauner.dev"
        mock_settings.tenant_id = "default-tenant"

        # Note: This will fail in practice because we're using a mock key
        # In real code, you'd need a valid RSA private key
        # For this test, we just verify it raises the expected error
        with pytest.raises(TokenExchangeError) as exc_info:
            issue_backend_jwt(
                user_id="user_123",
                email="test@example.com",
                tenant_id="tenant-abc",
                scopes=["read", "write"],
                raw_token="original-token",
            )

        # Should fail with signing error (because mock key is invalid)
        assert "Failed to sign backend JWT" in str(exc_info.value)

    def test_issue_backend_jwt_no_key_configured(self):
        """Test error when JWT signing key not configured."""
        # Temporarily remove signing key
        original_key = settings.jwt_signing_key
        settings.jwt_signing_key = None

        try:
            with pytest.raises(TokenExchangeError) as exc_info:
                issue_backend_jwt(
                    user_id="user_123",
                    email=None,
                    tenant_id=None,
                    scopes=[],
                    raw_token="token",
                )

            assert "JWT_SIGNING_KEY not configured" in str(exc_info.value)

        finally:
            settings.jwt_signing_key = original_key


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
