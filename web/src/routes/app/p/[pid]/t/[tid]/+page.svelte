<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { tick } from 'svelte';
	import Chip from '$lib/app/Chip.svelte';
	import Icon from '$lib/app/Icon.svelte';
	import IconButton from '$lib/app/IconButton.svelte';
	import Menu from '$lib/app/Menu.svelte';
	import MenuItem from '$lib/app/MenuItem.svelte';
	import TagDot from '$lib/app/TagDot.svelte';
	import Thumbnail from '$lib/app/Thumbnail.svelte';
	import { autosize } from '$lib/app/autosize';
	import { priorityIcon, statusIcon } from '$lib/app/icons';
	import { openTask, ui, wide } from '$lib/app/ui.svelte';
	import type { Op } from '$lib/core/types';
	import { fields, join, refs, type ImageRef } from '$lib/model/description';
	import {
		categoryPath,
		find,
		finished,
		isLine,
		isNote,
		isSeparator,
		nextStatus,
		priorities,
		priorityLabel,
		priorityOf,
		priorityTone,
		statusLabel,
		statuses,
		tagLabel,
		tags
	} from '$lib/model/project';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';

	const local = session.local!;
	const pid = $derived(page.params.pid ?? '');
	const tid = $derived(page.params.tid ?? '');
	const project = $derived(local.project(pid));
	const found = $derived(project ? find(project, tid) : undefined);
	const task = $derived(found?.task);
	const cat = $derived(found?.cat);

	let title = $state('');
	let titleBase = $state('');
	let details = $state('');
	let detailsBase = $state('');
	let descBase = $state('');
	let editingTitle = $state(false);
	let editingDetails = $state(false);
	let titleField: HTMLTextAreaElement | undefined = $state();
	let menu = $state('');
	let anchors: Record<string, HTMLElement | undefined> = $state({});
	let more: HTMLButtonElement | undefined = $state();
	let viewer: { url: string; name: string } | undefined = $state();
	let viewerDialog: HTMLDialogElement | undefined = $state();
	let picker: HTMLInputElement | undefined = $state();
	let subtask = $state('');

	const flat = (s: string) => s.replace(/\s+/g, ' ').trim();

	// A new task, or the text changed by an undo while the field isn't being typed in.
	let shownId = '';
	$effect(() => {
		if (!task) return;
		const fresh = task.id !== shownId;
		shownId = task.id;
		if (fresh || (!editingTitle && task.title !== titleBase)) {
			title = titleBase = task.title;
		}
		if (fresh || (!editingDetails && (task.description ?? '') !== descBase)) {
			descBase = task.description ?? '';
			details = detailsBase = fields(descBase).text;
		}
		if (fresh) menu = '';
	});

	$effect(() => {
		if (task && ui.editTitle) {
			ui.editTitle = false;
			void tick().then(() => titleField?.focus());
		}
	});

	$effect(() => {
		if (project && !found) void goto(app(`/p/${pid}`), { replaceState: true });
	});

	async function saveTitle() {
		if (!task) return;
		const next = flat(title);
		if ((next === '' && !isSeparator(task)) || next === titleBase) return;
		const base = titleBase;
		titleBase = next;
		const a = await local.perform(pid, { action: 'task.title', ids: [task.id], title: next, base }, { quiet: true });
		if (a.status !== 'applied' && a.status !== 'unchanged') titleBase = base;
	}

	async function saveDetails() {
		if (!task || details === detailsBase) return;
		const text = join(details, fields(task.description).images);
		const base = descBase;
		detailsBase = details;
		descBase = text;
		const a = await local.perform(pid, { action: 'task.description', ids: [task.id], text, base }, { quiet: true });
		if (a.status !== 'applied' && a.status !== 'unchanged') descBase = base;
	}

	$effect(() => {
		if (flat(title) === titleBase) return;
		const timer = setTimeout(saveTitle, 800);
		return () => clearTimeout(timer);
	});

	$effect(() => {
		if (details === detailsBase) return;
		const timer = setTimeout(saveDetails, 1000);
		return () => clearTimeout(timer);
	});

	// Esc keeps what was typed, as Enter does, and hands the keys back to the list.
	function fieldKey(e: KeyboardEvent, enterSaves: boolean) {
		if (e.key === 'Escape' || (enterSaves && e.key === 'Enter' && !e.shiftKey)) {
			e.preventDefault();
			e.stopPropagation();
			(e.currentTarget as HTMLElement).blur();
			if (e.key === 'Escape' && wide()) backToList();
		}
	}

	function backToList() {
		document.querySelector<HTMLElement>(`[data-key="t:${tid}"] a`)?.focus({ preventScroll: true });
	}

	function edit(op: Op, quiet = true) {
		void local.perform(pid, op, { quiet });
	}

	async function attach(files: FileList | null | undefined) {
		if (!task || !files?.length) return;
		await saveDetails();
		const a = await local.attach(pid, task.id, [...files]);
		if (typeof a === 'string') local.say(a);
		if (picker) picker.value = '';
	}

	async function removeImage(ref: ImageRef) {
		await saveDetails();
		const current = local.project(pid) && find(local.project(pid)!, tid)?.task;
		if (!current) return;
		const { text, images } = fields(current.description);
		const line = images.find((l) => l.includes(`(${ref.target})`));
		if (!line) return;
		await local.perform(
			pid,
			{ action: 'task.description', ids: [tid], text: join(text, images.filter((l) => l !== line)), base: current.description ?? '' },
			{ said: 'Image removed' }
		);
	}

	function addSubtask(e: SubmitEvent) {
		e.preventDefault();
		const text = flat(subtask);
		if (!text || !cat || !task) return;
		subtask = '';
		edit({ action: 'task.add', cat: cat.id, title: text, parent: task.id });
	}

	function close() {
		void goto(app(`/p/${pid}`), { noScroll: true });
	}

	$effect(() => {
		if (viewer && viewerDialog && !viewerDialog.open) viewerDialog.showModal();
	});

	const parent = $derived(task?.parent_id && cat ? cat.tasks.find((t) => t.id === task.parent_id) : undefined);
	const children = $derived(task && cat && !task.parent_id && !isLine(task) ? cat.tasks.filter((t) => t.parent_id === task.id) : []);
	const images = $derived(task && !isLine(task) ? refs(task.description) : []);
	const done = $derived(task ? finished(task) : false);
	const statusSet = $derived(task?.status === 'in_progress' || task?.status === 'cancelled');
</script>

{#if task && cat && project}
	<article
		aria-label="Task"
		class="relative flex min-h-dvh flex-col bg-surface wide:h-full wide:min-h-0 wide:overflow-y-auto wide:rounded-xl wide:bg-surface-container-low"
	>
		<header class="sticky top-0 z-10 flex h-16 shrink-0 items-center gap-1 bg-inherit px-1 pt-[env(safe-area-inset-top)] wide:h-14 wide:px-2">
			<IconButton icon="back" label="Back" class="wide:hidden" tone="plain" onclick={close} />
			<span bind:this={anchors.move} class="inline-flex min-w-0">
				<button
					type="button"
					class="text-title-medium flex h-10 min-w-0 cursor-pointer items-center gap-0.5 rounded-full pr-2 pl-3 text-primary outline-primary hover:bg-primary/8 focus-visible:outline-2"
					aria-label={`In ${cat.name}, move to another category`}
					onclick={() => (menu = menu === 'move' ? '' : 'move')}
				>
					<span class="truncate">{cat.name}</span>
					<Icon name="expandMore" class="size-5" />
				</button>
			</span>
			<span class="flex-1"></span>
			{#if !isLine(task)}
				<IconButton icon="addPhoto" label="Add an image" onclick={() => picker?.click()} />
				<input bind:this={picker} type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple class="hidden" onchange={(e) => attach(e.currentTarget.files)} />
			{/if}
			<IconButton bind:el={more} icon="more" label="More" onclick={() => (menu = menu === 'more' ? '' : 'more')} aria-haspopup="menu" />
			<IconButton icon="close" label="Close" class="max-wide:hidden" onclick={close} />
		</header>

		<Menu open={menu === 'move'} onclose={() => (menu = '')} anchor={anchors.move} label="Move to">
			{#each project.categories.filter((c) => c.id !== cat.id) as c (c.id)}
				<MenuItem label={categoryPath(project, c)} onclick={() => edit({ action: 'task.move', ids: [tid], cat: c.id }, false)} />
			{:else}
				<p class="text-body-medium px-3 py-2 text-on-surface-variant">No other category</p>
			{/each}
		</Menu>
		<Menu open={menu === 'more'} onclose={() => (menu = '')} anchor={more} align="end" label="Task">
			<MenuItem icon="arrowUp" label="Move up" keys="K" onclick={() => edit({ action: 'task.shift', ids: [tid], up: true }, false)} />
			<MenuItem icon="arrowDown" label="Move down" keys="J" onclick={() => edit({ action: 'task.shift', ids: [tid] }, false)} />
			{#if !isLine(task)}
				{#if task.parent_id}
					<MenuItem icon="outdent" label="Un-nest" keys="<" onclick={() => edit({ action: 'task.unnest', ids: [tid] }, false)} />
				{:else}
					<MenuItem icon="indent" label="Nest under the row above" keys=">" onclick={() => edit({ action: 'task.nest', ids: [tid] }, false)} />
				{/if}
			{/if}
			<MenuItem icon="move" label="Move to a category" keys="m" onclick={() => (ui.sheet = { kind: 'move', pid, tid })} />
			<MenuItem icon="delete" label="Delete" keys="d" onclick={() => edit({ action: 'task.delete', ids: [tid] }, false)} />
		</Menu>

		<div class="flex-1 pb-32 wide:mx-auto wide:w-full wide:max-w-[42rem]">
			{#if parent}
				<a
					href={app(`/p/${pid}/t/${parent.id}`)}
					onclick={(e) => {
						e.preventDefault();
						void openTask(pid, parent.id);
					}}
					class="text-label-large inline-block rounded-full px-4 py-2 text-primary hover:underline"
				>
					Subtask of {parent.title}
				</a>
			{/if}
			<textarea
				bind:this={titleField}
				bind:value={title}
				use:autosize={title}
				rows="1"
				aria-label="Title"
				placeholder={isSeparator(task) ? 'Separator label' : isNote(task) ? 'Note' : 'Title'}
				onfocus={() => (editingTitle = true)}
				onblur={() => {
					editingTitle = false;
					void saveTitle();
				}}
				oninput={() => (title = title.replace(/\n/g, ' '))}
				onkeydown={(e) => fieldKey(e, true)}
				class={[
					'text-headline-small block w-full resize-none overflow-hidden bg-transparent px-4 py-2 outline-none placeholder:text-on-surface-variant',
					done && 'line-through'
				]}
			></textarea>
			{#if !isLine(task)}
				<textarea
					bind:value={details}
					use:autosize={details}
					rows="1"
					aria-label="Description"
					placeholder="Add a description"
					onfocus={() => (editingDetails = true)}
					onblur={() => {
						editingDetails = false;
						void saveDetails();
					}}
					onkeydown={(e) => fieldKey(e, false)}
					class="text-body-large block w-full resize-none overflow-hidden bg-transparent px-4 py-2 outline-none placeholder:text-on-surface-variant"
				></textarea>
				{#if images.length}
					<div class="flex flex-wrap gap-2 px-4 py-2">
						{#each images as image (image.target)}
							<Thumbnail {local} {image} onOpen={(url) => (viewer = { url, name: image.name })} onRemove={() => removeImage(image)} />
						{/each}
					</div>
				{/if}
			{/if}

			<div class="flex flex-wrap items-center gap-2 px-4 py-1">
				{#if !isLine(task)}
					<span bind:this={anchors.status} class="inline-flex">
						{#if statusSet}
							<Chip label={statusLabel(task.status)} icon={statusIcon(task.status)} name="Status" onclick={() => (menu = 'status')} />
						{:else}
							<IconButton icon={statusIcon(task.status)} label={`Status: ${statusLabel(task.status)}`} onclick={() => (menu = 'status')} />
						{/if}
					</span>
					<span bind:this={anchors.priority} class="inline-flex">
						<Chip
							label={priorityLabel(priorityOf(task))}
							icon={priorityIcon(priorityOf(task))}
							tone={done ? '' : priorityTone(priorityOf(task))}
							name="Priority"
							onclick={() => (menu = 'priority')}
						/>
					</span>
				{/if}
				<span bind:this={anchors.tag} class="inline-flex">
					{#if task.tag_color}
						<Chip label={tagLabel(task.tag_color)} tag={task.tag_color} name="Colour" onclick={() => (menu = 'tag')} />
					{:else}
						<Chip label="Colour" icon="palette" onclick={() => (menu = 'tag')} />
					{/if}
				</span>
			</div>
			<Menu open={menu === 'status'} onclose={() => (menu = '')} anchor={anchors.status} label="Status">
				{#each statuses as s (s)}
					<MenuItem label={statusLabel(s)} icon={statusIcon(s)} checked={s === task.status} onclick={() => edit({ action: 'status.set', ids: [tid], value: s }, false)} />
				{/each}
			</Menu>
			<Menu open={menu === 'priority'} onclose={() => (menu = '')} anchor={anchors.priority} label="Priority">
				{#each priorities as p (p)}
					<MenuItem label={priorityLabel(p)} checked={p === priorityOf(task)} onclick={() => edit({ action: 'priority.set', ids: [tid], value: p })}>
						{#snippet leading()}<Icon name={priorityIcon(p)} class={['size-6', { up: 'text-mark-up', down: 'text-mark-down', '': 'text-on-surface-variant' }[priorityTone(p)]]} />{/snippet}
					</MenuItem>
				{/each}
			</Menu>
			<Menu open={menu === 'tag'} onclose={() => (menu = '')} anchor={anchors.tag} label="Colour">
				{#each [...tags, ''] as t (t)}
					<MenuItem label={tagLabel(t)} checked={t === (task.tag_color ?? '').replace(/-diamond$/, '')} onclick={() => edit({ action: 'tag.set', ids: [tid], value: t })}>
						{#snippet leading()}{#if t}<TagDot color={t} class="size-3" />{/if}{/snippet}
					</MenuItem>
				{/each}
			</Menu>

			{#if !isLine(task) && !task.parent_id}
				<ul class="pt-2" aria-label="Subtasks">
					{#each children as c (c.id)}
						<li class="flex items-start rounded-lg pr-4 pl-1 hover:bg-on-surface/8">
							<IconButton
								icon={statusIcon(c.status)}
								label={`${statusLabel(c.status)}, change status`}
								tone={c.status === 'in_progress' || c.status === 'completed' ? 'primary' : 'muted'}
								onclick={() => edit({ action: 'status.cycle', ids: [c.id] }, nextStatus(c.status) !== 'completed')}
							/>
							<a
								href={app(`/p/${pid}/t/${c.id}`)}
								onclick={(e) => {
									e.preventDefault();
									void openTask(pid, c.id);
								}}
								class={['text-body-large min-w-0 flex-1 py-3 pl-1', finished(c) && 'text-on-surface-variant line-through']}
							>
								{c.title}
							</a>
						</li>
					{/each}
					<li class="flex items-center pr-4 pl-1">
						<span class="grid size-12 place-items-center text-on-surface-variant wide:size-10"><Icon name="subtask" /></span>
						<form class="min-w-0 flex-1" onsubmit={addSubtask}>
							<input
								bind:value={subtask}
								placeholder="Add a subtask"
								aria-label="Add a subtask"
								enterkeyhint="done"
								onblur={() => {
									if (flat(subtask) && cat && task) edit({ action: 'task.add', cat: cat.id, title: flat(subtask), parent: task.id });
									subtask = '';
								}}
								onkeydown={(e) => {
									if (e.key === 'Escape') {
										e.preventDefault();
										e.currentTarget.blur();
									}
								}}
								class="text-body-large w-full bg-transparent py-3 pl-1 outline-none placeholder:text-on-surface-variant"
							/>
						</form>
					</li>
				</ul>
			{/if}
		</div>

		{#if !isLine(task)}
			<div class="pointer-events-none sticky bottom-0 mt-auto flex justify-end p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
				<button
					type="button"
					class="text-label-large pointer-events-auto flex h-14 cursor-pointer items-center gap-3 rounded-lg bg-primary-container pr-5 pl-4 text-on-primary-container shadow-md outline-primary hover:shadow-lg focus-visible:outline-2"
					onclick={() => edit({ action: 'status.set', ids: [tid], value: done ? 'todo' : 'completed' }, false)}
				>
					<Icon name={done ? 'todo' : 'check'} />{done ? 'Mark todo' : 'Mark done'}
				</button>
			</div>
		{/if}
	</article>

	{#if viewer}
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
		<dialog
			bind:this={viewerDialog}
			aria-label={viewer.name || 'Image'}
			onclose={() => (viewer = undefined)}
			onclick={() => viewerDialog?.close()}
			class="m-auto max-h-none max-w-none bg-transparent p-0 backdrop:bg-black/90"
		>
			<img src={viewer.url} alt={viewer.name} class="max-h-[92dvh] max-w-[96vw] object-contain" />
		</dialog>
	{/if}
{/if}
