<script lang="ts">
	import type { Snippet } from 'svelte';
	import Sheet from './Sheet.svelte';

	// A sheet around one name. Enter, Esc and a click outside all keep what was typed; empty text adds nothing.
	let {
		open,
		label,
		initial = '',
		placeholder,
		action,
		blankAllowed = false,
		onDone,
		onclose,
		extra
	}: {
		open: boolean;
		label: string;
		initial?: string;
		placeholder: string;
		action: string;
		blankAllowed?: boolean;
		onDone: (name: string) => Promise<boolean>;
		onclose: () => void;
		extra?: Snippet;
	} = $props();

	let text = $state('');
	let input: HTMLInputElement | undefined = $state();
	let busy = false;

	$effect(() => {
		if (open) {
			text = initial;
			queueMicrotask(() => input?.select());
		}
	});

	// Leaving a new row's sheet empty adds nothing; Enter adds a blank one where blank is allowed.
	async function done(leaving: boolean) {
		const name = text.replace(/\s+/g, ' ').trim();
		if (busy) return;
		if (name === initial.trim() && (initial.trim() !== '' || leaving || !blankAllowed)) {
			onclose();
			return;
		}
		if (!blankAllowed && name === '') {
			onclose();
			return;
		}
		busy = true;
		try {
			if (await onDone(name)) onclose();
		} finally {
			busy = false;
		}
	}
</script>

<Sheet {open} {label} onclose={() => done(true)}>
	<form
		class="flex items-center gap-2 py-2 pr-2 pl-4"
		onsubmit={(e) => {
			e.preventDefault();
			void done(false);
		}}
	>
		<input
			bind:this={input}
			bind:value={text}
			{placeholder}
			aria-label={placeholder}
			enterkeyhint="done"
			autocapitalize="sentences"
			class="text-title-medium min-w-0 flex-1 bg-transparent py-3 outline-none placeholder:text-on-surface-variant"
		/>
		<button
			class="text-label-large h-10 cursor-pointer rounded-full px-3 text-primary outline-primary hover:bg-primary/8 focus-visible:outline-2 disabled:cursor-default disabled:text-on-surface/38 disabled:hover:bg-transparent"
			disabled={!blankAllowed && text.trim() === ''}
		>
			{action}
		</button>
	</form>
	{@render extra?.()}
</Sheet>
