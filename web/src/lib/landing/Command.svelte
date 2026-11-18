<script lang="ts">
	import Icon from './Icon.svelte';

	let { command, label }: { command: string; label: string } = $props();

	let code: HTMLElement;
	let copied = $state(false);

	async function copy() {
		try {
			await navigator.clipboard.writeText(command);
		} catch {
			const selection = getSelection();
			selection?.selectAllChildren(code);
			return;
		}
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}
</script>

<div class="cmd">
	<code bind:this={code}>{command}</code>
	<button type="button" class="copy" aria-label={copied ? 'Copied' : label} onclick={copy}>
		<Icon name={copied ? 'check' : 'copy'} class={copied ? 'ok' : ''} />
	</button>
</div>

<style>
	.cmd {
		display: flex;
		align-items: center;
		background: var(--color-surface-container-lowest);
		border-radius: 12px;
		padding: 4px 4px 4px 16px;
		min-width: 0;
	}
	code {
		flex: 1;
		min-width: 0;
		overflow-x: auto;
		white-space: nowrap;
		font: 13.5px var(--font-code);
		padding: 8px 0;
		scrollbar-width: none;
	}
	code::-webkit-scrollbar {
		display: none;
	}
	.copy {
		width: 40px;
		height: 40px;
		flex: none;
		display: grid;
		place-items: center;
		border: 0;
		border-radius: 20px;
		background: transparent;
		color: var(--color-on-surface-variant);
		cursor: pointer;
	}
	.copy :global(svg) {
		width: 20px;
		height: 20px;
	}
	.copy :global(.ok) {
		color: var(--color-primary);
	}
	@media (hover: hover) {
		.copy:hover {
			background: var(--color-surface-container-high);
			color: var(--color-on-surface);
		}
	}
	.copy:active {
		background: var(--color-surface-container-highest);
	}
	.copy:focus-visible {
		outline: 2px solid var(--color-primary);
		outline-offset: 2px;
	}
	@media (max-width: 719px) {
		code {
			font-size: 12.5px;
		}
	}
</style>
