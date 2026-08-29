#!/bin/bash
# Simple HTTP API version to create OAuth app
# Uses Zitadel Management API v1 (REST)

set -e

PROJECT_ID="${1:-388435721750446087}"
ZITADEL_URL="${2:-http://127.0.0.1:8080}"

echo "Creating OAuth app in Zitadel via Management API..."
echo "URL: $ZITADEL_URL"
echo "Project: $PROJECT_ID"
echo ""

# Get bootstrap token
TOKEN_FILE=$(mktemp)
trap "rm -f $TOKEN_FILE" EXIT

echo "Getting bootstrap PAT..."
docker compose cp zitadel:/zitadel/bootstrap/login-client.pat "$TOKEN_FILE" 2>&1 | grep -v "Copying"
TOKEN=$(tr -d '\r\n' <"$TOKEN_FILE")

echo "Token obtained (length: ${#TOKEN})"
echo ""

# Create the app using management API
echo "Sending CreateApplication request..."
RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Host: localhost" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "app-bff-local",
    "redirectUris": ["http://localhost:8090/auth/callback"],
    "responseTypes": ["OIDC_RESPONSE_TYPE_CODE"],
    "grantTypes": ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"],
    "appType": "OIDC_APP_TYPE_WEB",
    "authMethodType": "OIDC_AUTH_METHOD_TYPE_BASIC",
    "postLogoutRedirectUris": ["http://localhost:8090/"],
    "devMode": true,
    "accessTokenType": "OIDC_TOKEN_TYPE_BEARER",
    "version": "OIDC_VERSION_1_0"
  }' \
  "$ZITADEL_URL/management/v1/projects/$PROJECT_ID/apps/oidc")

echo "Response:"
echo "$RESPONSE" | jq '.' || echo "$RESPONSE"
echo ""

# Check if successful
if echo "$RESPONSE" | grep -q "clientId"; then
    CLIENT_ID=$(echo "$RESPONSE" | jq -r '.clientId')
    CLIENT_SECRET=$(echo "$RESPONSE" | jq -r '.clientSecret')

    echo "✅ Success! OAuth app created!"
    echo ""
    echo "Update .env with:"
    echo "AUTH_CLIENT_ID=$CLIENT_ID"
    echo "AUTH_CLIENT_SECRET=$CLIENT_SECRET"
    echo ""
    echo "Then restart the app:"
    echo "docker-compose restart app"
elif echo "$RESPONSE" | grep -q "error"; then
    ERROR=$(echo "$RESPONSE" | jq -r '.error_description // .message // .error')
    echo "❌ Error creating app: $ERROR"
    exit 1
else
    echo "❌ Unexpected response"
    exit 1
fi
