<script>
  import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import { requestPasswordReset } from "$lib/api/registration.js"

  let loginName = $state("")
  let submitting = $state(false)
  let error = $state("")
  let sent = $state(false)

  async function submitRequest() {
    error = ""
    if (!loginName.trim()) {
      error = "Enter your login name."
      return
    }
    submitting = true
    try {
      await requestPasswordReset(loginName.trim())
      sent = true
    } catch {
      error = "Something went wrong. Please try again."
    } finally {
      submitting = false
    }
  }
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Reset your password</CardTitle>
      <CardDescription>
        {#if sent}
          Check your email for a reset link.
        {:else}
          Enter your login name and we'll email you a reset link.
        {/if}
      </CardDescription>
    </CardHeader>
    <CardContent>
      {#if sent}
        <p>If an account exists for that login name, a reset link is on its way.</p>
        <Button class="mt-4" href="/auth/login?return_to=/">Back to sign in</Button>
      {:else}
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); submitRequest() }}>
          <input
            type="text"
            autocomplete="username"
            placeholder="Login name"
            bind:value={loginName}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Sending…" : "Send reset link"}</Button>
          <Button type="button" variant="ghost" href="/auth/login?return_to=/">Back to sign in</Button>
        </form>
      {/if}
      {#if error}
        <p class="text-destructive mt-4" role="alert">{error}</p>
      {/if}
    </CardContent>
  </Card>
</div>
