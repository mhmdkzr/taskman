# Quick Start - Complete OAuth Setup (5 Minutes)

## The Situation
Everything is running. The auth system works **except** we need to register the OAuth app in Zitadel.

## What's the Issue?
When users try to log in, Zitadel says "app not found" because the OAuth app isn't registered yet.

## How to Fix (Choose One)

### 🟢 Option A: Web UI (Easiest - 3 clicks)

1. Go to `http://localhost:8080/ui/console`
2. **Projects** → **ZITADEL** → **Create New App**
3. Fill these fields:
   - Name: `app-bff-local`
   - App Type: `Web`
   - Auth Method: `Basic`
   - Response Type: ✓ `code`
   - Grant Types: ✓ `Authorization Code` + ✓ `Refresh Token`
   - Redirect URI: `http://localhost:8090/auth/callback`
   - Logout URI: `http://localhost:8090/`
4. Click Create
5. Copy Client ID and Secret
6. Update `.env`:
   ```
   AUTH_CLIENT_ID=<paste_id>
   AUTH_CLIENT_SECRET=<paste_secret>
   ```
7. Restart:
   ```bash
   docker-compose restart app
   ```

### 🔵 Option B: Use the Provision Script
Run (if permissions issue is fixed):
```bash
bash scripts/provision-zitadel-bff.sh
docker-compose restart app
```

## Verify It Works

```bash
# Should show redirect to login page (not error):
curl -v http://127.0.0.1:8090/auth/login
```

Then test in browser: `http://localhost:8090/auth/login`

## System Status
- ✅ All 9 services running
- ✅ App responding to requests  
- ✅ Auth service initialized
- ⚠️ OAuth app not registered (this is what we're fixing)

## Documentation
- Full guide: `SYSTEM_STATUS.md`
- Troubleshooting: `DEBUG_AUTH_SETUP.md`
- Implementation: `notes/design.md`

**That's it!** After these 5 minutes, the whole system will be fully functional.
