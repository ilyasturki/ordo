<script lang="ts">
	import { goto } from '$app/navigation';
	import BulkSheet from '$lib/BulkSheet.svelte';
	import CategoryCard from '$lib/CategoryCard.svelte';
	import FilterSheet from '$lib/FilterSheet.svelte';
	import Icon from '$lib/Icon.svelte';
	import { buildCategories, filtering, newID, openCount, type Category } from '$lib/model';
	import { Board } from '$lib/board.svelte';
	import { ordo } from '$lib/state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Confirm, { type ConfirmSpec } from '$lib/ui/Confirm.svelte';
	import IconButton from '$lib/ui/IconButton.svelte';
	import Menu, { type MenuItem } from '$lib/ui/Menu.svelte';
	import Prompt, { type PromptSpec } from '$lib/ui/Prompt.svelte';
	import { registerOverlay } from '$lib/ui/overlays';
	import { ripple } from '$lib/ui/ripple';

	const entry = $derived(ordo.current);
	const project = $derived(entry?.data);
	const board = new Board(
		() => entry?.id ?? '',
		() => entry?.data
	);

	let searching = $state(false);
	let query = $state('');
	let filterOpen = $state(false);
	let bulkOpen = $state(false);
	let prompt = $state<PromptSpec | null>(null);
	let confirm = $state<ConfirmSpec | null>(null);
	let tabs = $state<HTMLElement | null>(null);

	const filter = $derived(entry ? ordo.filterOf(entry.id) : undefined);
	const narrowed = $derived((filter !== undefined && filtering(filter)) || query.trim() !== '');
	const views = $derived(project ? buildCategories(project, filter, query) : []);

	$effect(() => (board.selecting ? registerOverlay(() => board.clear()) : undefined));
	$effect(() =>
		searching
			? registerOverlay(() => {
					searching = false;
					query = '';
				})
			: undefined
	);

	$effect(() => {
		const id = entry?.id;
		if (id && tabs) {
			tabs
				.querySelector(`[data-tab="${id}"]`)
				?.scrollIntoView({ inline: 'nearest', block: 'nearest' });
		}
	});

	function switchTo(id: string) {
		board.clear();
		board.adding = null;
		ordo.open(id);
		window.scrollTo({ top: 0 });
	}

	function newProject() {
		prompt = {
			title: 'New project',
			label: 'Name',
			confirm: 'Create',
			submit: async (name) => {
				const pid = await ordo.createProject(name);
				if (pid === null) {
					return false;
				}
				switchTo(pid);
			}
		};
	}

	function projectMenu(): MenuItem[] {
		const items: MenuItem[] = [];
		if (entry) {
			const id = entry.id;
			items.push(
				{
					label: 'Rename project',
					icon: 'edit',
					run: () =>
						(prompt = {
							title: 'Rename project',
							label: 'Name',
							value: entry.name,
							confirm: 'Rename',
							submit: async (name) =>
								(await ordo.edit(id, { action: 'project.rename', title: name }, { quiet: true }))
									.status !== 'refused'
						})
				},
				{
					label: 'Delete project',
					icon: 'delete',
					run: () =>
						(confirm = {
							title: `Delete “${entry.name}”?`,
							body: 'Its categories and tasks are deleted on every device. This can’t be undone.',
							confirm: 'Delete',
							run: () => void ordo.edit(id, { action: 'project.delete' })
						})
				}
			);
		}
		items.push(
			{ label: 'Sync now', icon: 'sync', run: () => void ordo.sync() },
			{ label: 'Devices', icon: 'devices', run: () => void goto('/devices') }
		);
		return items;
	}

	function categoryMenu(c: Category): MenuItem[] {
		const items: MenuItem[] = [
			{
				label: 'Rename',
				icon: 'edit',
				run: () =>
					(prompt = {
						title: 'Rename category',
						label: 'Name',
						value: c.name,
						confirm: 'Rename',
						submit: async (name) =>
							(await board.edit({ action: 'category.rename', cat: c.id, title: name })).status !==
							'refused'
					})
			},
			{
				label: 'Add separator',
				icon: 'separator',
				run: () =>
					(prompt = {
						title: 'Add separator',
						label: 'Label (optional)',
						confirm: 'Add',
						allowEmpty: true,
						submit: (title) => void board.edit({ action: 'separator.add', cat: c.id, title })
					})
			}
		];
		if (!c.parent_id) {
			items.push({
				label: 'Add sub-category',
				icon: 'create_new_folder',
				run: () => addCategory(c.id)
			});
		}
		items.push({
			label: 'Select completed',
			icon: 'checklist',
			run: () => {
				for (const t of c.tasks) {
					if (t.status === 'completed' || t.status === 'cancelled') {
						board.selected.add(t.id);
					}
				}
			}
		});
		items.push({
			label: 'Delete category',
			icon: 'delete',
			danger: true,
			run: () =>
				(confirm = {
					title: `Delete “${c.name}”?`,
					body: 'Its tasks and sub-categories are deleted too. This can’t be undone.',
					confirm: 'Delete',
					run: () => void board.edit({ action: 'category.delete', cat: c.id })
				})
		});
		return items;
	}

	function addCategory(parent = '') {
		prompt = {
			title: parent ? 'New sub-category' : 'New category',
			label: 'Name',
			confirm: 'Create',
			submit: async (title) =>
				(await board.edit({ action: 'category.add', new: newID(), parent, title })).status !==
				'refused'
		};
	}

	function removeSelected() {
		void board.onSelected({ action: 'task.delete' });
	}
</script>

{#if board.selecting}
	<header
		class="sticky top-0 z-30 flex h-[calc(64px+var(--safe-t))] items-center gap-1 bg-surface-high px-1 pt-safe-t"
	>
		<IconButton icon="close" label="Stop selecting" onclick={() => board.clear()} />
		<h1 class="flex-1 truncate pl-2 text-[22px]">{board.selected.size} selected</h1>
		<IconButton
			icon="completed"
			label="Cycle status"
			onclick={() => void board.onSelected({ action: 'status.cycle' }, { keep: true })}
		/>
		<IconButton icon="edit" label="Edit selected" onclick={() => (bulkOpen = true)} />
		<IconButton icon="delete" label="Delete selected" onclick={removeSelected} />
	</header>
{:else if searching}
	<header class="sticky top-0 z-30 bg-surface pt-safe-t">
		<div class="flex h-16 items-center gap-1 px-1">
			<IconButton
				icon="arrow_back"
				label="Close search"
				onclick={() => {
					searching = false;
					query = '';
				}}
			/>
			<!-- svelte-ignore a11y_autofocus -->
			<input
				bind:value={query}
				autofocus
				type="search"
				enterkeyhint="search"
				placeholder="Search {entry?.name ?? ''}"
				class="h-12 min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-on-surface-variant"
			/>
			{#if query}
				<IconButton icon="close" label="Clear" onclick={() => (query = '')} />
			{/if}
		</div>
	</header>
{:else}
	<header class="sticky top-0 z-30 bg-surface pt-safe-t">
		<div class="flex h-16 items-center gap-0.5 pr-1 pl-4">
			<h1 class="flex-1 truncate text-[22px] leading-7">Ordo</h1>
			{#if ordo.offline || ordo.syncError}
				<button
					class="flex h-8 items-center gap-1 rounded-full px-2 text-xs text-on-surface-variant"
					onclick={() => void ordo.sync()}
					title={ordo.syncError}
				>
					<Icon name="cloud_off" size={18} />{#if ordo.pending > 0}{ordo.pending}{/if}
				</button>
			{/if}
			<IconButton
				icon="search"
				label="Search"
				onclick={() => (searching = true)}
				disabled={!project}
			/>
			<IconButton
				icon="filter_list"
				label="Filter"
				onclick={() => (filterOpen = true)}
				disabled={!project}
				class={filter && filtering(filter) ? '!text-primary' : ''}
			/>
			<Menu label="More" items={projectMenu()} />
		</div>
		<nav
			bind:this={tabs}
			class="flex [scrollbar-width:none] overflow-x-auto border-b border-outline-variant"
			aria-label="Projects"
		>
			{#each ordo.projects as p (p.id)}
				<button
					use:ripple
					data-tab={p.id}
					class="state-layer relative flex h-12 shrink-0 items-center gap-2 px-4 text-sm font-medium {p.id ===
					entry?.id
						? 'text-primary'
						: 'text-on-surface-variant'}"
					aria-current={p.id === entry?.id ? 'page' : undefined}
					onclick={() => switchTo(p.id)}
				>
					{p.name}
					{#if openCount(p.data) > 0}<span class="text-xs opacity-70">{openCount(p.data)}</span
						>{/if}
					{#if p.id === entry?.id}
						<span class="absolute inset-x-2 bottom-0 h-[3px] rounded-t-full bg-primary"></span>
					{/if}
				</button>
			{/each}
			<button
				use:ripple
				class="state-layer flex h-12 shrink-0 items-center gap-2 px-4 text-sm font-medium text-on-surface-variant"
				onclick={newProject}
			>
				<Icon name="add" size={20} />New project
			</button>
		</nav>
	</header>
{/if}

<main class="mx-auto flex max-w-3xl flex-col gap-3 px-3 pt-3 pb-[calc(var(--safe-b)+96px)]">
	{#if !project}
		<div class="flex flex-col items-center gap-4 px-6 pt-24 text-center text-on-surface-variant">
			{#if ordo.syncing}
				<p>Loading your projects…</p>
			{:else if ordo.projects.length === 0}
				<p>No projects yet.</p>
				<Button variant="filled" icon="add" onclick={newProject}>New project</Button>
			{/if}
		</div>
	{:else}
		{#if filter && filtering(filter) && !searching}
			<div class="flex items-center gap-2 px-1 text-sm text-on-surface-variant">
				<Icon name="filter_list" size={18} />
				<span class="flex-1">Filtered</span>
				<Button
					onclick={() =>
						entry &&
						ordo.setFilter(entry.id, { statuses: [], priorities: [], categories: [], tags: [] })}
					>Clear</Button
				>
			</div>
		{/if}
		{#each views as view (view.cat.id)}
			<CategoryCard {view} {board} {narrowed} menu={categoryMenu(view.cat)} />
		{/each}
		{#if narrowed && views.length === 0}
			<p class="px-6 pt-16 text-center text-on-surface-variant">Nothing matches.</p>
		{/if}
		{#if !narrowed}
			{#if project.categories.length === 0}
				<p class="px-6 pt-16 text-center text-on-surface-variant">
					This project has no categories yet.
				</p>
			{/if}
			<div class="flex justify-center pt-2">
				<Button variant="tonal" icon="playlist_add" onclick={() => addCategory()}
					>New category</Button
				>
			</div>
		{/if}
	{/if}
</main>

{#if project}
	<FilterSheet bind:open={filterOpen} {project} />
	<BulkSheet bind:open={bulkOpen} {board} {project} remove={removeSelected} />
{/if}
<Prompt bind:spec={prompt} />
<Confirm bind:spec={confirm} />
