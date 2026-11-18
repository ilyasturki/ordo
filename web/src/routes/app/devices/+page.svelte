<script lang="ts">
	import { goto } from '$app/navigation';
	import Dialog from '$lib/app/Dialog.svelte';
	import Field from '$lib/app/Field.svelte';
	import Icon from '$lib/app/Icon.svelte';
	import IconButton from '$lib/app/IconButton.svelte';
	import Sheet from '$lib/app/Sheet.svelte';
	import TextButton from '$lib/app/TextButton.svelte';
	import { app } from '$lib/paths';
	import { restart, session } from '$lib/session.svelte';
	import type { DeviceInfo, Devices, PairCode } from '$lib/sync/api';

	// The phone's Devices: who syncs with the server, a code to pair one more, and the way out.
	const sync = session.sync;
	const local = session.local!;

	let devices: Devices | undefined = $state();
	let error: string | undefined = $state();
	let revoking: DeviceInfo | undefined = $state();
	let signingOut = $state(false);
	let pairing = $state(false);
	let code: PairCode | undefined = $state();
	let passphrase = $state('');
	let pairError: string | undefined = $state();
	let making = $state(false);

	const day = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' });
	const dayOf = (at: string) => (Number.isNaN(Date.parse(at)) ? '' : day.format(new Date(at)));

	async function load() {
		if (!sync) return;
		try {
			devices = await sync.devices();
			error = undefined;
		} catch (e) {
			error = e instanceof Error ? e.message : "Can't reach ordo serve";
		}
	}

	$effect(() => {
		if (!sync) void goto(app(), { replaceState: true });
		else void load();
	});

	async function make() {
		if (!sync || making) return;
		making = true;
		pairError = undefined;
		try {
			code = await sync.pairDevice(passphrase);
			passphrase = '';
		} catch (e) {
			pairError = e instanceof Error ? e.message : "Can't reach ordo serve";
		} finally {
			making = false;
		}
	}

	function openPairing() {
		pairing = true;
		code = undefined;
		pairError = undefined;
		if (!devices?.has_passphrase) void make();
	}

	async function revoke(d: DeviceInfo) {
		revoking = undefined;
		try {
			if (await sync!.revoke(d.id)) restart(app());
			else await load();
		} catch (e) {
			local.say(e instanceof Error ? e.message : "Can't reach ordo serve");
		}
	}

	async function signOut() {
		signingOut = false;
		await sync!.signOut();
		restart(app());
	}

	function revokeText(d: DeviceInfo): string {
		if (d.also?.length) return `Devices it paired sign out with it: ${d.also.join(', ')}.`;
		if (d.self) return 'This browser signs out and forgets its copy of your projects.';
		return 'It stops syncing until it pairs again.';
	}
</script>

<svelte:head><title>Devices · Ordo</title></svelte:head>

{#if sync}
	<main class="mx-auto flex min-h-dvh w-full max-w-2xl flex-col pt-[env(safe-area-inset-top)] pb-8">
		<header class="flex h-16 items-center gap-1 px-2">
			<IconButton icon="back" label="Back" onclick={() => void goto(app())} />
			<h1 class="text-title-large">Devices</h1>
		</header>

		{#if error}
			<p role="alert" class="text-body-medium px-6">{error}</p>
			<div class="px-3"><TextButton onclick={load}>Retry</TextButton></div>
		{/if}

		<ul aria-label="Devices">
			{#each devices?.devices ?? [] as d (d.id)}
				<li class="flex items-center gap-5 py-3.5 pr-3 pl-6">
					<Icon name="devices" class="size-6 text-on-surface-variant" />
					<div class="min-w-0 flex-1">
						<p class="text-body-large truncate">
							{d.name}{#if d.self}<span class="text-primary"> · this device</span>{/if}
						</p>
						<p class="text-body-medium text-on-surface-variant">
							Paired {dayOf(d.created_at)}{d.from ? ` by ${d.from}` : ''} · seen {dayOf(d.last_seen)}
						</p>
					</div>
					<IconButton icon="logout" label={`Sign out ${d.name}`} onclick={() => (revoking = d)} />
				</li>
			{/each}
		</ul>

		{#snippet item(icon: 'qrCode' | 'logout', title: string, text: string, onclick: () => void, danger = false)}
			<button
				type="button"
				class="flex min-h-18 w-full cursor-pointer items-center gap-5 px-6 py-3 text-left outline-primary hover:bg-on-surface/8 focus-visible:outline-2 focus-visible:-outline-offset-2"
				{onclick}
			>
				<Icon name={icon} class={['size-6', danger ? 'text-error' : 'text-on-surface-variant']} />
				<span class="min-w-0 flex-1">
					<span class={['text-body-large block', danger && 'text-error']}>{title}</span>
					<span class="text-body-medium block text-on-surface-variant">{text}</span>
				</span>
			</button>
		{/snippet}
		{@render item('qrCode', 'Pair a new device', 'Show a code to scan on the new device', openPairing)}
		{@render item('logout', 'Sign out', 'Drops this browser’s copy; the server keeps everything it got', () => (signingOut = true), true)}
	</main>

	<Dialog open={revoking !== undefined} title={`Sign out “${revoking?.name ?? ''}”?`} onclose={() => (revoking = undefined)}>
		{revoking ? revokeText(revoking) : ''}
		{#snippet actions()}
			<TextButton onclick={() => (revoking = undefined)}>Cancel</TextButton>
			<TextButton danger onclick={() => revoking && revoke(revoking)}>Sign out</TextButton>
		{/snippet}
	</Dialog>

	<Dialog open={signingOut} title="Sign out?" onclose={() => (signingOut = false)}>
		{local.waiting
			? `Not on the server yet: ${sync.waitingWords}. Signing out drops them.`
			: 'The server keeps your projects; pair this browser again to get them back.'}
		{#snippet actions()}
			<TextButton onclick={() => (signingOut = false)}>Cancel</TextButton>
			<TextButton danger onclick={signOut}>Sign out</TextButton>
		{/snippet}
	</Dialog>

	<Sheet open={pairing} label="Pair a new device" onclose={() => (pairing = false)}>
		<div class="flex flex-col items-center gap-4 px-6 pt-6 pb-6 text-center">
			<h2 class="text-title-large">Pair a new device</h2>
			{#if code}
				<img src={code.qr} alt="Pairing code" width="280" height="280" class="size-70 [image-rendering:pixelated]" />
				<p class="text-body-medium text-on-surface-variant">
					Scan it from the sign-in screen on the new device, or paste the link there. It works once, for {code.minutes} minutes.
				</p>
				<p class="text-body-small break-all select-all">{code.link}</p>
				<TextButton
					onclick={() =>
						void navigator.clipboard.writeText(code!.link).then(
							() => local.say('Link copied'),
							() => local.say('Select the link to copy it')
						)}
				>
					Copy the link
				</TextButton>
			{:else if devices?.has_passphrase}
				<p class="text-body-medium text-on-surface-variant">Type the passphrase again to make a pairing code.</p>
				<form
					class="flex w-full flex-col gap-4 text-left"
					onsubmit={(e) => {
						e.preventDefault();
						void make();
					}}
				>
					<Field label="Passphrase" type="password" autocomplete="current-password" bind:value={passphrase} error={pairError} oninput={() => (pairError = undefined)} />
					<TextButton filled type="submit" disabled={making || passphrase === ''}>{making ? 'Making a code…' : 'Make a code'}</TextButton>
				</form>
			{:else if making}
				<p class="text-body-medium text-on-surface-variant" aria-busy="true">Making a code…</p>
			{:else}
				<p class="text-body-medium">{pairError ?? ''}</p>
				<TextButton onclick={make}>Retry</TextButton>
			{/if}
		</div>
	</Sheet>
{/if}
