<script lang="ts">
	import { slide } from 'svelte/transition';
	import Composer from '$lib/Composer.svelte';
	import Icon from '$lib/Icon.svelte';
	import { formatEstimate, type CategoryView } from '$lib/model';
	import TaskRow from '$lib/TaskRow.svelte';
	import type { Board } from '$lib/board.svelte';
	import { ordo } from '$lib/state.svelte';
	import IconButton from '$lib/ui/IconButton.svelte';
	import Menu, { type MenuItem } from '$lib/ui/Menu.svelte';
	import { ripple } from '$lib/ui/ripple';

	let {
		view,
		board,
		menu,
		narrowed
	}: { view: CategoryView; board: Board; menu: MenuItem[]; narrowed: boolean } = $props();

	const cat = $derived(view.cat);
	const folded = $derived(!narrowed && ordo.prefs.folded[cat.id] === true);
	const doneOpen = $derived(narrowed || ordo.prefs.doneOpen[cat.id] === true);
	const adding = $derived(board.adding?.cat === cat.id ? board.adding : null);
	const est = $derived(formatEstimate(cat.estimate_minutes));
	const hasOpenTask = $derived(view.open.some((r) => r.task.kind !== 'separator'));

	function fold() {
		ordo.prefs.folded[cat.id] = !folded;
		ordo.savePrefs();
	}

	function toggleDone() {
		ordo.prefs.doneOpen[cat.id] = !doneOpen;
		ordo.savePrefs();
	}

	function add() {
		if (folded) {
			fold();
		}
		board.clear();
		board.adding = { cat: cat.id, after: '' };
	}
</script>

<section
	class="overflow-hidden rounded-3xl bg-surface-low {cat.parent_id ? 'ml-4' : ''}"
	data-cat={cat.id}
>
	<header class="flex items-center gap-0.5 py-1 pr-1 pl-4">
		<button
			class="flex min-h-12 min-w-0 flex-1 items-center gap-2 text-left"
			onclick={fold}
			aria-expanded={!folded}
		>
			<span class="flex min-w-0 flex-col">
				{#if view.parentName}
					<span
						class="text-[11px] leading-4 font-medium tracking-wider text-on-surface-variant uppercase"
						>{view.parentName}</span
					>
				{/if}
				<span class="truncate text-lg leading-6 font-medium text-on-surface">{cat.name}</span>
			</span>
			{#if est}<span class="shrink-0 text-sm text-on-surface-variant">~{est}</span>{/if}
			<Icon
				name="expand_more"
				size={20}
				class="shrink-0 text-on-surface-variant transition-transform {folded ? '-rotate-90' : ''}"
			/>
		</button>
		<IconButton icon="add" label="Add a task to {cat.name}" onclick={add} />
		<Menu label="More for {cat.name}" items={menu} />
	</header>

	{#if !folded}
		<div transition:slide={{ duration: 180 }}>
			{#if adding && adding.after === ''}
				<Composer {board} cat={cat.id} after="" />
			{/if}
			{#each view.open as row (row.task.id)}
				<TaskRow {row} {board} />
				{#if adding && adding.after === row.task.id}
					<Composer {board} cat={cat.id} after={row.task.id} />
				{/if}
			{/each}
			{#if !hasOpenTask && !adding}
				<p class="px-4 pt-1 pb-4 text-sm text-on-surface-variant">
					{view.completed.length > 0 ? 'All tasks complete' : 'No tasks yet'}
				</p>
			{/if}
			{#if view.completed.length > 0}
				<button
					use:ripple
					class="state-layer flex h-12 w-full items-center gap-3 px-4 text-left text-sm font-medium text-on-surface-variant"
					onclick={toggleDone}
					aria-expanded={doneOpen}
				>
					<Icon
						name="expand_more"
						size={20}
						class="transition-transform {doneOpen ? '' : '-rotate-90'}"
					/>
					Completed ({view.completed.length})
				</button>
				{#if doneOpen}
					<div transition:slide={{ duration: 180 }}>
						{#each view.completed as row (row.task.id)}
							<TaskRow {row} {board} />
						{/each}
					</div>
				{/if}
			{/if}
			<div class="h-2"></div>
		</div>
	{/if}
</section>
