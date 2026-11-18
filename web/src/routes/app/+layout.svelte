<script lang="ts">
	import { openSession, session } from '$lib/session.svelte';

	let { children } = $props();

	void openSession();
</script>

<svelte:head>
	<title>Ordo</title>
</svelte:head>

{#if session.local}
	{@render children()}
{:else if session.error}
	<p role="alert" class="text-body-large text-error p-4">Ordo couldn't start: {session.error}</p>
{:else}
	<p class="text-body-large text-on-surface-variant p-4" aria-busy="true">Opening…</p>
{/if}
