<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import SignIn from '$lib/SignIn.svelte';
	import { ordo } from '$lib/state.svelte';

	$effect(() => {
		if (ordo.signedIn) {
			void goto('/', { replaceState: true });
		}
	});
</script>

{#if !ordo.signedIn}
	<SignIn code={`${page.url.origin}/pair/${page.params.code}`} />
{/if}
