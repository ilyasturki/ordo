<script lang="ts">
	import { goto } from '$app/navigation';
	import type { DeviceInfo } from '$lib/api';
	import Icon from '$lib/Icon.svelte';
	import { ordo } from '$lib/state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Confirm, { type ConfirmSpec } from '$lib/ui/Confirm.svelte';
	import IconButton from '$lib/ui/IconButton.svelte';
	import Sheet from '$lib/ui/Sheet.svelte';

	let devices = $state<DeviceInfo[] | null>(null);
	let hasPass = $state(false);
	let error = $state('');
	let confirm = $state<ConfirmSpec | null>(null);
	let pairOpen = $state(false);
	let passphrase = $state('');
	let pairing = $state<{ link: string; qr: string; minutes: number } | null>(null);
	let pairError = $state('');
	let busy = $state(false);

	async function load() {
		error = '';
		try {
			const res = await ordo.api?.devices();
			devices = res?.devices ?? [];
			hasPass = res?.has_passphrase ?? false;
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		}
	}

	$effect(() => {
		void load();
	});

	function when(ts: string): string {
		const d = new Date(ts);
		return Number.isNaN(d.getTime())
			? ''
			: d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
	}

	function revoke(d: DeviceInfo) {
		confirm = {
			title: `Sign out “${d.name}”?`,
			body: d.also
				? `Devices it paired sign out with it: ${d.also}.`
				: d.self
					? 'This device signs out and forgets its copy of your projects.'
					: 'It stops syncing until it pairs again.',
			confirm: 'Sign out',
			run: async () => {
				try {
					const res = await ordo.api?.revoke(d.id);
					if (res?.self) {
						await ordo.signOut();
						await goto('/', { replaceState: true });
						return;
					}
					await load();
				} catch (err) {
					ordo.toast(err instanceof Error ? err.message : String(err));
				}
			}
		};
	}

	function signOut() {
		confirm = {
			title: 'Sign out?',
			body:
				ordo.pending > 0
					? `${ordo.pending} edits haven’t reached ordo serve yet and will be lost.`
					: 'This device forgets its copy of your projects.',
			confirm: 'Sign out',
			run: async () => {
				await ordo.signOut();
				await goto('/', { replaceState: true });
			}
		};
	}

	async function pair(e?: SubmitEvent) {
		e?.preventDefault();
		pairError = '';
		busy = true;
		try {
			pairing = (await ordo.api?.pairDevice(passphrase)) ?? null;
			passphrase = '';
		} catch (err) {
			pairError = err instanceof Error ? err.message : String(err);
		} finally {
			busy = false;
		}
	}

	function startPair() {
		pairing = null;
		pairError = '';
		pairOpen = true;
		if (!hasPass) {
			void pair();
		}
	}

	async function share() {
		if (pairing === null) {
			return;
		}
		try {
			if (navigator.share) {
				await navigator.share({ url: pairing.link });
			} else {
				await navigator.clipboard.writeText(pairing.link);
				ordo.toast('Link copied');
			}
		} catch {
			/* dismissed */
		}
	}
</script>

<header
	class="sticky top-0 z-30 flex h-[calc(64px+var(--safe-t))] items-center gap-1 bg-surface px-1 pt-safe-t"
>
	<IconButton
		icon="arrow_back"
		label="Back"
		onclick={() => (history.length > 1 ? history.back() : goto('/'))}
	/>
	<h1 class="flex-1 pl-2 text-[22px]">Devices</h1>
</header>

<main class="mx-auto flex max-w-2xl flex-col gap-4 px-4 pb-[calc(var(--safe-b)+48px)]">
	<p class="text-sm text-on-surface-variant">
		Synced with {ordo.session?.server || location.origin}{#if ordo.lastSync}
			· last sync {new Date(ordo.lastSync).toLocaleTimeString()}{/if}{#if ordo.pending > 0}
			· {ordo.pending} edits waiting{/if}
	</p>
	{#if error}
		<p class="rounded-xl bg-error-container p-4 text-sm">{error}</p>
	{/if}
	{#if devices}
		<ul class="overflow-hidden rounded-3xl bg-surface-low">
			{#each devices as d (d.id)}
				<li class="flex items-center gap-4 py-2 pr-1 pl-4">
					<Icon name="smartphone" class="text-on-surface-variant" />
					<div class="flex min-w-0 flex-1 flex-col">
						<span class="truncate"
							>{d.name}{#if d.self}
								<span class="text-sm text-primary">· this device</span>{/if}</span
						>
						<span class="truncate text-sm text-on-surface-variant">
							Paired {when(d.created_at)}{#if d.from}
								by {d.from}{/if} · seen {when(d.last_seen)}
						</span>
					</div>
					<IconButton icon="logout" label="Sign out {d.name}" onclick={() => revoke(d)} />
				</li>
			{/each}
		</ul>
	{/if}
	<div class="flex flex-wrap gap-2">
		<Button variant="filled" icon="qr_code" onclick={startPair}>Pair a new device</Button>
		<Button icon="logout" onclick={signOut}>Sign out</Button>
	</div>
</main>

<Sheet bind:open={pairOpen} title="Pair a new device">
	{#if pairing}
		<div class="flex flex-col items-center gap-4">
			<img
				src={pairing.qr}
				alt="Pairing code"
				class="w-64 rounded-xl bg-paper p-2 [image-rendering:pixelated]"
			/>
			<p class="text-center text-sm text-on-surface-variant">
				Scan it from the Ordo app’s sign-in screen, or open the link on the new device. It works
				once, for {pairing.minutes} minutes.
			</p>
			<code class="w-full rounded-lg bg-surface-high p-3 text-xs break-all">{pairing.link}</code>
			<Button variant="tonal" icon="share" onclick={share}>Share the link</Button>
		</div>
	{:else if hasPass}
		<form class="flex flex-col gap-3" onsubmit={pair}>
			<p class="text-sm text-on-surface-variant">
				Type the passphrase again to make a pairing code.
			</p>
			<input
				bind:value={passphrase}
				type="password"
				autocomplete="current-password"
				placeholder="Passphrase"
				class="h-14 rounded-sm border border-outline bg-transparent px-4 outline-none focus:border-primary"
			/>
			<Button variant="filled" type="submit" disabled={busy || passphrase === ''}
				>Make a code</Button
			>
		</form>
	{:else if busy}
		<p class="text-sm text-on-surface-variant">Making a code…</p>
	{/if}
	{#if pairError}
		<p class="mt-3 rounded-xl bg-error-container p-4 text-sm">{pairError}</p>
	{/if}
</Sheet>

<Confirm bind:spec={confirm} />
