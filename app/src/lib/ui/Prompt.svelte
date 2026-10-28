<script lang="ts" module>
	export type PromptSpec = {
		title: string;
		label: string;
		value?: string;
		confirm: string;
		/** Returns false to keep the dialog open. */
		submit: (value: string) => boolean | void | Promise<boolean | void>;
		allowEmpty?: boolean;
	};
</script>

<script lang="ts">
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';

	let { spec = $bindable(null) }: { spec: PromptSpec | null } = $props();

	let open = $state(false);
	let value = $state('');
	let input = $state<HTMLInputElement | null>(null);

	$effect(() => {
		if (spec !== null) {
			value = spec.value ?? '';
			open = true;
		}
	});

	$effect(() => {
		if (!open) {
			spec = null;
		}
	});

	$effect(() => {
		if (open && input !== null) {
			input.focus();
			input.select();
		}
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (spec === null || (!spec.allowEmpty && value.trim() === '')) {
			return;
		}
		if ((await spec.submit(value.trim())) !== false) {
			open = false;
		}
	}
</script>

<Dialog bind:open title={spec?.title ?? ''}>
	<form id="prompt-form" onsubmit={submit}>
		<label class="relative block">
			<input
				bind:this={input}
				bind:value
				maxlength="500"
				autocomplete="off"
				enterkeyhint="done"
				placeholder=" "
				class="peer h-14 w-full rounded-sm border border-outline bg-transparent px-4 pt-1 text-base text-on-surface outline-none focus:border-2 focus:border-primary"
			/>
			<span
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 bg-surface-high px-1 text-base text-on-surface-variant transition-all peer-focus:top-0 peer-focus:text-xs peer-focus:text-primary peer-[:not(:placeholder-shown)]:top-0 peer-[:not(:placeholder-shown)]:text-xs"
				>{spec?.label}</span
			>
		</label>
	</form>
	{#snippet actions()}
		<Button onclick={() => (open = false)}>Cancel</Button>
		<Button type="submit" form="prompt-form" disabled={!spec?.allowEmpty && value.trim() === ''}
			>{spec?.confirm}</Button
		>
	{/snippet}
</Dialog>
