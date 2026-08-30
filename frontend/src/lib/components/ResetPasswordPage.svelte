<script>
  import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import { resetPassword } from "$lib/api/registration.js"
  import { LoginSessionError } from "$lib/api/loginSession.js"

  /** @type {{ userId: string, code: string }} */
  let { userId, code } = $props()

  let newPassword = $state("")
  let submitting = $state(false)
  let error = $state("")
  let done = $state(false)

  async function submitReset() {
    error = ""
    if (!newPassword) {
      error = "Enter a new password."
      return
    }
    submitting = true
    try {
      await resetPassword(userId, code, newPassword)
      done = true
    } catch (err) {
      error = err instanceof LoginSessionError ? "That reset link is invalid or has expired." : "Something went wrong. Please try again."
    } finally {
      submitting = false
    }
  }
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Set a new password</CardTitle>
    </CardHeader>
    <CardContent>
      {#if done}
        <CardDescription class="mb-4">Your password has been changed.</CardDescription>
        <Button href="/auth/login?return_to=/">Continue to sign in</Button>
      {:else}
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); submitReset() }}>
          <input
            type="password"
            autocomplete="new-password"
            placeholder="New password"
            bind:value={newPassword}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Saving…" : "Save new password"}</Button>
        </form>
      {/if}
      {#if error}
        <p class="text-destructive mt-4" role="alert">{error}</p>
      {/if}
    </CardContent>
  </Card>
</div>
