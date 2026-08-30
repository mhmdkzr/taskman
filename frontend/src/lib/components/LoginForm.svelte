<script>
  import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import { createSession, checkPassword, checkTOTP, finalize, LoginSessionError } from "$lib/api/loginSession.js"

  /** @type {{ authRequestId: string }} */
  let { authRequestId } = $props()

  /** @type {"loginName" | "password" | "totp"} */
  let step = $state("loginName")
  let loginName = $state("")
  let password = $state("")
  let totpCode = $state("")
  let submitting = $state(false)
  let error = $state("")

  async function finalizeLogin() {
    const { redirectUrl } = await finalize(authRequestId)
    window.location.assign(redirectUrl || "/")
  }

  async function submitLoginName() {
    error = ""
    if (!loginName.trim()) {
      error = "Enter your login name."
      return
    }
    submitting = true
    try {
      await createSession(loginName.trim())
      step = "password"
    } catch (err) {
      error = err instanceof LoginSessionError ? "We couldn't find that account." : "Something went wrong. Please try again."
    } finally {
      submitting = false
    }
  }

  async function submitPassword() {
    error = ""
    if (!password) {
      error = "Enter your password."
      return
    }
    submitting = true
    try {
      const state = await checkPassword(password)
      if (!state.passwordVerified) {
        error = "Incorrect password."
        return
      }
      if (state.needsTotp) {
        step = "totp"
        return
      }
      await finalizeLogin()
    } catch (err) {
      error = err instanceof LoginSessionError ? "Incorrect password." : "Something went wrong. Please try again."
    } finally {
      submitting = false
    }
  }

  async function submitTOTP() {
    error = ""
    if (!totpCode.trim()) {
      error = "Enter the 6-digit code from your authenticator app."
      return
    }
    submitting = true
    try {
      const state = await checkTOTP(totpCode.trim())
      if (!state.totpVerified) {
        error = "Incorrect code."
        return
      }
      await finalizeLogin()
    } catch (err) {
      error = err instanceof LoginSessionError ? "Incorrect code." : "Something went wrong. Please try again."
    } finally {
      submitting = false
    }
  }

  function backToLoginName() {
    error = ""
    password = ""
    totpCode = ""
    step = "loginName"
  }
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Sign in</CardTitle>
      <CardDescription>
        {#if step === "loginName"}
          Enter your login name to continue.
        {:else if step === "password"}
          Signed in as {loginName}
        {:else}
          Enter the code from your authenticator app.
        {/if}
      </CardDescription>
    </CardHeader>
    <CardContent>
      {#if step === "loginName"}
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); submitLoginName() }}>
          <input
            type="text"
            autocomplete="username"
            placeholder="Login name"
            bind:value={loginName}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Checking…" : "Continue"}</Button>
        </form>
        <p class="mt-4 flex justify-between text-sm">
          <a class="underline" href="/register">Create account</a>
          <a class="underline" href="/forgot-password">Forgot password?</a>
        </p>
      {:else if step === "password"}
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); submitPassword() }}>
          <input
            type="password"
            autocomplete="current-password"
            placeholder="Password"
            bind:value={password}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Signing in…" : "Sign in"}</Button>
          <Button type="button" variant="ghost" disabled={submitting} onclick={backToLoginName}>Use a different account</Button>
        </form>
      {:else}
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); submitTOTP() }}>
          <input
            type="text"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="6-digit code"
            bind:value={totpCode}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Verifying…" : "Verify"}</Button>
        </form>
      {/if}
      {#if error}
        <p class="text-destructive mt-4" role="alert">{error}</p>
      {/if}
    </CardContent>
  </Card>
</div>
