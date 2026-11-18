<script lang="ts">
	import type { ClassValue } from 'svelte/elements';
	import { icons, type IconName } from './icons';

	let { name, class: cls = 'size-6' }: { name: IconName; class?: ClassValue } = $props();
</script>

<svg viewBox="0 0 24 24" class={['shrink-0 fill-current', cls]} aria-hidden="true"><path d={icons[name]} /></svg>
