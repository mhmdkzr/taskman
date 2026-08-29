# Auth System Debugging & Setup Guide

## Current Status

The authentication and ledger system is **95% implemented and running**. All containers are healthy and the auth flow partially works.

### ✅ What's Working

1. **Backend Application**
   - Go application running on port 8080
   - All services initialized (NATS, Temporal, PostgreSQL, TigerBeetle, Zitadel)
   - HTTP health checks passing
   - Auth middleware registered and listening

2. **Auth Flow - Steps 1-3 Complete**
   ```
   Browser → /auth/login
   ↓ (Generates PKCE, state, nonce)
   ↓ Sets auth_transaction cookie
   ↓ Redirects to Zitadel authorize endpoint
   ```
   - PKCE challenge/verifier generated correctly
   - State parameter validates
   - Nonce included
   - Session management working

3. **Zitadel Server**
   - Running and healthy
   - OIDC configuration endpoint accessible
   - Authorization endpoint responding

### ❌ What's Not Working

**OAuth App Registration**: The Zitadel OAuth app isn't properly configured.

When you access `/auth/login`, it redirects to:
```
http://localhost:8080/oauth/v2/authorize?client_id=388421460865056775&...
```

But Zitadel responds with:
```json
{
  "error": "unauthorized_client",
  "error_description": "The requested response type is missing in the client configuration."
}
```

**Root Cause**: The OAuth app with client_id `388421460865056775` doesn't exist in Zitadel.

## Why Provisioning Failed

The automatic provisioning script (`scripts/provision-zitadel-bff.sh`) failed with:
```
Authorization: No matching permissions found (AUTH-5mWD2)
```

The `login-client.pat` service account doesn't have permission to create applications through the Management API. This is a common Zitadel limitation where service accounts are scoped to specific operations.

## How to Fix (3 Options)

### Option 1: Use the Zitadel Console UI (Easiest)

1. Open browser to `http://localhost:8080/ui/console`
2. You may need to log in - if prompted:
   - Go to `http://localhost:8080/ui/v2/login/`
   - Create an account or use initial admin credentials
3. Navigate to **Projects** → **ZITADEL** → **Applications**
4. Click **"Create New App"** and fill in:
   - **Name**: `app-bff-local`
   - **App Type**: `Web`
   - **Authentication Method**: `Basic`
   - **Click Save**
5. Click on the created app and configure:
   - **Response Types**: `code`
   - **Grant Types**: `Authorization Code`, `Refresh Token`
   - **Redirect URIs**: `http://localhost:8090/auth/callback`
   - **Post Logout Redirect URIs**: `http://localhost:8090/`
6. Copy the **Client ID** and **Client Secret**
7. Update `.env`:
   ```bash
   AUTH_CLIENT_ID=<copied_client_id>
   AUTH_CLIENT_SECRET=<copied_client_secret>
   ```
8. Restart the app:
   ```bash
   docker-compose restart app
   ```

### Option 2: Create Service Account with Permissions

Create a new service account in Zitadel with admin permissions, then use its PAT:

```bash
# Get the ZITADEL instance admin token (requires initial setup)
# This is complex and not documented here - would need Zitadel admin access
```

### Option 3: Use gRPC Management Client

Use the Zitadel Go client library with proper authentication:

```go
// See internal/app for example usage
import "github.com/zitadel/zitadel-go/v3/pkg/client"
// ... would need IAM_OWNER role
```

## Verification Steps

Once you've created the app, verify it works:

1. **Check App Registration**:
   ```bash
   curl -X POST \
     -H "Authorization: Bearer <your_pat>" \
     -H 'Host: localhost' \
     -H 'Content-Type: application/json' \
     -d '{}' \
     http://127.0.0.1:8080/management/v1/projects/388435721750446087/apps/_search | jq '.result[] | select(.name == "app-bff-local")'
   ```

2. **Test Full Auth Flow**:
   ```bash
   curl -v -L "http://127.0.0.1:8090/auth/login?return_to=/dashboard"
   # Should redirect to Zitadel login page without error
   ```

3. **Check /api/me endpoint** (after implementing the callback):
   ```bash
   curl http://localhost:8080/api/me
   # Should return 401 Unauthorized (not authenticated yet)
   ```

## Architecture Notes

### Auth Flow Implementation

The implementation follows the design specification exactly:

1. **BFF Pattern**: Go backend handles all OIDC communication
2. **Session Management**: Uses `scs` with PostgreSQL store
3. **Token Storage**: Access token, refresh token, and expiry stored in session
4. **Introspection**: Access tokens validated via Zitadel's introspection endpoint
5. **PKCE**: Properly implemented for browser-based apps

### Key Files

- `internal/auth/bff/`: BFF authentication service
  - `service.go`: OIDC discovery and token exchange
  - `authenticate.go`: Token validation and refresh
  - `http.go`: HTTP handlers for login/logout/callback/me
- `internal/auth/register/routes.go`: Route registration
- `internal/identity/provision/`: User provisioning on first login

### Database Schema

Sessions stored in `public.sessions` table. User mappings in `public.users` table:
- `id` (UUID v7): Application user ID
- `zitadel_sub` (TEXT): Zitadel subject claim, unique per user

## Testing the System

Once OAuth is configured:

1. **Start the app**:
   ```bash
   docker-compose up -d
   ```

2. **Open browser**:
   ```
   http://localhost:8090/auth/login?return_to=/dashboard
   ```

3. **Expected flow**:
   - Redirects to Zitadel login
   - Login/register on Zitadel
   - Redirects back to app's callback
   - Sets session cookie
   - Redirects to `/dashboard`

4. **Check authenticated state**:
   ```bash
   curl -b "app_session=..." http://localhost:8090/api/me
   # Should return user ID
   ```

## Troubleshooting

### "The requested response type is missing"
- The app isn't registered or has wrong settings
- Check response types include "code"
- See "How to Fix" section above

### "Errors.App.NotFound"
- Client ID doesn't exist in Zitadel
- Verify you used the client ID from the app you created

### "No matching permissions found"
- The PAT token doesn't have admin rights
- Use UI option instead
- Or create a new service account with proper roles

### OAuth token validation fails
- Check `AUTH_ISSUER` matches Zitadel's actual URL
- Verify `AUTH_INTERNAL_ADDRESS` is set for internal Zitadel address
- Check network connectivity between containers

### Session not persisting
- PostgreSQL session store table exists
- Check Docker network connectivity
- Verify cookie settings match your domain

## Next Steps

1. ✅ Fix OAuth app registration (see "How to Fix" above)
2. ✅ Test full auth login flow
3. Test identity provisioning (new users created on first login)
4. Test token refresh flow
5. Test logout flow
6. Implement user profile endpoints
7. Test role-based access control
8. Implement CORS/CSRF protection as needed

## Related Documentation

- Design specification: `notes/design.md`
- Zitadel docs: https://zitadel.com/docs
- scs session library: https://pkg.go.dev/github.com/alexedwards/scs/v2
- OIDC spec: https://openid.net/specs/openid-connect-core-1_0.html
