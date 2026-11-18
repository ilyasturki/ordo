<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		danger = false,
		filled = false,
		children,
		class: cls = '',
		...rest
	}: { danger?: boolean; filled?: boolean; children: Snippet } & HTMLButtonAttributes = $props();
</script>

<button
	type="button"
	class={[
		'text-label-large h-10 cursor-pointer rounded-full px-3 outline-primary focus-visible:outline-2 disabled:cursor-default disabled:text-on-surface/38',
		filled
			? 'bg-primary px-6 text-on-primary hover:shadow-md'
			: danger
				? 'text-error hover:bg-error/8 active:bg-error/12'
				: 'text-primary hover:bg-primary/8 active:bg-primary/12',
		cls
	]}
	{...rest}
>
	{@render children()}
</button>
