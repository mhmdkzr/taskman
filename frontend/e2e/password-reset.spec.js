import { test, expect } from "@playwright/test"
import { createTestUser, deleteTestUser } from "./support/zitadel-admin.js"
import { waitForEmailBody, extractUserIDAndCode, clearAllEmails } from "./support/mailhog.js"

// End-to-end coverage for the "forgot password" flow
// (internal/auth/loginui's RequestPasswordReset/ResetPassword +
// ForgotPasswordForm/ResetPasswordPage.svelte), including the real email
// this app sends via MailHog. See e2e/README.md.

const ORIGINAL_PASSWORD = "SuperSecr3t!Pass"
const NEW_PASSWORD = "EvenNewerSecr3t!Pass"

test.describe("password reset", () => {
  /** @type {string} */
  let userName
  /** @type {string} */
  let userId

  test.beforeEach(async () => {
    userName = `e2e-reset-${Date.now()}`
    userId = await createTestUser({ userName, password: ORIGINAL_PASSWORD })
    await clearAllEmails()
  })

  test.afterEach(async () => {
    await deleteTestUser(userId)
  })

  test("resets via the emailed link and can sign in with the new password", async ({ page }) => {
    await page.goto("/forgot-password")
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByRole("button", { name: "Send reset link" }).click()
    await expect(page.getByText(/reset link is on its way/i)).toBeVisible()

    const body = await waitForEmailBody({ to: `${userName}@example.com`, subjectContains: "Reset your password" })
    const { userId: linkUserId, code } = extractUserIDAndCode(body)
    expect(linkUserId).toBe(userId)

    await page.goto(`/reset-password?userId=${linkUserId}&code=${code}`)
    await page.getByPlaceholder("New password").fill(NEW_PASSWORD)
    await page.getByRole("button", { name: "Save new password" }).click()
    await expect(page.getByText(/password has been changed/i)).toBeVisible()

    await page.getByRole("link", { name: "Continue to sign in" }).click()
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByRole("button", { name: "Continue" }).click()
    await page.getByPlaceholder("Password").fill(NEW_PASSWORD)
    await page.getByRole("button", { name: "Sign in" }).click()

    await expect(page).toHaveURL("http://localhost:8090/")
    await expect(page.getByText(/^Signed in as /)).toBeVisible()
  })

  test("does not email an unknown login name, and still reports success", async ({ page }) => {
    await page.goto("/forgot-password")
    await page.getByPlaceholder("Login name").fill(`nobody-${Date.now()}`)
    await page.getByRole("button", { name: "Send reset link" }).click()

    await expect(page.getByText(/reset link is on its way/i)).toBeVisible()
    await expect(waitForEmailBody({ to: "nobody", subjectContains: "Reset your password" }, { timeoutMs: 800 })).rejects.toThrow()
  })

  test("an old password no longer works after reset", async ({ page }) => {
    await page.goto("/forgot-password")
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByRole("button", { name: "Send reset link" }).click()

    const body = await waitForEmailBody({ to: `${userName}@example.com`, subjectContains: "Reset your password" })
    const { userId: linkUserId, code } = extractUserIDAndCode(body)
    await page.goto(`/reset-password?userId=${linkUserId}&code=${code}`)
    await page.getByPlaceholder("New password").fill(NEW_PASSWORD)
    await page.getByRole("button", { name: "Save new password" }).click()
    await expect(page.getByText(/password has been changed/i)).toBeVisible()

    await page.goto("/auth/login?return_to=/")
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByRole("button", { name: "Continue" }).click()
    await page.getByPlaceholder("Password").fill(ORIGINAL_PASSWORD)
    await page.getByRole("button", { name: "Sign in" }).click()

    await expect(page.getByRole("alert")).toContainText(/incorrect password/i)
  })
})
