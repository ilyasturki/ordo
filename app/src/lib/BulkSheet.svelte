<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import type { Project } from '$lib/model';
	import Pickers from '$lib/Pickers.svelte';
	import type { Board } from '$lib/board.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Sheet from '$lib/ui/Sheet.svelte';

	let {
		open = $bindable(false),
		board,
		project,
		remove
	}: { open?: boolean; board: Board; project: Project; remove: () => void } = $props();

	const n = $derived(board.selected.size);

	function set(action: string, value: string) {
		open = false;
		void board.onSelected({ action, value });
	}

	function structure(action: string, extra: { up?: boolean } = {}) {
		void board.onSelected({ action, ...extra }, { keep: true });
	}

	function cat(): string {
		const id = [...board.selected][0];
		return project.categories.find((c) => c.tasks.some((t) => t.id === id))?.id ?? '';
	}

	function move(cat: string) {
		open = false;
		void board.onSelected({ action: 'task.move', cat });
	}

	const section = 'mt-5 mb-2 text-sm font-medium text-on-surface-variant';
	const item =
		'state-layer flex h-12 w-full items-center gap-4 rounded-full px-4 text-left text-sm';
</script>

<Sheet bind:open title="Edit {n} selected">
	<h3 class={section}>Status</h3>
	<Pickers kind="status" pick={(v) => set('status.set', v)} />
	<h3 class={section}>Priority</h3>
	<Pickers kind="priority" pick={(v) => set('priority.set', v)} />
	<h3 class={section}>Tag</h3>
	<Pickers kind="tag" pick={(v) => set('tag.set', v)} />
	<h3 class={section}>Estimate</h3>
	<Pickers kind="estimate" pick={(v) => set('estimate.set', v)} />
	<h3 class={section}>Structure</h3>
	<button class={item} onclick={() => structure('task.shift', { up: true })}
		><Icon name="arrow_upward" class="text-on-surface-variant" />Move up</button
	>
	<button class={item} onclick={() => structure('task.shift')}
		><Icon name="arrow_downward" class="text-on-surface-variant" />Move down</button
	>
	<button class={item} onclick={() => structure('task.nest')}
		><Icon name="indent" class="text-on-surface-variant" />Nest under the task above</button
	>
	<button class={item} onclick={() => structure('task.unnest')}
		><Icon name="outdent" class="text-on-surface-variant" />Un-nest</button
	>
	{#if n === 1}
		<button
			class={item}
			onclick={() => {
				open = false;
				void board.edit({
					action: 'separator.add',
					ids: [...board.selected],
					cat: cat(),
					title: ''
				});
				board.clear();
			}}><Icon name="separator" class="text-on-surface-variant" />Separator below</button
		>
	{/if}
	<h3 class={section}>Move to</h3>
	{#each project.categories as c (c.id)}
		<button class="{item} {c.parent_id ? 'pl-10' : ''}" onclick={() => move(c.id)}
			><Icon name="move" class="text-on-surface-variant" />{c.name}</button
		>
	{/each}
	{#snippet actions()}
		<Button
			variant="danger"
			icon="delete"
			onclick={() => {
				open = false;
				remove();
			}}>Delete</Button
		>
		<Button variant="tonal" onclick={() => (open = false)}>Close</Button>
	{/snippet}
</Sheet>
