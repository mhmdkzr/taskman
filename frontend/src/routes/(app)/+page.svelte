<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageServerData } from './$types';

	let { data }: { data: PageServerData } = $props();
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-950">
	<div class="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-8 text-center shadow-lg">
		<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-blue-600 text-lg font-bold text-white">
			{data.user.name.charAt(0).toUpperCase()}
		</div>
		<h1 class="text-2xl font-semibold text-gray-100">Hi, {data.user.name}!</h1>
		<p class="mt-2 text-sm text-gray-400">Your user ID is {data.user.id}.</p>
		<form method="post" action="?/signOut" use:enhance class="mt-6">
			<button
				class="w-full rounded-lg bg-gray-800 px-4 py-2 text-sm font-medium text-gray-300 transition hover:bg-gray-700 focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 focus:ring-offset-gray-900 focus:outline-none"
			>
				Sign out
			</button>
		</form>
	</div>
</div>
