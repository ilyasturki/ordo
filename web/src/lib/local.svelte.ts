import type { Core } from '$lib/core/core';
import type { Answer, Category, Op, Project } from '$lib/core/types';
import { fromFile, noFilter, pruned, toFile, type Filter } from '$lib/model/filter';
import { rankProjects } from '$lib/model/project';
import type { DB, Queued } from '$lib/store';
import type { ProjectState, SyncOp, ViewFile } from '$lib/sync/api';

export type View = { folds: string[]; filter: Filter };

export type Snack = { id: number; msg: string; action?: { label: string; run: () => void } };

/** One edit the history can take back: the ops that undo it, and its own replayable op to redo it. */
type Step = { pid: string; forward: Op; back: Op[] };

/** What a round sends, and what adopting its answer needs to know of it. */
export type Outgoing = { ops: SyncOp[]; have: Record<string, string>; views: Record<string, ViewFile>; sent: Queued[]; last: number };

type Options = {
	/** A copy of a server's projects: every edit is queued for it. */
	synced?: boolean;
	fetchImage?: (name: string) => Promise<{ bytes: ArrayBuffer; type: string } | undefined>;
};

const emptyView: View = { folds: [], filter: noFilter };

/** The projects kept in this browser: every edit runs through the core, then the project goes back to IndexedDB. */
export class Local {
	/** In the order the tabs show them: ranked once at start, new ones after. */
	projects: Project[] = $state([]);
	/** Stored projects the core refused to open, kept untouched. */
	unreadable: { id: string; error: string }[] = $state([]);
	views: Record<string, View> = $state({});
	current = $state('');
	snack: Snack | undefined = $state();
	/** Edits, images and views the server hasn't had yet. */
	pending = $state(0);
	uploads = $state(0);
	viewsWaiting = $state(0);
	/** Projects whose file the server can't load, by ID: their copy here stays as it was. */
	broken: Record<string, string> = $state({});
	readonly synced: boolean;
	onWaiting: () => void = () => {};
	readonly #core: Core;
	readonly #db: DB;
	readonly #fetchImage: Options['fetchImage'];
	#order: string[] = [];
	#opened: Record<string, number> = {};
	#history: Step[] = [];
	#future: Step[] = [];
	#snacks = 0;
	#viewing = Promise.resolve();
	readonly #urls = new Map<string, string>();
	#stamps = new Map<string, string>();
	#queue: Queued[] = [];
	#seq = 0;
	#viewEdits: Record<string, ViewFile> = {};
	#unsent: string[] = [];

	private constructor(core: Core, db: DB, options: Options) {
		this.#core = core;
		this.#db = db;
		this.synced = options.synced ?? false;
		this.#fetchImage = options.fetchImage;
	}

	static async open(core: Core, db: DB, options: Options = {}): Promise<Local> {
		const local = new Local(core, db, options);
		const ids: string[] = [];
		for (const stored of await db.getAll('projects')) {
			try {
				ids.push(core.open(stored.raw));
				local.#stamps.set(stored.id, stored.stamp);
				if (stored.error) local.broken[stored.id] = stored.error;
			} catch (error) {
				local.unreadable.push({ id: stored.id, error: String(error) });
			}
		}
		for (const v of await db.getAll('views')) {
			local.views[v.id] = { folds: v.folds, filter: fromFile(v.filter) };
		}
		local.#opened = ((await db.get('meta', 'opened')) as Record<string, number> | undefined) ?? {};
		local.current = ((await db.get('meta', 'current')) as string | undefined) ?? '';
		local.#queue = await db.getAll('queue');
		local.#seq = local.#queue.at(-1)?.seq ?? 0;
		local.#viewEdits = ((await db.get('meta', 'viewEdits')) as Record<string, ViewFile> | undefined) ?? {};
		local.#unsent = ((await db.get('meta', 'unsent')) as string[] | undefined) ?? [];
		local.#counted();
		local.#order = rankProjects(local.#read(ids), local.#opened).map((p) => p.id);
		local.#show(ids);
		return local;
	}

	project(pid: string): Project | undefined {
		return this.projects.find((p) => p.id === pid);
	}

	/** The project as the core holds it this instant, ahead of the copy on screen while an edit is being kept. */
	held(pid: string): Project | undefined {
		const raw = this.#core.project(pid);
		return raw === null ? undefined : (JSON.parse(raw) as Project);
	}

	view(pid: string): View {
		const v = this.views[pid] ?? emptyView;
		const p = this.project(pid);
		return p ? { folds: v.folds, filter: pruned(v.filter, p) } : v;
	}

	/** The project on screen, which the next start opens on. */
	async setCurrent(pid: string): Promise<void> {
		if (pid === this.current) return;
		this.#opened = { ...this.#opened, [pid]: Date.now() };
		await this.#db.put('meta', pid, 'current');
		await this.#db.put('meta', this.#opened, 'opened');
		this.current = pid;
	}

	setFolded(pid: string, id: string, folded: boolean): Promise<void> {
		return this.#setView(pid, (v) =>
			v.folds.includes(id) === folded ? undefined : { folds: folded ? [...v.folds, id] : v.folds.filter((f) => f !== id) }
		);
	}

	setFilter(pid: string, filter: Filter): Promise<void> {
		return this.#setView(pid, () => ({ filter }));
	}

	/** One change at a time, each written before it shows, so a reload never loses one already on screen. */
	#setView(pid: string, change: (view: View) => Partial<View> | undefined): Promise<void> {
		const run = this.#viewing.then(async () => {
			const edit = change(this.view(pid));
			if (!edit) return;
			const view = { ...this.view(pid), ...edit };
			const tx = this.#db.transaction(['views', 'meta'], 'readwrite');
			void tx.objectStore('views').put({ id: pid, folds: [...view.folds].sort(), filter: toFile(view.filter) });
			if (this.synced) {
				const sent: ViewFile = {};
				if (edit.folds) sent.folds = [...edit.folds].sort();
				if (edit.filter) sent.filter = toFile(edit.filter);
				this.#viewEdits = { ...this.#viewEdits, [pid]: { ...this.#viewEdits[pid], ...sent } };
				void tx.objectStore('meta').put($state.snapshot(this.#viewEdits), 'viewEdits');
				this.#counted();
			}
			await tx.done;
			this.views[pid] = { ...(this.views[pid] ?? emptyView), ...edit };
			if (this.synced) this.onWaiting();
		});
		this.#viewing = run.catch(() => {});
		return run;
	}

	/**
	 * Runs a user's edit and keeps what it changed. A quiet edit says nothing; another says what it
	 * did with an Undo, or `said` when the core answers no message. A project.create given no pid
	 * answers the new one as row.
	 */
	async perform(pid: string, op: Op, { quiet = false, said = '' } = {}): Promise<Answer> {
		const answer = this.#core.perform(pid, op);
		const target = op.action === 'project.create' && pid === '' ? (answer.row ?? '') : pid;
		if (answer.status === 'applied') {
			await this.#land(target, answer.queue ? [answer.queue] : []);
		}
		const step = this.#record(target, answer);
		if (answer.status === 'conflict' && answer.theirs !== undefined) {
			this.offerMine(pid, op, answer.theirs, answer.msg);
		} else if (!quiet) {
			this.say(answer.msg || said, step);
		}
		return answer;
	}

	#record(pid: string, answer: Answer): Step | undefined {
		if (answer.status !== 'applied' || !answer.queue || !answer.undo?.length) return undefined;
		const step = { pid, forward: answer.queue, back: answer.undo };
		this.#history.push(step);
		this.#future = [];
		return step;
	}

	say(msg: string, step?: Step): void {
		if (msg === '') return;
		this.snack = { id: ++this.#snacks, msg, action: step && { label: 'Undo', run: () => void this.#takeBack(step) } };
	}

	offerMine(pid: string, op: Op, theirs: string, msg?: string): void {
		const run = () => void this.perform(pid, { ...op, base: theirs });
		this.snack = { id: ++this.#snacks, msg: msg || 'Changed on another device', action: { label: 'Keep mine', run } };
	}

	get canUndo(): boolean {
		return this.#history.length > 0;
	}

	/** Takes the last edit back, as the TUI's u does. */
	async undo(): Promise<boolean> {
		const step = this.#history.at(-1);
		if (!step) {
			this.say('Nothing to undo');
			return false;
		}
		return this.#takeBack(step);
	}

	async #takeBack(step: Step): Promise<boolean> {
		this.#history = this.#history.filter((s) => s !== step);
		const landed: Op[] = [];
		for (const op of step.back) {
			const answer = this.#core.perform(step.pid, op);
			if (answer.status === 'applied') landed.push(answer.queue ?? op);
		}
		if (landed.length === 0) {
			this.say("Can't undo that: it changed since");
			return false;
		}
		await this.#land(step.pid, landed);
		this.#future.push(step);
		this.say('Undone');
		return true;
	}

	/** Does again what the last undo took back, as the TUI's ctrl+r does. */
	async redo(): Promise<boolean> {
		const step = this.#future.pop();
		if (!step) {
			this.say('Nothing to redo');
			return false;
		}
		const answer = this.#core.perform(step.pid, step.forward);
		if (answer.status !== 'applied') {
			this.say("Can't redo that: it changed since");
			return false;
		}
		await this.#land(step.pid, [answer.queue ?? step.forward]);
		if (answer.queue && answer.undo?.length) this.#history.push({ pid: step.pid, forward: answer.queue, back: answer.undo });
		this.say('Redone');
		return true;
	}

	/** Adds a project file as a new project and answers its ID; throws on a file ordo can't import. */
	async import(raw: string): Promise<string> {
		const id = this.#core.import(raw);
		await this.#land(id, this.synced ? importOps(this.held(id)!) : []);
		return id;
	}

	/** The project as ordo writes its file. */
	export(pid: string): string | undefined {
		return this.#core.export(pid) ?? undefined;
	}

	/** Keeps each image and adds its line to the task's description. */
	async attach(pid: string, tid: string, files: File[]): Promise<Answer | string> {
		const lines: string[] = [];
		for (const file of files) {
			if (file.size > maxImage) return 'Images over 25 MB can’t be attached';
			const bytes = await file.arrayBuffer();
			const ext = extension(new Uint8Array(bytes));
			if (!ext) return 'Only png, jpeg, gif and webp images can be attached';
			const digest = new Uint8Array(await crypto.subtle.digest('SHA-256', bytes));
			const name = [...digest.slice(0, 16)].map((b) => b.toString(16).padStart(2, '0')).join('') + '.' + ext;
			try {
				if (!(await this.#db.get('images', name))) {
					const tx = this.#db.transaction(['images', 'meta'], 'readwrite');
					void tx.objectStore('images').put({ name, bytes, type: { jpg: 'image/jpeg' }[ext] ?? 'image/' + ext, pending: this.synced });
					if (this.synced && !this.#unsent.includes(name)) {
						this.#unsent = [...this.#unsent, name];
						void tx.objectStore('meta').put(this.#unsent, 'unsent');
					}
					await tx.done;
					this.#counted();
				}
			} catch {
				return 'This browser can’t keep that image';
			}
			lines.push(`![${altOf(file.name)}](attachments/${name})`);
		}
		return this.perform(pid, { action: 'task.attach', ids: [tid], lines }, { said: lines.length === 1 ? 'Image added' : 'Images added' });
	}

	/** A URL for a kept image, undefined when this browser doesn't hold it and can't get it. */
	async imageURL(name: string): Promise<string | undefined> {
		const known = this.#urls.get(name);
		if (known) return known;
		let image = await this.#db.get('images', name);
		if (!image && this.#fetchImage) {
			const got = await this.#fetchImage(name).catch(() => undefined);
			if (!got) return undefined;
			image = { name, bytes: got.bytes, type: got.type, pending: false };
			await this.#db.put('images', image).catch(() => {});
		}
		if (!image) return undefined;
		const url = URL.createObjectURL(new Blob([image.bytes], { type: image.type }));
		this.#urls.set(name, url);
		return url;
	}

	/** The next image the server hasn't stored, ahead of the ops naming it. */
	async nextUpload(): Promise<{ name: string; bytes: ArrayBuffer } | undefined> {
		for (const name of this.#unsent) {
			const image = await this.#db.get('images', name);
			if (image) return { name, bytes: image.bytes };
			await this.uploaded(name);
		}
		return undefined;
	}

	async uploaded(name: string): Promise<void> {
		this.#unsent = this.#unsent.filter((n) => n !== name);
		const tx = this.#db.transaction(['images', 'meta'], 'readwrite');
		const images = tx.objectStore('images');
		const image = await images.get(name);
		if (image) void images.put({ ...image, pending: false });
		void tx.objectStore('meta').put(this.#unsent, 'unsent');
		await tx.done;
		this.#counted();
	}

	get waiting(): boolean {
		return this.#queue.length > 0 || this.#unsent.length > 0 || Object.keys(this.#viewEdits).length > 0;
	}

	/** What reaches the server next: the queue, the stamp of each copy no op waits on, the view changes. */
	outgoing(): Outgoing {
		const sent = [...this.#queue];
		const busy = new Set(sent.map((q) => q.project));
		const have: Record<string, string> = {};
		for (const [id, stamp] of this.#stamps) {
			if (!busy.has(id) && stamp !== '' && !this.broken[id]) have[id] = stamp;
		}
		return {
			ops: sent.map((q) => ({ ...q.op, id: q.qid, project: q.project })),
			have,
			views: { ...this.#viewEdits },
			sent,
			last: sent.at(-1)?.seq ?? this.#seq
		};
	}

	/** Takes the server's copy of each project, with the edits queued since the request put back on top. */
	adopt(states: ProjectState[], out: Outgoing): Promise<void> {
		this.#queue = this.#queue.filter((q) => q.seq > out.last);
		const later = this.#queue;
		const held = new Set(this.projects.map((p) => p.id));
		const saves: string[] = [];
		const keep: string[] = [];
		for (const st of states) {
			if (!st.data) {
				if (!held.has(st.id)) continue;
				keep.push(st.id);
				if ((this.broken[st.id] ?? '') !== (st.error ?? '')) {
					if (st.error) this.broken[st.id] = st.error;
					else delete this.broken[st.id];
					saves.push(st.id);
				}
				continue;
			}
			try {
				this.#core.open(JSON.stringify(st.data));
			} catch (error) {
				if (!held.has(st.id)) continue;
				keep.push(st.id);
				this.broken[st.id] = error instanceof Error ? error.message : "Can't open this project";
				saves.push(st.id);
				continue;
			}
			for (const q of later) if (q.project === st.id) this.#core.replay(st.id, q.op);
			this.#stamps.set(st.id, st.stamp);
			delete this.broken[st.id];
			keep.push(st.id);
			saves.push(st.id);
		}
		const createdSince = new Set(later.filter((q) => q.op.action === 'project.create').map((q) => q.project));
		const gone = [...held].filter((id) => !keep.includes(id) && !createdSince.has(id));
		for (const id of gone) {
			this.#core.close(id);
			this.#stamps.delete(id);
			delete this.broken[id];
			delete this.views[id];
			delete this.#viewEdits[id];
		}
		this.#counted();
		if (saves.length === 0 && gone.length === 0) {
			return out.sent.length ? this.#db.delete('queue', IDBKeyRange.upperBound(out.last)) : Promise.resolve();
		}
		this.#show([...keep, ...[...createdSince].filter((id) => !keep.includes(id))]);
		const tx = this.#db.transaction(['projects', 'queue', 'views', 'meta'], 'readwrite');
		void tx.objectStore('queue').delete(IDBKeyRange.upperBound(out.last));
		for (const id of saves) {
			const raw = this.#core.project(id);
			if (raw !== null) void tx.objectStore('projects').put(this.#stored(id, raw));
		}
		for (const id of gone) {
			void tx.objectStore('projects').delete(id);
			void tx.objectStore('views').delete(id);
		}
		if (gone.length) void tx.objectStore('meta').put($state.snapshot(this.#viewEdits), 'viewEdits');
		return tx.done;
	}

	/** docs/sync.md#views; a view that just went out isn't sent again, or a server that can't write it gets it every round. */
	adoptViews(theirs: Record<string, ViewFile> | undefined, out: Outgoing): Promise<void> {
		const before = this.#viewEdits;
		const edits: Record<string, ViewFile> = {};
		for (const [pid, edit] of Object.entries(before)) {
			if (out.views[pid] !== edit) edits[pid] = edit;
		}
		const changed: string[] = [];
		if (theirs) {
			for (const p of this.projects) {
				const view = theirs[p.id];
				const mine = this.views[p.id];
				if (view) {
					const next = { ...view, ...edits[p.id] };
					const shown: View = { folds: next.folds ?? [], filter: fromFile(next.filter) };
					if (!mine || !sameView(mine, shown)) {
						this.views[p.id] = shown;
						changed.push(p.id);
					}
				} else if (mine && (mine.folds.length > 0 || Object.keys(toFile(mine.filter)).length > 0) && !edits[p.id] && !out.views[p.id]) {
					edits[p.id] = { folds: [...mine.folds].sort(), filter: toFile(mine.filter) };
				}
			}
		}
		this.#viewEdits = edits;
		this.#counted();
		const keys = Object.keys(edits);
		if (changed.length === 0 && keys.length === Object.keys(before).length && keys.every((k) => before[k] === edits[k])) return Promise.resolve();
		const tx = this.#db.transaction(['views', 'meta'], 'readwrite');
		for (const id of changed) {
			const v = this.views[id];
			void tx.objectStore('views').put({ id, folds: [...v.folds].sort(), filter: toFile(v.filter) });
		}
		void tx.objectStore('meta').put($state.snapshot(this.#viewEdits), 'viewEdits');
		return tx.done;
	}

	#counted(): void {
		this.pending = this.#queue.length;
		this.uploads = this.#unsent.length;
		this.viewsWaiting = Object.keys(this.#viewEdits).length;
	}

	#stored(id: string, raw: string) {
		const { name } = JSON.parse(raw) as Project;
		return { id, name, stamp: this.#stamps.get(id) ?? '', raw, error: this.broken[id] };
	}

	/** Keeps what the core holds of the project, and queues ops for the server: both in one write, made before any await. */
	async #land(pid: string, ops: Op[]): Promise<void> {
		const raw = this.#core.project(pid);
		const queued = this.synced ? ops.map((op) => ({ seq: ++this.#seq, qid: crypto.randomUUID(), project: pid, op })) : [];
		this.#queue.push(...queued);
		const tx = this.#db.transaction(['projects', 'queue', 'views', 'meta'], 'readwrite');
		for (const q of queued) void tx.objectStore('queue').put($state.snapshot(q));
		if (raw === null) {
			this.#stamps.delete(pid);
			delete this.broken[pid];
			void tx.objectStore('projects').delete(pid);
			void tx.objectStore('views').delete(pid);
			delete this.views[pid];
			if (this.#viewEdits[pid]) {
				delete this.#viewEdits[pid];
				void tx.objectStore('meta').put($state.snapshot(this.#viewEdits), 'viewEdits');
			}
		} else {
			void tx.objectStore('projects').put(this.#stored(pid, raw));
		}
		this.#counted();
		if (queued.length) this.onWaiting();
		await tx.done;
		const ids = this.projects.map((p) => p.id).filter((id) => id !== pid);
		this.#show(this.#core.project(pid) === null ? ids : [...ids, pid]);
	}

	#read(ids: string[]): Project[] {
		return ids.map((id) => JSON.parse(this.#core.project(id) ?? 'null') as Project | null).filter((p) => p !== null);
	}

	#show(ids: string[]): void {
		const held = this.#read(ids);
		const known = new Set(this.#order);
		const fresh = rankProjects(held.filter((p) => !known.has(p.id)), this.#opened).map((p) => p.id);
		this.#order = [...this.#order.filter((id) => ids.includes(id)), ...fresh];
		const at = new Map(this.#order.map((id, i) => [id, i]));
		this.projects = held.sort((a, b) => at.get(a.id)! - at.get(b.id)!);
	}
}

function sameView(a: View, b: View): boolean {
	const key = (v: View) => JSON.stringify([[...v.folds].sort(), toFile(v.filter)]);
	return key(a) === key(b);
}

/** The ops that build an imported project on the server: no op imports a file. */
function importOps(p: Project): Op[] {
	const ops: Op[] = [{ action: 'project.create', title: p.name }];
	if (p.group) ops.push({ action: 'project.group', value: p.group });
	const under = (parent: Category): Category[] =>
		p.categories.filter((c) => c.parent_id === parent.id).flatMap((c) => [c, ...under(c)]);
	for (const top of p.categories.filter((c) => !c.parent_id)) {
		ops.push({ action: 'category.restore', cats: [top, ...under(top)] });
	}
	return ops;
}

const maxImage = 25 << 20;

/** As the server sniffs them (http.DetectContentType): anything else, it refuses. */
export function extension(head: Uint8Array): string | undefined {
	const starts = (bytes: number[], at = 0) => bytes.every((b, i) => head[at + i] === b);
	const ascii = (s: string, at = 0) => starts([...s].map((c) => c.charCodeAt(0)), at);
	if (starts([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])) return 'png';
	if (starts([0xff, 0xd8, 0xff])) return 'jpg';
	if (ascii('GIF87a') || ascii('GIF89a')) return 'gif';
	if (ascii('RIFF') && ascii('WEBPVP', 8)) return 'webp';
	return undefined;
}

/** The alt text of a file's line, as attachments.Line cleans it. */
export function altOf(fileName: string): string {
	const dot = fileName.lastIndexOf('.');
	const stem = dot > 0 ? fileName.slice(0, dot) : fileName;
	const flat = stem.replace(/[[\]\u0000-\u001f\u007f]/g, ' ').trim().replace(/\s+/g, ' ');
	return flat === 'image' || /^\d*$/.test(flat) ? 'photo' : flat;
}
