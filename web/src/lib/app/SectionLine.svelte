<script lang="ts">
	import type { Section } from '#lib/model/board.ts';
	import TagDot from './TagDot.svelte';

	let { section, selected = false, onclick }: { section: Section; selected?: boolean; onclick: () => void } = $props();

	const label = $derived(section.row.task.title);
	const count = $derived(section.count === 0 ? '' : section.revealed ? `hide ${section.count}` : `${section.count} hidden`);
	const tag = $derived(section.row.task.tag_color ?? '');
</script>

{#snippet rule()}
	{#if label || count}
		<span class="h-0.5 w-5 shrink-0 rounded-full bg-on-surface-variant"></span>
		<span class="text-title-small truncate"><span class="font-bold">{label}</span>{label && count ? ' · ' : ''}{count}</span>
	{/if}
	<span class="h-0.5 min-w-5 flex-1 rounded-full bg-on-surface-variant"></span>
	{#if tag}<TagDot color={tag} class="ml-1 size-2.5" />{/if}
{/snippet}

<div
	data-key={section.key}
	class={['mx-2 flex rounded-lg text-on-surface-variant', selected ? 'bg-secondary-container' : section.count > 0 && 'hover:bg-on-surface/8']}
>
	{#if section.count > 0}
		<button
			type="button"
			{onclick}
			aria-expanded={section.revealed}
			class="flex h-12 min-w-0 flex-1 cursor-pointer items-center gap-2.5 rounded-lg pr-4 pl-3 outline-primary focus-visible:outline-2"
		>
			{@render rule()}
		</button>
	{:else}
		<div class="flex h-12 min-w-0 flex-1 items-center gap-2.5 pr-4 pl-3">{@render rule()}</div>
	{/if}
</div>
