<script>
  import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import { verifyEmail, startTOTPEnrollment, confirmTOTPEnrollment } from "$lib/api/registration.js"

  /** @type {{ userId: string, code: string }} */
  let { userId, code } = $props()

  /** @type {"verifying" | "verified" | "error" | "totp-setup" | "totp-done"} */
  let step = $state("verifying")
  let totpSecret = $state("")
  let totpCode = $state("")
  let submitting = $state(false)
  let error = $state("")

  async function runVerify() {
    try {
      await verifyEmail(userId, code)
      step = "verified"
    } catch {
      step = "error"
    }
  }

  async function enableTOTP() {
    error = ""
    submitting = true
    try {
      const { secret } = await startTOTPEnrollment(userId)
      totpSecret = secret
      step = "totp-setup"
    } catch {
      error = "Couldn't start 2FA setup. You can add it later."
    } finally {
      submitting = false
    }
  }

  async function confirmTOTP() {
    error = ""
    if (!totpCode.trim()) {
      error = "Enter the 6-digit code from your authenticator app."
      return
    }
    submitting = true
    try {
      await confirmTOTPEnrollment(userId, totpCode.trim())
      step = "totp-done"
    } catch {
      error = "That code didn't match. Try again."
    } finally {
      submitting = false
    }
  }

  runVerify()
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Verify your email</CardTitle>
    </CardHeader>
    <CardContent>
      {#if step === "verifying"}
        <p>Verifying…</p>
      {:else if step === "error"}
        <p class="text-destructive" role="alert">
          That verification link is invalid or has expired. Please register again or contact support.
        </p>
      {:else if step === "verified"}
        <CardDescription class="mb-4">Your email is verified. Want to add an extra layer of security?</CardDescription>
        <div class="flex flex-col gap-2">
          <Button onclick={enableTOTP} disabled={submitting}>Enable two-factor authentication</Button>
          <Button variant="ghost" href="/auth/login?return_to=/">Skip for now</Button>
        </div>
      {:else if step === "totp-setup"}
        <CardDescription class="mb-2">
          Scan this into your authenticator app, or enter the secret manually, then confirm the code it shows.
        </CardDescription>
        <p class="border-input mb-3 break-all rounded-2xl border bg-transparent px-3 py-2 text-sm">{totpSecret}</p>
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); confirmTOTP() }}>
          <input
            type="text"
            inputmode="numeric"
            placeholder="6-digit code"
            bind:value={totpCode}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Confirming…" : "Confirm"}</Button>
        </form>
      {:else if step === "totp-done"}
        <CardDescription class="mb-4">Two-factor authentication is enabled for your account.</CardDescription>
        <Button href="/auth/login?return_to=/">Continue to sign in</Button>
      {/if}
      {#if error}
        <p class="text-destructive mt-4" role="alert">{error}</p>
      {/if}
    </CardContent>
  </Card>
</div>
