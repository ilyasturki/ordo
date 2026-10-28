<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import {
		PRIORITY_LABELS,
		STATUS_LABELS,
		formatEstimate,
		isSeparator,
		type Row
	} from '$lib/model';
	import StatusIcon from '$lib/StatusIcon.svelte';
	import TagMark from '$lib/TagMark.svelte';
	import type { Board } from '$lib/board.svelte';
	import { hold } from '$lib/ui/hold';
	import { ripple } from '$lib/ui/ripple';
	import TaskRow from './TaskRow.svelte';

	let {
		row,
		board,
		cat,
		child = false
	}: { row: Row; board: Board; cat: string; child?: boolean } = $props();

	const t = $derived(row.task);
	const selected = $derived(board.selected.has(t.id));
	const done = $derived(t.status === 'completed' || t.status === 'cancelled');
	const pri = $derived(t.priority && t.priority !== 'medium' ? t.priority : '');
	const est = $derived(formatEstimate(t.estimate_minutes));
	const meta = $derived(pri !== '' || est !== '' || row.childTotal > 0 || row.images > 0);
	const drag = $derived(board.drag?.moved ? board.drag : null);
	const lifted = $derived(drag?.id === t.id ? drag : null);
	const gapBefore = $derived(drag !== null && drag.spot.cat === cat && drag.spot.before === t.id);
	const gapInside = $derived(
		drag !== null && drag.spot.cat === cat && drag.spot.parent === t.id && drag.spot.before === ''
	);

	function cycle() {
		void board.edit({ action: 'status.cycle', ids: [t.id] });
	}

	function open(e: MouseEvent) {
		if (board.selecting) {
			e.preventDefault();
			board.toggle(t.id);
		}
	}
</script>

{#if gapBefore && drag}
	<div
		class="mx-2 rounded-2xl bg-on-surface/5 {drag.spot.parent ? 'ml-10' : ''}"
		style:height="{drag.height}px"
	></div>
{/if}
<div
	data-block={t.id}
	class={lifted ? 'fixed z-50 touch-none rounded-2xl bg-surface-highest shadow-xl' : ''}
	style:top={lifted ? `${lifted.top}px` : undefined}
	style:left={lifted ? `${lifted.left}px` : undefined}
	style:width={lifted ? `${lifted.width}px` : undefined}
	style:transform={lifted
		? `translateX(${(lifted.spot.parent ? 36 : 0) - (child ? 36 : 0)}px)`
		: undefined}
	style:transition={lifted ? 'transform 120ms' : undefined}
>
	{#if isSeparator(t)}
		<div
			class="flex min-h-10 items-center gap-3 py-1 pr-4 {child ? 'pl-14' : 'pl-4'} {selected
				? 'bg-secondary-container'
				: ''}"
			data-row={t.id}
			data-sep
			data-parent={t.parent_id ?? ''}
			use:hold={(e) => board.press(t.id, e)}
			role="button"
			tabindex="-1"
			onclick={() => board.selecting && board.toggle(t.id)}
			onkeydown={(e) => e.key === 'Enter' && board.selecting && board.toggle(t.id)}
		>
			{#if t.title}<span class="text-xs font-medium tracking-wide text-on-surface-variant uppercase"
					>{t.title}</span
				>{/if}
			<span class="h-px flex-1 bg-outline-variant"></span>
		</div>
	{:else}
		<div
			class="flex items-start {child ? 'pl-10' : 'pl-1'} pr-3 {selected
				? 'bg-secondary-container'
				: ''}"
			data-row={t.id}
			data-parent={t.parent_id ?? ''}
		>
			{#if board.selecting}
				<button
					class="grid size-12 shrink-0 place-items-center"
					aria-label={selected ? 'Deselect' : 'Select'}
					onclick={() => board.toggle(t.id)}
				>
					<span
						class="grid size-5 place-items-center rounded-[2px] border-2 {selected
							? 'border-primary bg-primary text-on-primary'
							: 'border-on-surface-variant'}"
					>
						{#if selected}<Icon name="check" size={16} />{/if}
					</span>
				</button>
			{:else}
				<button
					use:ripple
					class="state-layer grid size-12 shrink-0 place-items-center rounded-full"
					aria-label="{STATUS_LABELS[t.status] ?? t.status}, change status"
					onclick={cycle}
				>
					<StatusIcon status={t.status} />
				</button>
			{/if}
			<a
				href="/p/{board.pid()}/t/{t.id}"
				class="flex min-h-12 min-w-0 flex-1 flex-col justify-center py-3 pr-1"
				use:hold={(e) => board.press(t.id, e)}
				onclick={open}
				oncontextmenu={(e) => e.preventDefault()}
				draggable="false"
			>
				<span
					class="text-base leading-6 break-words {done
						? 'text-on-surface-variant line-through decoration-on-surface-variant/60'
						: 'text-on-surface'}">{t.title}</span
				>
				{#if row.preview && !done}
					<span class="line-clamp-2 text-sm leading-5 whitespace-pre-line text-on-surface-variant"
						>{row.preview}</span
					>
				{/if}
				{#if meta && !done}
					<span class="mt-1.5 flex flex-wrap gap-1.5">
						{#if pri}
							<span
								class="inline-flex h-6 items-center gap-1 rounded-md border border-outline-variant px-1.5 text-xs font-medium {pri ===
									'critical' || pri === 'high'
									? 'text-pri-up'
									: 'text-pri-down'}"
								><Icon name="priority_{pri}" size={14} />{PRIORITY_LABELS[pri]}</span
							>
						{/if}
						{#if est}
							<span
								class="inline-flex h-6 items-center gap-1 rounded-md border border-outline-variant px-1.5 text-xs font-medium text-on-surface-variant"
								><Icon name="schedule" size={14} />{est}</span
							>
						{/if}
						{#if row.childTotal > 0}
							<span
								class="inline-flex h-6 items-center gap-1 rounded-md border border-outline-variant px-1.5 text-xs font-medium text-on-surface-variant"
								><Icon name="subtask" size={14} />{row.childDone}/{row.childTotal}</span
							>
						{/if}
						{#if row.images > 0}
							<span
								class="inline-flex h-6 items-center gap-1 rounded-md border border-outline-variant px-1.5 text-xs font-medium text-on-surface-variant"
								><Icon name="image" size={14} />{row.images}</span
							>
						{/if}
					</span>
				{/if}
			</a>
			{#if t.tag_color}
				<span class="flex h-12 items-center pl-2"><TagMark color={t.tag_color} /></span>
			{/if}
		</div>
	{/if}
	{#each row.children as c (c.task.id)}
		<TaskRow row={c} {board} {cat} child />
	{/each}
	{#if gapInside && drag}
		<div class="mx-2 ml-10 rounded-2xl bg-on-surface/5" style:height="{drag.height}px"></div>
	{/if}
</div>
