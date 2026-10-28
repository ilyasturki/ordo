<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Drawer } from 'vaul-svelte';
	import { registerOverlay } from '$lib/ui/overlays';

	type Props = {
		open?: boolean;
		title: string;
		children: Snippet;
		actions?: Snippet;
	};

	let { open = $bindable(false), title, children, actions }: Props = $props();

	$effect(() => (open ? registerOverlay(() => (open = false)) : undefined));
</script>

<Drawer.Root bind:open repositionInputs={false}>
	<Drawer.Portal>
		<Drawer.Overlay class="fixed inset-0 z-40 bg-scrim" />
		<Drawer.Content
			class="fixed inset-x-0 bottom-0 z-40 mx-auto flex max-h-[92dvh] max-w-2xl flex-col rounded-t-[28px] bg-surface-low pb-safe-b outline-none"
		>
			<Drawer.Handle class="mx-auto mt-4 mb-2 !h-1 !w-8 rounded-full !bg-outline-variant" />
			<Drawer.Title class="px-6 pt-2 pb-2 text-[22px] leading-7 text-on-surface"
				>{title}</Drawer.Title
			>
			<div class="min-h-0 flex-1 overflow-y-auto px-6 pb-4">
				{@render children()}
			</div>
			{#if actions}
				<div class="flex justify-end gap-2 px-6 pt-1 pb-4">{@render actions()}</div>
			{/if}
		</Drawer.Content>
	</Drawer.Portal>
</Drawer.Root>
