<script lang="ts">
	import type { Project } from '$lib/core/types';
	import type { Local } from '$lib/local.svelte';
	import type { Line } from '$lib/model/board';
	import { active } from '$lib/model/filter';
	import { nextStatus } from '$lib/model/project';
	import { app } from '$lib/paths';
	import CategoryHeader from './CategoryHeader.svelte';
	import TaskRow from './TaskRow.svelte';
	import { hold, openTask, ui } from './ui.svelte';

	let { local, project, lines }: { local: Local; project: Project; lines: Line[] } = $props();

	const pid = $derived(project.id);
	const searching = $derived(ui.query.trim() !== '');

	function cycle(id: string, status: string) {
		ui.cursor = 't:' + id;
		if (active(local.view(pid).filter)) hold(id);
		// The step shows in place; done keeps its Undo, since the row may leave the view.
		void local.perform(pid, { action: 'status.cycle', ids: [id] }, { quiet: nextStatus(status) !== 'completed' });
	}

	function open(e: MouseEvent, id: string) {
		if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
		e.preventDefault();
		void openTask(pid, id);
	}

	function fold(id: string, folded: boolean) {
		void local.setFolded(pid, id, !folded);
	}
</script>

{#if project.categories.length === 0}
	<div class="flex flex-col items-center gap-4 px-6 py-24 text-center">
		<p class="text-body-large text-on-surface-variant">No categories yet</p>
		<button
			type="button"
			class="text-label-large h-10 cursor-pointer rounded-full bg-secondary-container px-6 text-on-secondary-container hover:shadow-sm"
			onclick={() => (ui.sheet = { kind: 'newCategory', pid, parent: '' })}
		>
			New category
		</button>
		<p class="text-body-medium hidden text-on-surface-variant wide:block"><kbd>A</kbd> adds one, <kbd>?</kbd> shows the keys</p>
	</div>
{:else if searching && lines.length === 0}
	<p class="text-body-large px-6 py-24 text-center text-on-surface-variant">Nothing matches</p>
{:else}
	<div role="list" aria-label={project.name} class="pb-28 wide:pb-24">
		{#each lines as line (line.key)}
			<div role="listitem">
				{#if line.kind === 'header'}
					<CategoryHeader
						header={line}
						selected={ui.cursor === line.key}
						onclick={() => {
							ui.cursor = line.key;
							ui.sheet = { kind: 'category', pid, cat: line.cat };
						}}
						onFold={searching ? undefined : () => fold(line.cat, line.folded)}
					/>
				{:else if line.kind === 'block'}
					<TaskRow
						row={line.row}
						href={app(`/p/${pid}/t/${line.row.task.id}`)}
						selected={ui.cursor === line.key}
						onCycle={() => cycle(line.row.task.id, line.row.task.status)}
						onOpen={(e) => open(e, line.row.task.id)}
						onFold={searching ? undefined : () => fold(line.row.task.id, line.row.folded)}
					/>
					{#each line.children as c (c.task.id)}
						<TaskRow
							row={c}
							child
							href={app(`/p/${pid}/t/${c.task.id}`)}
							selected={ui.cursor === 't:' + c.task.id}
							onCycle={() => cycle(c.task.id, c.task.status)}
							onOpen={(e) => open(e, c.task.id)}
						/>
					{/each}
				{:else if line.kind === 'hidden'}
					<button
						type="button"
						data-key={line.key}
						class={[
							'text-label-large mx-2 block w-[calc(100%-1rem)] cursor-pointer rounded-lg py-3 pr-4 pl-12 text-left text-primary outline-primary hover:bg-primary/8 focus-visible:outline-2',
							ui.cursor === line.key && 'bg-secondary-container'
						]}
						onclick={() => (ui.peek = line.peeking ? undefined : line.cat)}
					>
						{line.peeking ? `Hide the ${line.count} filtered out` : `${line.count} hidden by the filter`}
					</button>
				{:else}
					<p class="text-body-medium pt-1 pr-4 pb-3 pl-14 text-on-surface-variant">No tasks</p>
				{/if}
			</div>
		{/each}
	</div>
{/if}
