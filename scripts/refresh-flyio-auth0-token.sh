#!/usr/bin/env bash
# Refresh Auth0 access token in Fly.io MCP deployment
#
# WORKAROUND: This script manually refreshes the Auth0 access token
# used by the Fly.io MCP server. Auth0 tokens expire after 24 hours,
# so this needs to run daily until the proper fix is implemented.
#
# Proper fix: See docs/WORKAROUND-AUTH0-TOKEN-REFRESH.md
#
# Usage:
#   ./scripts/refresh-flyio-auth0-token.sh
#
# Prerequisites:
#   - fly CLI authenticated
#   - terraform outputs available in homelab-k8s/terraform/auth0/
#   - jq and curl installed

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FLY_APP="toolbridge-mcp-staging"
AUTH0_DOMAIN="dev-zysv6k3xo7pkwmcb.us.auth0.com"
AUTH0_AUDIENCE="https://toolbridgeapi.erauner.dev"
TERRAFORM_DIR="${HOME}/git/side/homelab-k8s/terraform/auth0"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Fly.io Auth0 Token Refresh (Workaround)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}⚠️  WORKAROUND: This script manually refreshes Auth0 tokens${NC}"
echo -e "${YELLOW}   See docs/WORKAROUND-AUTH0-TOKEN-REFRESH.md for proper fix${NC}"
echo ""

# Step 1: Check prerequisites
echo -e "${BLUE}[1/5]${NC} Checking prerequisites..."

if ! command -v fly &> /dev/null; then
    echo -e "${RED}✗ fly CLI not found${NC}"
    echo "Install: https://fly.io/docs/hands-on/install-flyctl/"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${RED}✗ jq not found${NC}"
    echo "Install: brew install jq"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    echo -e "${RED}✗ curl not found${NC}"
    exit 1
fi

if [ ! -d "$TERRAFORM_DIR" ]; then
    echo -e "${RED}✗ Terraform directory not found: $TERRAFORM_DIR${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All prerequisites satisfied${NC}"

# Step 2: Extract Auth0 client credentials from terraform
echo -e "${BLUE}[2/5]${NC} Extracting Auth0 credentials from terraform..."

cd "$TERRAFORM_DIR"

CLIENT_ID=$(terraform output -json mcp_introspection_client 2>/dev/null | jq -r '.client_id')
CLIENT_SECRET=$(terraform output -json mcp_introspection_client 2>/dev/null | jq -r '.client_secret')

if [ -z "$CLIENT_ID" ] || [ "$CLIENT_ID" = "null" ]; then
    echo -e "${RED}✗ Failed to extract client_id from terraform${NC}"
    exit 1
fi

if [ -z "$CLIENT_SECRET" ] || [ "$CLIENT_SECRET" = "null" ]; then
    echo -e "${RED}✗ Failed to extract client_secret from terraform${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Auth0 client credentials extracted${NC}"
echo "  Client ID: ${CLIENT_ID:0:20}..."

# Return to original directory
cd - > /dev/null

# Step 3: Request new access token from Auth0
echo -e "${BLUE}[3/5]${NC} Requesting new Auth0 access token..."

TOKEN_RESPONSE=$(curl -s -X POST "https://${AUTH0_DOMAIN}/oauth/token" \
  -H 'content-type: application/json' \
  -d "{
    \"client_id\":\"${CLIENT_ID}\",
    \"client_secret\":\"${CLIENT_SECRET}\",
    \"audience\":\"${AUTH0_AUDIENCE}\",
    \"grant_type\":\"client_credentials\"
  }")

ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo -e "${RED}✗ Failed to get access token${NC}"
    echo "Response: $TOKEN_RESPONSE"
    exit 1
fi

# Decode token to show expiry
EXPIRES_IN=$(echo "$TOKEN_RESPONSE" | jq -r '.expires_in')
EXPIRES_AT=$(date -u -v+${EXPIRES_IN}S '+%Y-%m-%d %H:%M:%S UTC' 2>/dev/null || date -u -d "+${EXPIRES_IN} seconds" '+%Y-%m-%d %H:%M:%S UTC')

echo -e "${GREEN}✓ Auth0 access token obtained${NC}"
echo "  Token: ${ACCESS_TOKEN:0:50}..."
echo "  Expires in: ${EXPIRES_IN}s (~$(($EXPIRES_IN / 3600)) hours)"
echo "  Expires at: $EXPIRES_AT"

# Step 4: Update Fly.io secret
echo -e "${BLUE}[4/5]${NC} Updating Fly.io secret..."

fly secrets set "TOOLBRIDGE_JWT_TOKEN=${ACCESS_TOKEN}" -a "$FLY_APP"

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to update Fly.io secret${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Fly.io secret updated${NC}"

# Step 5: Verify deployment
echo -e "${BLUE}[5/5]${NC} Verifying deployment..."

sleep 5

STATUS=$(fly status -a "$FLY_APP" 2>&1)

if echo "$STATUS" | grep -q "passing"; then
    echo -e "${GREEN}✓ Deployment healthy${NC}"
    echo "$STATUS" | grep -E "(Machines|CHECKS)"
else
    echo -e "${YELLOW}⚠ Deployment status unclear${NC}"
    echo "$STATUS"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ Token refresh complete${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📅 Next refresh needed before: $EXPIRES_AT${NC}"
echo -e "${YELLOW}💡 Set up cron job to run this daily:${NC}"
echo "   0 0 * * * cd /path/to/toolbridge-api && ./scripts/refresh-flyio-auth0-token.sh"
echo ""
echo -e "${YELLOW}🔧 For permanent fix, see: docs/WORKAROUND-AUTH0-TOKEN-REFRESH.md${NC}"
echo ""
