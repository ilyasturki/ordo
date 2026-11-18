<script lang="ts">
	import type { Local } from '$lib/local.svelte';

	let { local }: { local: Local } = $props();

	const snack = $derived(local.snack);

	$effect(() => {
		const shown = snack;
		if (!shown) return;
		const timer = setTimeout(() => {
			if (local.snack === shown) local.snack = undefined;
		}, 6000);
		return () => clearTimeout(timer);
	});
</script>

<div class="pointer-events-none fixed inset-x-0 bottom-0 z-40 flex justify-center px-4 pb-[max(1rem,env(safe-area-inset-bottom))] wide:justify-start wide:pl-6">
	{#if snack}
		{#key snack.id}
			<div
				role="status"
				class="snack text-body-medium pointer-events-auto flex min-h-12 w-full max-w-xl items-center gap-2 rounded-xs bg-inverse-surface py-1 pr-2 pl-4 text-inverse-on-surface shadow-lg wide:w-auto wide:min-w-80"
			>
				<span class="flex-1 py-2">{snack.msg}</span>
				{#if snack.action}
					{@const action = snack.action}
					<button
						type="button"
						class="text-label-large h-10 cursor-pointer rounded-full px-3 text-inverse-primary hover:bg-inverse-primary/8"
						onclick={() => {
							local.snack = undefined;
							action.run();
						}}
					>
						{action.label}
					</button>
				{/if}
			</div>
		{/key}
	{/if}
</div>

<style>
	@media (prefers-reduced-motion: no-preference) {
		.snack {
			transition: translate 150ms ease-out;
			@starting-style {
				translate: 0 1rem;
			}
		}
	}
</style>
