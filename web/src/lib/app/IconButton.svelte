<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Icon from './Icon.svelte';
	import type { IconName } from './icons';

	let {
		icon,
		label,
		tone = 'muted',
		class: cls = '',
		el = $bindable(),
		...rest
	}: { icon: IconName; label: string; tone?: 'muted' | 'plain' | 'primary' | 'error'; el?: HTMLButtonElement } & HTMLButtonAttributes = $props();
</script>

<button
	bind:this={el}
	type="button"
	aria-label={label}
	title={label}
	class={[
		'grid size-12 shrink-0 cursor-pointer place-items-center rounded-full outline-primary focus-visible:outline-2 disabled:cursor-default disabled:opacity-40 wide:size-10',
		'hover:bg-on-surface/8 active:bg-on-surface/12',
		{ muted: 'text-on-surface-variant', plain: 'text-on-surface', primary: 'text-primary', error: 'text-error' }[tone],
		cls
	]}
	{...rest}
>
	<Icon name={icon} />
</button>
