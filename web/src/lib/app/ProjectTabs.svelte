<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Local } from '#lib/local.svelte.ts';
	import { groupKey } from '#lib/model/project.ts';
	import { app } from '#lib/paths.ts';
	import Icon from './Icon.svelte';
	import { ui } from './ui.svelte';

	// The phone's row of projects: a tap on the open one opens its sheet, the + adds one.
	let { local, pid }: { local: Local; pid: string } = $props();

	let row: HTMLElement;
	const grouped = $derived(local.projects.some((p) => groupKey(p) !== ''));

	$effect(() => {
		row.querySelector('[aria-current="page"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
	});

	function tap(e: MouseEvent, id: string) {
		if (e.metaKey || e.ctrlKey || e.shiftKey) return;
		e.preventDefault();
		if (id === pid) ui.sheet = { kind: 'project', pid };
		else void goto(app(`/p/${id}`), { replace: true });
	}
</script>

<nav bind:this={row} aria-label="Projects" class="flex overflow-x-auto px-1 [scrollbar-width:none]">
	{#each local.projects as p, i (p.id)}
		<a
			href={app(`/p/${p.id}`)}
			aria-current={p.id === pid ? 'page' : undefined}
			onclick={(e) => tap(e, p.id)}
			class={[
				'text-title-medium relative flex h-12 shrink-0 items-center px-4 whitespace-nowrap outline-primary focus-visible:outline-2 focus-visible:-outline-offset-2',
				p.id === pid ? 'text-primary' : 'text-on-surface-variant',
				grouped &&
					i > 0 &&
					groupKey(local.projects[i - 1]) !== groupKey(p) &&
					'before:absolute before:top-3.5 before:left-0 before:h-5 before:w-px before:bg-outline-variant'
			]}
		>
			{p.name}
			{#if p.id === pid}<span class="absolute inset-x-4 bottom-0 h-[3px] rounded-t-[3px] bg-primary"></span>{/if}
		</a>
	{/each}
	<button
		type="button"
		aria-label="New project"
		class="grid h-12 shrink-0 cursor-pointer place-items-center px-4 text-on-surface-variant"
		onclick={() => (ui.sheet = { kind: 'newProject' })}
	>
		<Icon name="add" />
	</button>
</nav>
