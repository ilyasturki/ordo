<script lang="ts">
	import type { Header } from '$lib/model/board';
	import Chip from './Chip.svelte';
	import Icon from './Icon.svelte';
	import IconButton from './IconButton.svelte';
	import { statusIcon } from './icons';

	let {
		header,
		selected = false,
		onclick,
		onFold
	}: { header: Header; selected?: boolean; onclick: () => void; onFold?: () => void } = $props();
</script>

<div
	data-key={header.key}
	class={['mx-2 mt-1 flex min-h-12 items-center rounded-lg', selected ? 'bg-secondary-container' : 'hover:bg-on-surface/8']}
>
	<button
		type="button"
		{onclick}
		aria-label={`${header.category.name}, category menu`}
		class="text-title-small flex min-h-12 min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-lg py-2 pl-3 text-left text-primary outline-primary focus-visible:outline-2"
	>
		<h2 class="min-w-0 break-words">
			{#if header.parentName}<span class="text-on-surface-variant">{header.parentName}&nbsp; ›&nbsp; </span>{/if}{header.category
				.name}{#if header.category.pinned}<Icon name="pin" class="ml-1 inline size-[1em] align-[-0.125em] text-on-surface-variant" />{/if}
		</h2>
		{#if header.folded && header.status}
			<Chip label={header.total > 0 ? `${header.done}/${header.total}` : ''} icon={statusIcon(header.status)} />
		{/if}
	</button>
	{#if onFold}
		<IconButton
			icon={header.folded ? 'expandMore' : 'expandLess'}
			label={`${header.folded ? 'Unfold' : 'Fold'} ${header.category.name}`}
			class="wide:m-1"
			onclick={onFold}
		/>
	{:else}
		<span class="w-3"></span>
	{/if}
</div>
