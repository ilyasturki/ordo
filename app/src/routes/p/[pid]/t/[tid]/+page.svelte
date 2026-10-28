<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Icon from '$lib/Icon.svelte';
	import { keepImage } from '$lib/images';
	import {
		descriptionFields,
		findTask,
		finished,
		imagesOf,
		isSeparator,
		joinDescription,
		type Task
	} from '$lib/model';
	import Pickers from '$lib/Pickers.svelte';
	import StatusIcon from '$lib/StatusIcon.svelte';
	import Thumb from '$lib/Thumb.svelte';
	import { ordo } from '$lib/state.svelte';
	import Confirm, { type ConfirmSpec } from '$lib/ui/Confirm.svelte';
	import IconButton from '$lib/ui/IconButton.svelte';
	import Sheet from '$lib/ui/Sheet.svelte';
	import { ripple } from '$lib/ui/ripple';

	const pid = $derived(page.params.pid ?? '');
	const tid = $derived(page.params.tid ?? '');
	const entry = $derived(ordo.project(pid));
	const found = $derived(entry ? findTask(entry.data, tid) : null);
	const task = $derived(found?.task);
	const cat = $derived(found?.cat);
	const parent = $derived(
		task?.parent_id && cat ? cat.tasks.find((t) => t.id === task.parent_id) : undefined
	);
	const children = $derived(
		cat && task && !task.parent_id ? cat.tasks.filter((t) => t.parent_id === task.id) : []
	);
	const fields = $derived(descriptionFields(task?.description ?? ''));

	let title = $state('');
	let details = $state('');
	let titleBase = '';
	let detailsBase = '';
	let editingTitle = false;
	let editingDetails = false;
	let titleTimer: ReturnType<typeof setTimeout> | undefined;
	let detailsTimer: ReturnType<typeof setTimeout> | undefined;
	let moveOpen = $state(false);
	let confirm = $state<ConfirmSpec | null>(null);
	let subtask = $state('');
	let picker = $state<HTMLInputElement | null>(null);

	$effect(() => {
		const t = task;
		if (t === undefined) {
			return;
		}
		if (!editingTitle && t.title !== titleBase) {
			title = titleBase = t.title;
		}
		const d = descriptionFields(t.description ?? '').text;
		if (!editingDetails && d !== detailsBase) {
			details = detailsBase = d;
		}
	});

	function edit(op: Parameters<typeof ordo.edit>[1], quiet = true) {
		return ordo.edit(pid, op, { quiet });
	}

	async function saveTitle() {
		clearTimeout(titleTimer);
		const next = title.replace(/\s+/gu, ' ').trim();
		if (task === undefined || next === '' || next === titleBase) {
			return;
		}
		const base = titleBase;
		titleBase = next;
		const a = await edit({ action: 'task.title', ids: [tid], title: next, base });
		if (a.status !== 'applied' && a.status !== 'unchanged') {
			titleBase = base;
		}
	}

	async function saveDetails() {
		clearTimeout(detailsTimer);
		if (task === undefined || details === detailsBase) {
			return;
		}
		const { images } = descriptionFields(task.description ?? '');
		const base = task.description ?? '';
		const text = joinDescription(details, images);
		detailsBase = details;
		await edit({ action: 'task.description', ids: [tid], text, base });
	}

	function dropImage(line: string) {
		if (task === undefined) {
			return;
		}
		const base = task.description ?? '';
		const text = joinDescription(
			fields.text,
			fields.images.filter((l) => l !== line)
		);
		void ordo.edit(pid, { action: 'task.description', ids: [tid], text, base });
	}

	async function attach(files: FileList | null) {
		if (files === null || files.length === 0) {
			return;
		}
		const lines: string[] = [];
		for (const f of files) {
			try {
				lines.push(await keepImage(f));
			} catch (err) {
				ordo.toast(err instanceof Error ? err.message : String(err));
			}
		}
		if (lines.length > 0) {
			await saveDetails();
			await edit({ action: 'task.attach', ids: [tid], lines }, false);
		}
		if (picker) {
			picker.value = '';
		}
	}

	function remove() {
		if (task === undefined) {
			return;
		}
		const go = () => {
			void ordo.edit(pid, { action: 'task.delete', ids: [tid] });
			back();
		};
		if (children.length > 0) {
			confirm = {
				title: 'Delete this task?',
				body: `Its ${children.length} subtask${children.length === 1 ? '' : 's'} go with it.`,
				confirm: 'Delete',
				run: go
			};
			return;
		}
		go();
	}

	function back() {
		if (history.length > 1) {
			history.back();
		} else {
			void goto('/', { replaceState: true });
		}
	}

	async function addSubtask(e: SubmitEvent) {
		e.preventDefault();
		const text = subtask.trim();
		if (text === '' || cat === undefined) {
			return;
		}
		const a = await edit({ action: 'task.add', cat: cat.id, parent: tid, title: text });
		if (a.status === 'applied') {
			subtask = '';
		}
	}

	function cycle(t: Task) {
		void ordo.edit(pid, { action: 'status.cycle', ids: [t.id] });
	}

	const section = 'mb-2 text-sm font-medium text-on-surface-variant';
</script>

<svelte:window
	onpagehide={() => {
		void saveTitle();
		void saveDetails();
	}}
/>

<header
	class="sticky top-0 z-30 flex h-[calc(64px+var(--safe-t))] items-center gap-1 bg-surface px-1 pt-safe-t"
>
	<IconButton icon="arrow_back" label="Back" onclick={back} />
	<div class="flex-1"></div>
	{#if task}
		<IconButton icon="delete" label="Delete task" onclick={remove} />
	{/if}
</header>

{#if !entry || !task || !cat}
	<main class="px-6 pt-16 text-center text-on-surface-variant">
		<p>This task is gone: it was deleted, here or on another device.</p>
	</main>
{:else}
	<main class="mx-auto flex max-w-2xl flex-col gap-6 px-4 pb-[calc(var(--safe-b)+48px)]">
		<div class="flex flex-col gap-1">
			<button
				use:ripple
				class="state-layer -ml-1 inline-flex h-8 items-center gap-1 self-start rounded-lg pr-2 pl-3 text-sm font-medium text-primary"
				onclick={() => (moveOpen = true)}
			>
				{cat.name}<Icon name="expand_more" size={18} />
			</button>
			{#if parent}
				<a
					href="/p/{pid}/t/{parent.id}"
					class="flex items-center gap-1 text-sm text-on-surface-variant"
				>
					<Icon name="subtask" size={16} />Subtask of {parent.title}
				</a>
			{/if}
			<textarea
				bind:value={title}
				rows="1"
				maxlength="500"
				enterkeyhint="done"
				aria-label="Title"
				class="field-sizing-content w-full resize-none bg-transparent text-2xl leading-8 outline-none {finished(
					task.status
				)
					? 'text-on-surface-variant line-through'
					: ''}"
				onfocus={() => (editingTitle = true)}
				onblur={() => {
					editingTitle = false;
					void saveTitle();
				}}
				oninput={() => {
					clearTimeout(titleTimer);
					titleTimer = setTimeout(() => void saveTitle(), 800);
				}}
				onkeydown={(e) => {
					if (e.key === 'Enter') {
						e.preventDefault();
						(e.currentTarget as HTMLTextAreaElement).blur();
					}
				}}></textarea>
		</div>

		{#if !isSeparator(task)}
			<div class="flex items-start gap-3">
				<Icon name="notes" class="mt-0.5 text-on-surface-variant" />
				<div class="flex min-w-0 flex-1 flex-col gap-3">
					<textarea
						bind:value={details}
						rows="1"
						placeholder="Add details"
						aria-label="Details"
						class="field-sizing-content min-h-6 w-full resize-none bg-transparent text-base leading-6 outline-none placeholder:text-on-surface-variant"
						onfocus={() => (editingDetails = true)}
						onblur={() => {
							editingDetails = false;
							void saveDetails();
						}}
						oninput={() => {
							clearTimeout(detailsTimer);
							detailsTimer = setTimeout(() => void saveDetails(), 1000);
						}}></textarea>
					{#each imagesOf(task.description) as image (image.target)}
						{@const line = fields.images.find((l) => l.includes(`(${image.target})`))}
						<Thumb {image} remove={line ? () => dropImage(line) : undefined} />
					{/each}
					<label
						class="state-layer inline-flex h-8 cursor-pointer items-center gap-2 self-start rounded-lg px-2 text-sm font-medium text-primary"
					>
						<Icon name="add_photo" size={20} />Add image
						<input
							bind:this={picker}
							type="file"
							accept="image/png,image/jpeg,image/gif,image/webp"
							multiple
							class="hidden"
							onchange={(e) => void attach(e.currentTarget.files)}
						/>
					</label>
				</div>
			</div>

			<section>
				<h2 class={section}>Status</h2>
				<Pickers
					kind="status"
					value={task.status}
					pick={(v) => void edit({ action: 'status.set', ids: [tid], value: v }, false)}
				/>
			</section>
			<section>
				<h2 class={section}>Priority</h2>
				<Pickers
					kind="priority"
					value={task.priority ?? ''}
					pick={(v) => void edit({ action: 'priority.set', ids: [tid], value: v })}
				/>
			</section>
			<section>
				<h2 class={section}>Tag</h2>
				<Pickers
					kind="tag"
					value={task.tag_color ?? ''}
					pick={(v) => void edit({ action: 'tag.set', ids: [tid], value: v })}
				/>
			</section>
			<section>
				<h2 class={section}>Estimate</h2>
				<Pickers
					kind="estimate"
					value={String(task.estimate_minutes ?? 0)}
					pick={(v) => void edit({ action: 'estimate.set', ids: [tid], value: v })}
				/>
			</section>

			{#if !task.parent_id}
				<section>
					<h2 class={section}>Subtasks</h2>
					{#each children as c (c.id)}
						<div class="flex items-center">
							<button
								use:ripple
								class="state-layer grid size-12 shrink-0 place-items-center rounded-full"
								aria-label="Change status"
								onclick={() => cycle(c)}
							>
								<StatusIcon status={c.status} />
							</button>
							<a
								href="/p/{pid}/t/{c.id}"
								class="min-w-0 flex-1 py-3 {finished(c.status)
									? 'text-on-surface-variant line-through'
									: ''}">{c.title}</a
							>
						</div>
					{/each}
					<form class="flex items-center" onsubmit={addSubtask}>
						<span class="grid size-12 shrink-0 place-items-center text-on-surface-variant"
							><Icon name="add" /></span
						>
						<input
							bind:value={subtask}
							maxlength="500"
							enterkeyhint="enter"
							placeholder="Add subtask"
							class="h-12 min-w-0 flex-1 bg-transparent outline-none placeholder:text-on-surface-variant"
						/>
					</form>
				</section>
			{/if}
		{/if}
	</main>

	<Sheet bind:open={moveOpen} title="Move to">
		{#each entry.data.categories as c (c.id)}
			<button
				use:ripple
				class="state-layer flex h-12 w-full items-center gap-4 rounded-full px-4 text-left text-sm {c.parent_id
					? 'pl-10'
					: ''} {c.id === cat.id ? 'bg-secondary-container text-on-secondary-container' : ''}"
				onclick={() => {
					moveOpen = false;
					if (c.id !== cat.id) {
						void ordo.edit(pid, { action: 'task.move', ids: [tid], cat: c.id });
					}
				}}
			>
				<Icon name={c.id === cat.id ? 'check' : 'move'} class="text-on-surface-variant" />{c.name}
			</button>
		{/each}
	</Sheet>
{/if}

<Confirm bind:spec={confirm} />
