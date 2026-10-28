<script lang="ts">
	import { Api, ServerError, parsePairing } from '$lib/api';
	import Icon from '$lib/Icon.svelte';
	import Scanner from '$lib/Scanner.svelte';
	import { ordo } from '$lib/state.svelte';
	import Button from '$lib/ui/Button.svelte';

	let { code: given = '' }: { code?: string } = $props();

	const native = import.meta.env.APP_BUILD;
	let link = $state('');
	let server = $state(ordo.session?.server ?? '');
	let passphrase = $state('');
	let usePass = $state(false);
	let hasPass = $state(false);
	let busy = $state(false);
	let error = $state('');
	let scanning = $state(false);

	const expired = $derived(ordo.session?.expired === true);

	$effect(() => {
		if (!native) {
			new Api({ server: '', token: '' }).hello().then(
				(h) => (hasPass = h.has_passphrase),
				() => undefined
			);
		}
	});

	function fail(err: unknown) {
		error = err instanceof Error ? err.message : String(err);
	}

	async function pair(text: string) {
		error = '';
		const target = parsePairing(text);
		if (target === null) {
			error = native
				? 'Paste the whole link, like https://ordo.example.com/pair/…'
				: 'Paste the code or the whole link.';
			return;
		}
		if (native && target.server === '') {
			error = 'Paste the whole link: the app needs the address of your ordo serve too.';
			return;
		}
		const at = native ? target.server : '';
		busy = true;
		try {
			const { token } = await new Api({ server: at, token: '' }).pair(target.code, '', !native);
			await ordo.signIn({ server: at, token: native ? token : '' });
		} catch (err) {
			fail(err);
		} finally {
			busy = false;
		}
	}

	async function login(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		const at = native ? server.trim().replace(/\/+$/u, '') : '';
		if (native && !/^https?:\/\//u.test(at)) {
			error = 'Type the address of your ordo serve, like https://ordo.example.com';
			return;
		}
		busy = true;
		try {
			const { token } = await new Api({ server: at, token: '' }).login(passphrase, '', !native);
			await ordo.signIn({ server: at, token: native ? token : '' });
		} catch (err) {
			if (err instanceof ServerError && err.status === 404) {
				error = 'That address answers, but not as ordo serve.';
			} else {
				fail(err);
			}
		} finally {
			busy = false;
		}
	}

	async function paste() {
		try {
			link = await navigator.clipboard.readText();
		} catch {
			/* the field takes a long-press paste too */
		}
	}

	$effect(() => {
		if (given !== '') {
			void pair(given);
		}
	});
</script>

<main
	class="mx-auto flex min-h-dvh max-w-md flex-col gap-6 px-6 pt-[calc(var(--safe-t)+48px)] pb-safe-b"
>
	<div class="flex flex-col items-center gap-4 text-center">
		<img src="/icon.svg" alt="" class="size-16 rounded-2xl" />
		<h1 class="text-[28px] leading-9">{expired ? 'Sign in again' : 'Sign in to Ordo'}</h1>
		<p class="text-sm text-on-surface-variant">
			{#if expired}
				ordo serve stopped taking this device. Your projects and {ordo.pending} waiting edits stay here
				until you sign in to the same server.
			{:else}
				Run <code class="rounded bg-surface-high px-1">ordo serve pair</code> on your computer, or open
				Devices on a signed-in device, then scan the code or paste its link.
			{/if}
		</p>
	</div>

	{#if !usePass}
		<div class="flex flex-col gap-3">
			<Button
				variant="filled"
				icon="qr_code_scanner"
				onclick={() => (scanning = true)}
				disabled={busy}>Scan the code</Button
			>
			<form
				class="flex flex-col gap-3"
				onsubmit={(e) => {
					e.preventDefault();
					void pair(link);
				}}
			>
				<div
					class="flex items-center gap-1 rounded-sm border border-outline focus-within:border-primary"
				>
					<input
						bind:value={link}
						inputmode="url"
						autocomplete="off"
						autocapitalize="off"
						spellcheck="false"
						placeholder={native ? 'Pairing link' : 'Pairing code or link'}
						class="h-14 min-w-0 flex-1 bg-transparent px-4 outline-none"
					/>
					<button
						type="button"
						class="state-layer grid size-12 place-items-center rounded-full text-on-surface-variant"
						aria-label="Paste"
						onclick={paste}
					>
						<Icon name="content_paste" />
					</button>
				</div>
				<Button variant="tonal" type="submit" disabled={busy || link.trim() === ''}>Pair</Button>
			</form>
		</div>
		{#if native || hasPass}
			<Button onclick={() => (usePass = true)}>Use the passphrase instead</Button>
		{/if}
	{:else}
		<form class="flex flex-col gap-3" onsubmit={login}>
			{#if native}
				<input
					bind:value={server}
					inputmode="url"
					autocomplete="url"
					autocapitalize="off"
					spellcheck="false"
					placeholder="https://ordo.example.com"
					class="h-14 rounded-sm border border-outline bg-transparent px-4 outline-none focus:border-primary"
				/>
			{/if}
			<input
				bind:value={passphrase}
				type="password"
				autocomplete="current-password"
				placeholder="Passphrase"
				class="h-14 rounded-sm border border-outline bg-transparent px-4 outline-none focus:border-primary"
			/>
			<Button variant="filled" type="submit" disabled={busy || passphrase === ''}>Sign in</Button>
		</form>
		<Button onclick={() => (usePass = false)}>Pair with a code instead</Button>
	{/if}

	{#if error}
		<p role="alert" class="rounded-xl bg-error-container p-4 text-sm">{error}</p>
	{/if}
</main>

<Scanner bind:open={scanning} found={(text) => void pair(text)} />
