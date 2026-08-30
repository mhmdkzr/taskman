// MailHog API helper: specs read the transactional emails
// internal/auth/loginui sends directly (see e2e/README.md) instead of a
// mocked mailer, to exercise the real send path end to end.
const MAILHOG_URL = "http://localhost:8025"

/**
 * Polls MailHog until a message to `to` with a subject containing
 * `subjectContains` appears, then returns its plain-text body. Throws after
 * timeoutMs if none arrives.
 * @param {{to: string, subjectContains: string}} match
 * @param {{timeoutMs?: number, pollMs?: number}} [options]
 */
export async function waitForEmailBody(match, options = {}) {
  const timeoutMs = options.timeoutMs ?? 5000
  const pollMs = options.pollMs ?? 200
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const response = await fetch(`${MAILHOG_URL}/api/v2/messages?limit=50`)
    const data = await response.json()
    for (const item of data.items ?? []) {
      const to = (item.To ?? []).map((addr) => `${addr.Mailbox}@${addr.Domain}`).join(",")
      const subject = item.Content?.Headers?.Subject?.[0] ?? ""
      if (to.includes(match.to) && subject.includes(match.subjectContains)) {
        return item.Content.Body
      }
    }
    await new Promise((resolve) => setTimeout(resolve, pollMs))
  }
  throw new Error(`no email to ${match.to} with subject containing ${JSON.stringify(match.subjectContains)} within ${timeoutMs}ms`)
}

/** Extracts `{ userId, code }` from a verify-email or reset-password link in an email body. */
export function extractUserIDAndCode(body) {
  const match = body.match(/userId=([^&\s]+)&code=([^&\s]+)/)
  if (!match) throw new Error(`no userId/code link found in email body: ${body}`)
  return { userId: match[1], code: match[2] }
}

/** Deletes all messages, so specs start from a clean inbox. */
export async function clearAllEmails() {
  await fetch(`${MAILHOG_URL}/api/v1/messages`, { method: "DELETE" })
}
