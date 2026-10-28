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
		formatEstimate,
		isSeparator,
		joinDescription,
		PRIORITY_LABELS,
		STATUS_LABELS,
		tagName,
		type Task
	} from '$lib/model';
	import Pickers, { type PickerKind } from '$lib/Pickers.svelte';
	import TagMark from '$lib/TagMark.svelte';
	import Menu, { type MenuItem } from '$lib/ui/Menu.svelte';
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
	let choosing = $state<PickerKind | null>(null);
	let choosingOpen = $state(false);

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

	function choose(kind: PickerKind) {
		choosing = kind;
		choosingOpen = true;
	}

	const ACTIONS: Record<PickerKind, string> = {
		status: 'status.set',
		priority: 'priority.set',
		tag: 'tag.set',
		estimate: 'estimate.set'
	};

	const TITLES: Record<PickerKind, string> = {
		status: 'Status',
		priority: 'Priority',
		tag: 'Tag',
		estimate: 'Estimate'
	};

	function picked(value: string) {
		if (choosing !== null) {
			void edit({ action: ACTIONS[choosing], ids: [tid], value }, choosing === 'status');
		}
		choosingOpen = false;
	}

	function valueOf(kind: PickerKind, t: Task): string {
		switch (kind) {
			case 'status':
				return t.status;
			case 'priority':
				return t.priority ?? '';
			case 'tag':
				return t.tag_color ?? '';
			default:
				return String(t.estimate_minutes ?? 0);
		}
	}

	function menu(t: Task): MenuItem[] {
		const items: MenuItem[] = [
			{
				label: 'Move up',
				icon: 'arrow_upward',
				run: () => void edit({ action: 'task.shift', ids: [tid], up: true }, false)
			},
			{
				label: 'Move down',
				icon: 'arrow_downward',
				run: () => void edit({ action: 'task.shift', ids: [tid] }, false)
			}
		];
		if (t.parent_id) {
			items.push({
				label: 'Un-nest',
				icon: 'outdent',
				run: () => void edit({ action: 'task.unnest', ids: [tid] }, false)
			});
		} else if (!isSeparator(t)) {
			items.push({
				label: 'Nest under the task above',
				icon: 'indent',
				run: () => void edit({ action: 'task.nest', ids: [tid] }, false)
			});
		}
		items.push({ label: 'Delete', icon: 'delete', danger: true, run: remove });
		return items;
	}

	const rowClass = 'state-layer flex min-h-14 w-full items-center gap-4 rounded-2xl px-3 text-left';
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
		<Menu label="More for this task" items={menu(task)} />
	{/if}
</header>

{#if !entry || !task || !cat}
	<main class="px-6 pt-16 text-center text-on-surface-variant">
		<p>This task is gone: it was deleted, here or on another device.</p>
	</main>
{:else}
	<main class="mx-auto flex max-w-2xl flex-col gap-2 px-4 pb-[calc(var(--safe-b)+96px)]">
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
			placeholder={isSeparator(task) ? 'Separator label' : 'Title'}
			class="field-sizing-content w-full resize-none bg-transparent py-2 text-2xl leading-8 outline-none placeholder:text-on-surface-variant {finished(
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

		{#if !isSeparator(task)}
			<div class="flex items-start gap-4 px-3 py-2">
				<Icon name="notes" class="mt-0.5 text-on-surface-variant" />
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
			</div>
			{#each imagesOf(task.description) as image (image.target)}
				{@const line = fields.images.find((l) => l.includes(`(${image.target})`))}
				<div class="pl-13"><Thumb {image} remove={line ? () => dropImage(line) : undefined} /></div>
			{/each}
			<label use:ripple class="{rowClass} cursor-pointer text-on-surface-variant">
				<Icon name="add_photo" />Add image
				<input
					bind:this={picker}
					type="file"
					accept="image/png,image/jpeg,image/gif,image/webp"
					multiple
					class="hidden"
					onchange={(e) => void attach(e.currentTarget.files)}
				/>
			</label>

			<button use:ripple class={rowClass} onclick={() => choose('status')}>
				<StatusIcon status={task.status} />
				<span class="flex-1">{STATUS_LABELS[task.status] ?? task.status}</span>
			</button>
			<button use:ripple class={rowClass} onclick={() => choose('priority')}>
				<Icon
					name={task.priority ? `priority_${task.priority}` : 'priority_none'}
					class={task.priority === 'critical' || task.priority === 'high'
						? 'text-pri-up'
						: task.priority === 'low' || task.priority === 'trivial'
							? 'text-pri-down'
							: 'text-on-surface-variant'}
				/>
				<span class="flex-1 {task.priority ? '' : 'text-on-surface-variant'}"
					>{task.priority ? PRIORITY_LABELS[task.priority] : 'Add priority'}</span
				>
			</button>
			<button use:ripple class={rowClass} onclick={() => choose('tag')}>
				<span class="grid size-6 place-items-center text-on-surface-variant">
					{#if task.tag_color}<TagMark color={task.tag_color} />{:else}<Icon name="label" />{/if}
				</span>
				<span class="flex-1 {task.tag_color ? '' : 'text-on-surface-variant'}"
					>{task.tag_color ? tagName(task.tag_color) : 'Add tag'}</span
				>
			</button>
			<button use:ripple class={rowClass} onclick={() => choose('estimate')}>
				<Icon name="schedule" class="text-on-surface-variant" />
				<span class="flex-1 {task.estimate_minutes ? '' : 'text-on-surface-variant'}"
					>{task.estimate_minutes ? formatEstimate(task.estimate_minutes) : 'Add estimate'}</span
				>
			</button>

			{#if !task.parent_id}
				<div class="mt-2">
					{#each children as c (c.id)}
						<div class="flex items-center">
							<button
								use:ripple
								class="state-layer grid size-12 shrink-0 place-items-center rounded-full"
								aria-label="{STATUS_LABELS[c.status]}, change status"
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
					<form class="flex items-center px-3" onsubmit={addSubtask}>
						<Icon name="subtask" class="text-on-surface-variant" />
						<input
							bind:value={subtask}
							maxlength="500"
							enterkeyhint="enter"
							placeholder="Add subtasks"
							aria-label="Add a subtask"
							class="h-14 min-w-0 flex-1 bg-transparent pl-4 outline-none placeholder:text-on-surface-variant"
						/>
					</form>
				</div>
			{/if}
		{/if}
	</main>

	{#if !isSeparator(task)}
		<div
			class="fixed inset-x-0 bottom-0 flex justify-end bg-surface px-4 pt-2 pb-[calc(var(--safe-b)+12px)]"
		>
			<button
				use:ripple
				class="state-layer h-10 rounded-full px-4 text-sm font-medium text-primary"
				onclick={() =>
					void edit(
						{
							action: 'status.set',
							ids: [tid],
							value: finished(task.status) ? 'todo' : 'completed'
						},
						false
					)}
			>
				{finished(task.status) ? 'Mark open' : 'Mark done'}
			</button>
		</div>
	{/if}

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

	<Sheet bind:open={choosingOpen} title={choosing ? TITLES[choosing] : ''}>
		{#if choosing}
			<Pickers kind={choosing} value={valueOf(choosing, task)} pick={picked} />
		{/if}
	</Sheet>
{/if}

<Confirm bind:spec={confirm} />
