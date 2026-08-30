#!/usr/bin/env sh
# Provisions the local Compose ZITADEL BFF client. Safe to rerun.
set -eu

app_name=app-bff-local
project_name=ZITADEL
redirect_url=http://localhost:8090/auth/callback
post_logout_url=http://localhost:8090/
login_ui_base_uri=http://localhost:8090/
login_client_pat_path=/zitadel/bootstrap/login-client.pat
token_file=$(mktemp)
response_file=$(mktemp)
trap 'rm -f "$token_file" "$response_file"' EXIT

docker compose cp zitadel:/zitadel/bootstrap/admin-provisioner.pat "$token_file" >/dev/null
token=$(tr -d '\r\n' <"$token_file")
auth_header="Authorization: Bearer $token"

project_id=$(curl -fsS -X POST -H "$auth_header" -H 'Host: localhost' -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8080/management/v1/projects/_search |
	jq -er --arg name "$project_name" '.result[] | select(.name == $name) | .id')

apps=$(curl -fsS -X POST -H "$auth_header" -H 'Host: localhost' -H 'Content-Type: application/json' -d '{}' "http://127.0.0.1:8080/management/v1/projects/$project_id/apps/_search")
app_id=$(printf '%s' "$apps" | jq -r --arg name "$app_name" '.result[] | select(.name == $name) | .id' | head -n1)
client_id=$(printf '%s' "$apps" | jq -r --arg name "$app_name" '.result[] | select(.name == $name) | .oidcConfig.clientId' | head -n1)

if [ -z "$app_id" ] || [ "$app_id" = null ]; then
	curl -fsS -X POST -H "$auth_header" -H 'Host: localhost' -H 'Content-Type: application/json' \
		-d "$(jq -n --arg name "$app_name" --arg redirect "$redirect_url" --arg logout "$post_logout_url" '{name: $name, redirectUris: [$redirect], responseTypes: ["OIDC_RESPONSE_TYPE_CODE"], grantTypes: ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"], appType: "OIDC_APP_TYPE_WEB", authMethodType: "OIDC_AUTH_METHOD_TYPE_BASIC", postLogoutRedirectUris: [$logout], version: "OIDC_VERSION_1_0", devMode: true, accessTokenType: "OIDC_TOKEN_TYPE_BEARER"}')" \
		"http://127.0.0.1:8080/management/v1/projects/$project_id/apps/oidc" >"$response_file"
	app_id=$(jq -er '.appId' <"$response_file")
	client_id=$(jq -er '.clientId' <"$response_file")
	client_secret=$(jq -er '.clientSecret' <"$response_file")
else
	client_secret=$(sed -n 's/^AUTH_CLIENT_SECRET=//p' .env)
	if [ -z "$client_secret" ]; then
		curl -fsS -X POST -H "$auth_header" -H 'Host: localhost' "http://127.0.0.1:8080/management/v1/projects/$project_id/apps/$app_id/oidc_config/_generate_client_secret" >"$response_file"
		client_secret=$(jq -er '.clientSecret' <"$response_file")
	fi
fi

update_env() {
	key=$1
	value=$2
	awk -v key="$key" -v value="$value" 'BEGIN { found = 0 } $0 ~ "^" key "=" { print key "=" value; found = 1; next } { print } END { if (!found) print key "=" value }' .env >.env.provisioned
	mv .env.provisioned .env
}

update_env AUTH_ENABLED true
update_env AUTH_ISSUER http://localhost:8080
update_env AUTH_CLIENT_ID "$client_id"
update_env AUTH_CLIENT_SECRET "$client_secret"
update_env AUTH_REDIRECT_URL "$redirect_url"
update_env AUTH_POST_LOGOUT_REDIRECT_URL "$post_logout_url"
update_env AUTH_LOGIN_CLIENT_PAT_PATH "$login_client_pat_path"
update_env AUTH_COOKIE_SECURE false

# Point ZITADEL's Login V2 base URI at our own custom login UI
# (internal/auth/loginui + frontend/src/lib/components/LoginForm.svelte)
# instead of the zitadel-login container, so authorization requests never
# redirect to ZITADEL's hosted UI. Idempotent: safe to rerun.
curl -fsS -X PUT -H "$auth_header" -H 'Host: localhost' -H 'Content-Type: application/json' \
	-d "$(jq -n --arg uri "$login_ui_base_uri" '{loginV2: {required: true, baseUri: $uri}}')" \
	http://127.0.0.1:8080/v2/features/instance >/dev/null

printf 'ZITADEL BFF client %s configured. Client secret was written only to .env.\n' "$app_id"
printf 'Login V2 base URI set to %s (zitadel-login is no longer used for authorization requests).\n' "$login_ui_base_uri"
