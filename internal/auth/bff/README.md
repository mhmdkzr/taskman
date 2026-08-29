# BFF authentication

The Go backend is the sole Zitadel OIDC client. `GET /auth/login` starts an
authorization-code flow with PKCE; `GET /auth/callback` exchanges the code on
the server; `POST /auth/logout` destroys the local session; and `GET /api/me`
returns the app-owned user ID.

Tokens are stored only in the PostgreSQL-backed SCS session. The browser gets
one opaque `app_session` cookie with `HttpOnly`, `Secure`, and `SameSite=Strict`.
A separate, ten-minute `auth_transaction` cookie is used only to carry OAuth
state/PKCE across the cross-site top-level callback, then is destroyed. It is
`HttpOnly`, `Secure`, and `SameSite=Lax` as required for that callback.

Configure a Zitadel confidential Web application with exact redirect and
post-logout URLs. The backend uses OIDC discovery and ZITADEL's SDK PKCE helper;
the Svelte SPA never receives an access or refresh token.

For local Compose, run `scripts/provision-zitadel-bff.sh` to create that
application via the Management API and populate `AUTH_CLIENT_ID`/
`AUTH_CLIENT_SECRET` in `.env`. It authenticates with the `admin-provisioner`
machine user's PAT (bootstrapped with the `IAM_OWNER` role via
`ZITADEL_FIRSTINSTANCE_ORG_MACHINE_*` in `.env.example`) rather than the
`login-client` account, which is scoped to login operations only and cannot
create applications.

Zitadel's `FirstInstance.Org.Machine` bootstrap has no scoped-role option, so
`admin-provisioner` is a standing full instance-admin credential (PAT and
machine key both set to expire 2099-01-01, i.e. effectively never) written to
the `zitadel-bootstrap` Docker volume. This is an accepted tradeoff for local
Compose only — nothing here revokes or rotates it after
`provision-zitadel-bff.sh` runs. Do not carry this bootstrap config into a
shared or long-lived environment without adding rotation/revocation.

```sh
curl -i http://localhost:8090/auth/login
curl -i --cookie app-session.txt http://localhost:8090/api/me
curl -i -X POST --cookie app-session.txt -H 'X-Requested-With: XMLHttpRequest' http://localhost:8090/auth/logout
```
