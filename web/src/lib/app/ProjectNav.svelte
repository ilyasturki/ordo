<script lang="ts">
	import type { Local } from '$lib/local.svelte';
	import { groupKey } from '$lib/model/project';
	import { app, site } from '$lib/paths';
	import { importProject } from './actions';
	import Icon from './Icon.svelte';
	import { ui } from './ui.svelte';

	// The wide screen's list of projects, where the phone has its tabs: each group together, under its name.
	let { local, pid }: { local: Local; pid: string } = $props();

	const grouped = $derived(local.projects.some((p) => groupKey(p) !== ''));
</script>

<nav aria-label="Projects" class="flex h-full flex-col overflow-y-auto px-3 pt-3 pb-4">
	<p class="text-title-medium flex h-14 items-center gap-3 px-4">
		<img src="/icon.svg" alt="" class="size-6" />Ordo
	</p>
	<ul class="flex flex-col">
		{#each local.projects as p, i (p.id)}
			{@const group = groupKey(p)}
			{#if grouped && (i === 0 || groupKey(local.projects[i - 1]) !== group)}
				<li class={['text-title-small px-4 pb-1 text-on-surface-variant', i > 0 && 'mt-2 border-t border-outline-variant pt-3']}>
					{group === '' ? 'Other' : (p.group ?? '').trim()}
				</li>
			{/if}
			<li>
				<a
					href={app(`/p/${p.id}`)}
					aria-current={p.id === pid ? 'page' : undefined}
					class={[
						'text-label-large flex h-12 items-center rounded-full px-4 outline-primary focus-visible:outline-2',
						p.id === pid ? 'bg-secondary-container text-on-secondary-container' : 'text-on-surface-variant hover:bg-on-surface/8'
					]}
				>
					<span class="truncate">{p.name}</span>
				</a>
			</li>
		{/each}
	</ul>
	<div class="mt-2 flex flex-col border-t border-outline-variant pt-2">
		<button
			type="button"
			class="text-label-large flex h-12 cursor-pointer items-center gap-3 rounded-full px-4 text-on-surface-variant outline-primary hover:bg-on-surface/8 focus-visible:outline-2"
			onclick={() => (ui.sheet = { kind: 'newProject' })}
		>
			<Icon name="add" />New project
		</button>
		<button
			type="button"
			class="text-label-large flex h-12 cursor-pointer items-center gap-3 rounded-full px-4 text-on-surface-variant outline-primary hover:bg-on-surface/8 focus-visible:outline-2"
			onclick={() => importProject(local)}
		>
			<Icon name="upload" />Import a project
		</button>
		{#if local.synced || site}
			<a
				href={app(local.synced ? '/devices' : '/signin')}
				class="text-label-large flex h-12 items-center gap-3 rounded-full px-4 text-on-surface-variant outline-primary hover:bg-on-surface/8 focus-visible:outline-2"
			>
				<Icon name={local.synced ? 'devices' : 'sync'} />{local.synced ? 'Devices' : 'Connect to ordo serve'}
			</a>
		{/if}
	</div>
</nav>
