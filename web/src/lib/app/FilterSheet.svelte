<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Local } from '#lib/local.svelte.ts';
	import { active, flip, hasValues, noFilter, type Filter } from '#lib/model/filter.ts';
	import { categoryPath, priorities, priorityLabel, priorityTone, statusLabel, statuses, tagLabel, tags } from '#lib/model/project.ts';
	import Icon from './Icon.svelte';
	import Sheet from './Sheet.svelte';
	import TagDot from './TagDot.svelte';
	import TextButton from './TextButton.svelte';
	import { priorityIcon, statusIcon } from './icons';

	// A change shows at once; Esc or a click outside only closes, as a filter keeps Esc for cancel.
	let { local, pid, onclose }: { local: Local; pid: string; onclose: () => void } = $props();

	const project = $derived(local.project(pid)!);
	const filter = $derived(local.view(pid).filter);

	function set(next: Filter) {
		void local.setFilter(pid, next);
	}
</script>

{#snippet toggle(label: string, on: boolean, flipIt: () => void, leading?: Snippet)}
	<button
		type="button"
		aria-pressed={on}
		onclick={flipIt}
		class={[
			'text-label-large flex h-8 max-w-full cursor-pointer items-center gap-2 rounded-sm border px-3 outline-primary focus-visible:outline-2',
			on ? 'border-transparent bg-secondary-container text-on-secondary-container' : 'border-outline-variant text-on-surface-variant hover:bg-on-surface/8',
			(leading || on) && 'pl-2'
		]}
	>
		{#if leading}{@render leading()}{:else if on}<Icon name="check" class="size-[18px]" />{/if}
		<span class="truncate">{label}</span>
	</button>
{/snippet}

{#snippet section(title: string, body: Snippet)}
	<h3 class="text-label-large pt-4 pb-2 text-on-surface-variant">{title}</h3>
	<div class="flex flex-wrap gap-2">{@render body()}</div>
{/snippet}

<Sheet open label="Filter" onclose={() => onclose()}>
	<div class="px-5 pt-4 pb-5">
		<div class="flex items-center gap-2">
			<h2 class="text-title-large flex-1">Filter</h2>
			<label class="relative flex items-center">
				<input
					type="checkbox"
					role="switch"
					aria-label="Filter on"
					checked={active(filter)}
					disabled={!hasValues(filter)}
					onchange={() => set({ ...filter, off: !filter.off })}
					class="peer absolute inset-0 z-10 cursor-pointer opacity-0 disabled:cursor-default"
				/>
				<span
					aria-hidden="true"
					class="relative h-8 w-13 cursor-pointer rounded-full border-2 border-outline bg-surface-container-highest transition-colors peer-checked:border-primary peer-checked:bg-primary peer-disabled:cursor-default peer-disabled:opacity-38 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary after:absolute after:top-1/2 after:left-1.5 after:size-4 after:-translate-y-1/2 after:rounded-full after:bg-outline after:transition-all peer-checked:after:left-6 peer-checked:after:size-6 peer-checked:after:bg-on-primary"
				></span>
			</label>
			<TextButton disabled={!hasValues(filter)} onclick={() => set(noFilter)}>Clear</TextButton>
		</div>

		{#snippet statusChips()}
			{#each statuses as s (s)}
				{#snippet mark()}<Icon name={statusIcon(s)} class="size-[18px]" />{/snippet}
				{@render toggle(statusLabel(s), filter.statuses.includes(s), () => set({ ...filter, off: false, statuses: flip(filter.statuses, s) }), mark)}
			{/each}
		{/snippet}
		{@render section('Status', statusChips)}

		{#snippet priorityChips()}
			{#each priorities as p (p)}
				{#snippet mark()}<Icon
						name={priorityIcon(p)}
						class={['size-[18px]', { up: 'text-mark-up', down: 'text-mark-down', '': '' }[priorityTone(p)]]}
					/>{/snippet}
				{@render toggle(priorityLabel(p), filter.priorities.includes(p), () => set({ ...filter, off: false, priorities: flip(filter.priorities, p) }), mark)}
			{/each}
		{/snippet}
		{@render section('Priority', priorityChips)}

		{#if project.categories.length}
			{#snippet categoryChips()}
				{#each project.categories as c (c.id)}
					{@render toggle(categoryPath(project, c), filter.categories.includes(c.id), () =>
						set({ ...filter, off: false, categories: flip(filter.categories, c.id) })
					)}
				{/each}
			{/snippet}
			{@render section('Category', categoryChips)}
		{/if}

		{#snippet tagChips()}
			{#each [...tags, ''] as t (t)}
				{#snippet mark()}<TagDot color={t} class="mx-1 size-2.5" />{/snippet}
				{@render toggle(t === '' ? 'Untagged' : tagLabel(t), filter.tags.includes(t), () => set({ ...filter, off: false, tags: flip(filter.tags, t) }), t ? mark : undefined)}
			{/each}
		{/snippet}
		{@render section('Tag', tagChips)}
	</div>
</Sheet>
