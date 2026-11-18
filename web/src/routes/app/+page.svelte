<script lang="ts">
	import { goto } from '$app/navigation';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';

	const local = session.local!;
	let name = $state('');
	let message = $state('');

	async function create(event: SubmitEvent) {
		event.preventDefault();
		const answer = await local.perform('', { action: 'project.create', title: name });
		message = answer.status === 'applied' ? '' : (answer.msg ?? '');
		if (answer.status === 'applied' && answer.row) {
			name = '';
			await goto(app(`/p/${answer.row}`));
		}
	}

	async function pick(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) {
			return;
		}
		try {
			await goto(app(`/p/${await local.import(await file.text())}`));
		} catch (error) {
			message = `Can't import ${file.name}: ${String(error)}`;
		}
	}
</script>

<header class="bg-surface sticky top-0 flex h-16 items-center px-4">
	<h1 class="text-title-large">Projects</h1>
</header>

<main class="mx-auto max-w-2xl px-4 pb-16">
	<ul aria-label="Projects">
		{#each local.projects as project (project.id)}
			<li>
				<a href={app(`/p/${project.id}`)} class="text-body-large hover:bg-surface-container-high flex min-h-14 items-center rounded-md px-4">
					{project.name}
				</a>
			</li>
		{:else}
			<li class="text-body-large text-on-surface-variant px-4 py-4">No projects yet.</li>
		{/each}
	</ul>

	<form onsubmit={create} class="mt-6 flex gap-2">
		<label class="sr-only" for="new-project">Project name</label>
		<input
			id="new-project"
			bind:value={name}
			placeholder="New project"
			class="text-body-large border-outline focus:border-primary min-w-0 flex-1 rounded-xs border bg-transparent px-4 py-3 outline-none focus:border-2"
		/>
		<button class="text-label-large bg-primary text-on-primary rounded-full px-6">Create</button>
	</form>

	<label class="text-label-large text-primary mt-4 inline-flex cursor-pointer items-center rounded-full px-3 py-2.5 hover:bg-primary/8">
		Import a project file
		<input type="file" accept=".json,application/json" class="sr-only" onchange={pick} />
	</label>

	{#if message}
		<p role="alert" class="text-body-medium text-error mt-2">{message}</p>
	{/if}
</main>
