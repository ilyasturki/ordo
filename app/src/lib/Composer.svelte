<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import type { Board } from '$lib/board.svelte';

	let { board, cat, after }: { board: Board; cat: string; after: string } = $props();

	let title = $state('');
	let input = $state<HTMLInputElement | null>(null);
	let busy = false;

	$effect(() => {
		input?.focus();
	});

	async function add(keepOpen: boolean) {
		const text = title.trim();
		const current = board.adding?.cat === cat && board.adding.after === after;
		if (busy || !current) {
			return;
		}
		if (text === '') {
			if (!keepOpen) {
				board.adding = null;
			}
			return;
		}
		busy = true;
		const answer = await board.edit(
			{ action: 'task.insert', cat, ids: after ? [after] : [], title: text },
			{ quiet: true }
		);
		busy = false;
		if (answer.status !== 'applied') {
			return;
		}
		title = '';
		board.adding = keepOpen && answer.row ? { cat, after: answer.row } : null;
	}
</script>

<form
	class="flex items-center pr-3 pl-1"
	onsubmit={(e) => {
		e.preventDefault();
		void add(true);
	}}
>
	<span class="grid size-12 shrink-0 place-items-center text-on-surface-variant"
		><Icon name="todo" /></span
	>
	<input
		bind:this={input}
		bind:value={title}
		maxlength="500"
		autocomplete="off"
		enterkeyhint="enter"
		placeholder="Title"
		aria-label="New task"
		class="h-12 min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-on-surface-variant"
		onblur={() => void add(false)}
		onkeydown={(e) => {
			if (e.key === 'Escape') {
				title = '';
				board.adding = null;
			}
		}}
	/>
</form>
