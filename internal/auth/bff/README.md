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

```sh
curl -i http://localhost:8090/auth/login
curl -i --cookie app-session.txt http://localhost:8090/api/me
curl -i -X POST --cookie app-session.txt -H 'X-Requested-With: XMLHttpRequest' http://localhost:8090/auth/logout
```
