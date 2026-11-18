<script lang="ts">
	import { app } from '$lib/paths';
</script>

<svelte:head>
	<title>Ordo</title>
	<meta name="description" content="Plan projects from the terminal, the browser and your phone." />
</svelte:head>

<main class="mx-auto flex min-h-dvh max-w-2xl flex-col justify-center gap-6 px-4 py-16">
	<img src="/icon.svg" alt="" width="56" height="56" />
	<h1 class="text-display-small">Ordo</h1>
	<p class="text-body-large text-on-surface-variant">Plan projects from the terminal, the browser and your phone.</p>
	<a
		href={app()}
		class="text-label-large bg-primary text-on-primary self-start rounded-full px-6 py-2.5 hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
	>
		Open in the browser
	</a>
</main>
