import { test, expect } from '@playwright/test'
import { createTestUser, deleteTestUser } from './support/zitadel-admin.js'

// End-to-end coverage for the custom Session-API login UI
// (internal/auth/loginui + LoginForm.svelte) against the real Compose stack:
// live ZITADEL, the Go BFF, and the Svelte frontend. See e2e/README.md for
// what this does and does not cover.

const PASSWORD = 'SuperSecr3t!Pass'

test.describe('custom login UI', () => {
  /** @type {string} */
  let userId
  /** @type {string} */
  let userName

  test.beforeAll(async () => {
    userName = `e2e-login-${Date.now()}`
    userId = await createTestUser({ userName, password: PASSWORD })
  })

  test.afterAll(async () => {
    await deleteTestUser(userId)
  })

  test('GET /auth/login renders only our custom login UI, never ZITADEL hosted UI', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')

    await expect(page).toHaveURL(/^http:\/\/localhost:8090\/login\?authRequest=/)
    await expect(page.getByText('Sign in', { exact: true })).toBeVisible()
    await expect(page.getByPlaceholder('Login name')).toBeVisible()
  })

  test('logs in with a valid login name and password', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')

    await page.getByPlaceholder('Login name').fill(userName)
    await page.getByRole('button', { name: 'Continue' }).click()
    await expect(page.getByText(`Signed in as ${userName}`)).toBeVisible()

    await page.getByPlaceholder('Password').fill(PASSWORD)
    await page.getByRole('button', { name: 'Sign in' }).click()

    await expect(page).toHaveURL('http://localhost:8090/')
    await expect(page.getByText(/^Signed in as /)).toBeVisible()
  })

  test('shows an error for an unknown login name', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')

    await page.getByPlaceholder('Login name').fill('nobody-e2e-test')
    await page.getByRole('button', { name: 'Continue' }).click()

    await expect(page.getByRole('alert')).toContainText(/couldn't find that account/i)
    await expect(page.getByPlaceholder('Login name')).toBeVisible()
  })

  test('shows an error for an incorrect password', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')

    await page.getByPlaceholder('Login name').fill(userName)
    await page.getByRole('button', { name: 'Continue' }).click()
    await page.getByPlaceholder('Password').fill('definitely-wrong-password')
    await page.getByRole('button', { name: 'Sign in' }).click()

    await expect(page.getByRole('alert')).toContainText(/incorrect password/i)
  })

  test('rejects an empty login name without calling the backend', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')

    await page.getByRole('button', { name: 'Continue' }).click()

    await expect(page.getByRole('alert')).toContainText(/enter your login name/i)
  })

  test('"use a different account" returns to the login-name step', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')

    await page.getByPlaceholder('Login name').fill(userName)
    await page.getByRole('button', { name: 'Continue' }).click()
    await expect(page.getByPlaceholder('Password')).toBeVisible()

    await page.getByRole('button', { name: 'Use a different account' }).click()

    await expect(page.getByPlaceholder('Login name')).toBeVisible()
    await expect(page.getByPlaceholder('Password')).toHaveCount(0)
  })

  test('logs out and clears the app session', async ({ page }) => {
    await page.goto('/auth/login?return_to=/')
    await page.getByPlaceholder('Login name').fill(userName)
    await page.getByRole('button', { name: 'Continue' }).click()
    await page.getByPlaceholder('Password').fill(PASSWORD)
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page.getByText(/^Signed in as /)).toBeVisible()

    // Wait for the logout request itself (which destroys the server-side
    // session before it responds) rather than racing the click against the
    // navigation it triggers.
    await Promise.all([
      page.waitForResponse((response) => response.url().endsWith('/auth/logout')),
      page.getByRole('button', { name: 'Sign out' }).click(),
    ])

    // Logout hands off to ZITADEL's own end-session page (pre-existing bff
    // behavior, unrelated to this custom login UI) — the thing this test
    // owns is confirming the local app session was actually cleared.
    await page.goto('/')
    await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible()
  })
})
