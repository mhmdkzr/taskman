<script>
  import { Card, CardContent, CardHeader, CardTitle } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import LoginForm from "$lib/components/LoginForm.svelte"
  import RegisterForm from "$lib/components/RegisterForm.svelte"
  import VerifyEmailPage from "$lib/components/VerifyEmailPage.svelte"
  import ForgotPasswordForm from "$lib/components/ForgotPasswordForm.svelte"
  import ResetPasswordPage from "$lib/components/ResetPasswordPage.svelte"

  const params = new URLSearchParams(window.location.search)
  const authRequestId = params.get("authRequest")
  const path = window.location.pathname

  let user = $state(null)
  let loading = $state(true)
  let authAvailable = $state(true)
  let actionError = $state("")

  async function loadUser() {
    try {
      const response = await fetch("/api/me", { credentials: "same-origin" })
      if (response.ok) user = await response.json()
      else if (response.status === 404) authAvailable = false
    } catch {
      actionError = "Unable to check your session. Please try again."
    } finally {
      loading = false
    }
  }

  async function logout() {
    actionError = ""
    try {
      const response = await fetch("/auth/logout", {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Requested-With": "XMLHttpRequest" },
      })
      if (!response.ok && !response.redirected) {
        actionError = "Unable to sign out. Please try again."
        return
      }
      if (response.status === 204) {
        window.location.assign("/")
        return
      }
      const { redirect_url } = await response.json()
      window.location.assign(redirect_url || "/")
    } catch {
      actionError = "Unable to sign out. Please try again."
    }
  }

  const isRegister = path === "/register"
  const isVerifyEmail = path === "/verify-email"
  const isForgotPassword = path === "/forgot-password"
  const isResetPassword = path === "/reset-password"
  const isStandaloneRoute = authRequestId || isRegister || isVerifyEmail || isForgotPassword || isResetPassword

  if (!isStandaloneRoute) loadUser()
</script>

{#if authRequestId}
  <LoginForm {authRequestId} />
{:else if isRegister}
  <RegisterForm />
{:else if isVerifyEmail}
  <VerifyEmailPage userId={params.get("userId") ?? ""} code={params.get("code") ?? ""} />
{:else if isForgotPassword}
  <ForgotPasswordForm />
{:else if isResetPassword}
  <ResetPasswordPage userId={params.get("userId") ?? ""} code={params.get("code") ?? ""} />
{:else}
  <div class="flex min-h-dvh items-center justify-center p-4">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <CardTitle>App</CardTitle>
      </CardHeader>
      <CardContent>
        {#if loading}
          Loading…
        {:else if user}
          <p class="mb-4">Signed in as {user.id}</p>
          <Button onclick={logout}>Sign out</Button>
        {:else if !authAvailable}
          <p>Authentication is not configured.</p>
        {:else}
          <Button href="/auth/login?return_to=/">Sign in</Button>
        {/if}
        {#if actionError}
          <p class="mt-4 text-destructive" role="alert">{actionError}</p>
        {/if}
      </CardContent>
    </Card>
  </div>
{/if}
