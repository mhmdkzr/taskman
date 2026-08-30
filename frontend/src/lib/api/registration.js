// Client for registration, email verification, password reset, and TOTP
// enrollment (internal/auth/loginui). Unlike loginSession.js, none of these
// calls need an in-progress ZITADEL session — registration and password
// reset happen before one exists.
import { LoginSessionError } from "./loginSession.js"

/**
 * @param {string} path
 * @param {unknown} body
 */
async function post(path, body) {
  const response = await fetch(path, {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  const payload = response.status === 204 ? {} : await response.json().catch(() => ({}))
  if (!response.ok) {
    throw new LoginSessionError(payload.error || `request to ${path} failed`, response.status)
  }
  return payload
}

/**
 * Creates a new account and emails a verification link. Returns
 * `{ userId }`.
 * @param {{loginName: string, email: string, password: string, givenName: string, familyName: string}} fields
 */
export function register(fields) {
  return post("/auth/register", fields)
}

/**
 * Confirms a registration's emailed verification code.
 * @param {string} userId
 * @param {string} code
 */
export function verifyEmail(userId, code) {
  return post("/auth/verify-email", { userId, code })
}

/**
 * Emails a password-reset link for loginName, if an account exists. Always
 * succeeds regardless of whether the account exists.
 * @param {string} loginName
 */
export function requestPasswordReset(loginName) {
  return post("/auth/password-reset", { loginName })
}

/**
 * Sets a new password using an emailed reset code.
 * @param {string} userId
 * @param {string} code
 * @param {string} newPassword
 */
export function resetPassword(userId, code, newPassword) {
  return post("/auth/password-reset/confirm", { userId, code, newPassword })
}

/**
 * Begins TOTP enrollment for userId. Returns `{ uri, secret }`.
 * @param {string} userId
 */
export function startTOTPEnrollment(userId) {
  return post("/auth/mfa/totp", { userId })
}

/**
 * Completes TOTP enrollment.
 * @param {string} userId
 * @param {string} code
 */
export function confirmTOTPEnrollment(userId, code) {
  return post("/auth/mfa/totp/verify", { userId, code })
}
