<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Icon from '$lib/Icon.svelte';
	import { ripple } from '$lib/ui/ripple';

	type Props = HTMLButtonAttributes & { icon: string; label: string };

	let { icon, label, class: cls = '', ...rest }: Props = $props();
</script>

<button
	use:ripple
	aria-label={label}
	class="state-layer grid size-12 shrink-0 place-items-center rounded-full text-on-surface-variant disabled:opacity-40 {cls}"
	{...rest}
>
	<Icon name={icon} />
</button>
