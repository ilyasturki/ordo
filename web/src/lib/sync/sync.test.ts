import 'fake-indexeddb/auto';
import { readFileSync } from 'node:fs';
import { afterEach, beforeAll, describe, expect, it } from 'vitest';
import { nodeCore } from '$lib/core/node';
import type { Project } from '$lib/core/types';
import { Local } from '$lib/local.svelte';
import { openStore } from '$lib/store';
import { ordo, serve, type Served } from '../../../e2e/serve';
import { Api, parsePairing, parseServer, pairingCode, type Session } from './api';
import { delays, Sync } from './sync.svelte';

const ferry = readFileSync(new URL('../../../../assets/demo/ferry.json', import.meta.url), 'utf8');

const fast = { ...delays, sync: 20, poll: 60_000, firstBackoff: 50, maxBackoff: 200, reconnect: 50, maxReconnect: 200, failuresBeforeTrouble: 2 };

let stores = 0;
const open: { local: Local; sync: Sync }[] = [];
const served: Served[] = [];

beforeAll(() => void ordo(), 300_000);

afterEach(async () => {
	const core = await nodeCore();
	for (const { local, sync } of open.splice(0)) {
		sync.stop();
		for (const p of local.projects) core.close(p.id);
	}
	for (const s of served.splice(0)) await s.close();
});

async function server(seeded = true): Promise<Served> {
	const s = await serve({ seeded });
	served.push(s);
	return s;
}

/** A browser signed in with a bearer token, its copy in a store of its own. */
async function browser(s: Served, name = `sync-${stores++}`): Promise<{ local: Local; sync: Sync; session: Session }> {
	const session: Session = { server: s.url, token: await s.token(), cookie: false };
	const core = await nodeCore();
	let sync: Sync | undefined;
	const local = await Local.open(core, await openStore(name), { synced: true, fetchImage: (n) => sync!.api.image(n) });
	sync = new Sync(local, session, fast);
	open.push({ local, sync });
	return { local, sync, session };
}

/** The server's copy as another client gets it. */
async function onServer(s: Served): Promise<Project[]> {
	const answer = await new Api({ server: s.url, token: await s.token(), cookie: false }).sync([], {}, {});
	return answer.projects.map((p) => p.data!);
}

const titles = (p: Project | undefined) => p?.categories.flatMap((c) => c.tasks.map((t) => t.title)) ?? [];

async function until(check: () => boolean | Promise<boolean>, ms = 5000): Promise<void> {
	const end = Date.now() + ms;
	while (!(await check())) {
		if (Date.now() > end) throw new Error('timed out');
		await new Promise((r) => setTimeout(r, 20));
	}
}

describe('sync', () => {
	it('takes the server projects, and sends an edit back', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		expect(local.projects.map((p) => p.name)).toEqual(['Ordo']);
		const p = local.projects[0];
		await local.perform(p.id, { action: 'task.add', cat: p.categories[0].id, title: 'From the browser' });
		expect(local.pending).toBe(1);
		await sync.sync();
		expect(local.pending).toBe(0);
		expect(titles((await onServer(s))[0])).toContain('From the browser');
	});

	it('keeps edits made while the server is down and sends them once it is back', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		const p = local.projects[0];
		await s.stop();
		await local.perform(p.id, { action: 'task.add', cat: p.categories[0].id, title: 'Offline one' });
		await local.perform(p.id, { action: 'project.rename', title: 'Ordo offline' });
		await sync.sync();
		await sync.sync();
		expect(sync.trouble).toMatchObject({ kind: 'failing', offline: true });
		expect(local.pending).toBe(2);
		await s.start();
		await sync.sync();
		expect(sync.trouble).toBeUndefined();
		const [theirs] = await onServer(s);
		expect(theirs.name).toBe('Ordo offline');
		expect(titles(theirs)).toContain('Offline one');
	});

	it('sends an undo, and a redo, as the edits that take it back', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		const p = local.projects[0];
		const task = p.categories[0].tasks[0];
		await local.perform(p.id, { action: 'task.delete', ids: [task.id] });
		await sync.sync();
		expect(titles((await onServer(s))[0])).not.toContain(task.title);
		await local.undo();
		await sync.sync();
		expect(titles((await onServer(s))[0])[0]).toBe(task.title);
		await local.redo();
		await sync.sync();
		expect(titles((await onServer(s))[0])).not.toContain(task.title);
	});

	it('takes back a created project on the server too', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		const pid = (await local.perform('', { action: 'project.create', title: 'Fresh' })).row!;
		await sync.sync();
		expect((await onServer(s)).map((p) => p.id)).toContain(pid);
		await local.undo();
		await sync.sync();
		expect((await onServer(s)).map((p) => p.id)).not.toContain(pid);
		expect(local.project(pid)).toBeUndefined();
	});

	it('keeps the queue through a reload', async () => {
		const s = await server();
		const name = `sync-${stores++}`;
		const first = await browser(s, name);
		await first.sync.sync();
		const p = first.local.projects[0];
		await s.stop();
		await first.local.perform(p.id, { action: 'task.add', cat: p.categories[0].id, title: 'Kept' });
		first.sync.stop();
		const core = await nodeCore();
		core.close(p.id);
		const again = await Local.open(core, await openStore(name), { synced: true });
		const sync = new Sync(again, first.session, fast);
		open.push({ local: again, sync });
		expect(again.pending).toBe(1);
		await s.start();
		await sync.sync();
		expect(titles((await onServer(s))[0])).toContain('Kept');
	});

	it('puts the edits queued during a round back on the server copy', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		const p = local.projects[0];
		const cat = p.categories[0].id;
		await local.perform(p.id, { action: 'task.add', cat, title: 'Sent' });
		const round = sync.sync();
		await local.perform(p.id, { action: 'task.add', cat, title: 'Queued meanwhile' });
		await round;
		expect(titles(local.project(p.id))).toEqual(expect.arrayContaining(['Sent', 'Queued meanwhile']));
		await sync.sync();
		expect(local.pending).toBe(0);
		expect(titles((await onServer(s))[0])).toEqual(expect.arrayContaining(['Sent', 'Queued meanwhile']));
	});

	it('offers to keep a title typed over one another device changed', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		const p = local.projects[0];
		const task = p.categories[0].tasks[0];
		const other = new Api({ server: s.url, token: await s.token(), cookie: false });
		await other.sync([{ id: 'o1', project: p.id, action: 'task.title', ids: [task.id], title: 'Theirs', base: task.title }], {}, {});
		await local.perform(p.id, { action: 'task.title', ids: [task.id], title: 'Mine', base: task.title }, { quiet: true });
		await sync.sync();
		expect(local.project(p.id)!.categories[0].tasks[0].title).toBe('Theirs');
		expect(local.snack?.action?.label).toBe('Keep mine');
		local.snack!.action!.run();
		await until(() => local.pending === 1);
		await sync.sync();
		expect((await onServer(s))[0].categories[0].tasks[0].title).toBe('Mine');
	});

	it('syncs folds and the filter', async () => {
		const s = await server();
		const a = await browser(s);
		const b = await browser(s);
		await a.sync.sync();
		const p = a.local.projects[0];
		await a.local.setFolded(p.id, p.categories[0].id, true);
		await a.local.setFilter(p.id, { off: false, statuses: ['todo'], priorities: [''], categories: [], tags: [] });
		expect(a.local.viewsWaiting).toBe(1);
		await a.sync.sync();
		expect(a.local.viewsWaiting).toBe(0);
		await b.sync.sync();
		expect(b.local.view(p.id).folds).toEqual([p.categories[0].id]);
		expect(b.local.view(p.id).filter.priorities).toEqual(['']);
	});

	it('imports a project file onto the server', async () => {
		const s = await server(false);
		const { local, sync } = await browser(s);
		await sync.sync();
		const id = await local.import(ferry);
		await sync.sync();
		const [theirs] = await onServer(s);
		expect(theirs.id).toBe(id);
		expect(titles(theirs)).toEqual(titles(JSON.parse(ferry) as Project));
		expect(theirs.categories.map((c) => [c.name, c.parent_id ?? ''])).toEqual(
			(JSON.parse(ferry) as Project).categories.map((c) => [c.name, c.parent_id ?? ''])
		);
	});

	it('uploads an image before the line naming it, and another browser fetches it', async () => {
		const s = await server();
		const a = await browser(s);
		await a.sync.sync();
		const p = a.local.projects[0];
		const tid = p.categories[0].tasks[0].id;
		const png = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d]);
		await a.local.attach(p.id, tid, [new File([png], 'shot.png')]);
		expect(a.local.uploads).toBe(1);
		await a.sync.sync();
		expect(a.local.uploads).toBe(0);
		const name = /attachments\/([0-9a-f]{32}\.png)/.exec((await onServer(s))[0].categories[0].tasks[0].description ?? '')![1];
		const b = await browser(s);
		await b.sync.sync();
		expect(await b.local.imageURL(name)).toMatch(/^blob:/);
		expect(await b.local.imageURL('0'.repeat(32) + '.png')).toBeUndefined();
	});

	it('hears another client’s change on the live stream', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		sync.start();
		const p = local.projects[0];
		await new Promise((r) => setTimeout(r, 200));
		const other = new Api({ server: s.url, token: await s.token(), cookie: false });
		await other.sync([{ id: 'o1', project: p.id, action: 'project.rename', title: 'Renamed elsewhere' }], {}, {});
		await until(() => local.project(p.id)?.name === 'Renamed elsewhere');
	});

	it('drops the token once the server signs this browser out, and keeps what waits', async () => {
		const s = await server();
		const { local, sync } = await browser(s);
		await sync.sync();
		const me = (await sync.devices()).devices.find((d) => d.self)!;
		const other = new Api({ server: s.url, token: await s.token(), cookie: false });
		await other.revoke(me.id);
		const p = local.projects[0];
		await local.perform(p.id, { action: 'task.add', cat: p.categories[0].id, title: 'Waits' });
		await sync.sync();
		expect(sync.trouble).toEqual({ kind: 'expired' });
		expect(sync.session.token).toBe('');
		expect(local.pending).toBe(1);
	});
});

describe('pairing links', () => {
	it('read the server and the code', () => {
		expect(parsePairing('https://ordo.example.com/pair/ABC')).toEqual({ server: 'https://ordo.example.com', code: 'ABC' });
		expect(parsePairing(' http://127.0.0.1:7319/sub/pair/ABC/ ')).toEqual({ server: 'http://127.0.0.1:7319/sub', code: 'ABC' });
		expect(parsePairing('ftp://x/pair/ABC')).toBeUndefined();
		expect(pairingCode('ABC234')).toBe('ABC234');
		expect(pairingCode('not a code')).toBeUndefined();
		expect(parseServer('ordo.example.com')).toBe('https://ordo.example.com');
		expect(parseServer('http://localhost:7319/')).toBe('http://localhost:7319');
		expect(parseServer('ordo')).toBeUndefined();
		expect(parseServer('http://nas:7319')).toBe('http://nas:7319');
		expect(parseServer('https://user:pw@ordo.example.com')).toBeUndefined();
	});
});
