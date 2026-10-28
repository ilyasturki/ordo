<script lang="ts" module>
	export type MenuItem = {
		label: string;
		icon?: string;
		danger?: boolean;
		run: () => void;
	};
</script>

<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import Icon from '$lib/Icon.svelte';
	import { registerOverlay } from '$lib/ui/overlays';

	type Props = { label: string; items: MenuItem[]; icon?: string; class?: string };

	let { label, items, icon = 'more_vert', class: cls = '' }: Props = $props();
	let open = $state(false);

	$effect(() => (open ? registerOverlay(() => (open = false)) : undefined));
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger
		class="state-layer grid size-12 place-items-center rounded-full text-on-surface-variant {cls}"
		aria-label={label}
	>
		<Icon name={icon} />
	</DropdownMenu.Trigger>
	<DropdownMenu.Portal>
		<DropdownMenu.Content
			align="end"
			sideOffset={4}
			class="z-50 min-w-48 rounded-sm bg-surface-mid py-2 shadow-lg outline-none"
		>
			{#each items as item (item.label)}
				<DropdownMenu.Item
					onSelect={item.run}
					class="flex h-12 items-center gap-3 px-3 text-sm outline-none select-none data-highlighted:bg-on-surface/10 {item.danger
						? 'text-error'
						: 'text-on-surface'}"
				>
					{#if item.icon}<Icon
							name={item.icon}
							class={item.danger ? '' : 'text-on-surface-variant'}
						/>{/if}
					<span class="pr-3">{item.label}</span>
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>
