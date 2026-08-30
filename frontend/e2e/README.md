# End-to-end tests (Playwright)

These tests drive the real Compose stack in a real browser: the Go backend,
live ZITADEL, and the built/served Svelte frontend — not a mock. Registration
and password-reset specs also read the real transactional emails this app
sends via MailHog (`e2e/support/mailhog.js`), rather than mocking mail
delivery. Run them with:

```sh
docker compose up -d --build app   # ensure the backend has the latest code
deno task test:e2e                 # from frontend/, or `make fe-test-e2e`
```

## Preconditions

- The full Compose stack must already be running (`docker compose up -d`
  from the repo root) and reachable at `http://localhost:8090`, including
  `mailhog` (`http://localhost:8025`).
- `AUTH_ENABLED=true` in `.env`, with `scripts/provision-zitadel-bff.sh`
  already run at least once (creates the OIDC app, points ZITADEL's Login V2
  base URI at this frontend instead of `zitadel-login` — see that script and
  `internal/auth/loginui/README.md`).
- `WEBHOOKS_ZITADEL_PATH_SECRET` need not be set for these specs —
  registration/password-reset emails are sent directly by this app's own
  mailer (see `internal/auth/loginui/registration.go`), not via ZITADEL's
  notification pipeline.
- Docker CLI available on the machine running the tests (`docker compose
  exec app ...` is used to read the `admin-provisioner` PAT — see
  `e2e/support/zitadel-admin.js` — the same credential
  `scripts/provision-zitadel-bff.sh` uses).

Each spec provisions and deletes its own throwaway ZITADEL user (either via
the management API in `zitadel-admin.js`, or through the real `/register`
form), so these are safe to run repeatedly against a shared local dev
instance without touching real accounts. Tests run serially (`workers: 1` in
`playwright.config.js`) since they share that one instance and one MailHog
inbox.

## What's covered

`login.spec.js` — the custom Session-API login UI
(`internal/auth/loginui` + `LoginForm.svelte`):

- `GET /auth/login` renders our own UI and never ZITADEL's hosted UI
- successful login with a valid login name + password
- unknown login name → error, stays on the login-name step
- incorrect password → error
- empty login name is rejected client-side without a backend call
- "use a different account" returns to the login-name step
- logout clears the app session

`registration.spec.js` — sign-up and email verification
(`RegisterForm`/`VerifyEmailPage.svelte`):

- registers, verifies via the link emailed through MailHog, then signs in
- an invalid/unknown verification code shows an error
- enrolling TOTP right after verification (secret shown, confirmed with a
  real computed code via `support/totp.js`) makes the *next* login require
  that TOTP code (`LoginForm`'s TOTP step)

`password-reset.spec.js` — "forgot password"
(`ForgotPasswordForm`/`ResetPasswordPage.svelte`):

- requests a reset, follows the emailed link, sets a new password, signs in
  with it
- an unknown login name still reports success and sends no email (avoids
  account-existence enumeration)
- the old password no longer works after a reset

## What's NOT covered (not implemented yet)

- **Other login methods** — passkeys, MFA via SMS/email/U2F, and external
  IdP (Google/GitHub/etc.) login are documented as follow-on work in
  `internal/auth/loginui/README.md` but not implemented (TOTP is; see
  `registration.spec.js`). The user explicitly confirmed external IdP is out
  of scope for this app.

Writing Playwright specs for unimplemented features would just be specs
that can never pass. Once any of the above ship, add a corresponding
`*.spec.js` here.
