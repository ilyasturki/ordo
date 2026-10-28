<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Icon from '$lib/Icon.svelte';
	import { ripple } from '$lib/ui/ripple';

	type Props = HTMLButtonAttributes & {
		variant?: 'filled' | 'tonal' | 'text' | 'outlined' | 'danger';
		icon?: string;
		children: Snippet;
	};

	let { variant = 'text', icon, children, class: cls = '', ...rest }: Props = $props();

	const paint = {
		filled: 'bg-primary text-on-primary px-6',
		tonal: 'bg-secondary-container text-on-secondary-container px-6',
		text: 'text-primary px-3',
		outlined: 'border border-outline text-primary px-6',
		danger: 'text-error px-3'
	};
</script>

<button
	use:ripple
	class="state-layer inline-flex h-10 items-center justify-center gap-2 rounded-full text-sm font-medium tracking-wide disabled:opacity-40 {paint[
		variant
	]} {icon ? 'pl-4' : ''} {cls}"
	{...rest}
>
	{#if icon}<Icon name={icon} size={18} />{/if}
	{@render children()}
</button>
