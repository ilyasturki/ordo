<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import {
		NO_FILTER,
		PRIORITIES,
		PRIORITY_LABELS,
		STATUSES,
		STATUS_LABELS,
		TAGS,
		tagName,
		type Filter,
		type Project
	} from '$lib/model';
	import TagMark from '$lib/TagMark.svelte';
	import { ordo } from '$lib/state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Sheet from '$lib/ui/Sheet.svelte';

	let { open = $bindable(false), project }: { open?: boolean; project: Project } = $props();

	const f = $derived(ordo.filterOf(project.id));

	function flip(key: keyof Filter, v: string) {
		const has = f[key].includes(v);
		ordo.setFilter(project.id, {
			...f,
			[key]: has ? f[key].filter((x) => x !== v) : [...f[key], v]
		});
	}

	const section = 'mt-5 mb-2 text-sm font-medium text-on-surface-variant';
	const chip =
		'state-layer inline-flex h-8 items-center gap-1.5 rounded-lg border px-3 text-sm font-medium';

	function chipClass(on: boolean) {
		return on
			? `${chip} border-transparent bg-secondary-container text-on-secondary-container`
			: `${chip} border-outline-variant text-on-surface-variant`;
	}
</script>

<Sheet bind:open title="Filter">
	<h3 class={section}>Status</h3>
	<div class="flex flex-wrap gap-2">
		{#each STATUSES as s (s)}
			<button class={chipClass(f.statuses.includes(s))} onclick={() => flip('statuses', s)}>
				{#if f.statuses.includes(s)}<Icon name="check" size={18} />{/if}{STATUS_LABELS[s]}
			</button>
		{/each}
	</div>
	<h3 class={section}>Priority</h3>
	<div class="flex flex-wrap gap-2">
		{#each PRIORITIES as p (p)}
			<button class={chipClass(f.priorities.includes(p))} onclick={() => flip('priorities', p)}>
				{#if f.priorities.includes(p)}<Icon name="check" size={18} />{/if}{PRIORITY_LABELS[p]}
			</button>
		{/each}
	</div>
	<h3 class={section}>Tag</h3>
	<div class="flex flex-wrap gap-2">
		{#each TAGS as t (t)}
			<button
				class={chipClass(f.tags.includes(t))}
				onclick={() => flip('tags', t)}
				aria-label={tagName(t)}
			>
				{#if t}<TagMark color={t} />{/if}{tagName(t)}
			</button>
		{/each}
	</div>
	<h3 class={section}>Category</h3>
	<div class="flex flex-wrap gap-2">
		{#each project.categories as c (c.id)}
			<button
				class={chipClass(f.categories.includes(c.id))}
				onclick={() => flip('categories', c.id)}
			>
				{#if f.categories.includes(c.id)}<Icon name="check" size={18} />{/if}{c.name}
			</button>
		{/each}
	</div>
	{#snippet actions()}
		<Button onclick={() => ordo.setFilter(project.id, NO_FILTER)}>Clear all</Button>
		<Button variant="tonal" onclick={() => (open = false)}>Done</Button>
	{/snippet}
</Sheet>
