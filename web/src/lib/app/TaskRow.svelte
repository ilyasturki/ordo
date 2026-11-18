<script lang="ts">
	import type { Row } from '$lib/model/board';
	import { finished, isNote, isSeparator, priorityLabel, priorityOf, priorityTone, statusLabel } from '$lib/model/project';
	import Chip from './Chip.svelte';
	import IconButton from './IconButton.svelte';
	import TagDot from './TagDot.svelte';
	import { priorityIcon, statusIcon } from './icons';

	let {
		row,
		child = false,
		href,
		selected = false,
		onCycle,
		onOpen,
		onFold
	}: {
		row: Row;
		child?: boolean;
		href: string;
		selected?: boolean;
		onCycle: () => void;
		onOpen: (e: MouseEvent) => void;
		onFold?: () => void;
	} = $props();

	const t = $derived(row.task);
	const done = $derived(finished(t));
	const priority = $derived(priorityOf(t));
	const tag = $derived(t.tag_color ?? '');
</script>

<div
	data-key={'t:' + t.id}
	class={[
		'group relative mx-2 flex min-h-12 items-start rounded-lg',
		selected ? 'bg-secondary-container text-on-secondary-container' : 'hover:bg-on-surface/8',
		row.muted && 'opacity-50'
	]}
>
	{#if isSeparator(t)}
		<a
			{href}
			onclick={onOpen}
			aria-label={t.title || 'Separator'}
			class={['flex h-12 min-w-0 flex-1 items-center gap-2.5 rounded-lg pr-4 outline-primary focus-visible:outline-2', child ? 'pl-12' : 'pl-3']}
		>
			{#if t.title}
				<span class="h-[3px] w-5 rounded-full bg-on-surface"></span>
				<span class="text-title-small truncate font-bold">{t.title}</span>
			{/if}
			<span class="h-[3px] min-w-5 flex-1 rounded-full bg-on-surface"></span>
			{#if tag}<TagDot color={tag} class="ml-1 size-2.5" />{/if}
		</a>
	{:else if isNote(t)}
		<a {href} onclick={onOpen} class="flex min-h-12 min-w-0 flex-1 items-start rounded-lg pr-3 outline-primary focus-visible:outline-2">
			<span class="grid size-12 shrink-0 place-items-center" aria-label="Note"><span class="h-0.5 w-2.5 rounded-full bg-on-surface-variant"></span></span>
			<span class="text-body-medium min-w-0 flex-1 py-3.5 break-words whitespace-pre-line text-on-surface-variant">{t.title}</span>
			{#if tag}<span class="grid h-12 place-items-center"><TagDot color={tag} /></span>{/if}
		</a>
	{:else}
		<IconButton
			icon={statusIcon(t.status)}
			label={`${statusLabel(t.status)}, change status`}
			tone={t.status === 'in_progress' || t.status === 'completed' ? 'primary' : 'muted'}
			class={['wide:m-1', child && 'ml-8']}
			onclick={onCycle}
		/>
		<a {href} onclick={onOpen} class="flex min-h-12 min-w-0 flex-1 items-start gap-1 rounded-lg pr-3 outline-primary focus-visible:outline-2">
			<span class="min-w-0 flex-1 py-3 pl-1">
				<span class={['text-body-large line-clamp-2 break-words', done && 'text-on-surface-variant line-through']}>{t.title}</span>
				{#if row.preview}
					<span class="text-body-medium line-clamp-2 whitespace-pre-line text-on-surface-variant">{row.preview}</span>
				{/if}
				{#if priority || row.childTotal > 0 || row.images > 0}
					<span class="mt-1.5 flex flex-wrap gap-1.5">
						{#if priority}<Chip label={priorityLabel(priority)} icon={priorityIcon(priority)} tone={done ? '' : priorityTone(priority)} />{/if}
						{#if row.childTotal > 0}<Chip label={`${row.childDone}/${row.childTotal}`} icon="subtask" />{/if}
						{#if row.images > 0}<Chip label={String(row.images)} icon="image" />{/if}
					</span>
				{/if}
			</span>
			{#if tag}<span class={['grid h-12 place-items-center', done && 'opacity-50']}><TagDot color={tag} /></span>{/if}
		</a>
		{#if onFold && row.childTotal > 0}
			<IconButton
				icon={row.folded ? 'expandMore' : 'expandLess'}
				label={`${row.folded ? 'Unfold' : 'Fold'} subtasks of ${t.title}`}
				class="wide:m-1"
				onclick={onFold}
			/>
		{/if}
	{/if}
</div>
