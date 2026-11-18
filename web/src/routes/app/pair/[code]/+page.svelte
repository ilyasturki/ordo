<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { app, site } from '$lib/paths';
	import { restart } from '$lib/session.svelte';
	import { pair } from '$lib/sync/session';

	// A pairing link opened in the browser. A link preview fetches it too, so only a tap spends it.
	const code = page.params.code ?? '';
	let error: string | undefined = $state();
	let busy = $state(false);

	onMount(() => {
		history.replaceState(history.state, '', app('/signin'));
		if (site) restart(app('/signin'));
	});

	async function go() {
		busy = true;
		error = await pair('', code);
		busy = false;
		if (!error) restart(app());
	}
</script>

<svelte:head><title>Pair · Ordo</title></svelte:head>

<main class="mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center gap-4 px-4 py-8">
	<img src="/icon.svg" alt="" width="48" height="48" />
	<h1 class="text-headline-small">Pair this browser?</h1>
	<p class="text-body-large text-on-surface-variant">It then syncs your projects with this ordo serve. The link works once.</p>
	{#if error}<p role="alert" class="text-body-medium text-error">{error}</p>{/if}
	<button
		type="button"
		class="text-label-large h-10 cursor-pointer self-start rounded-full bg-primary px-6 text-on-primary outline-primary hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-default disabled:bg-on-surface/12 disabled:text-on-surface/38"
		disabled={busy}
		onclick={go}
	>
		{busy ? 'Pairing…' : 'Pair'}
	</button>
</main>
