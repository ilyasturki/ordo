<script lang="ts">
	import { untrack } from 'svelte';
	import type { Task } from '$lib/core/types';
	import type { Local } from '$lib/local.svelte';
	import { taskVisible } from '$lib/model/filter';
	import { categoryPath, priorities, priorityLabel, priorityTone, tagLabel, tags } from '$lib/model/project';
	import Chip from './Chip.svelte';
	import Icon from './Icon.svelte';
	import IconButton from './IconButton.svelte';
	import Menu from './Menu.svelte';
	import MenuItem from './MenuItem.svelte';
	import Sheet from './Sheet.svelte';
	import TagDot from './TagDot.svelte';
	import { priorityIcon } from './icons';
	import { ui } from './ui.svelte';

	// The phone's add sheet: a title, then the category, details, priority and colour, each one tap away.
	let { local, pid: shownPid, cat: initialCat, after: initialAfter, onclose }: { local: Local; pid: string; cat: string; after?: string; onclose: () => void } =
		$props();

	// Its save outlives the sheet when Esc closes it mid-save.
	const pid = untrack(() => shownPid);

	const project = $derived(local.project(pid)!);
	let title = $state('');
	let details: string | undefined = $state();
	let cat = $state(untrack(() => initialCat));
	let after = $state(untrack(() => initialAfter ?? ''));
	let priority = $state('');
	let tag = $state('');
	let menu = $state('');
	let field: HTMLInputElement | undefined = $state();
	let anchors: Record<string, HTMLElement | undefined> = $state({});

	const category = $derived(project.categories.find((c) => c.id === cat));
	const hidden = $derived(
		!!category && !taskVisible(local.view(pid).filter, { id: '', title: '', status: 'todo', priority, tag_color: tag } as Task, category)
	);

	$effect(() => {
		field?.focus();
	});

	async function save(): Promise<boolean> {
		const text = title.replace(/\s+/g, ' ').trim();
		const into = category;
		if (text === '' || !into) return true;
		if (hidden) return false;
		title = '';
		const a = await local.perform(pid, { action: 'task.insert', cat: into.id, ids: after ? [after] : [], title: text }, { quiet: true });
		if (a.status !== 'applied' || !a.row) {
			title = text;
			local.say(a.msg ?? '');
			return false;
		}
		const row = a.row;
		after = row;
		if (details?.trim()) await local.perform(pid, { action: 'task.description', ids: [row], text: details.trim(), base: '' }, { quiet: true });
		if (priority) await local.perform(pid, { action: 'priority.set', ids: [row], value: priority }, { quiet: true });
		if (tag) await local.perform(pid, { action: 'tag.set', ids: [row], value: tag }, { quiet: true });
		details = undefined;
		await local.setFolded(pid, into.id, false);
		if (into.parent_id) await local.setFolded(pid, into.parent_id, false);
		ui.cursor = 't:' + row;
		return true;
	}

	async function leave() {
		if (await save()) onclose();
	}
</script>

<Sheet open label="New task" onclose={leave}>
	<form
		class="pt-2 pb-1"
		onsubmit={(e) => {
			e.preventDefault();
			void save();
		}}
	>
		<input
			bind:this={field}
			bind:value={title}
			placeholder="New task"
			aria-label="New task"
			enterkeyhint="done"
			autocapitalize="sentences"
			class="text-title-medium w-full bg-transparent px-4 py-3 outline-none placeholder:text-on-surface-variant"
		/>
		{#if hidden}
			<p class="text-body-small px-4 text-error">Clear the filter to add a task here</p>
		{/if}
		{#if details !== undefined}
			<!-- svelte-ignore a11y_autofocus -->
			<textarea
				bind:value={details}
				placeholder="Add a description"
				aria-label="Description"
				rows="2"
				autofocus
				class="text-body-large wide:text-body-medium field-sizing-content max-h-40 w-full resize-none bg-transparent px-4 py-2 outline-none placeholder:text-on-surface-variant"
			></textarea>
		{/if}
		<div class="flex items-center gap-1 px-2">
			<div class="flex min-w-0 flex-1 items-center">
				<span bind:this={anchors.cat} class="ml-2 inline-flex min-w-0">
					<Chip label={category?.name ?? 'Category'} trailing="expandMore" name="Category" onclick={() => (menu = menu === 'cat' ? '' : 'cat')} />
				</span>
				{#if details === undefined}
					<IconButton icon="notes" label="Description" onclick={() => (details = '')} />
				{/if}
				<span bind:this={anchors.priority} class="inline-flex">
					{#if priority === ''}
						<IconButton icon="flag" label="Priority" onclick={() => (menu = 'priority')} />
					{:else}
						<span class="mx-1"><Chip icon={priorityIcon(priority)} tone={priorityTone(priority)} name="Priority" label="" onclick={() => (menu = 'priority')} /></span>
					{/if}
				</span>
				<span bind:this={anchors.tag} class="inline-flex">
					{#if tag === ''}
						<IconButton icon="palette" label="Colour" onclick={() => (menu = 'tag')} />
					{:else}
						<span class="mx-1"><Chip {tag} name="Colour" label="" onclick={() => (menu = 'tag')} /></span>
					{/if}
				</span>
			</div>
			<button
				class="text-label-large h-10 cursor-pointer rounded-full px-3 text-primary hover:bg-primary/8 disabled:cursor-default disabled:text-on-surface/38 disabled:hover:bg-transparent"
				disabled={title.trim() === '' || hidden}
			>
				Save
			</button>
		</div>
	</form>
	<Menu open={menu === 'cat'} onclose={() => (menu = '')} anchor={anchors.cat} label="Category">
		{#each project.categories as c (c.id)}
			<MenuItem
				label={categoryPath(project, c)}
				checked={c.id === cat}
				onclick={() => {
					if (c.id !== cat) after = '';
					cat = c.id;
					menu = '';
					field?.focus();
				}}
			/>
		{/each}
	</Menu>
	<Menu open={menu === 'priority'} onclose={() => (menu = '')} anchor={anchors.priority} label="Priority">
		{#each priorities as p (p)}
			<MenuItem
				label={priorityLabel(p)}
				checked={p === priority}
				onclick={() => {
					priority = p;
					menu = '';
					field?.focus();
				}}
			>
				{#snippet leading()}<Icon name={priorityIcon(p)} class={['size-6', { up: 'text-mark-up', down: 'text-mark-down', '': 'text-on-surface-variant' }[priorityTone(p)]]} />{/snippet}
			</MenuItem>
		{/each}
	</Menu>
	<Menu open={menu === 'tag'} onclose={() => (menu = '')} anchor={anchors.tag} label="Colour">
		{#each ['', ...tags] as t (t)}
			<MenuItem
				label={tagLabel(t)}
				checked={t === tag}
				onclick={() => {
					tag = t;
					menu = '';
					field?.focus();
				}}
			>
				{#snippet leading()}{#if t}<TagDot color={t} class="size-3" />{/if}{/snippet}
			</MenuItem>
		{/each}
	</Menu>
</Sheet>
