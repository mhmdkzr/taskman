<script lang="ts">
	import { enhance } from '$app/forms';
	import type { ActionData } from './$types';

	let { form }: { form: ActionData } = $props();
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-950">
	<div class="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-8 shadow-lg">
		<h1 class="mb-6 text-center text-2xl font-semibold text-gray-100">Sign in</h1>

		<form method="post" use:enhance class="space-y-4">
			<label class="block">
				<span class="text-sm font-medium text-gray-300">Email</span>
				<input
					type="email"
					name="email"
					required
					class="mt-1 block w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
					placeholder="you@example.com"
				/>
			</label>
			<label class="block">
				<span class="text-sm font-medium text-gray-300">Password</span>
				<input
					type="password"
					name="password"
					required
					class="mt-1 block w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
					placeholder="Enter your password"
				/>
			</label>
			<button
				class="w-full rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-900 focus:outline-none"
			>
				Sign in
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-gray-400">
			Don't have an account?
			<a href="/register" class="font-medium text-blue-400 hover:text-blue-300">Sign up</a>
		</p>

		{#if form?.message}
			<p class="mt-4 text-center text-sm text-red-400">{form.message}</p>
		{/if}
	</div>
</div>
