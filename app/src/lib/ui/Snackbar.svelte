<script lang="ts">
	import { fly } from 'svelte/transition';
	import { ordo } from '$lib/state.svelte';

	const snack = $derived(ordo.snack);

	$effect(() => {
		if (snack === null) {
			return;
		}
		const timer = setTimeout(
			() => {
				if (ordo.snack?.key === snack.key) {
					ordo.snack = null;
				}
			},
			snack.action ? 6000 : 4000
		);
		return () => clearTimeout(timer);
	});

	function act() {
		const run = snack?.action?.run;
		ordo.snack = null;
		run?.();
	}
</script>

{#key snack?.key}
	{#if snack}
		<div
			class="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex justify-center px-4 pb-[calc(var(--safe-b)+16px)]"
		>
			<div
				role="status"
				transition:fly={{ y: 24, duration: 200 }}
				class="pointer-events-auto flex min-h-12 w-full max-w-xl items-center gap-2 rounded-sm bg-inverse-surface py-1 pr-2 pl-4 text-sm text-inverse-on-surface shadow-lg"
			>
				<span class="flex-1 py-2">{snack.msg}</span>
				{#if snack.action}
					<button
						class="state-layer h-10 rounded-full px-3 text-sm font-medium text-inverse-primary"
						onclick={act}>{snack.action.label}</button
					>
				{/if}
			</div>
		</div>
	{/if}
{/key}
