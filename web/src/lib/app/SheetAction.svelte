<script lang="ts">
	import Icon from './Icon.svelte';
	import type { IconName } from './icons';

	let { icon, label, danger = false, onclick }: { icon: IconName; label: string; danger?: boolean; onclick: () => void } = $props();
</script>

<button
	type="button"
	{onclick}
	class={[
		'text-body-large flex h-14 w-full cursor-pointer items-center gap-4 px-4 text-left outline-none hover:bg-on-surface/8 focus-visible:bg-on-surface/12 active:bg-on-surface/12',
		danger ? 'text-error' : 'text-on-surface'
	]}
>
	<Icon name={icon} class={['size-6', danger ? 'text-error' : 'text-on-surface-variant']} />
	{label}
</button>
