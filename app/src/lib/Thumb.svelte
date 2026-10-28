<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import type { ImageRef } from '$lib/model';
	import { ordo } from '$lib/state.svelte';

	let { image, remove }: { image: ImageRef; remove?: () => void } = $props();

	let url = $state<string | null>(null);
	let missing = $state(false);

	$effect(() => {
		const file = image.file;
		if (file === undefined) {
			missing = true;
			return;
		}
		let live = true;
		let made: string | null = null;
		void ordo.image(file).then((blob) => {
			if (!live) {
				return;
			}
			if (blob === null) {
				missing = true;
				return;
			}
			made = URL.createObjectURL(blob);
			url = made;
		});
		return () => {
			live = false;
			if (made !== null) {
				URL.revokeObjectURL(made);
			}
		};
	});
</script>

<figure class="relative overflow-hidden rounded-xl bg-surface-high">
	{#if url}
		<a href={url} target="_blank" rel="noreferrer">
			<img src={url} alt={image.name} class="max-h-80 w-full object-contain" />
		</a>
	{:else}
		<div class="flex h-24 items-center gap-2 px-4 text-sm text-on-surface-variant">
			<Icon name="image" />
			<span class="truncate">{image.name || 'image'}{missing ? ' (not synced yet)' : ''}</span>
		</div>
	{/if}
	{#if remove}
		<button
			class="absolute top-2 right-2 grid size-8 place-items-center rounded-full bg-inverse-surface/70 text-inverse-on-surface"
			aria-label="Remove {image.name}"
			onclick={remove}
		>
			<Icon name="close" size={18} />
		</button>
	{/if}
</figure>
