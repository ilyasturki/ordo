<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Answer, Op } from '$lib/core/types';
	import type { Local } from '$lib/local.svelte';
	import { addedLineShows } from '$lib/model/board';
	import { categoryPath, category as findCategory, deletedWith, find, groupKey, isLine, projectGroups, versioned } from '$lib/model/project';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';
	import AddSheet from './AddSheet.svelte';
	import Dialog from './Dialog.svelte';
	import FilterSheet from './FilterSheet.svelte';
	import HelpSheet from './HelpSheet.svelte';
	import NameSheet from './NameSheet.svelte';
	import Sheet from './Sheet.svelte';
	import SheetAction from './SheetAction.svelte';
	import SyncSheet from './SyncSheet.svelte';
	import TextButton from './TextButton.svelte';
	import { ui } from './ui.svelte';

	let { local }: { local: Local } = $props();

	const sheet = $derived(ui.sheet);
	// A handler reads its sheet's values before this: they are gone after it.
	const close = () => (ui.sheet = undefined);

	let clash: { taken: string; mine: string; other: string; pick: (p: 'mine' | 'other' | undefined) => void } | undefined = $state();

	/** A taken category name asks what frees it; the sheet stays open until a pick lands or is cancelled. */
	async function named(pid: string, name: string, edit: (name: string) => Op): Promise<boolean> {
		const a = await local.perform(pid, edit(name), { quiet: true });
		if (!a.clash) {
			if (a.status !== 'applied' && a.status !== 'unchanged') local.say(a.msg ?? '');
			return a.status === 'applied';
		}
		const taken = a.clash;
		const pick = await new Promise<'mine' | 'other' | undefined>((resolve) => {
			clash = { taken: findCategory(local.project(pid)!, taken.taken)?.name ?? '', mine: taken.this, other: taken.other, pick: resolve };
		});
		clash = undefined;
		if (pick === 'mine') return applied(await local.perform(pid, edit(taken.this), { quiet: true }));
		if (pick === 'other') {
			return (
				applied(await local.perform(pid, { action: 'category.rename', cat: taken.taken, title: taken.other }, { quiet: true })) &&
				applied(await local.perform(pid, edit(name), { quiet: true }))
			);
		}
		return false;
	}

	function applied(a: Answer): boolean {
		if (a.status !== 'applied') local.say(a.msg ?? '');
		return a.status === 'applied';
	}

	async function unfold(pid: string, cat: string) {
		await local.setFolded(pid, cat, false);
	}

	function lineSheet(pid: string, cat: string, type: 'note' | 'separator') {
		const c = findCategory(local.project(pid)!, cat);
		if (c && !addedLineShows(c, local.view(pid).filter, type)) {
			close();
			local.say(`Clear the filter to add a ${type} here`);
		} else ui.sheet = { kind: 'line', pid, cat, type };
	}
</script>

{#if sheet?.kind === 'add' && local.project(sheet.pid)}
	<AddSheet {local} pid={sheet.pid} cat={sheet.cat} after={sheet.after} onclose={close} />
{:else if sheet?.kind === 'filter' && local.project(sheet.pid)}
	<FilterSheet {local} pid={sheet.pid} onclose={close} />
{:else if sheet?.kind === 'sync' && session.sync}
	<SyncSheet sync={session.sync} {local} onclose={close} />
{:else if sheet?.kind === 'help'}
	<HelpSheet onclose={close} />
{:else if sheet?.kind === 'newProject'}
	<NameSheet
		open
		label="New project"
		placeholder="New project"
		action="Add"
		onclose={close}
		onDone={async (name) => {
			const a = await local.perform('', { action: 'project.create', title: name }, { quiet: true });
			if (!applied(a)) return false;
			await goto(app(`/p/${a.row}`));
			return true;
		}}
	/>
{:else if sheet?.kind === 'project' && local.project(sheet.pid)}
	{@const p = local.project(sheet.pid)!}
	{@const pid = sheet.pid}
	<NameSheet
		open
		label="Project"
		initial={p.name}
		placeholder="Project name"
		action="Rename"
		onclose={close}
		onDone={async (name) => applied(await local.perform(pid, { action: 'project.rename', title: name }, { quiet: true }))}
	>
		{#snippet extra()}
			<SheetAction icon="group" label="Group" onclick={() => (ui.sheet = { kind: 'group', pid })} />
			<SheetAction icon="newFolder" label="New category" onclick={() => (ui.sheet = { kind: 'newCategory', pid, parent: '' })} />
			<SheetAction icon="delete" label="Delete project" danger onclick={() => (ui.sheet = { kind: 'deleteProject', pid })} />
		{/snippet}
	</NameSheet>
{:else if sheet?.kind === 'group' && local.project(sheet.pid)}
	{@const p = local.project(sheet.pid)!}
	{@const pid = sheet.pid}
	{@const group = p.group ?? ''}
	<NameSheet
		open
		label="Group"
		initial={group}
		placeholder="Group"
		action="Set"
		blankAllowed
		onclose={close}
		onDone={async (name) => name === group || applied(await local.perform(pid, { action: 'project.group', value: name }))}
	>
		{#snippet extra()}
			{@const groups = projectGroups(local.projects)}
			{#if groups.length}
				<div class="flex flex-wrap gap-2 px-4 pb-4">
					{#each groups as g (g)}
						{@const on = g.toLowerCase() === groupKey(p)}
						<button
							type="button"
							aria-pressed={on}
							class={[
								'text-label-large flex h-8 cursor-pointer items-center rounded-sm border px-3',
								on ? 'border-transparent bg-secondary-container text-on-secondary-container' : 'border-outline-variant text-on-surface-variant hover:bg-on-surface/8'
							]}
							onclick={() => {
								const pick = on ? '' : g;
								if (pick !== group) void local.perform(pid, { action: 'project.group', value: pick });
								close();
							}}
						>
							{g}
						</button>
					{/each}
				</div>
			{/if}
		{/snippet}
	</NameSheet>
{:else if sheet?.kind === 'category' && local.project(sheet.pid) && findCategory(local.project(sheet.pid)!, sheet.cat)}
	{@const p = local.project(sheet.pid)!}
	{@const pid = sheet.pid}
	{@const cat = findCategory(p, sheet.cat)!}
	<NameSheet
		open
		label="Category"
		initial={cat.name}
		placeholder="Category name"
		action="Rename"
		onclose={close}
		onDone={(name) => named(pid, name, (title) => ({ action: 'category.rename', cat: cat.id, title }))}
	>
		{#snippet extra()}
			<SheetAction icon="newFolder" label="New sub-category" onclick={() => (ui.sheet = { kind: 'newCategory', pid, parent: cat.parent_id || cat.id })} />
			{#if versioned(p, cat)}
				<SheetAction
					icon="add"
					label="Add next version"
					onclick={() => {
						void local.perform(pid, { action: 'version.next', cat: cat.id });
						close();
					}}
				/>
			{/if}
			<SheetAction icon="separator" label="Add a separator" onclick={() => lineSheet(pid, cat.id, 'separator')} />
			<SheetAction icon="note" label="Add a note" onclick={() => lineSheet(pid, cat.id, 'note')} />
			<SheetAction
				icon="pin"
				label={cat.pinned ? 'Unpin' : 'Pin'}
				onclick={() => {
					void local.perform(pid, { action: 'category.pin', cat: cat.id, value: String(!cat.pinned) });
					close();
				}}
			/>
			<SheetAction icon="delete" label="Delete category" danger onclick={() => (ui.sheet = { kind: 'deleteCategory', pid, cat: cat.id })} />
		{/snippet}
	</NameSheet>
{:else if sheet?.kind === 'newCategory' && local.project(sheet.pid)}
	{@const pid = sheet.pid}
	{@const parent = sheet.parent}
	<NameSheet
		open
		label={parent ? 'New sub-category' : 'New category'}
		placeholder={parent ? 'New sub-category' : 'New category'}
		action="Add"
		onclose={close}
		onDone={async (name) => {
			const ok = await named(pid, name, (title) => ({ action: 'category.add', title, parent }));
			if (ok && parent) await unfold(pid, parent);
			return ok;
		}}
	/>
{:else if sheet?.kind === 'line' && local.project(sheet.pid)}
	{@const { pid, cat, after, type } = sheet}
	<NameSheet
		open
		label={type === 'note' ? 'New note' : 'New separator'}
		placeholder={type === 'note' ? 'Note' : 'Separator label'}
		action="Add"
		blankAllowed={type === 'separator'}
		onclose={close}
		onDone={async (text) => {
			const a = await local.perform(pid, { action: type === 'note' ? 'note.add' : 'separator.add', cat, ids: after ? [after] : [], title: text }, { quiet: true });
			if (!applied(a)) return false;
			await unfold(pid, cat);
			if (a.row) ui.cursor = 't:' + a.row;
			return true;
		}}
	/>
{:else if sheet?.kind === 'move' && local.project(sheet.pid) && find(local.project(sheet.pid)!, sheet.tid)}
	{@const p = local.project(sheet.pid)!}
	{@const { pid, tid } = sheet}
	{@const here = find(p, tid)!}
	<Sheet open label="Move to" onclose={close}>
		<h2 class="text-title-large px-6 pt-5 pb-2">Move to</h2>
		<div class="pb-2">
			{#each p.categories.filter((c) => c.id !== here.cat.id) as c (c.id)}
				<button
					type="button"
					class="text-body-large flex h-12 w-full cursor-pointer items-center px-6 text-left hover:bg-on-surface/8 focus-visible:bg-on-surface/12 focus-visible:outline-none"
					onclick={() => {
						void local.perform(pid, { action: 'task.move', ids: [tid], cat: c.id });
						close();
					}}
				>
					{categoryPath(p, c)}
				</button>
			{:else}
				<p class="text-body-medium px-6 py-3 text-on-surface-variant">No other category</p>
			{/each}
		</div>
	</Sheet>
{:else if sheet?.kind === 'deleteCategory' && local.project(sheet.pid) && findCategory(local.project(sheet.pid)!, sheet.cat)}
	{@const p = local.project(sheet.pid)!}
	{@const pid = sheet.pid}
	{@const cat = findCategory(p, sheet.cat)!}
	<Dialog open title={`Delete category “${cat.name}”?`} onclose={close}>
		{deletedWith(p, cat)}
		{#snippet actions()}
			<TextButton onclick={close}>Cancel</TextButton>
			<TextButton
				danger
				onclick={() => {
					void local.perform(pid, { action: 'category.delete', cat: cat.id });
					close();
				}}
			>
				Delete
			</TextButton>
		{/snippet}
	</Dialog>
{:else if sheet?.kind === 'deleteProject' && local.project(sheet.pid)}
	{@const p = local.project(sheet.pid)!}
	{@const pid = sheet.pid}
	{@const n = p.categories.reduce((n, c) => n + c.tasks.filter((t) => !isLine(t)).length, 0)}
	<Dialog open title={`Delete project “${p.name}”?`} onclose={close}>
		Its {p.categories.length} categories and {n} tasks go too. This can't be undone.
		{#snippet actions()}
			<TextButton onclick={close}>Cancel</TextButton>
			<TextButton
				danger
				onclick={() => {
					void local.perform(pid, { action: 'project.delete' });
					close();
				}}
			>
				Delete
			</TextButton>
		{/snippet}
	</Dialog>
{/if}

{#if clash}
	{@const c = clash}
	<Dialog open title={`“${c.taken}” is taken`} onclose={() => c.pick(undefined)}>
		Another category here has that name.
		{#snippet actions()}
			<div class="flex flex-col items-end gap-1">
				<TextButton onclick={() => c.pick('mine')}>Name it “{c.mine}”</TextButton>
				<TextButton onclick={() => c.pick('other')}>Rename the other to “{c.other}”</TextButton>
				<TextButton onclick={() => c.pick(undefined)}>Cancel</TextButton>
			</div>
		{/snippet}
	</Dialog>
{/if}
