<script lang="ts">
	import type { Snippet } from 'svelte';

	// An M3 alert dialog: Esc and a click outside cancel it.
	let {
		open,
		title,
		onclose,
		children,
		actions
	}: { open: boolean; title: string; onclose: () => void; children?: Snippet; actions: Snippet } = $props();

	let dialog: HTMLDialogElement;

	$effect(() => {
		if (open && !dialog.open) dialog.showModal();
		else if (!open && dialog.open) dialog.close();
	});
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog
	bind:this={dialog}
	aria-label={title}
	oncancel={(e) => {
		e.preventDefault();
		onclose();
	}}
	onclick={(e) => {
		if (e.target === dialog) onclose();
	}}
	class="m-auto w-[min(28rem,calc(100vw-3rem))] bg-transparent p-0 text-on-surface backdrop:bg-scrim/32"
>
	<div class="rounded-xl bg-surface-container-high p-6">
		<h2 class="text-headline-small text-balance">{title}</h2>
		{#if children}
			<div class="text-body-medium mt-4 text-on-surface-variant">{@render children()}</div>
		{/if}
		<div class="mt-6 flex flex-wrap justify-end gap-2">{@render actions()}</div>
	</div>
</dialog>
