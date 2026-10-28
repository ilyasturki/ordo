import { Api, Offline, ServerError, type ProjectState, type SyncResult } from '$lib/api';
import { loadCore, type Core } from '$lib/core';
import { openStore, type DB, type Queued, type Session, type StoredProject } from '$lib/db';
import { NO_FILTER, newID, type Answer, type Filter, type Op, type Project } from '$lib/model';

export type Entry = {
	id: string;
	name: string;
	stamp: string;
	data: Project;
	error?: string;
};

export type Snack = {
	key: number;
	msg: string;
	action?: { label: string; run: () => void };
};

type Prefs = {
	current: string;
	folded: Record<string, boolean>;
	doneOpen: Record<string, boolean>;
	filters: Record<string, Filter>;
};

const PREFS = 'ordo.prefs';
const SYNC_DELAY_MS = 400;
const POLL_MS = 30_000;
const MAX_BACKOFF_MS = 60_000;

function readPrefs(): Prefs {
	const fallback: Prefs = { current: '', folded: {}, doneOpen: {}, filters: {} };
	try {
		const raw = localStorage.getItem(PREFS);
		return raw === null ? fallback : { ...fallback, ...(JSON.parse(raw) as Partial<Prefs>) };
	} catch {
		return fallback;
	}
}

function byName(a: Entry, b: Entry): number {
	return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
}

function entryOf(s: StoredProject): Entry | null {
	try {
		return {
			id: s.id,
			name: s.name,
			stamp: s.stamp,
			data: JSON.parse(s.raw) as Project,
			error: s.error
		};
	} catch {
		return null;
	}
}

class Ordo {
	#db: DB | undefined;
	#core: Core | undefined;
	#opened = new Set<string>();
	#queue: Queued[] = [];
	#seq = 0;
	#timer: ReturnType<typeof setTimeout> | undefined;
	#inflight: Promise<void> | undefined;
	#again = false;
	#backoff = 0;
	#snackKey = 0;

	booted = $state(false);
	session = $state<Session | null>(null);
	projects = $state.raw<Entry[]>([]);
	pending = $state(0);
	syncing = $state(false);
	offline = $state(false);
	syncError = $state('');
	lastSync = $state(0);
	snack = $state<Snack | null>(null);
	prefs = $state<Prefs>(readPrefs());

	get api(): Api | null {
		return this.session === null || this.session.expired ? null : new Api(this.session);
	}

	get signedIn(): boolean {
		return this.session !== null && !this.session.expired;
	}

	get current(): Entry | undefined {
		return this.projects.find((p) => p.id === this.prefs.current) ?? this.projects[0];
	}

	project(id: string): Entry | undefined {
		return this.projects.find((p) => p.id === id);
	}

	savePrefs(): void {
		try {
			localStorage.setItem(PREFS, JSON.stringify(this.prefs));
		} catch {
			/* private mode */
		}
	}

	open(id: string): void {
		this.prefs.current = id;
		this.savePrefs();
	}

	filterOf(id: string): Filter {
		return this.prefs.filters[id] ?? NO_FILTER;
	}

	setFilter(id: string, f: Filter): void {
		this.prefs.filters[id] = f;
		this.savePrefs();
	}

	toast(msg: string, action?: Snack['action']): void {
		this.snack = { key: ++this.#snackKey, msg, action };
	}

	async boot(): Promise<void> {
		void loadCore().then(
			(core) => (this.#core = core),
			() => undefined
		);
		const db = await openStore();
		this.#db = db;
		this.session = ((await db.get('meta', 'session')) as Session | undefined) ?? null;
		const stored = await db.getAll('projects');
		this.projects = stored
			.map(entryOf)
			.filter((e) => e !== null)
			.sort(byName);
		this.#queue = await db.getAll('queue');
		this.#seq = this.#queue.at(-1)?.seq ?? 0;
		this.pending = this.#queue.length;
		this.booted = true;
		if (this.signedIn) {
			void this.sync();
		}
	}

	async core(): Promise<Core> {
		this.#core ??= await loadCore();
		return this.#core;
	}

	#ensureOpen(core: Core, e: Entry): void {
		if (!this.#opened.has(e.id)) {
			core.open(JSON.stringify(e.data));
			this.#opened.add(e.id);
		}
	}

	#setEntry(e: Entry): void {
		const rest = this.projects.filter((p) => p.id !== e.id);
		this.projects = [...rest, e].sort(byName);
	}

	#raw(core: Core, pid: string): string {
		const raw = core.project(pid);
		if (raw === null) {
			throw new Error('The project is not open');
		}
		return raw;
	}

	/** Runs a user's edit on the local copy and queues what the server replays. */
	async edit(pid: string, op: Op, opts: { quiet?: boolean } = {}): Promise<Answer> {
		const core = await this.core();
		const known = this.project(pid);
		if (known !== undefined) {
			this.#ensureOpen(core, known);
		}
		const answer = core.perform(pid, op);
		if (answer.status !== 'applied' || answer.queue === undefined) {
			if (answer.status !== 'unchanged' && answer.msg) {
				this.toast(answer.msg);
			}
			return answer;
		}
		this.#land(core, pid, [answer.queue]);
		if (!opts.quiet && answer.msg) {
			const undo = answer.undo;
			this.toast(
				answer.msg,
				undo && undo.length > 0
					? { label: 'Undo', run: () => void this.replay(pid, undo) }
					: undefined
			);
		}
		return answer;
	}

	/** Applies ops as they are, an undo's, and queues those that changed something. */
	async replay(pid: string, ops: Op[]): Promise<void> {
		const core = await this.core();
		const landed = ops.filter((op) => core.replay(pid, op).status === 'applied');
		if (landed.length > 0) {
			this.#land(core, pid, landed);
		}
	}

	#land(core: Core, pid: string, ops: Op[]): void {
		const db = this.#db;
		const queued = ops.map((op) => ({ seq: ++this.#seq, qid: newID(), project: pid, op }));
		this.#queue.push(...queued);
		this.pending = this.#queue.length;
		const deleting = ops.some((op) => op.action === 'project.delete');
		const tx = db?.transaction(['projects', 'queue'], 'readwrite');
		if (deleting) {
			core.close(pid);
			this.#opened.delete(pid);
			this.projects = this.projects.filter((p) => p.id !== pid);
			void tx?.objectStore('projects').delete(pid);
		} else {
			const raw = this.#raw(core, pid);
			const data = JSON.parse(raw) as Project;
			const prev = this.project(pid);
			this.#opened.add(pid);
			this.#setEntry({
				id: pid,
				name: data.name,
				stamp: prev?.stamp ?? '',
				data,
				error: prev?.error
			});
			void tx
				?.objectStore('projects')
				.put({ id: pid, name: data.name, stamp: prev?.stamp ?? '', raw });
		}
		for (const q of queued) {
			void tx?.objectStore('queue').put(q);
		}
		this.#schedule(SYNC_DELAY_MS);
	}

	async createProject(name: string): Promise<string | null> {
		const pid = newID();
		const answer = await this.edit(pid, { action: 'project.create', title: name }, { quiet: true });
		return answer.status === 'applied' ? pid : null;
	}

	#schedule(ms: number): void {
		clearTimeout(this.#timer);
		if (this.signedIn) {
			this.#timer = setTimeout(() => void this.sync(), ms);
		}
	}

	async sync(): Promise<void> {
		if (this.#inflight !== undefined) {
			this.#again = true;
			return this.#inflight;
		}
		this.#inflight = this.#run().finally(() => {
			this.#inflight = undefined;
			if (this.#again) {
				this.#again = false;
				void this.sync();
			}
		});
		return this.#inflight;
	}

	async #run(): Promise<void> {
		const api = this.api;
		const db = this.#db;
		if (api === null || db === undefined) {
			return;
		}
		clearTimeout(this.#timer);
		this.syncing = true;
		try {
			await this.#upload(api, db);
			const sent = this.#queue.slice();
			const last = sent.at(-1)?.seq ?? this.#seq;
			const busy = new Set(sent.map((q) => q.project));
			const have: Record<string, string> = {};
			for (const p of this.projects) {
				if (!busy.has(p.id) && p.stamp !== '' && p.error === undefined) {
					have[p.id] = p.stamp;
				}
			}
			const answer = await api.sync(
				sent.map((q) => ({ ...q.op, id: q.qid, project: q.project })),
				have
			);
			const core = await this.core();
			this.#adopt(core, db, answer.projects, last);
			this.#report(sent, answer.results);
			this.offline = false;
			this.syncError = '';
			this.lastSync = Date.now();
			this.#backoff = 0;
			this.#schedule(this.#queue.length > 0 ? SYNC_DELAY_MS : POLL_MS);
		} catch (err) {
			this.#failed(err);
		} finally {
			this.syncing = false;
		}
	}

	#failed(err: unknown): void {
		if (err instanceof ServerError && err.status === 401) {
			this.expire();
			return;
		}
		this.offline = err instanceof Offline;
		this.syncError = err instanceof Error ? err.message : String(err);
		this.#backoff = Math.min(MAX_BACKOFF_MS, this.#backoff === 0 ? 2000 : this.#backoff * 2);
		this.#schedule(this.#backoff);
	}

	/** Takes the server's copy of each project, with the edits queued since the request put back on top. */
	#adopt(core: Core, db: DB, states: ProjectState[], last: number): void {
		this.#queue = this.#queue.filter((q) => q.seq > last);
		this.pending = this.#queue.length;
		const later = this.#queue;
		const onServer = new Set(states.map((s) => s.id));
		const tx = db.transaction(['projects', 'queue'], 'readwrite');
		void tx.objectStore('queue').delete(IDBKeyRange.upperBound(last));
		const next: Entry[] = [];
		for (const st of states) {
			const mine = this.project(st.id);
			if (st.data === undefined) {
				if (mine !== undefined) {
					const e = { ...mine, error: st.error };
					next.push(e);
					if (st.error !== mine.error) {
						void tx.objectStore('projects').put({
							id: e.id,
							name: e.name,
							stamp: e.stamp,
							raw: JSON.stringify(e.data),
							error: e.error
						});
					}
				}
				continue;
			}
			core.open(JSON.stringify(st.data));
			this.#opened.add(st.id);
			for (const q of later) {
				if (q.project === st.id) {
					core.replay(st.id, q.op);
				}
			}
			const raw = this.#raw(core, st.id);
			const data = JSON.parse(raw) as Project;
			next.push({ id: st.id, name: data.name, stamp: st.stamp, data });
			void tx.objectStore('projects').put({ id: st.id, name: data.name, stamp: st.stamp, raw });
		}
		const createdSince = new Set(
			later.filter((q) => q.op.action === 'project.create').map((q) => q.project)
		);
		for (const p of this.projects) {
			if (onServer.has(p.id)) {
				continue;
			}
			if (createdSince.has(p.id)) {
				next.push(p);
				continue;
			}
			core.close(p.id);
			this.#opened.delete(p.id);
			void tx.objectStore('projects').delete(p.id);
		}
		this.projects = next.sort(byName);
	}

	#report(sent: Queued[], results: SyncResult[]): void {
		for (const r of results) {
			const q = sent.find((s) => s.qid === r.id);
			if (q === undefined || r.status === 'applied' || r.status === 'unchanged') {
				continue;
			}
			if (r.status === 'conflict' && r.theirs !== undefined) {
				const mine = { ...q.op, base: r.theirs };
				this.toast(r.msg ?? 'Changed on another device', {
					label: 'Keep mine',
					run: () => void this.edit(q.project, mine)
				});
				continue;
			}
			if (r.msg) {
				this.toast(r.msg);
			}
		}
	}

	async #upload(api: Api, db: DB): Promise<void> {
		const images = await db.getAll('images');
		for (const img of images) {
			if (img.pending) {
				await api.putImage(img.name, img.blob);
				await db.put('images', { ...img, pending: false });
			}
		}
	}

	/** An image the app stored, or fetched from the server and kept: images download when viewed. */
	async image(name: string): Promise<Blob | null> {
		const db = this.#db;
		const held = await db?.get('images', name);
		if (held !== undefined) {
			return held.blob;
		}
		const api = this.api;
		if (api === null || db === undefined) {
			return null;
		}
		try {
			const blob = await api.image(name);
			await db.put('images', { name, blob, pending: false });
			return blob;
		} catch {
			return null;
		}
	}

	async keepImage(name: string, blob: Blob): Promise<void> {
		await this.#db?.put('images', { name, blob, pending: true });
	}

	async signIn(session: Session): Promise<void> {
		const db = this.#db;
		if (db === undefined) {
			return;
		}
		const previous = this.session;
		if (previous !== null && previous.server !== session.server) {
			await this.#wipe(db);
		}
		this.session = session;
		await db.put('meta', session, 'session');
		await this.sync();
	}

	expire(): void {
		if (this.session === null) {
			return;
		}
		this.session = { ...this.session, expired: true };
		void this.#db?.put('meta', $state.snapshot(this.session), 'session');
	}

	async signOut(): Promise<void> {
		const db = this.#db;
		try {
			await this.api?.logout();
		} catch {
			/* signed out locally either way */
		}
		if (db !== undefined) {
			await this.#wipe(db);
			await db.delete('meta', 'session');
		}
		this.session = null;
	}

	async #wipe(db: DB): Promise<void> {
		const tx = db.transaction(['projects', 'queue', 'images'], 'readwrite');
		await Promise.all([
			tx.objectStore('projects').clear(),
			tx.objectStore('queue').clear(),
			tx.objectStore('images').clear(),
			tx.done
		]);
		for (const id of this.#opened) {
			this.#core?.close(id);
		}
		this.#opened.clear();
		this.#queue = [];
		this.pending = 0;
		this.projects = [];
	}
}

export const ordo = new Ordo();
