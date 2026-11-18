<script lang="ts">
	import { page } from '$app/state';
	import SignIn from '$lib/app/SignIn.svelte';
	import Snackbar from '$lib/app/Snackbar.svelte';
	import { openSession, session } from '$lib/session.svelte';

	let { children } = $props();

	void openSession();
</script>

<svelte:head>
	<title>Ordo</title>
</svelte:head>

{#if session.local}
	{@render children()}
	<Snackbar local={session.local} />
{:else if session.signIn}
	{#if page.route.id === '/app/pair/[code]'}
		{@render children()}
	{:else}
		<SignIn />
	{/if}
{:else if session.error}
	<p role="alert" class="text-body-large p-4 text-error">Ordo couldn't start: {session.error}</p>
{:else}
	<p class="text-body-large p-4 text-on-surface-variant" aria-busy="true">Opening…</p>
{/if}
