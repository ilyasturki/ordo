import 'fake-indexeddb/auto';
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { nodeCore } from '$lib/core/node';
import type { Core } from '$lib/core/core';
import type { Project } from '$lib/core/types';
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

describe('Core', () => {
	it('refuses a file a newer ordo wrote', async () => {
		const core: Core = await nodeCore();
		expect(() => core.open('{"schema": 99, "id": "p9", "name": "x", "categories": []}')).toThrow(/newer ordo/);
	});
});
