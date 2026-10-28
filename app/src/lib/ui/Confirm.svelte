<script lang="ts" module>
	export type ConfirmSpec = {
		title: string;
		body: string;
		confirm: string;
		run: () => void;
	};
</script>

<script lang="ts">
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';

	let { spec = $bindable(null) }: { spec: ConfirmSpec | null } = $props();

	let open = $state(false);

	$effect(() => {
		if (spec !== null) {
			open = true;
		}
	});

	$effect(() => {
		if (!open) {
			spec = null;
		}
	});

	function go() {
		spec?.run();
		open = false;
	}
</script>

<Dialog bind:open title={spec?.title ?? ''}>
	{spec?.body}
	{#snippet actions()}
		<Button onclick={() => (open = false)}>Cancel</Button>
		<Button variant="danger" onclick={go}>{spec?.confirm}</Button>
	{/snippet}
</Dialog>
