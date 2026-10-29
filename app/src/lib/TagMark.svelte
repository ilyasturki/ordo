<script lang="ts">
	import { canonicalTag, tagName } from '$lib/model';

	let { color }: { color: string } = $props();

	const hue = $derived(canonicalTag(color));
	const fill = $derived(
		{ green: 'bg-tag-green', blue: 'bg-tag-blue', magenta: 'bg-tag-magenta', cyan: 'bg-tag-cyan' }[
			hue
		] ?? 'bg-outline'
	);
</script>

<span
	role="img"
	aria-label="{tagName(color)} tag"
	class="inline-block size-2.5 shrink-0 rounded-full {fill}"
></span>
