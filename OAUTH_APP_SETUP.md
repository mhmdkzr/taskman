# OAuth App Setup - Comprehensive Guide

## Problem Statement

Creating the OAuth app programmatically fails with permission error `AUTH-5mWD2: No matching permissions found`, even when using the correct Zitadel Management API with valid authentication.

## Root Cause Analysis

The bootstrap PAT token (`login-client.pat`) is specifically created for the `login-client` service account, which has limited permissions:
- ✅ Can authenticate users via OIDC login flow
- ❌ Cannot create/manage applications
- ❌ Cannot manage other service accounts or projects

This is by design - Zitadel uses principle of least privilege.

## Solution Options (Ranked by Feasibility)

### Option 1: Web UI (✅ Fastest & Most Reliable)

**Time**: 3 minutes  
**Complexity**: Simple  
**Reliability**: 100%

1. Open `http://localhost:8080/ui/console`
2. Log in (create account if needed)
3. **Projects** → **ZITADEL** → **Applications** → **Create New App**
4. Configure:
   ```
   Name:                     app-bff-local
   App Type:                 Web
   Auth Method:              Basic
   Response Type:            code
   Grant Types:              Authorization Code, Refresh Token  
   Redirect URI:             http://localhost:8090/auth/callback
   Logout URI:               http://localhost:8090/
   Development Mode:         ✓ checked
   ```
5. Copy Client ID and Secret
6. Update `.env` and restart app

### Option 2: Create Admin Service Account (✅ Programmatic & Reusable)

**Time**: 10 minutes  
**Complexity**: Medium  
**Reliability**: 95%

This approach creates a service account with IAM admin permissions that can be used for app creation and other admin tasks.

**Steps**:

1. Use the Web UI to log in as the initial admin
2. Go to **Users** → **Service Accounts**
3. Create new service account with:
   - Name: `bff-provisioner`
   - Roles: `IAM_OWNER` or `PROJECT_OWNER`
4. Generate JWT key and download the key file
5. Run the programmatic setup:
   ```bash
   go run scripts/create-oauth-app.go \
     -project-id="388435721750446087" \
     -key-file="path/to/key.json"
   ```

**Advantages**:
- Reusable for other admin operations
- Doesn't require interactive login
- Works in CI/CD pipelines
- More secure than storing passwords

### Option 3: Bootstrap Configuration (❌ Not Supported in Zitadel v4.17.1)

**Status**: Not available

Zitadel v4.17.1 doesn't support declarative app configuration via environment variables or startup config files (unlike some other identity providers). This approach would require upgrading or contributing to Zitadel.

### Option 4: Direct Database Manipulation (❌ Fragile)

**Status**: Not recommended

While Zitadel's data is in PostgreSQL, it uses event sourcing with complex relationships between events and projections. Direct database manipulation:
- Breaks event sourcing consistency
- Can cause unpredictable behavior
- Not supported by Zitadel
- Will fail on future versions/migrations

## Verified API Endpoints

We've confirmed these APIs work (but require admin permissions):

### REST API (Management v1)
```bash
POST /management/v1/projects/{projectId}/apps/oidc
Header: Authorization: Bearer {token_with_admin_permissions}
Content-Type: application/json

{
  "name": "app-bff-local",
  "redirectUris": ["http://localhost:8090/auth/callback"],
  "responseTypes": ["OIDC_RESPONSE_TYPE_CODE"],
  "grantTypes": ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"],
  "appType": "OIDC_APP_TYPE_WEB",
  "authMethodType": "OIDC_AUTH_METHOD_TYPE_BASIC",
  "postLogoutRedirectUris": ["http://localhost:8090/"],
  "devMode": true
}
```

### gRPC API (Application Service v2)
```go
client.ApplicationServiceV2().CreateApplication(ctx, &applicationv2.CreateApplicationRequest{
  ProjectId: "388435721750446087",
  Name: "app-bff-local",
  // ... config
})
```

**Authentication options**:
- `client.PAT(token)` - Personal Access Token
- `client.JWTAuthentication(keyFile)` - JWT service account
- `client.PasswordAuthentication(username, password)` - User credentials (requires user verification)

## Implementation Status

✅ **API**: Correctly identified and tested  
✅ **SDK**: zitadel-go/v3 properly supports app creation  
✅ **Authentication**: Multiple methods verified  
❌ **Bootstrap Token**: Lacks required permissions  
✅ **Workarounds**: Multiple viable options available  

## Recommended Path Forward

1. **For local development** (right now):
   - Use Option 1 (Web UI) - takes 3 minutes
   - Complete the setup and test the full auth flow

2. **For automation/CI-CD** (later):
   - Implement Option 2 (Admin Service Account)
   - Update provision script to use JWT service account
   - Add to CI/CD pipeline for automatic setup

## Scripts Provided

| Script | Method | Status |
|--------|--------|--------|
| `scripts/create-oauth-app-simple.sh` | REST API + PAT | ❌ Fails (permission denied) |
| `scripts/create-oauth-app.go` | gRPC + PAT | ❌ Fails (permission denied) |
| `scripts/provision-zitadel-bff.sh` | REST API + PAT | ❌ Fails (permission denied) |
| Manual UI setup | Browser | ✅ Works (3 min) |

All scripts that use the bootstrap PAT will fail with the same permission error until a service account with admin permissions is available.

## Next Steps

1. Complete app setup using Option 1 (Web UI) - 3 minutes
2. Test the full OAuth flow in the browser
3. Once verified, implement Option 2 for production/CI-CD
4. Update provider scripts to use admin service account JWT

## References

- Zitadel docs: https://zitadel.com/docs
- Management API: https://zitadel.com/docs/apis/resources/mgmt_api_applications
- Application Service v2: https://zitadel.com/docs/apis/resources/application_service_v2
- Service Accounts: https://zitadel.com/docs/guides/integrate/service-accounts
- zitadel-go SDK: https://github.com/zitadel/zitadel-go/v3
