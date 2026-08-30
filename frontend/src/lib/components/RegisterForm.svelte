<script>
  import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import { register } from "$lib/api/registration.js"
  import { LoginSessionError } from "$lib/api/loginSession.js"

  let loginName = $state("")
  let email = $state("")
  let givenName = $state("")
  let familyName = $state("")
  let password = $state("")
  let submitting = $state(false)
  let error = $state("")
  let registered = $state(false)

  async function submitRegister() {
    error = ""
    if (!loginName.trim() || !email.trim() || !givenName.trim() || !familyName.trim() || !password) {
      error = "Fill in every field."
      return
    }
    submitting = true
    try {
      await register({
        loginName: loginName.trim(),
        email: email.trim(),
        givenName: givenName.trim(),
        familyName: familyName.trim(),
        password,
      })
      registered = true
    } catch (err) {
      error = err instanceof LoginSessionError ? "That login name or email is already taken." : "Something went wrong. Please try again."
    } finally {
      submitting = false
    }
  }
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Create an account</CardTitle>
      <CardDescription>
        {#if registered}
          Check your email to verify your account.
        {:else}
          Fill in your details to sign up.
        {/if}
      </CardDescription>
    </CardHeader>
    <CardContent>
      {#if registered}
        <p>We sent a verification link to <strong>{email}</strong>. Open it to finish creating your account.</p>
        <Button class="mt-4" href="/auth/login?return_to=/">Back to sign in</Button>
      {:else}
        <form class="flex flex-col gap-3" onsubmit={(event) => { event.preventDefault(); submitRegister() }}>
          <input
            type="text"
            autocomplete="username"
            placeholder="Login name"
            bind:value={loginName}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <input
            type="email"
            autocomplete="email"
            placeholder="Email"
            bind:value={email}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <div class="flex gap-2">
            <input
              type="text"
              autocomplete="given-name"
              placeholder="First name"
              bind:value={givenName}
              disabled={submitting}
              class="border-input h-9 w-1/2 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
            />
            <input
              type="text"
              autocomplete="family-name"
              placeholder="Last name"
              bind:value={familyName}
              disabled={submitting}
              class="border-input h-9 w-1/2 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
            />
          </div>
          <input
            type="password"
            autocomplete="new-password"
            placeholder="Password"
            bind:value={password}
            disabled={submitting}
            class="border-input h-9 rounded-2xl border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Creating account…" : "Create account"}</Button>
        </form>
        <p class="mt-4 text-sm">
          Already have an account? <a class="underline" href="/auth/login?return_to=/">Sign in</a>
        </p>
      {/if}
      {#if error}
        <p class="text-destructive mt-4" role="alert">{error}</p>
      {/if}
    </CardContent>
  </Card>
</div>
