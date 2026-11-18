<script lang="ts">
	import { page } from '$app/state';
	import SignIn from '$lib/app/SignIn.svelte';
	import Snackbar from '$lib/app/Snackbar.svelte';
	import { openSession, session, useHere } from '$lib/session.svelte';

	let { children } = $props();

	void openSession();
</script>

<svelte:head>
	<title>Ordo</title>
</svelte:head>

{#if session.elsewhere}
	<main class="flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
		<img src="/icon.svg" alt="" width="48" height="48" />
		<h1 class="text-headline-small">Open in another tab</h1>
		<p class="text-body-medium max-w-80 text-on-surface-variant">Ordo syncs from one tab at a time.</p>
		<button
			type="button"
			class="text-label-large mt-2 h-10 cursor-pointer rounded-full bg-primary px-6 text-on-primary outline-primary hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2"
			onclick={() => void useHere()}
		>
			Use here
		</button>
	</main>
{:else if session.local}
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
