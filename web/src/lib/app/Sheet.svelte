<script lang="ts">
	import type { Snippet } from 'svelte';

	// A bottom sheet on a phone, a dialog on a wide screen. Esc and a click outside both call onclose:
	// a sheet holding what the user typed saves then, as Enter does.
	let {
		open,
		label,
		onclose,
		children
	}: { open: boolean; label: string; onclose: (how: 'esc' | 'outside') => void; children: Snippet } = $props();

	let dialog: HTMLDialogElement;

	$effect(() => {
		if (open && !dialog.open) dialog.showModal();
		else if (!open && dialog.open) dialog.close();
	});

	function cancel(e: Event) {
		e.preventDefault();
		onclose('esc');
	}

	function click(e: MouseEvent) {
		if (e.target === dialog) onclose('outside');
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog
	bind:this={dialog}
	aria-label={label}
	oncancel={cancel}
	onclick={click}
	class="sheet m-0 mt-auto w-full max-w-none overflow-visible bg-transparent p-0 text-on-surface backdrop:bg-scrim/32 wide:m-auto wide:w-[min(32rem,calc(100vw-2rem))]"
>
	<div class="max-h-[85dvh] overflow-y-auto rounded-t-xl bg-surface-container-low pb-[env(safe-area-inset-bottom)] wide:rounded-xl">
		{@render children()}
	</div>
</dialog>

<style>
	@media (prefers-reduced-motion: no-preference) {
		.sheet[open] {
			transition: translate 200ms cubic-bezier(0.05, 0.7, 0.1, 1);
			@starting-style {
				translate: 0 100%;
			}
		}
		@media (width >= 52.5rem) {
			.sheet[open] {
				transition: none;
			}
		}
	}
</style>
