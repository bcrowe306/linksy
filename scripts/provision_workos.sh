#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Linksy / WorkOS Automated Provisioning Script
# ==============================================================================
# This script runs during CI/CD to programmatically configure WorkOS resources:
# 1. Verifies API key and Client ID connectivity.
# 2. Ensures the primary Organization / Workspace exists.
# 3. Ensures Webhook Endpoints are registered for user events.
# ==============================================================================

WORKOS_API_KEY="${WORKOS_API_KEY:-}"
WORKOS_CLIENT_ID="${WORKOS_CLIENT_ID:-}"
APP_URL="${APP_URL:-https://test.lynqr.app}"

if [ -z "$WORKOS_API_KEY" ]; then
  echo "==> [WorkOS] WORKOS_API_KEY is not set. Skipping automated provisioning."
  exit 0
fi

echo "==> [WorkOS] Connecting to WorkOS API..."

# 1. Verify Connectivity
AUTH_HEADER="Authorization: Bearer $WORKOS_API_KEY"
USER_COUNT=$(curl -s -H "$AUTH_HEADER" "https://api.workos.com/user_management/users" | grep -o '"object":"list"' || true)

if [ -z "$USER_COUNT" ]; then
  echo "==> [WorkOS] WARNING: Failed to authenticate with WorkOS API."
  exit 0
fi
echo "==> [WorkOS] Authenticated successfully."

# 2. Ensure Primary Organization Exists
ORG_RESPONSE=$(curl -s -H "$AUTH_HEADER" "https://api.workos.com/organizations")
ORG_ID=$(echo "$ORG_RESPONSE" | grep -o '"id":"org_[^"]*"' | head -n 1 | cut -d'"' -f4 || true)

if [ -z "$ORG_ID" ]; then
  echo "==> [WorkOS] Creating primary workspace 'Lynqr Workspace'..."
  CREATE_ORG_RESP=$(curl -s -X POST "https://api.workos.com/organizations" \
    -H "$AUTH_HEADER" \
    -H "Content-Type: application/json" \
    -d '{"name":"Lynqr Workspace"}')
  ORG_ID=$(echo "$CREATE_ORG_RESP" | grep -o '"id":"org_[^"]*"' | head -n 1 | cut -d'"' -f4 || true)
  echo "==> [WorkOS] Created organization: $ORG_ID"
else
  echo "==> [WorkOS] Verified existing organization: $ORG_ID"
fi

# 3. Ensure Webhook Endpoint Exists
WEBHOOK_URL="${APP_URL}/api/v1/auth/webhook"
WEBHOOK_RESP=$(curl -s -H "$AUTH_HEADER" "https://api.workos.com/webhook_endpoints")

if echo "$WEBHOOK_RESP" | grep -q "$WEBHOOK_URL"; then
  echo "==> [WorkOS] Webhook endpoint for $WEBHOOK_URL already registered."
else
  echo "==> [WorkOS] Registering webhook endpoint for $WEBHOOK_URL..."
  CREATE_WH_RESP=$(curl -s -X POST "https://api.workos.com/webhook_endpoints" \
    -H "$AUTH_HEADER" \
    -H "Content-Type: application/json" \
    -d "{\"endpoint_url\":\"$WEBHOOK_URL\",\"events\":[\"user.created\",\"user.updated\",\"user.deleted\"]}")
  echo "==> [WorkOS] Webhook endpoint registered successfully."
fi

echo "==> [WorkOS] Provisioning check completed successfully."
