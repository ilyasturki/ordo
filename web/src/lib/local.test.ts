import 'fake-indexeddb/auto';
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { nodeCore } from '$lib/core/node';
import type { Core } from '$lib/core/core';
import type { Op, Project } from '$lib/core/types';
import { Local } from '$lib/local.svelte';
import { openStore } from '$lib/store';

let stores = 0;
let last: Local | undefined;

async function fresh(): Promise<{ local: Local; reopen: () => Promise<Local> }> {
	const name = `test-${stores++}`;
	const core = await nodeCore();
	// One core serves every test: drop what the last one held.
	for (const p of last?.projects ?? []) core.close(p.id);
	const local = await Local.open(core, await openStore(name));
	last = local;
	return {
		local,
		// Another tab: a core holding nothing yet, the same IndexedDB.
		reopen: async () => {
			for (const p of local.projects) core.close(p.id);
			last = await Local.open(core, await openStore(name));
			return last;
		}
	};
}

const ferry = readFileSync(new URL('../../../assets/demo/ferry.json', import.meta.url), 'utf8');

describe('Local', () => {
	it('keeps a created project and its edits through a reload', async () => {
		const { local, reopen } = await fresh();
		const created = await local.perform('', { action: 'project.create', title: 'Ferry' });
		expect(created.status).toBe('applied');
		const pid = created.row!;
		const cat = await local.perform(pid, { action: 'category.add', title: 'Fix' });
		await local.perform(pid, { action: 'task.add', cat: cat.row, title: 'Dock' });

		const again = await reopen();
		expect(again.projects.map((p) => p.name)).toEqual(['Ferry']);
		expect(again.project(pid)?.categories[0].tasks.map((t) => t.title)).toEqual(['Dock']);
	});

	it('answers a refused edit without keeping anything', async () => {
		const { local } = await fresh();
		const pid = (await local.perform('', { action: 'project.create', title: 'Ferry' })).row!;
		const taken = await local.perform('', { action: 'project.create', title: 'ferry' });
		expect(taken.status).toBe('refused');
		expect(taken.msg).toBe('A project with that name already exists');
		expect(local.projects.map((p) => p.id)).toEqual([pid]);
	});

	it('imports a project file and exports it as ordo writes it', async () => {
		const { local, reopen } = await fresh();
		const id = await local.import(ferry);
		const exported = local.export(id)!;
		expect(exported).toContain('\n  "schema": ');
		const titles = (raw: string) => (JSON.parse(raw) as Project).categories.flatMap((c) => c.tasks.map((t) => t.title));
		expect(titles(exported)).toEqual(titles(ferry));
		await expect(local.import(ferry)).rejects.toThrow('A project with that name already exists');
		expect((await reopen()).project(id)?.name).toBe(JSON.parse(ferry).name);
	});

	it('forgets a deleted project', async () => {
		const { local, reopen } = await fresh();
		const pid = (await local.perform('', { action: 'project.create', title: 'Ferry' })).row!;
		await local.perform(pid, { action: 'project.delete' });
		expect(local.projects).toEqual([]);
		expect((await reopen()).projects).toEqual([]);
	});
});

const strip = (p: Project | undefined) =>
	JSON.parse(JSON.stringify(p ?? null, (k, v) => (k === 'updated_at' || k === 'created_at' || k === 'completion_date' ? undefined : v)));

describe('undo and redo', () => {
	it('take back each kind of edit and do it again', async () => {
		const { local } = await fresh();
		const pid = await local.import(ferry);
		const p = () => local.project(pid)!;
		const first = p().categories[0];
		const parent = p().categories.flatMap((c) => c.tasks).find((t) => p().categories.some((c) => c.tasks.some((x) => x.parent_id === t.id)))!;
		const edits: Op[] = [
			{ action: 'task.add', cat: first.id, title: 'New one' },
			{ action: 'task.title', ids: [first.tasks[0].id], title: 'Renamed', base: first.tasks[0].title },
			{ action: 'status.cycle', ids: [first.tasks[0].id] },
			{ action: 'priority.set', ids: [first.tasks[0].id], value: 'high' },
			{ action: 'tag.set', ids: [first.tasks[0].id], value: 'blue' },
			{ action: 'task.shift', ids: [first.tasks[0].id] },
			{ action: 'task.delete', ids: [parent.id] },
			{ action: 'note.add', cat: first.id, title: 'A note' },
			{ action: 'separator.add', cat: first.id, title: '' },
			{ action: 'category.add', title: 'Later' },
			{ action: 'category.rename', cat: first.id, title: 'Renamed cat' },
			{ action: 'category.pin', cat: p().categories[1].id, value: 'true' },
			{ action: 'category.delete', cat: first.id },
			{ action: 'project.rename', title: 'Boat' },
			{ action: 'project.group', value: 'Work' }
		];
		for (const op of edits) {
			const before = strip(p());
			const answer = await local.perform(pid, op, { quiet: true });
			expect(answer.status, op.action).toBe('applied');
			const after = strip(p());
			expect(await local.undo(), op.action).toBe(true);
			expect(strip(p()), `undo ${op.action}`).toEqual(before);
			expect(await local.redo(), op.action).toBe(true);
			expect(strip(p()), `redo ${op.action}`).toEqual(after);
		}
		expect(await local.undo()).toBe(true);
		expect(p().group ?? '').toBe('');
	});

	it('takes a created project back and brings it again', async () => {
		const { local } = await fresh();
		const pid = (await local.perform('', { action: 'project.create', title: 'Ferry' })).row!;
		await local.undo();
		expect(local.projects).toEqual([]);
		await local.redo();
		expect(local.project(pid)?.name).toBe('Ferry');
	});

	it('offers an Undo with what an edit says', async () => {
		const { local } = await fresh();
		const pid = await local.import(ferry);
		const id = local.project(pid)!.categories[0].tasks[0].id;
		await local.perform(pid, { action: 'task.delete', ids: [id] });
		expect(local.snack?.msg).toBeTruthy();
		expect(local.snack?.action?.label).toBe('Undo');
		local.snack!.action!.run();
		await new Promise((r) => setTimeout(r, 0));
		expect(local.project(pid)!.categories[0].tasks[0].id).toBe(id);
	});
});

describe('views and images', () => {
	it('keeps folds and the filter through a reload', async () => {
		const { local, reopen } = await fresh();
		const pid = await local.import(ferry);
		const cat = local.project(pid)!.categories[0].id;
		await local.setFolded(pid, cat, true);
		await local.setFilter(pid, { off: false, statuses: ['todo'], priorities: [''], categories: [], tags: [] });
		await local.setCurrent(pid);
		const again = await reopen();
		expect(again.view(pid).folds).toEqual([cat]);
		expect(again.view(pid).filter.priorities).toEqual(['']);
		expect(again.current).toBe(pid);
	});

	it('keeps an attached image under its hash and refuses what is not one', async () => {
		const { local } = await fresh();
		const pid = await local.import(ferry);
		const tid = local.project(pid)!.categories[0].tasks[0].id;
		const png = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3]);
		const answer = await local.attach(pid, tid, [new File([png], 'Screen shot.png', { type: 'image/png' })]);
		expect(typeof answer === 'string' ? answer : answer.status).toBe('applied');
		const description = local.project(pid)!.categories[0].tasks[0].description ?? '';
		expect(description).toMatch(/!\[Screen shot\]\(attachments\/[0-9a-f]{32}\.png\)$/);
		expect(await local.attach(pid, tid, [new File(['hello'], 'a.txt')])).toBe('Only png, jpeg, gif and webp images can be attached');
	});
});

describe('Core', () => {
	it('refuses a file a newer ordo wrote', async () => {
		const core: Core = await nodeCore();
		expect(() => core.open('{"schema": 99, "id": "p9", "name": "x", "categories": []}')).toThrow(/newer ordo/);
	});
});
