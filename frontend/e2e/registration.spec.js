import { test, expect } from "@playwright/test"
import { deleteTestUser } from "./support/zitadel-admin.js"
import { waitForEmailBody, extractUserIDAndCode, clearAllEmails } from "./support/mailhog.js"
import { generateTOTP } from "./support/totp.js"

// End-to-end coverage for account registration, email verification, and
// optional TOTP enrollment (internal/auth/loginui's registration.go +
// RegisterForm/VerifyEmailPage.svelte), including the real email this app
// sends via MailHog rather than a mocked mailer. See e2e/README.md.

const PASSWORD = "SuperSecr3t!Pass"

test.describe("registration and email verification", () => {
  /** @type {string} */
  let userName
  /** @type {string | undefined} */
  let userId

  test.beforeEach(async () => {
    userName = `e2e-register-${Date.now()}`
    userId = undefined
    await clearAllEmails()
  })

  test.afterEach(async () => {
    if (userId) await deleteTestUser(userId)
  })

  test("registers, verifies via the emailed link, and can then sign in", async ({ page }) => {
    const email = `${userName}@example.com`

    await page.goto("/register")
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByPlaceholder("Email").fill(email)
    await page.getByPlaceholder("First name").fill("E2E")
    await page.getByPlaceholder("Last name").fill("Register")
    await page.getByPlaceholder("Password").fill(PASSWORD)
    await page.getByRole("button", { name: "Create account" }).click()
    await expect(page.getByText(/check your email/i)).toBeVisible()

    const body = await waitForEmailBody({ to: email, subjectContains: "Verify your email" })
    const parsed = extractUserIDAndCode(body)
    userId = parsed.userId

    await page.goto(`/verify-email?userId=${parsed.userId}&code=${parsed.code}`)
    await expect(page.getByText(/your email is verified/i)).toBeVisible()

    await page.getByRole("link", { name: "Skip for now" }).click();
    await expect(page).toHaveURL(/^http:\/\/localhost:8090\/login\?authRequest=/)
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByRole("button", { name: "Continue" }).click()
    await page.getByPlaceholder("Password").fill(PASSWORD)
    await page.getByRole("button", { name: "Sign in" }).click()

    await expect(page).toHaveURL("http://localhost:8090/")
    await expect(page.getByText(/^Signed in as /)).toBeVisible()
  })

  test("shows an error for an invalid verification code", async ({ page }) => {
    await page.goto("/verify-email?userId=does-not-exist&code=WRONGCODE")

    await expect(page.getByRole("alert")).toContainText(/invalid or has expired/i)
  })

  test("enrolling TOTP after verification requires it on the next login", async ({ page }) => {
    const email = `${userName}@example.com`

    await page.goto("/register")
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByPlaceholder("Email").fill(email)
    await page.getByPlaceholder("First name").fill("E2E")
    await page.getByPlaceholder("Last name").fill("Totp")
    await page.getByPlaceholder("Password").fill(PASSWORD)
    await page.getByRole("button", { name: "Create account" }).click()

    const body = await waitForEmailBody({ to: email, subjectContains: "Verify your email" })
    const parsed = extractUserIDAndCode(body)
    userId = parsed.userId
    await page.goto(`/verify-email?userId=${parsed.userId}&code=${parsed.code}`)

    await page.getByRole("button", { name: "Enable two-factor authentication" }).click()
    const secret = await page.getByText(/^[A-Z2-7]+$/).textContent()
    await page.getByPlaceholder("6-digit code").fill(generateTOTP(secret))
    await page.getByRole("button", { name: "Confirm" }).click()
    await expect(page.getByText(/two-factor authentication is enabled/i)).toBeVisible()

    await page.getByRole("link", { name: "Continue to sign in" }).click()
    await page.getByPlaceholder("Login name").fill(userName)
    await page.getByRole("button", { name: "Continue" }).click()
    await page.getByPlaceholder("Password").fill(PASSWORD)
    await page.getByRole("button", { name: "Sign in" }).click()
    await expect(page.getByPlaceholder("6-digit code")).toBeVisible()

    await page.getByPlaceholder("6-digit code").fill(generateTOTP(secret))
    await page.getByRole("button", { name: "Verify" }).click()

    await expect(page).toHaveURL("http://localhost:8090/")
    await expect(page.getByText(/^Signed in as /)).toBeVisible()
  })
})
