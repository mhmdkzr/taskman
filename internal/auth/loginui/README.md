# Custom login UI (Session API)

This slice implements ZITADEL's [custom login UI pattern](https://zitadel.com/docs/guides/integrate/login-ui):
instead of redirecting the browser to ZITADEL's hosted login page, the
Svelte frontend drives ZITADEL's Session API through this slice's endpoints,
then finalizes against the OIDC authorization request that triggered login.
It also owns registration, email verification, password reset, and TOTP
enrollment — all self-service account operations ZITADEL's v2 REST API
supports directly, driven from our own UI instead of ZITADEL's hosted pages.

It complements [`internal/auth/bff`](../bff/README.md), which remains the
sole OIDC client and owns token exchange, introspection, and the app-owned
session cookie. This slice only gets the browser to a ZITADEL session that's
sufficient to authenticate; `bff.Service.CompleteAuthorizationCallback`
(exported for exactly this reuse) does the rest.

## Login flow

1. The frontend requests `GET /auth/login?return_to=...` as usual (see
   `bff` README). Because the ZITADEL instance is configured with a custom
   login UI base URI (see **Required ZITADEL configuration** below), ZITADEL
   redirects the browser to that URI with `?authRequest=V2_...` instead of
   rendering its hosted login page. That URI points at the Svelte frontend's
   own login route.
2. The frontend calls `POST /auth/session` with `{"loginName": "..."}`. This
   creates a ZITADEL session (`checks.user`), resolves the account's ZITADEL
   user ID and whether it has TOTP enrolled, and stores all of it
   server-side in a short-lived transaction cookie (`login_session`,
   `HttpOnly`, path `/auth/session`, 10 minute lifetime). The response is a
   `State` — `{"loginName": "...", "passwordVerified": false, "needsTotp": bool, "totpVerified": false}`
   — that never reports sufficient right after creation.
3. The frontend calls `PATCH /auth/session/password` with
   `{"password": "..."}`. This adds a `checks.password` check to the same
   ZITADEL session.
4. If `State.needsTotp` is true, the frontend calls `PATCH /auth/session/totp`
   with `{"code": "..."}` (the 6-digit code from the account's authenticator
   app), adding a `checks.totp` check. This requires the password check to
   have already succeeded.
5. Once `State.Sufficient()` holds (password verified, and TOTP verified if
   required), the frontend calls `POST /auth/session/finalize` with
   `{"authRequestId": "..."}` (the value captured from the `authRequest`
   query parameter in step 1). This links the ZITADEL session to the auth
   request via ZITADEL's `CreateCallback`, which returns a `callback_url`
   carrying the same `code`/`state` ZITADEL would otherwise have redirected
   the browser with. The handler feeds those into
   `bff.Service.CompleteAuthorizationCallback` — the exact same code path
   `GET /auth/callback` uses — to exchange the code, provision/load the app
   identity, and establish the app session cookie. The login transaction is
   destroyed at this point. The response is `{"redirectUrl": "..."}`, which
   the frontend should navigate to (`window.location.assign(...)`).

Each step requires the previous one's session cookie; calling
`PATCH .../password`, `PATCH .../totp`, or `POST .../finalize` without an
active/sufficient session returns `409 Conflict`. A rejected login name,
password, or TOTP code returns `401 Unauthorized` without indicating which
factor was wrong.

## Registration, verification, password reset, and TOTP enrollment

These don't need an in-progress ZITADEL session (no login has started yet),
so they're stateless from this slice's perspective — no transaction cookie
involved. They authenticate to ZITADEL with a *different*, broader
credential than the login flow above (see **Required ZITADEL configuration**).

- **`POST /auth/register`** — `{"loginName", "email", "password", "givenName", "familyName"}`.
  Creates the ZITADEL user (`AddHumanUser`), requests email verification in
  ZITADEL's "return code" mode (the code comes back in the API response
  instead of ZITADEL emailing it), and sends the verification email itself
  via this app's own SMTP client (`Mailer`/`internal/notifications` is a
  separate, unrelated webhook-relay path — this one never touches ZITADEL's
  notification pipeline). Returns `{"userId": "..."}`; the emailed link is
  `{frontendBaseURL}/verify-email?userId=...&code=...`.
- **`POST /auth/verify-email`** — `{"userId", "code"}`. Calls ZITADEL's
  `VerifyEmail`. Returns `204` on success.
- **`POST /auth/password-reset`** — `{"loginName"}`. Resolves the login name
  to a user ID (`ListUsers` by `loginNameQuery`), requests a reset code in
  "return code" mode, and emails it the same way as registration. **Always
  returns `204`** regardless of whether the login name exists, so this
  endpoint can't be used to enumerate accounts (no email is sent for an
  unknown login name, but the caller can't tell from the response).
- **`POST /auth/password-reset/confirm`** — `{"userId", "code", "newPassword"}`.
  Calls ZITADEL's `SetPassword` with the emailed verification code.
- **`POST /auth/mfa/totp`** — `{"userId"}`. Starts TOTP enrollment
  (`RegisterTOTP`), returning `{"uri", "secret"}` — the otpauth: URI (for a
  QR code) and raw secret (for manual entry). Enrollment isn't active until
  confirmed.
- **`POST /auth/mfa/totp/verify`** — `{"userId", "code"}`. Completes
  enrollment (`VerifyTOTPRegistration`) with a code from the authenticator
  app. Only after this does `POST /auth/session`'s `hasTOTP` check (and
  therefore `State.needsTotp` on subsequent logins) see it. The frontend
  offers this as an optional step right after email verification
  (`VerifyEmailPage.svelte`), not as a separate account-settings page — there
  isn't one yet.

## Required ZITADEL configuration

### Credentials (already wired in local Compose)

This slice uses two different ZITADEL credentials, matching the least
privilege each REST call actually needs:

| Credential | Config | Used for |
|---|---|---|
| `IAM_LOGIN_CLIENT`-scoped PAT | `AUTH_LOGIN_CLIENT_PAT_PATH` | `CreateSession`/`SetSession`/`CreateCallback` (the login flow) |
| Broader admin PAT | `AUTH_ADMIN_PAT_PATH` | `AddHumanUser`/`VerifyEmail`/`PasswordReset`/`SetPassword`/`ListUsers`/TOTP enrollment |

Local Compose already provisions the first for ZITADEL's own official
`zitadel-login` container, via `start-from-init`'s
`FirstInstance.Org.LoginClient` bootstrap (`.env.example`):

```
ZITADEL_FIRSTINSTANCE_LOGINCLIENTPATPATH=/zitadel/bootstrap/login-client.pat
ZITADEL_FIRSTINSTANCE_ORG_LOGINCLIENT_MACHINE_USERNAME=login-client
```

ZITADEL writes that machine user's PAT to `/zitadel/bootstrap/login-client.pat`
in the shared `zitadel-bootstrap` Docker volume. `compose.yaml` mounts that
volume read-only into the `app` service too (alongside `zitadel-login`).

For the second, this slice reuses the same `admin-provisioner` PAT
`bff`/`scripts/provision-zitadel-bff.sh` already use for management-API
calls (also in the same mounted volume) — no separate credential is
provisioned. This is a pragmatic reuse, not a least-privilege ideal: outside
local Compose, provision a narrower service account (e.g. scoped to
`ORG_USER_MANAGER` rather than full instance admin) and point
`AUTH_ADMIN_PAT_PATH` at it instead; confirm the provisioning approach with
whoever owns that instance.

No separate credential needs provisioning for local Compose; `app` just
needs both PAT-path env vars set and the volume mounted, both already done
in `compose.yaml`/`.env.example`.

### Custom login UI base URI

ZITADEL's instance-level `loginV2` feature flag controls where authorization
requests redirect for login — by default, ZITADEL's own hosted
`zitadel-login` container. `scripts/provision-zitadel-bff.sh` points it at
this frontend instead (`PUT /v2/features/instance` with
`{"loginV2": {"required": true, "baseUri": "http://localhost:8090/"}}`),
idempotently, so rerunning that script is how to re-apply this after ZITADEL
data is reset. `.env.example`'s `ZITADEL_DEFAULTINSTANCE_FEATURES_LOGINV2_BASEURI`
sets the same thing at first-instance bootstrap time, for a fresh instance
that hasn't run the script yet.

## Design notes

- **Sufficiency policy** lives in `State.Sufficient()` (`session.go`):
  ZITADEL deliberately leaves it up to the calling client to decide when a
  session has enough verified factors. Adding passkey/external-IdP support
  later means extending `State` and this method, not the endpoints.
- **The ZITADEL session token never reaches the browser.** It lives only in
  the server-side `login_session` transaction (backed by the same
  PostgreSQL-backed SCS store `bff` uses for its own OAuth transaction). The
  browser only ever sees the client-safe `State` view.
- **REST, not gRPC**: this slice calls ZITADEL's v2 REST API directly over
  HTTP using `bff.Service.HTTPClient()`/`Issuer()` (so it inherits the same
  private Compose address override), rather than the vendored gRPC service
  clients.
- **This app owns email delivery**, not ZITADEL. Registration/password-reset
  use ZITADEL's "return code" verification mode specifically so the code
  never leaves ZITADEL via its own notification pipeline; `Mailer` (an
  interface over the app's SMTP client) sends the actual email. This is
  unrelated to `internal/notifications/zitadel`, which relays ZITADEL's
  *own* generated notifications (e.g. for flows this slice doesn't cover)
  through a webhook.
- **Testability seams**: `AuthorizationCompleter`, `TokenSource`, and
  `Mailer` are small interfaces at the points this slice reaches out to
  unrelated dependencies (`bff.Service`, ZITADEL credentials, SMTP), so unit
  tests can substitute stubs and an `httptest.Server` for ZITADEL's REST API
  without a live ZITADEL instance, database, or mail server. `StaticToken` is
  the production `TokenSource`: a pre-issued PAT read once at startup rather
  than a refreshed OAuth2 token, since PATs don't need refreshing.

## Example

```sh
# 1. Create a session for a login name.
curl -i -c cookies.txt -X POST http://localhost:8090/auth/session \
  -H 'Content-Type: application/json' \
  -d '{"loginName":"minnie-mouse@fabi.zitadel.app"}'

# 2. Verify the password on the same session.
curl -i -b cookies.txt -c cookies.txt -X PATCH http://localhost:8090/auth/session/password \
  -H 'Content-Type: application/json' \
  -d '{"password":"Secr3tP4ssw0rd!"}'

# 2b. If State.needsTotp was true, also verify a TOTP code.
curl -i -b cookies.txt -c cookies.txt -X PATCH http://localhost:8090/auth/session/totp \
  -H 'Content-Type: application/json' \
  -d '{"code":"123456"}'

# 3. Finalize against the auth request captured from the ?authRequest= redirect.
curl -i -b cookies.txt -X POST http://localhost:8090/auth/session/finalize \
  -H 'Content-Type: application/json' \
  -d '{"authRequestId":"V2_224908753244265546"}'
```

```sh
# Register, then verify using the code from the emailed link.
curl -s -X POST http://localhost:8090/auth/register -H 'Content-Type: application/json' \
  -d '{"loginName":"minnie","email":"minnie@example.com","password":"Secr3tP4ssw0rd!","givenName":"Minnie","familyName":"Mouse"}'
curl -s -X POST http://localhost:8090/auth/verify-email -H 'Content-Type: application/json' \
  -d '{"userId":"<id>","code":"<code>"}'

# Forgot password, then set a new one using the code from the emailed link.
curl -s -X POST http://localhost:8090/auth/password-reset -H 'Content-Type: application/json' \
  -d '{"loginName":"minnie"}'
curl -s -X POST http://localhost:8090/auth/password-reset/confirm -H 'Content-Type: application/json' \
  -d '{"userId":"<id>","code":"<code>","newPassword":"NewSecr3tP4ssw0rd!"}'

# Enroll TOTP for an account, then confirm with a code from the authenticator app.
curl -s -X POST http://localhost:8090/auth/mfa/totp -H 'Content-Type: application/json' -d '{"userId":"<id>"}'
curl -s -X POST http://localhost:8090/auth/mfa/totp/verify -H 'Content-Type: application/json' \
  -d '{"userId":"<id>","code":"123456"}'
```
