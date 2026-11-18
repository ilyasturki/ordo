<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Icon from '$lib/app/Icon.svelte';
	import IconButton from '$lib/app/IconButton.svelte';
	import { ui, wide } from '$lib/app/ui.svelte';
	import type { Category } from '$lib/core/types';
	import { categoryVisible } from '$lib/model/filter';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';

	// The phone's categories page: each category with its menu, and moves where the phone drags.
	const local = session.local!;
	const pid = $derived(page.params.pid ?? '');
	const project = $derived(local.project(pid));
	const tops = $derived(project ? project.categories.filter((c) => !c.parent_id) : []);

	function subs(cat: Category): Category[] {
		return project ? project.categories.filter((c) => c.parent_id === cat.id) : [];
	}

	/** Moves a category one place among its siblings, by naming the sibling it lands above. */
	function shift(cat: Category, siblings: Category[], up: boolean) {
		const at = siblings.findIndex((c) => c.id === cat.id);
		const to = at + (up ? -1 : 1);
		if (to < 0 || to >= siblings.length) return;
		const before = up ? siblings[to].id : (siblings[to + 1]?.id ?? '');
		void local.perform(pid, { action: 'category.place', cat: cat.id, before }, { quiet: true });
	}

	function go(cat: Category) {
		if (!categoryVisible(local.view(pid).filter, cat)) {
			local.say(`“${cat.name}” is hidden by the filter`);
			return;
		}
		ui.jump = cat.id;
		if (!wide()) void goto(app(`/p/${pid}`), { noScroll: true });
	}

	function back() {
		void goto(app(`/p/${pid}`), { noScroll: true });
	}
</script>

{#snippet line(cat: Category, siblings: Category[], child: boolean)}
	{@const at = siblings.findIndex((c) => c.id === cat.id)}
	<div class={['flex min-h-14 items-center rounded-lg pr-1 hover:bg-on-surface/8', child ? 'pl-10' : 'pl-2']}>
		<button
			type="button"
			class={['min-w-0 flex-1 cursor-pointer truncate rounded-lg px-3 py-3 text-left outline-primary focus-visible:outline-2', child ? 'text-body-large' : 'text-title-medium']}
			onclick={() => go(cat)}
		>
			{cat.name}{#if cat.pinned}<Icon name="pin" class="ml-1 inline size-[1em] align-[-0.125em] text-on-surface-variant" />{/if}
		</button>
		<IconButton icon="arrowUp" label={`Move ${cat.name} up`} disabled={at === 0} onclick={() => shift(cat, siblings, true)} />
		<IconButton icon="arrowDown" label={`Move ${cat.name} down`} disabled={at === siblings.length - 1} onclick={() => shift(cat, siblings, false)} />
		<IconButton icon="more" label={`Edit ${cat.name}`} onclick={() => (ui.sheet = { kind: 'category', pid, cat: cat.id })} />
	</div>
{/snippet}

{#if project}
	<article aria-label="Categories" class="flex min-h-dvh flex-col bg-surface wide:h-full wide:min-h-0 wide:overflow-y-auto wide:rounded-xl wide:bg-surface-container-low">
		<header class="sticky top-0 z-10 flex h-16 shrink-0 items-center gap-1 bg-inherit px-1 wide:h-14 wide:px-2">
			<IconButton icon="back" label="Back" tone="plain" class="wide:hidden" onclick={back} />
			<h1 class="text-title-large min-w-0 flex-1 truncate px-3">{project.name} categories</h1>
			<IconButton icon="add" label="New category" onclick={() => (ui.sheet = { kind: 'newCategory', pid, parent: '' })} />
			<IconButton icon="close" label="Close" class="max-wide:hidden" onclick={back} />
		</header>
		<div class="px-2 pb-24">
			{#each tops as cat (cat.id)}
				{@render line(cat, tops, false)}
				{@const children = subs(cat)}
				{#each children as sub (sub.id)}
					{@render line(sub, children, true)}
				{/each}
			{:else}
				<p class="text-body-large px-4 py-16 text-center text-on-surface-variant">No categories yet</p>
			{/each}
		</div>
	</article>
{/if}
