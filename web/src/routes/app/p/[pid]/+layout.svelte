<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { tick, untrack } from 'svelte';
	import IconButton from '$lib/app/IconButton.svelte';
	import Icon from '$lib/app/Icon.svelte';
	import ProjectList from '$lib/app/ProjectList.svelte';
	import ProjectMenu from '$lib/app/ProjectMenu.svelte';
	import ProjectNav from '$lib/app/ProjectNav.svelte';
	import ProjectTabs from '$lib/app/ProjectTabs.svelte';
	import Sheets from '$lib/app/Sheets.svelte';
	import { onKey } from '$lib/app/keys';
	import { resetList, ui } from '$lib/app/ui.svelte';
	import { lines, search, target } from '$lib/model/board';
	import { active, hasValues, toFile } from '$lib/model/filter';
	import { app } from '$lib/paths';
	import { session } from '$lib/session.svelte';

	let { children } = $props();

	const local = session.local!;
	const pid = $derived(page.params.pid ?? '');
	const project = $derived(local.project(pid));
	const view = $derived(local.view(pid));
	const filterKey = $derived(JSON.stringify(toFile(view.filter)));
	const shown = $derived.by(() => {
		if (!project) return [];
		const held = new Set(ui.held);
		if (ui.query.trim()) return search(lines(project, view.filter, { held }), ui.query);
		return lines(project, view.filter, { peek: ui.peek, held, folds: new Set(view.folds) });
	});
	const taskOpen = $derived(page.route.id !== '/app/p/[pid]');
	const title = $derived(project ? (target(project, view.filter, ui.focus)?.name ?? '') : '');

	let list: HTMLElement | undefined = $state();
	let bar: HTMLElement | undefined = $state();
	let scrolled = $state(false);
	let searchField: HTMLInputElement | undefined = $state();

	$effect(() => {
		const id = pid;
		untrack(() => {
			if (local.project(id)) void local.setCurrent(id);
			resetList();
			ui.cursor = page.params.tid ? 't:' + page.params.tid : '';
			ui.focus = undefined;
			ui.searching = false;
			ui.query = '';
		});
	});

	$effect(() => {
		filterKey;
		resetList();
	});

	$effect(() => {
		if (!project && !local.projects.some((p) => p.id === pid)) {
			const next = local.projects[0];
			void goto(next ? app(`/p/${next.id}`) : app(), { replaceState: true });
		}
	});

	$effect(() => {
		if (ui.searching) void tick().then(() => searchField?.focus());
	});

	// The category with the most rows in sight names the bar; a jump holds it until the next scroll.
	function measure() {
		if (!list) return;
		const box = list.getBoundingClientRect();
		const top = Math.max(box.top, bar && getComputedStyle(bar).position === 'sticky' ? bar.getBoundingClientRect().bottom : 0);
		const bottom = Math.min(box.bottom, innerHeight);
		const cats = new Map<string, string>();
		for (const l of shown) {
			cats.set(l.key, l.cat);
			if (l.kind === 'block') for (const c of l.children) cats.set('t:' + c.task.id, l.cat);
		}
		const pixels = new Map<string, number>();
		for (const el of list.querySelectorAll<HTMLElement>('[data-key]')) {
			const cat = cats.get(el.dataset.key!);
			if (!cat) continue;
			const r = el.getBoundingClientRect();
			const seen = Math.min(r.bottom, bottom) - Math.max(r.top, top);
			if (seen > 0) pixels.set(cat, (pixels.get(cat) ?? 0) + seen);
		}
		let best: string | undefined;
		for (const [cat, n] of pixels) if (best === undefined || n > pixels.get(best)!) best = cat;
		ui.focus = best;
		scrolled = (list.scrollTop || scrollY) > 0;
	}

	let frame = 0;
	function onScroll() {
		cancelAnimationFrame(frame);
		frame = requestAnimationFrame(measure);
	}

	$effect(() => {
		shown;
		void tick().then(measure);
	});

	$effect(() => {
		const cat = ui.jump;
		if (!cat || !list) return;
		void tick().then(() => {
			list?.querySelector(`[data-key="c:${cat}"]`)?.scrollIntoView({ block: 'start' });
			ui.cursor = 'c:' + cat;
			ui.jump = undefined;
		});
	});

	function keydown(e: KeyboardEvent) {
		if (project && onKey(e, { local, project, lines: shown, taskOpen })) e.preventDefault();
	}

	function closeSearch() {
		ui.searching = false;
		ui.query = '';
	}
</script>

<svelte:window onkeydown={keydown} onscroll={onScroll} onresize={onScroll} />

{#if project}
	<div class="wide:grid wide:h-dvh wide:grid-cols-[15rem_minmax(0,1fr)_minmax(0,1fr)] wide:overflow-hidden">
		<div class="hidden border-r border-outline-variant wide:block wide:min-h-0">
			<ProjectNav {local} {pid} />
		</div>

		<section
			bind:this={list}
			onscroll={onScroll}
			aria-label="Tasks"
			class={['relative min-w-0 wide:min-h-0 wide:overflow-y-auto', taskOpen && 'max-wide:hidden']}
		>
			<header
				bind:this={bar}
				class={['sticky top-0 z-20 pt-[env(safe-area-inset-top)]', scrolled && !ui.searching ? 'bg-surface-container' : 'bg-surface']}
			>
				{#if ui.searching}
					<div class="flex h-16 items-center px-2">
						<label class="flex h-12 min-w-0 flex-1 items-center rounded-full bg-surface-container-high pr-1">
							<IconButton icon="back" label="Close search" onclick={closeSearch} />
							<input
								bind:this={searchField}
								bind:value={ui.query}
								type="search"
								placeholder={`Search ${project.name}`}
								aria-label="Search"
								enterkeyhint="search"
								onkeydown={(e) => {
									if (e.key === 'Escape') {
										e.preventDefault();
										closeSearch();
									} else if (e.key === 'Enter') searchField?.blur();
								}}
								class="text-body-large min-w-0 flex-1 bg-transparent outline-none placeholder:text-on-surface-variant [&::-webkit-search-cancel-button]:hidden"
							/>
							{#if ui.query}<IconButton icon="close" label="Clear" onclick={() => (ui.query = '')} />{/if}
						</label>
					</div>
				{:else}
					<div class="flex h-16 items-center pr-1 pl-4">
						<h1 class="text-title-large min-w-0 flex-1 truncate" aria-live="polite">
							<span class="sr-only">{project.name}: </span>{title}
						</h1>
						{#if session.sync?.trouble}
							<IconButton icon="cloudOff" label="Not synced" tone="error" onclick={() => (ui.sheet = { kind: 'sync' })} />
						{/if}
						<IconButton icon="search" label="Search" onclick={() => (ui.searching = true)} />
						<IconButton
							icon={hasValues(view.filter) && !active(view.filter) ? 'filterOff' : 'filter'}
							label={active(view.filter) ? 'Filter, on' : hasValues(view.filter) ? 'Filter, off' : 'Filter'}
							tone={active(view.filter) ? 'primary' : 'muted'}
							onclick={() => (ui.sheet = { kind: 'filter', pid })}
						/>
						<ProjectMenu {local} {pid} />
					</div>
					<div class="wide:hidden"><ProjectTabs {local} {pid} /></div>
				{/if}
			</header>

			<ProjectList {local} {project} lines={shown} top={() => bar?.getBoundingClientRect().bottom ?? 0} />

			{#if project.categories.length > 0 && !ui.searching}
				<button
					type="button"
					aria-label="Add a task"
					title="Add a task (a)"
					class={[
						'fixed right-4 z-30 grid size-14 cursor-pointer place-items-center rounded-lg bg-primary-container text-on-primary-container shadow-md transition-[bottom] outline-primary hover:shadow-lg focus-visible:outline-2 wide:sticky wide:ml-auto wide:mr-4 wide:-mt-14',
						local.snack ? 'bottom-20' : 'bottom-[max(1rem,env(safe-area-inset-bottom))]',
						'wide:bottom-4'
					]}
					onclick={() => {
						const cat = target(project, view.filter, ui.focus);
						if (cat) ui.sheet = { kind: 'add', pid, cat: cat.id };
					}}
				>
					<Icon name="add" />
				</button>
			{/if}
		</section>

		<div class={['min-w-0 wide:min-h-0 wide:p-2 wide:pl-0', !taskOpen && 'max-wide:hidden']}>
			{@render children()}
		</div>
	</div>

	<Sheets {local} />
{/if}
