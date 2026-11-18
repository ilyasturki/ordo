<script lang="ts">
	import Sheet from './Sheet.svelte';
	import TextButton from './TextButton.svelte';
	import { keyHelp } from './keys';

	let { onclose }: { onclose: () => void } = $props();
</script>

<Sheet open label="Keyboard shortcuts" onclose={() => onclose()}>
	<div class="px-6 pt-5 pb-3">
		<h2 class="text-title-large">Keys</h2>
		<p class="text-body-medium mt-1 text-on-surface-variant">The terminal's, on the list.</p>
		<div class="mt-2 grid gap-x-8 wide:grid-cols-2">
			{#each keyHelp as [section, keys] (section)}
				<section>
					<h3 class="text-label-large pt-4 pb-1 text-primary">{section}</h3>
					<dl class="text-body-medium grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5">
						{#each keys as [key, what] (key)}
							<dt><kbd class="text-label-large font-sans whitespace-nowrap">{key}</kbd></dt>
							<dd class="text-on-surface-variant">{what}</dd>
						{/each}
					</dl>
				</section>
			{/each}
		</div>
		<div class="mt-4 flex justify-end"><TextButton onclick={onclose}>Close</TextButton></div>
	</div>
</Sheet>
