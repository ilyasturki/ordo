<script lang="ts">
	import { bump, KeyReader, type Part } from './bump';

	let version = $state('v1.3.0');
	let message = $state('');
	const reader = new KeyReader();

	function press(part: Part, delta: number) {
		const next = bump(version, part, delta);
		if (next instanceof Error) {
			message = next.message;
			return;
		}
		version = next;
		message = '';
	}

	function onkeydown(event: KeyboardEvent) {
		const hit = reader.read(event);
		if (hit) {
			event.preventDefault();
			press(...hit);
		}
	}

	const keys: { label: string; hint: string; part: Part; delta: number }[] = [
		{ label: 'ctrl+a', hint: 'patch', part: 2, delta: 1 },
		{ label: 'ctrl+x', hint: 'back one', part: 2, delta: -1 },
		{ label: 'gm', hint: 'minor', part: 1, delta: 1 },
		{ label: 'gM', hint: 'major', part: 0, delta: 1 }
	];
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
<div
	class="bump tty"
	role="application"
	tabindex="0"
	aria-label="Version bump demo. Focus it and press ctrl+a, ctrl+x, g m or g M."
	{onkeydown}
>
	<pre aria-hidden="true">  <b>▼ <span class="v" data-testid="bump-version">{version}</span> - polish</b> <span class="t-blue">[/]</span>
  <span class="t-blue">[/]</span> <span class="t-red">▲ Encrypt secrets with age</span>
  [ ] Per-host overrides <span class="t-dim">¶</span>

  <b>▶ v1.2.1</b> <span class="t-dim">[x]</span>
  <b>▶ v1.2.0</b> <span class="t-dim">[x]</span></pre>
	<p class="sr-only" aria-live="polite">{version} - polish</p>
	<p class="msg" role="status">{message}</p>
	<div class="keys">
		{#each keys as key (key.label)}
			<button type="button" class="key" onclick={() => press(key.part, key.delta)}>{key.label} <small>{key.hint}</small></button>
		{/each}
	</div>
</div>

<style>
	.bump {
		border-radius: 16px;
		padding: 22px 22px 20px;
		font-size: 15px;
		line-height: 1.7;
		min-width: 0;
	}
	.bump:focus-visible {
		outline: 2px solid var(--color-primary);
		outline-offset: 4px;
	}
	pre {
		margin: 0;
		font: inherit;
		white-space: pre;
		overflow: hidden;
	}
	.msg {
		height: 1.7em;
		color: var(--t-muted);
		font-size: 13px;
		margin-top: 6px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.keys {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 14px;
	}
	.key {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 40px;
		padding: 0 14px;
		border-radius: 10px;
		border: 1px solid color-mix(in srgb, var(--t-fg) 30%, transparent);
		background: transparent;
		color: var(--t-fg);
		font: 600 14px/1 var(--font-code);
		cursor: pointer;
	}
	.key small {
		font: 400 13px/1 var(--font-sans);
		color: var(--t-muted);
	}
	@media (hover: hover) {
		.key:hover {
			background: color-mix(in srgb, var(--t-fg) 8%, transparent);
		}
	}
	.key:active {
		background: color-mix(in srgb, var(--t-fg) 14%, transparent);
	}
	.key:focus-visible {
		outline: 2px solid var(--color-primary);
		outline-offset: 2px;
	}
	@media (max-width: 719px) {
		.bump {
			font-size: 14px;
			padding: 18px 16px;
		}
	}
</style>
