import 'fake-indexeddb/auto';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import type { SyncOp } from '$lib/api';
import type { Core } from '$lib/core';
import type { Project, Task } from '$lib/model';
import { Ordo } from '$lib/state.svelte';
import { nodeCore } from '$lib/testing/core';

const SERVER = 'https://ordo.test';

let core: Core;
let dbs = 0;

beforeAll(async () => {
	core = await nodeCore();
});

function task(id: string, title: string, extra: Partial<Task> = {}): Task {
	return {
		id,
		title,
		status: 'todo',
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z',
		...extra
	};
}

function project(id: string, tasks: Task[]): Project {
	return {
		schema: 1,
		id,
		name: id,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z',
		categories: [{ id: 'c1', name: 'Feature', created_at: '2026-01-01T00:00:00Z', tasks }]
	};
}

type Request = { ops: SyncOp[]; have: Record<string, string> };
type Reply =
	| {
			results?: { id: string; status: string; msg?: string; theirs?: string }[];
			projects: unknown[];
	  }
	| number;

/** A stand-in ordo serve: each sync answers with the next reply, built from the request. */
function serve(...replies: ((req: Request) => Reply | Promise<Reply>)[]) {
	const requests: Request[] = [];
	vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
		if (url === `${SERVER}/api/events`) {
			return new Response(new ReadableStream());
		}
		expect(url).toBe(`${SERVER}/api/sync`);
		expect(new Headers(init.headers).get('Authorization')).toBe('Bearer t');
		const req = JSON.parse(String(init.body)) as Request;
		requests.push(req);
		const next = replies.shift();
		if (next === undefined) {
			throw new Error('no reply left');
		}
		const reply = await next(req);
		if (typeof reply === 'number') {
			return new Response(JSON.stringify({ error: 'nope' }), { status: reply });
		}
		const results = reply.results ?? req.ops.map((op) => ({ id: op.id, status: 'applied' }));
		return Response.json({ protocol: 1, results, projects: reply.projects });
	});
	return requests;
}

async function signedIn(pid: string, tasks: Task[]): Promise<Ordo> {
	const ordo = new Ordo();
	ordo.useCore(core);
	await ordo.boot(`test-${++dbs}`);
	serve(() => ({ projects: [{ id: pid, name: pid, stamp: 's1', data: project(pid, tasks) }] }));
	await ordo.signIn({ server: SERVER, token: 't' });
	return ordo;
}

function statusOf(ordo: Ordo, pid: string, id: string): string | undefined {
	return ordo.project(pid)?.data.categories[0].tasks.find((t) => t.id === id)?.status;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('the store', () => {
	it('shows an edit at once and sends the replayable op it queued', async () => {
		const ordo = await signedIn('p1', [task('a', 'one')]);
		await ordo.edit('p1', { action: 'status.cycle', ids: ['a'] });
		expect(statusOf(ordo, 'p1', 'a')).toBe('in_progress');
		expect(ordo.pending).toBe(1);

		const sent = serve((req) => ({
			projects: [
				{
					id: 'p1',
					name: 'p1',
					stamp: 's2',
					data: project('p1', [task('a', 'one', { status: 'in_progress' })])
				}
			]
		}));
		await ordo.sync();
		expect(sent[0].ops.map((o) => [o.action, o.value, o.project])).toEqual([
			['status.set', 'in_progress', 'p1']
		]);
		expect(sent[0].have).toEqual({});
		expect(ordo.pending).toBe(0);
		expect(ordo.project('p1')?.stamp).toBe('s2');
	});

	it('puts the edits made during a sync back on the copy the server sends', async () => {
		const ordo = await signedIn('p2', [task('a', 'one'), task('b', 'two'), task('c', 'three')]);
		await ordo.edit('p2', { action: 'status.set', ids: ['a'], value: 'completed' });
		let release!: () => void;
		let arrived!: () => void;
		const gate = new Promise<void>((r) => (release = r));
		const inFlight = new Promise<void>((r) => (arrived = r));
		serve(async () => {
			arrived();
			await gate;
			return {
				projects: [
					{
						id: 'p2',
						name: 'p2',
						stamp: 's2',
						data: project('p2', [
							task('a', 'one', { status: 'completed' }),
							task('b', 'two'),
							task('c', 'renamed on the laptop')
						])
					}
				]
			};
		});
		const syncing = ordo.sync();
		await inFlight;
		await ordo.edit('p2', {
			action: 'task.title',
			ids: ['b'],
			title: 'typed meanwhile',
			base: 'two'
		});
		release();
		await syncing;
		const titles = ordo.project('p2')?.data.categories[0].tasks.map((t) => t.title);
		expect(titles).toEqual(['one', 'typed meanwhile', 'renamed on the laptop']);
		expect(statusOf(ordo, 'p2', 'a')).toBe('completed');
		expect(ordo.pending).toBe(1);
	});

	it('names the stamp of each project with nothing queued, so the server can leave it out', async () => {
		const ordo = await signedIn('p3', [task('a', 'one')]);
		const sent = serve(() => ({ projects: [{ id: 'p3', name: 'p3', stamp: 's1' }] }));
		await ordo.sync();
		expect(sent[0].have).toEqual({ p3: 's1' });
		expect(ordo.project('p3')?.data.categories[0].tasks[0].title).toBe('one');
	});

	it('drops a project the server no longer has', async () => {
		const ordo = await signedIn('p4', [task('a', 'one')]);
		serve(() => ({ projects: [] }));
		await ordo.sync();
		expect(ordo.project('p4')).toBeUndefined();
	});

	it('offers to keep a title another device changed first', async () => {
		const ordo = await signedIn('p5', [task('a', 'one')]);
		await ordo.edit('p5', { action: 'task.title', ids: ['a'], title: 'mine', base: 'one' });
		serve((req) => ({
			results: [
				{
					id: req.ops[0].id,
					status: 'conflict',
					msg: 'Changed on another device',
					theirs: 'theirs'
				}
			],
			projects: [{ id: 'p5', name: 'p5', stamp: 's2', data: project('p5', [task('a', 'theirs')]) }]
		}));
		await ordo.sync();
		expect(ordo.project('p5')?.data.categories[0].tasks[0].title).toBe('theirs');
		expect(ordo.snack?.action?.label).toBe('Keep mine');
		ordo.snack?.action?.run();
		await vi.waitFor(() => expect(ordo.pending).toBe(1));
		expect(ordo.project('p5')?.data.categories[0].tasks[0].title).toBe('mine');
	});

	it('undoes a delete by putting the rows back', async () => {
		const ordo = await signedIn('p6', [task('a', 'one'), task('b', 'two')]);
		await ordo.edit('p6', { action: 'task.delete', ids: ['a'] });
		expect(ordo.project('p6')?.data.categories[0].tasks.map((t) => t.id)).toEqual(['b']);
		ordo.snack?.action?.run();
		await vi.waitFor(() =>
			expect(ordo.project('p6')?.data.categories[0].tasks.map((t) => t.id)).toEqual(['a', 'b'])
		);
		expect(ordo.pending).toBe(2);
	});

	it('keeps the data and the queue when the server stops taking the device', async () => {
		const ordo = await signedIn('p7', [task('a', 'one')]);
		await ordo.edit('p7', { action: 'status.cycle', ids: ['a'] });
		serve(() => 401);
		await ordo.sync();
		expect(ordo.signedIn).toBe(false);
		expect(ordo.session?.expired).toBe(true);
		expect(ordo.pending).toBe(1);
		expect(statusOf(ordo, 'p7', 'a')).toBe('in_progress');
	});

	it('creates a project offline and keeps it until the server has it', async () => {
		const ordo = await signedIn('p8', []);
		const pid = await ordo.createProject('Errands');
		expect(pid).not.toBeNull();
		expect(ordo.project(pid!)?.name).toBe('Errands');
		vi.stubGlobal('fetch', () => Promise.reject(new TypeError('offline')));
		await ordo.sync();
		expect(ordo.offline).toBe(true);
		expect(ordo.project(pid!)?.name).toBe('Errands');
	});
});
