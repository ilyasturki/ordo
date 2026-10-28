<script lang="ts">
	import '../app.css';
	import { onMount, type Snippet } from 'svelte';
	import { App } from '@capacitor/app';
	import { page } from '$app/state';
	import SignIn from '$lib/SignIn.svelte';
	import { ordo } from '$lib/state.svelte';
	import { wireNative } from '$lib/ui/native';
	import Snackbar from '$lib/ui/Snackbar.svelte';

	let { children }: { children: Snippet } = $props();

	onMount(() => {
		void ordo.boot().then(() => ordo.listen(ordo.signedIn));
		const unwire = wireNative();
		const wake = () => {
			const visible = document.visibilityState === 'visible';
			ordo.listen(visible && ordo.signedIn);
			if (visible) {
				void ordo.sync();
			}
		};
		document.addEventListener('visibilitychange', wake);
		window.addEventListener('online', wake);
		const resume = import.meta.env.APP_BUILD ? App.addListener('resume', wake) : undefined;
		return () => {
			unwire();
			document.removeEventListener('visibilitychange', wake);
			window.removeEventListener('online', wake);
			void resume?.then((h) => h.remove());
		};
	});

	const pairing = $derived(page.url.pathname.startsWith('/pair/'));
</script>

{#if ordo.booted}
	{#if ordo.signedIn || pairing}
		{@render children()}
	{:else}
		<SignIn />
	{/if}
{/if}
<Snackbar />
