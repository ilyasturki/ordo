<script lang="ts">
	import { page } from '$app/state';
	import type { Task } from '$lib/core/types';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';

	const local = session.local!;
	const project = $derived(local.project(page.params.pid ?? ''));
	let title = $state('');
	let message = $state('');

	const marks: Record<Task['status'], string> = { todo: '○', in_progress: '◐', completed: '●', cancelled: '⊘' };

	async function add(event: SubmitEvent) {
		event.preventDefault();
		if (!project) {
			return;
		}
		const cat = project.categories[0];
		const answer = await local.perform(
			project.id,
			cat ? { action: 'task.add', cat: cat.id, title } : { action: 'category.add', title }
		);
		message = answer.status === 'applied' ? '' : (answer.msg ?? '');
		if (answer.status === 'applied') {
			title = '';
		}
	}

	function download() {
		const raw = project && local.export(project.id);
		if (!raw) {
			return;
		}
		const link = document.createElement('a');
		link.href = URL.createObjectURL(new Blob([raw], { type: 'application/json' }));
		link.download = `${project.id}.json`;
		link.click();
		URL.revokeObjectURL(link.href);
	}
</script>

<header class="bg-surface sticky top-0 flex h-16 items-center gap-2 px-1">
	<a href={app()} aria-label="Projects" class="hover:bg-on-surface/8 grid size-12 place-items-center rounded-full">
		<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor" aria-hidden="true">
			<path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z" />
		</svg>
	</a>
	<h1 class="text-title-large flex-1 truncate">{project?.name ?? 'Gone'}</h1>
	{#if project}
		<button onclick={download} class="text-label-large text-primary hover:bg-primary/8 rounded-full px-3 py-2.5">Export</button>
	{/if}
</header>

<main class="mx-auto max-w-2xl px-4 pb-16">
	{#if !project}
		<p class="text-body-large text-on-surface-variant py-4">This project isn't in this browser.</p>
	{:else}
		{#each project.categories as category (category.id)}
			<section aria-label={category.name}>
				<h2 class="text-title-small text-primary flex min-h-12 items-center">
					{#if category.parent_id}<span class="text-on-surface-variant">›&nbsp;</span>{/if}{category.name}
				</h2>
				<ul>
					{#each category.tasks as task (task.id)}
						{#if task.kind === 'separator'}
							<li class="border-outline-variant my-2 border-t" aria-label="Separator"></li>
						{:else}
							<li class="flex min-h-12 items-center gap-3" class:pl-10={task.parent_id}>
								<span class="w-6 text-center" class:text-primary={task.status !== 'todo'} aria-label={task.kind === 'note' ? 'Note' : task.status}>
									{task.kind === 'note' ? '–' : marks[task.status]}
								</span>
								<span class="text-body-large" class:line-through={task.status === 'completed'}>{task.title}</span>
							</li>
						{/if}
					{/each}
				</ul>
			</section>
		{:else}
			<p class="text-body-large text-on-surface-variant py-4">Empty project: add a category to start.</p>
		{/each}

		<form onsubmit={add} class="mt-6 flex gap-2">
			<label class="sr-only" for="add">{project.categories.length ? 'Task title' : 'Category name'}</label>
			<input
				id="add"
				bind:value={title}
				placeholder={project.categories.length ? 'Add a task' : 'Add a category'}
				class="text-body-large border-outline focus:border-primary min-w-0 flex-1 rounded-xs border bg-transparent px-4 py-3 outline-none focus:border-2"
			/>
			<button class="text-label-large bg-primary text-on-primary rounded-full px-6">Add</button>
		</form>
		{#if message}
			<p role="alert" class="text-body-medium text-error mt-2">{message}</p>
		{/if}
	{/if}
</main>
