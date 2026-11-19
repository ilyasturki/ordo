<script lang="ts">
	import { tagDot, tags } from '#lib/model/project.ts';

	let { color, class: cls = 'size-2.5' }: { color: string; class?: string } = $props();
	const dot = $derived(tagDot(color));
</script>

<span
	class={['inline-block shrink-0 rounded-full', cls]}
	style:background={tags.includes(dot) ? `var(--color-mark-${dot})` : 'var(--color-outline)'}
></span>
