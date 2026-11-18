<script lang="ts">
	import type { Project } from '$lib/core/types';
	import type { Local } from '$lib/local.svelte';
	import type { Block, Line } from '$lib/model/board';
	import { dropPlace, entered, placeOp, slotAllowed } from '$lib/model/drag';
	import { active } from '$lib/model/filter';
	import { nextStatus } from '$lib/model/project';
	import { app } from '$lib/paths';
	import CategoryHeader from './CategoryHeader.svelte';
	import TaskRow from './TaskRow.svelte';
	import { Reorder } from './reorder.svelte';
	import { hold, openTask, ui } from './ui.svelte';

	let { local, project, lines, top }: { local: Local; project: Project; lines: Line[]; top?: () => number } = $props();

	const pid = $derived(project.id);
	const searching = $derived(ui.query.trim() !== '');

	const drag = new Reorder();
	let list: HTMLElement | undefined = $state();
	let lifted: Line[] = $state.raw([]);
	// A drop's order stays on screen until its edit lands.
	let landing: Line[] | undefined = $state.raw();
	const shown = $derived(drag.key ? lifted : (landing ?? lines));
	const blank = $derived.by(() => {
		if (!drag.key) return entered(shown);
		const at = new Map(lifted.map((l) => [l.key, l]));
		return entered(drag.order().map((k) => at.get(k)!));
	});

	function press(e: PointerEvent, block: Block) {
		if (searching || !list) return;
		const from = shown;
		const others = from.filter((l) => l.key !== block.key);
		drag.press(e, {
			key: block.key,
			items: [...list.children] as HTMLElement[],
			allowed: (slot) => slotAllowed(others, slot),
			top,
			drop: (order) => {
				const at = new Map(from.map((l) => [l.key, l]));
				void land(order.map((k) => at.get(k)!), block, from);
			}
		});
		lifted = from;
	}

	async function land(order: Line[], block: Block, from: Line[]) {
		const drop = dropPlace(order, block, from);
		if (!drop) return;
		landing = order;
		ui.cursor = block.key;
		if (active(local.view(pid).filter)) hold(drop.id);
		const answer = await local.perform(pid, placeOp(drop), { quiet: true });
		if (answer.status === 'applied' && drop.unfold) await local.setFolded(pid, drop.cat, false);
		else if (answer.status === 'refused') local.say(answer.msg ?? '');
		if (landing === order) landing = undefined;
	}

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
	<div
		bind:this={list}
		{@attach drag.surface}
		role="list"
		aria-label={project.name}
		class={['pb-28 select-none [-webkit-touch-callout:none] wide:pb-24', drag.key && 'cursor-grabbing [&_*]:cursor-grabbing']}
	>
		{#each shown as line (line.key)}
			<div
				role="listitem"
				data-line={line.key}
				class={drag.look(line.key)}
				style:transform={drag.transform(line.key)}
				onpointerdown={line.kind === 'block' ? (e) => press(e, line) : undefined}
			>
				{#if drag.key === line.key}<span class="absolute inset-x-2 inset-y-0 -z-10 rounded-lg bg-surface-container-high shadow-lg"></span>{/if}
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
					<p class={['text-body-medium pt-1 pr-4 pb-3 pl-14 text-on-surface-variant', blank.has(line.key) && 'invisible']}>No tasks</p>
				{/if}
			</div>
		{/each}
	</div>
{/if}
