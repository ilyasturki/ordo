<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	// An M3 outlined text field: its label above the box, the error under it.
	let {
		label,
		value = $bindable(''),
		error,
		el = $bindable(),
		...rest
	}: { label: string; value?: string; error?: string; el?: HTMLInputElement } & HTMLInputAttributes = $props();

	const id = $props.id();
</script>

<div class="flex flex-col gap-1">
	<label for={id} class={['text-body-small px-1', error ? 'text-error' : 'text-on-surface-variant']}>{label}</label>
	<input
		{id}
		bind:this={el}
		bind:value
		aria-invalid={error ? true : undefined}
		aria-describedby={error ? `${id}-error` : undefined}
		autocomplete="off"
		autocapitalize="off"
		spellcheck="false"
		class={[
			'text-body-large h-14 w-full rounded-xs border bg-transparent px-4 outline-none placeholder:text-on-surface-variant',
			error ? 'border-error focus:border-2 focus:px-[15px]' : 'border-outline focus:border-2 focus:border-primary focus:px-[15px]'
		]}
		{...rest}
	/>
	{#if error}<p id={`${id}-error`} class="text-body-small px-4 text-error">{error}</p>{/if}
</div>
