<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';
	import type { IconName } from './icons';

	let {
		label,
		icon,
		danger = false,
		checked,
		keys = '',
		leading,
		onclick
	}: {
		label: string;
		icon?: IconName;
		danger?: boolean;
		/** Set on a picker's choices: the one that holds now. */
		checked?: boolean;
		/** The key that does the same, shown on a wide screen. */
		keys?: string;
		leading?: Snippet;
		onclick: () => void;
	} = $props();
</script>

<button
	type="button"
	role={checked === undefined ? 'menuitem' : 'menuitemradio'}
	aria-checked={checked}
	{onclick}
	class={[
		'text-body-large flex h-12 w-full cursor-pointer items-center gap-3 px-3 text-left outline-none hover:bg-on-surface/8 focus-visible:bg-on-surface/12 active:bg-on-surface/12',
		checked && 'bg-secondary-container text-on-secondary-container hover:bg-secondary-container',
		danger && 'text-error'
	]}
>
	{#if leading}
		<span class="grid size-6 place-items-center">{@render leading()}</span>
	{:else if icon}
		<Icon name={icon} class={['size-6', danger ? 'text-error' : 'text-on-surface-variant']} />
	{/if}
	<span class="flex-1 truncate">{label}</span>
	{#if keys}<kbd class="text-label-medium hidden font-sans text-on-surface-variant wide:inline">{keys}</kbd>{/if}
</button>
