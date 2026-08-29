#!/usr/bin/env sh
# Provision Zitadel BFF OAuth app - V2 with proper bootstrap PAT usage
# Based on Zitadel research: bootstrap PAT has full admin privileges
set -eu

app_name=app-bff-local
project_name=ZITADEL
redirect_url=http://localhost:8090/auth/callback
post_logout_url=http://localhost:8090/

# Extract bootstrap PAT - this has full admin privileges
echo "Extracting bootstrap PAT from Zitadel container..."
token_file=$(mktemp)
trap 'rm -f "$token_file"' EXIT

docker compose cp zitadel:/zitadel/bootstrap/login-client.pat "$token_file" 2>&1 | grep -v "Copying" || true
token=$(tr -d '\r\n' <"$token_file")

if [ -z "$token" ]; then
    echo "❌ Failed to extract bootstrap PAT"
    exit 1
fi

echo "✓ Bootstrap PAT extracted (length: ${#token})"

# Get project ID
echo "Finding ZITADEL project..."
project_id=$(curl -fsS -X POST \
    -H "Authorization: Bearer $token" \
    -H 'Host: localhost' \
    -H 'Content-Type: application/json' \
    -d '{}' \
    http://127.0.0.1:8080/management/v1/projects/_search | \
    jq -er --arg name "$project_name" '.result[] | select(.name == $name) | .id' 2>/dev/null)

if [ -z "$project_id" ]; then
    echo "❌ Could not find ZITADEL project"
    exit 1
fi

echo "✓ Project ID: $project_id"

# Check for existing app
echo "Checking for existing app..."
existing_app=$(curl -fsS -X POST \
    -H "Authorization: Bearer $token" \
    -H 'Host: localhost' \
    -H 'Content-Type: application/json' \
    -d '{}' \
    "http://127.0.0.1:8080/management/v1/projects/$project_id/apps/_search" | \
    jq -r --arg name "$app_name" '.result[] | select(.name == $name) | .id' 2>/dev/null | head -n1)

if [ -n "$existing_app" ]; then
    echo "⚠️  App already exists: $existing_app"
    client_id=$(curl -fsS -X POST \
        -H "Authorization: Bearer $token" \
        -H 'Host: localhost' \
        -H 'Content-Type: application/json' \
        -d '{}' \
        "http://127.0.0.1:8080/management/v1/projects/$project_id/apps/_search" | \
        jq -r --arg name "$app_name" '.result[] | select(.name == $name) | .oidcConfig.clientId' 2>/dev/null | head -n1)

    if [ -z "$client_id" ]; then
        echo "❌ Could not get client ID from existing app"
        exit 1
    fi

    echo "Using existing client ID: $client_id"
else
    echo "Creating new OAuth app..."
    response=$(curl -fsS -X POST \
        -H "Authorization: Bearer $token" \
        -H 'Host: localhost' \
        -H 'Content-Type: application/json' \
        -d "$(jq -n \
            --arg name "$app_name" \
            --arg redirect "$redirect_url" \
            --arg logout "$post_logout_url" \
            '{name: $name, redirectUris: [$redirect], responseTypes: ["OIDC_RESPONSE_TYPE_CODE"], grantTypes: ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"], appType: "OIDC_APP_TYPE_WEB", authMethodType: "OIDC_AUTH_METHOD_TYPE_BASIC", postLogoutRedirectUris: [$logout], version: "OIDC_VERSION_1_0", devMode: true, accessTokenType: "OIDC_TOKEN_TYPE_BEARER"}')" \
        "http://127.0.0.1:8080/management/v1/projects/$project_id/apps/oidc" 2>&1)

    if echo "$response" | grep -q "error"; then
        error_msg=$(echo "$response" | jq -r '.error_description // .message // .error' 2>/dev/null || echo "$response")
        echo "❌ Failed to create app: $error_msg"
        echo "Full response: $response"
        exit 1
    fi

    client_id=$(echo "$response" | jq -er '.clientId' 2>/dev/null)
    client_secret=$(echo "$response" | jq -er '.clientSecret' 2>/dev/null)

    if [ -z "$client_id" ] || [ -z "$client_secret" ]; then
        echo "❌ Invalid response from API"
        echo "Response: $response"
        exit 1
    fi

    echo "✓ OAuth app created"
    echo "  Client ID: $client_id"
fi

# Update .env
update_env() {
    key=$1
    value=$2
    awk -v key="$key" -v value="$value" \
        'BEGIN { found = 0 } $0 ~ "^" key "=" { print key "=" value; found = 1; next } { print } END { if (!found) print key "=" value }' \
        .env >.env.provisioned
    mv .env.provisioned .env
}

echo "Updating .env..."
update_env AUTH_ENABLED true
update_env AUTH_ISSUER http://localhost:8080
update_env AUTH_CLIENT_ID "$client_id"
if [ -n "${client_secret:-}" ]; then
    update_env AUTH_CLIENT_SECRET "$client_secret"
fi
update_env AUTH_REDIRECT_URL "$redirect_url"
update_env AUTH_POST_LOGOUT_REDIRECT_URL "$post_logout_url"
update_env AUTH_COOKIE_SECURE false

echo ""
echo "✅ Setup complete!"
echo "App configuration:"
echo "  Name: app-bff-local"
echo "  Client ID: $client_id"
echo ""
echo "Restart the app to apply changes:"
echo "  docker-compose restart app"
