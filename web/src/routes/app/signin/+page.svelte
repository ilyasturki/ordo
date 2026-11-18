<script lang="ts">
	import { goto } from '$app/navigation';
	import SignIn from '$lib/app/SignIn.svelte';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';
</script>

<svelte:head><title>Sign in · Ordo</title></svelte:head>

<SignIn onBack={session.local ? () => void goto(app()) : undefined} />
