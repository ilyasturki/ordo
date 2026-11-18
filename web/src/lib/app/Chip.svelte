<script lang="ts">
	import Icon from './Icon.svelte';
	import TagDot from './TagDot.svelte';
	import type { IconName } from './icons';

	// A task's value, small and read only on the list, bigger wherever a click opens its picker.
	let {
		label = '',
		icon,
		tag = '',
		tone = '',
		trailing,
		name = '',
		onclick
	}: {
		label?: string;
		icon?: IconName;
		tag?: string;
		tone?: 'up' | 'down' | '';
		trailing?: IconName;
		name?: string;
		onclick?: (event: MouseEvent) => void;
	} = $props();

	const color = $derived({ up: 'text-mark-up', down: 'text-mark-down', '': 'text-on-surface-variant' }[tone]);
</script>

{#snippet body(big: boolean)}
	{#if icon}<Icon name={icon} class={big ? 'size-[18px]' : 'size-3.5'} />{/if}
	{#if tag}<TagDot color={tag} class={big ? 'mx-0.5 size-3' : 'mx-0.5 size-2.5'} />{/if}
	{#if label}<span class="truncate">{label}</span>{/if}
	{#if trailing}<Icon name={trailing} class="size-[18px] text-on-surface-variant" />{/if}
{/snippet}

{#if onclick}
	<button
		type="button"
		{onclick}
		aria-label={name ? `${name}: ${label}` : undefined}
		class={[
			'text-label-large inline-flex h-8 max-w-full min-w-10 cursor-pointer items-center justify-center gap-1 rounded-sm border border-outline pl-1.5 outline-primary hover:bg-on-surface/8 focus-visible:outline-2 active:bg-on-surface/12',
			trailing ? 'pr-1' : 'pr-2',
			color
		]}
	>
		{@render body(true)}
	</button>
{:else}
	<span class={['text-label-medium inline-flex h-6 max-w-full items-center gap-1 rounded-[6px] border border-outline px-1.5', color]}>
		{@render body(false)}
	</span>
{/if}
