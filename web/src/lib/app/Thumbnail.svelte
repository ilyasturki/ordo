<script lang="ts">
	import type { Local } from '$lib/local.svelte';
	import type { ImageRef } from '$lib/model/description';
	import Icon from './Icon.svelte';

	// A reference this browser holds no file for shows its name: never a request that fails.
	let { local, image, onOpen, onRemove }: { local: Local; image: ImageRef; onOpen: (url: string) => void; onRemove: () => void } = $props();

	let url: string | undefined = $state();
	let looked = $state(false);

	$effect(() => {
		const file = image.file;
		looked = false;
		url = undefined;
		if (!file) {
			looked = true;
			return;
		}
		void local.imageURL(file).then((u) => {
			url = u;
			looked = true;
		});
	});
</script>

<div class="group/thumb relative size-24 shrink-0 overflow-hidden rounded-md bg-surface-container-highest">
	{#if url}
		<button type="button" class="size-full cursor-zoom-in" aria-label={`Open ${image.name || 'image'}`} onclick={() => onOpen(url!)}>
			<img src={url} alt={image.name} class="size-full object-cover" />
		</button>
	{:else if looked}
		<div class="text-label-small flex size-full flex-col items-center justify-center gap-1 p-2 text-center text-on-surface-variant" title="Not in this browser">
			<Icon name="image" class="size-5" />
			<span class="line-clamp-2 break-all">{image.name || 'image'}</span>
		</div>
	{/if}
	<button
		type="button"
		aria-label={`Remove ${image.name || 'image'}`}
		title="Remove"
		class="absolute top-1 right-1 grid size-7 cursor-pointer place-items-center rounded-full bg-scrim/60 text-white"
		onclick={onRemove}
	>
		<Icon name="close" class="size-4" />
	</button>
</div>
