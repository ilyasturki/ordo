<script lang="ts">
	import jsQR from 'jsqr';
	import { onDestroy } from 'svelte';
	import Sheet from '$lib/ui/Sheet.svelte';
	import Button from '$lib/ui/Button.svelte';

	let { open = $bindable(false), found }: { open?: boolean; found: (text: string) => void } =
		$props();

	let video = $state<HTMLVideoElement | null>(null);
	let error = $state('');
	let stream: MediaStream | null = null;
	let timer: ReturnType<typeof setInterval> | undefined;
	const canvas = document.createElement('canvas');

	function stop() {
		clearInterval(timer);
		stream?.getTracks().forEach((t) => t.stop());
		stream = null;
	}

	async function start(el: HTMLVideoElement) {
		error = '';
		try {
			stream = await navigator.mediaDevices.getUserMedia({
				video: { facingMode: 'environment' },
				audio: false
			});
		} catch (err) {
			error =
				err instanceof DOMException && err.name === 'NotAllowedError'
					? 'Ordo may not use the camera. Allow it in the app settings, or paste the link instead.'
					: 'No camera to scan with. Paste the link instead.';
			return;
		}
		el.srcObject = stream;
		await el.play().catch(() => undefined);
		const ctx = canvas.getContext('2d', { willReadFrequently: true });
		timer = setInterval(() => {
			if (ctx === null || el.videoWidth === 0) {
				return;
			}
			canvas.width = el.videoWidth;
			canvas.height = el.videoHeight;
			ctx.drawImage(el, 0, 0);
			const img = ctx.getImageData(0, 0, canvas.width, canvas.height);
			const code = jsQR(img.data, img.width, img.height, { inversionAttempts: 'dontInvert' });
			if (code?.data) {
				stop();
				open = false;
				found(code.data);
			}
		}, 150);
	}

	$effect(() => {
		if (open && video !== null) {
			void start(video);
			return stop;
		}
	});

	onDestroy(stop);
</script>

<Sheet bind:open title="Scan the pairing code">
	<p class="mb-3 text-sm text-on-surface-variant">
		Point the camera at the code <code>ordo serve pair</code> prints, or at the one Devices shows on a
		signed-in device.
	</p>
	{#if error}
		<p class="rounded-xl bg-error-container p-4 text-sm">{error}</p>
	{:else}
		<!-- svelte-ignore a11y_media_has_caption -->
		<video
			bind:this={video}
			playsinline
			muted
			class="aspect-square w-full rounded-2xl bg-black object-cover"
		></video>
	{/if}
	{#snippet actions()}
		<Button onclick={() => (open = false)}>Cancel</Button>
	{/snippet}
</Sheet>
