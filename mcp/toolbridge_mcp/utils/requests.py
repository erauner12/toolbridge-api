"""
HTTP request helpers for calling the Go API.

These helpers extract the Authorization header from the current MCP request
context and forward it to the Go API along with tenant headers (which are
added automatically by TenantDirectTransport).
"""

from typing import Any, Dict, Optional

import httpx
from fastmcp.server.dependencies import get_http_headers
from loguru import logger


class AuthorizationError(Exception):
    """Raised when Authorization header is missing from MCP request."""
    pass


async def get_auth_header() -> str:
    """
    Extract Authorization header from current MCP request context.
    
    Uses FastMCP's dependency injection to access request headers.
    
    Returns:
        Authorization header value (e.g., "Bearer eyJ...")
    
    Raises:
        AuthorizationError: If Authorization header is missing
    """
    headers = get_http_headers()
    auth = headers.get("authorization") or headers.get("Authorization")
    
    if not auth:
        logger.error("Missing Authorization header in MCP request")
        raise AuthorizationError(
            "Missing Authorization header. MCP client must provide JWT token."
        )
    
    return auth


async def call_get(
    client: httpx.AsyncClient,
    path: str,
    params: Optional[Dict[str, Any]] = None,
) -> httpx.Response:
    """
    Make GET request to Go API.
    
    Args:
        client: httpx client (with TenantDirectTransport)
        path: API endpoint path (e.g., "/v1/notes")
        params: Query parameters
    
    Returns:
        HTTP response
    
    Raises:
        httpx.HTTPStatusError: If request fails
        AuthorizationError: If Authorization header missing
    """
    auth_header = await get_auth_header()
    headers = {"Authorization": auth_header}
    
    logger.debug(f"GET {path} params={params}")
    response = await client.get(path, params=params, headers=headers)
    response.raise_for_status()
    return response


async def call_post(
    client: httpx.AsyncClient,
    path: str,
    json: Optional[Dict[str, Any]] = None,
) -> httpx.Response:
    """
    Make POST request to Go API.
    
    Args:
        client: httpx client (with TenantDirectTransport)
        path: API endpoint path (e.g., "/v1/notes")
        json: JSON request body
    
    Returns:
        HTTP response
    
    Raises:
        httpx.HTTPStatusError: If request fails
        AuthorizationError: If Authorization header missing
    """
    auth_header = await get_auth_header()
    headers = {"Authorization": auth_header}
    
    logger.debug(f"POST {path}")
    response = await client.post(path, json=json, headers=headers)
    response.raise_for_status()
    return response


async def call_put(
    client: httpx.AsyncClient,
    path: str,
    json: Optional[Dict[str, Any]] = None,
    if_match: Optional[int] = None,
) -> httpx.Response:
    """
    Make PUT request to Go API.
    
    Args:
        client: httpx client (with TenantDirectTransport)
        path: API endpoint path (e.g., "/v1/notes/{uid}")
        json: JSON request body
        if_match: Optional version for optimistic locking
    
    Returns:
        HTTP response
    
    Raises:
        httpx.HTTPStatusError: If request fails
        AuthorizationError: If Authorization header missing
    """
    auth_header = await get_auth_header()
    headers = {"Authorization": auth_header}
    
    if if_match is not None:
        headers["If-Match"] = str(if_match)
    
    logger.debug(f"PUT {path} if_match={if_match}")
    response = await client.put(path, json=json, headers=headers)
    response.raise_for_status()
    return response


async def call_patch(
    client: httpx.AsyncClient,
    path: str,
    json: Optional[Dict[str, Any]] = None,
) -> httpx.Response:
    """
    Make PATCH request to Go API.
    
    Args:
        client: httpx client (with TenantDirectTransport)
        path: API endpoint path (e.g., "/v1/notes/{uid}")
        json: JSON request body (partial update)
    
    Returns:
        HTTP response
    
    Raises:
        httpx.HTTPStatusError: If request fails
        AuthorizationError: If Authorization header missing
    """
    auth_header = await get_auth_header()
    headers = {"Authorization": auth_header}
    
    logger.debug(f"PATCH {path}")
    response = await client.patch(path, json=json, headers=headers)
    response.raise_for_status()
    return response


async def call_delete(
    client: httpx.AsyncClient,
    path: str,
) -> httpx.Response:
    """
    Make DELETE request to Go API.
    
    Args:
        client: httpx client (with TenantDirectTransport)
        path: API endpoint path (e.g., "/v1/notes/{uid}")
    
    Returns:
        HTTP response
    
    Raises:
        httpx.HTTPStatusError: If request fails
        AuthorizationError: If Authorization header missing
    """
    auth_header = await get_auth_header()
    headers = {"Authorization": auth_header}
    
    logger.debug(f"DELETE {path}")
    response = await client.delete(path, headers=headers)
    response.raise_for_status()
    return response
