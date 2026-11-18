<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Local } from '$lib/local.svelte';
	import { app } from '$lib/paths';
	import type { Sync } from '$lib/sync/sync.svelte';
	import Icon from './Icon.svelte';
	import Sheet from './Sheet.svelte';
	import TextButton from './TextButton.svelte';

	let { sync, local, onclose }: { sync: Sync; local: Local; onclose: () => void } = $props();

	const broken = $derived(Object.entries(local.broken).map(([id, error]) => ({ id, name: local.project(id)?.name ?? id, error })));
	const troubled = $derived(!!sync.trouble || broken.length > 0);
	const words = $derived.by(() => {
		const t = sync.trouble;
		if (!t && broken.length > 0) {
			return broken.length === 1
				? ["A project can't sync", "Your server can't open its file, so this browser keeps its copy as it was. Fix the file on the server and it syncs again."]
				: ["Some projects can't sync", "Your server can't open their files, so this browser keeps its copies as they were. Fix the files on the server and they sync again."];
		}
		switch (t?.kind) {
			case 'failing':
				return t.offline
					? ["Can't reach your server", "This browser can't get to ordo serve right now: no network, or the server is down. It keeps trying while the page is open."]
					: ["Your server isn't answering", `ordo serve answered with an error: ${t.message}. This browser keeps trying while the page is open.`];
			case 'refused':
				return ['Your server refused the sync', `ordo serve said: ${t.message}`];
			case 'outdated':
				return ['Update needed', `This app and your server speak different versions of the sync: ${t.message}`];
			case 'expired':
				return ['Sign in again', 'Your server no longer knows this browser: sign in again to keep syncing.'];
			default:
				return ['Synced', 'Everything in this browser has reached your server.'];
		}
	});
	const waiting = $derived(
		!local.waiting
			? 'No edit is waiting.'
			: `${sync.waitingWords} ${local.pending + local.uploads + local.viewsWaiting === 1 ? 'waits' : 'wait'} safely in this browser.`
	);

	function ago(at: number): string {
		const s = Math.round((Date.now() - at) / 1000);
		if (s < 60) return 'just now';
		if (s < 3600) return `${Math.floor(s / 60)} min ago`;
		if (s < 86_400) return `${Math.floor(s / 3600)} h ago`;
		return `${Math.floor(s / 86_400)} days ago`;
	}
</script>

<Sheet open label="Sync" onclose={() => onclose()}>
	<div class="flex flex-col gap-3 px-6 pt-6 pb-4">
		<h2 class="text-title-large flex items-center gap-3">
			<Icon name={troubled ? 'cloudOff' : 'sync'} class={['size-6', troubled ? 'text-error' : 'text-primary']} />
			{words[0]}
		</h2>
		<p class="text-body-medium">{words[1]}</p>
		{#each broken as b (b.id)}
			<p class="text-body-medium break-words"><span class="font-medium">{b.name}</span>: {b.error}</p>
		{/each}
		<p class="text-body-medium">{waiting}</p>
		<p class="text-body-small text-on-surface-variant">
			Last synced: {sync.lastSync === 0 ? 'not since the page opened' : ago(sync.lastSync)}<br />
			Server: {sync.session.server || location.origin}
		</p>
		<div class="flex justify-end pt-2">
			{#if sync.trouble?.kind === 'expired'}
				<TextButton
					onclick={() => {
						onclose();
						void goto(app('/signin'));
					}}
				>
					Sign in again
				</TextButton>
			{:else}
				<TextButton onclick={() => void sync.sync()}>Retry</TextButton>
			{/if}
		</div>
	</div>
</Sheet>
