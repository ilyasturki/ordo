<script lang="ts">
	import Icon from '$lib/Icon.svelte';

	let { status, size = 24 }: { status: string; size?: number } = $props();

	const tone = $derived(
		status === 'completed'
			? 'text-primary'
			: status === 'in_progress'
				? 'text-primary'
				: 'text-on-surface-variant'
	);
</script>

<Icon name={status} {size} class={tone} />
