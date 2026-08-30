// Client for the custom Session-API login UI backend
// (internal/auth/loginui). Each call drives one step of the ZITADEL session:
// create it with a login name, verify the password, then finalize it
// against the OIDC authorization request captured from the `authRequest`
// query parameter ZITADEL redirected the browser with.

/** Thrown when the backend responds with a non-2xx status. */
export class LoginSessionError extends Error {
  /**
   * @param {string} message
   * @param {number} status
   */
  constructor(message, status) {
    super(message)
    this.name = "LoginSessionError"
    this.status = status
  }
}

/**
 * @param {string} path
 * @param {unknown} body
 */
async function post(path, method, body) {
  const response = await fetch(path, {
    method,
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    throw new LoginSessionError(payload.error || `request to ${path} failed`, response.status)
  }
  return payload
}

/**
 * Starts a login for loginName. Returns the current login State.
 * @param {string} loginName
 */
export function createSession(loginName) {
  return post("/auth/session", "POST", { loginName })
}

/**
 * Verifies the password for the in-progress login. Returns the updated
 * login State.
 * @param {string} password
 */
export function checkPassword(password) {
  return post("/auth/session/password", "PATCH", { password })
}

/**
 * Verifies a TOTP code for the in-progress login, when State.needsTotp is
 * true. Returns the updated login State.
 * @param {string} code
 */
export function checkTOTP(code) {
  return post("/auth/session/totp", "PATCH", { code })
}

/**
 * Finalizes a sufficient login against authRequestId. Returns
 * `{ redirectUrl }`, where the caller should navigate on success.
 * @param {string} authRequestId
 */
export function finalize(authRequestId) {
  return post("/auth/session/finalize", "POST", { authRequestId })
}
