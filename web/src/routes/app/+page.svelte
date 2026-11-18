<script lang="ts">
	import { goto } from '$app/navigation';
	import { importProject } from '$lib/app/actions';
	import Sheets from '$lib/app/Sheets.svelte';
	import { ui } from '$lib/app/ui.svelte';
	import { app, site } from '$lib/paths';
	import { session } from '$lib/session.svelte';

	const local = session.local!;

	$effect(() => {
		const next = local.project(local.current) ?? local.projects[0];
		if (next) void goto(app(`/p/${next.id}`), { replaceState: true });
	});
</script>

{#if local.projects.length === 0}
	<main class="flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
		<img src="/icon.svg" alt="" width="48" height="48" />
		<h1 class="text-headline-small">No projects yet</h1>
		<p class="text-body-medium max-w-80 text-on-surface-variant">
			{local.synced ? 'They sync with your ordo serve.' : 'They stay in this browser.'} A project file from ordo opens here too.
		</p>
		<div class="mt-2 flex flex-wrap justify-center gap-2">
			<button
				type="button"
				class="text-label-large h-10 cursor-pointer rounded-full bg-primary px-6 text-on-primary outline-primary hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2"
				onclick={() => (ui.sheet = { kind: 'newProject' })}
			>
				New project
			</button>
			<button
				type="button"
				class="text-label-large h-10 cursor-pointer rounded-full border border-outline px-6 text-primary outline-primary hover:bg-primary/8 focus-visible:outline-2"
				onclick={() => importProject(local)}
			>
				Import a project
			</button>
			{#if site && !local.synced}
				<a
					href={app('/signin')}
					class="text-label-large flex h-10 items-center rounded-full px-4 text-primary outline-primary hover:bg-primary/8 focus-visible:outline-2"
				>
					Connect to ordo serve
				</a>
			{/if}
		</div>
		{#each local.unreadable as u (u.id)}
			<p role="alert" class="text-body-small text-error">Can't open a stored project ({u.id}): {u.error}</p>
		{/each}
	</main>
	<Sheets {local} />
{/if}
