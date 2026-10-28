<script lang="ts">
	import { ICONS } from '$lib/icons';

	let {
		name,
		size = 24,
		class: cls = ''
	}: { name: string; size?: number; class?: string } = $props();
</script>

<svg
	class="shrink-0 fill-current {cls}"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	aria-hidden="true"
>
	<path d={ICONS[name] ?? ICONS.todo} />
</svg>
