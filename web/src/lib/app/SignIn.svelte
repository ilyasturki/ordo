<script lang="ts">
	import { app, site } from '#lib/paths.ts';
	import { restart, session } from '#lib/session.svelte.ts';
	import { parsePairing, parseServer } from '#lib/sync/api.ts';
	import { login, offersPassphrase, pair } from '#lib/sync/session.ts';
	import Field from './Field.svelte';
	import IconButton from './IconButton.svelte';

	let { onBack }: { onBack?: () => void } = $props();

	const sync = session.sync;
	const known = sync?.session.server;
	let address = $state(known ?? '');
	let link = $state('');
	let passphrase = $state('');
	let error: string | undefined = $state();
	let passError: string | undefined = $state();
	let busy: '' | 'pair' | 'login' = $state('');
	let offering = $state(false);

	const server = $derived(site ? (parseServer(address) ?? '') : '');

	$effect(() => {
		const at = server;
		offering = false;
		if (site && at === '') return;
		const timer = setTimeout(() => void offersPassphrase(at).then((yes) => at === server && (offering = yes)), site && !known ? 400 : 0);
		return () => clearTimeout(timer);
	});

	function linkChanged() {
		error = undefined;
		const from = parsePairing(link)?.server;
		if (site && from && address.trim() === '') address = from;
	}

	async function done(failure: string | undefined): Promise<boolean> {
		busy = '';
		if (failure) return false;
		restart(app());
		return true;
	}

	async function pairNow() {
		if (busy || link.trim() === '') return;
		busy = 'pair';
		error = undefined;
		const failure = await pair(server, link, sync);
		if (!(await done(failure))) error = failure;
	}

	async function loginNow() {
		if (busy || passphrase === '') return;
		busy = 'login';
		passError = undefined;
		const failure = await login(server, passphrase, sync);
		if (!(await done(failure))) passError = failure;
	}
</script>

<main class="mx-auto flex min-h-dvh w-full max-w-md flex-col gap-4 px-4 pt-[env(safe-area-inset-top)] pb-8">
	<header class="-mx-2 flex h-16 items-center gap-1">
		{#if onBack}<IconButton icon="back" label="Back" onclick={onBack} />{/if}
		<h1 class={['text-title-large', !onBack && 'px-2']}>
			{sync?.session.expired ? 'Sign in again' : site ? 'Connect to ordo serve' : 'Ordo'}
		</h1>
	</header>

	<p class="text-body-large text-on-surface-variant">
		{#if site}
			Work on the projects of your own ordo serve from this browser. The ones kept here wait until you sign out.
		{:else}
			Pair this browser with your ordo serve. On a paired device, open Devices and pick “Pair a new device”, or run
			“ordo serve pair” on the server, then paste its link.
		{/if}
	</p>
	{#if sync?.session.expired && sync.waitingWords}
		<p class="text-body-medium">{sync.waitingWords} in this browser: they sync once it signs in again to the same server.</p>
	{/if}

	{#if site}
		<Field label="Server address" placeholder="https://…" type="url" inputmode="url" bind:value={address} oninput={() => (error = undefined)} />
	{/if}
	<form
		class="flex flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();
			void pairNow();
		}}
	>
		<Field label="Pairing link or code" placeholder="https://…/pair/…" enterkeyhint="go" bind:value={link} oninput={linkChanged} {error} />
		<button
			class="text-label-large h-10 cursor-pointer rounded-full bg-primary px-6 text-on-primary outline-primary hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-default disabled:bg-on-surface/12 disabled:text-on-surface/38 disabled:shadow-none"
			disabled={busy !== '' || link.trim() === ''}
		>
			{busy === 'pair' ? 'Pairing…' : 'Pair'}
		</button>
	</form>

	{#if offering}
		<form
			class="mt-4 flex flex-col gap-4"
			onsubmit={(e) => {
				e.preventDefault();
				void loginNow();
			}}
		>
			<Field label="Passphrase" type="password" autocomplete="current-password" enterkeyhint="go" bind:value={passphrase} oninput={() => (passError = undefined)} error={passError} />
			<button
				class="text-label-large h-10 cursor-pointer rounded-full bg-secondary-container px-6 text-on-secondary-container outline-primary hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-default disabled:bg-on-surface/12 disabled:text-on-surface/38 disabled:shadow-none"
				disabled={busy !== '' || passphrase === ''}
			>
				{busy === 'login' ? 'Signing in…' : 'Sign in'}
			</button>
		</form>
	{/if}
</main>
