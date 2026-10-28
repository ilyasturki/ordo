<script lang="ts" module>
	export type PickerKind = 'status' | 'priority' | 'tag' | 'estimate';
</script>

<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import {
		ESTIMATE_PRESETS,
		PRIORITIES,
		PRIORITY_LABELS,
		STATUSES,
		STATUS_LABELS,
		TAGS,
		formatEstimate,
		tagName
	} from '$lib/model';
	import TagMark from '$lib/TagMark.svelte';

	type Props = {
		kind: PickerKind;
		/** The current value, absent for a selection of several. */
		value?: string;
		pick: (value: string) => void;
	};

	let { kind, value, pick }: Props = $props();

	const chip =
		'state-layer inline-flex h-8 items-center gap-1.5 rounded-lg border px-3 text-sm font-medium';

	function chipClass(on: boolean) {
		return on
			? `${chip} border-transparent bg-secondary-container text-on-secondary-container`
			: `${chip} border-outline-variant text-on-surface-variant`;
	}
</script>

{#if kind === 'status'}
	<div
		class="flex overflow-hidden rounded-full border border-outline"
		role="group"
		aria-label="Status"
	>
		{#each STATUSES as s, i (s)}
			<button
				class="state-layer flex h-10 flex-1 items-center justify-center gap-1 text-sm font-medium {i >
				0
					? 'border-l border-outline'
					: ''} {value === s
					? 'bg-secondary-container text-on-secondary-container'
					: 'text-on-surface'}"
				aria-pressed={value === s}
				onclick={() => pick(s)}
			>
				{#if value === s}<Icon name="check" size={18} />{/if}{STATUS_LABELS[s]}
			</button>
		{/each}
	</div>
{:else if kind === 'priority'}
	<div class="flex flex-wrap gap-2" role="group" aria-label="Priority">
		{#each PRIORITIES as p (p)}
			<button class={chipClass(value === p)} aria-pressed={value === p} onclick={() => pick(p)}>
				<Icon
					name={value === p ? 'check' : p === '' ? 'priority_none' : `priority_${p}`}
					size={18}
					class={value === p
						? ''
						: p === 'critical' || p === 'high'
							? 'text-pri-up'
							: p === 'low' || p === 'trivial'
								? 'text-pri-down'
								: ''}
				/>{PRIORITY_LABELS[p]}
			</button>
		{/each}
	</div>
{:else if kind === 'tag'}
	<div class="flex flex-wrap gap-2" role="group" aria-label="Tag">
		{#each TAGS as tag (tag)}
			<button
				class="state-layer grid size-10 place-items-center rounded-full border {value === tag
					? 'border-2 border-primary'
					: 'border-outline-variant'}"
				aria-label={tagName(tag)}
				aria-pressed={value === tag}
				onclick={() => pick(tag)}
			>
				{#if tag}<TagMark color={tag} />{:else}<Icon
						name="close"
						size={18}
						class="text-on-surface-variant"
					/>{/if}
			</button>
		{/each}
	</div>
{:else}
	<div class="flex flex-wrap gap-2" role="group" aria-label="Estimate">
		{#each ESTIMATE_PRESETS as m (m)}
			<button
				class={chipClass(value === String(m))}
				aria-pressed={value === String(m)}
				onclick={() => pick(String(m))}
			>
				{#if value === String(m)}<Icon name="check" size={18} />{/if}{m === 0
					? 'None'
					: formatEstimate(m)}
			</button>
		{/each}
	</div>
{/if}
