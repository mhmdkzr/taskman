#!/bin/bash
# Create OAuth app in Zitadel programmatically using Go SDK
#
# This attempts to use the Zitadel Management API v2 to create the OAuth app.
# It handles several authentication methods:
# 1. PAT (Personal Access Token) - from bootstrap
# 2. JWT Service Account - from key file
# 3. User credentials - prompts for username/password

set -e

PROJECT_ID="${1:-388435721750446087}"
ISSUER="${2:-http://localhost:8080}"

echo "Creating OAuth app in Zitadel..."
echo "Project ID: $PROJECT_ID"
echo "Issuer: $ISSUER"
echo ""

# Try method 1: Use bootstrap PAT
echo "Attempting authentication with bootstrap PAT..."
TOKEN_FILE=$(mktemp)
trap "rm -f $TOKEN_FILE" EXIT

if docker-compose cp zitadel:/zitadel/bootstrap/login-client.pat "$TOKEN_FILE" 2>/dev/null; then
    TOKEN=$(tr -d '\r\n' <"$TOKEN_FILE")

    echo "✓ Found bootstrap PAT"
    echo "Building Go program..."

    # Build and run the Go program
    cd "$(dirname "$0")/.."
    go run scripts/create-oauth-app.go \
        -issuer="$ISSUER" \
        -project-id="$PROJECT_ID" \
        -token="$TOKEN"

    exit $?
else
    echo "✗ Could not get bootstrap PAT"
fi

echo ""
echo "Alternative methods:"
echo "1. Create app manually via UI: http://localhost:8080/ui/console"
echo "2. Provide JWT service account key file"
echo "3. Log in as admin user and generate a PAT"
echo ""
echo "See DEBUG_AUTH_SETUP.md for detailed instructions"
