<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Dialog } from 'bits-ui';
	import { registerOverlay } from '$lib/ui/overlays';

	type Props = {
		open?: boolean;
		title: string;
		children?: Snippet;
		actions: Snippet;
	};

	let { open = $bindable(false), title, children, actions }: Props = $props();

	$effect(() => (open ? registerOverlay(() => (open = false)) : undefined));
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-50 bg-scrim" />
		<Dialog.Content
			class="fixed top-1/2 left-1/2 z-50 flex w-[min(calc(100vw-48px),560px)] -translate-x-1/2 -translate-y-1/2 flex-col gap-4 rounded-[28px] bg-surface-high p-6 shadow-xl outline-none"
		>
			<Dialog.Title class="text-2xl leading-8 text-on-surface">{title}</Dialog.Title>
			{#if children}
				<div class="text-sm text-on-surface-variant">{@render children()}</div>
			{/if}
			<div class="flex justify-end gap-2">{@render actions()}</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
