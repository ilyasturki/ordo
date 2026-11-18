import type { Core } from '$lib/core/core';
import type { Answer, Op, Project } from '$lib/core/types';
import { fromFile, noFilter, pruned, toFile, type Filter } from '$lib/model/filter';
import { rankProjects } from '$lib/model/project';
import type { DB } from '$lib/store';

export type View = { folds: string[]; filter: Filter };

export type Snack = { id: number; msg: string; undo?: () => void };

/** One edit the history can take back: the ops that undo it, and its own replayable op to redo it. */
type Step = { pid: string; forward: Op; back: Op[] };

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
	readonly #core: Core;
	readonly #db: DB;
	#order: string[] = [];
	#opened: Record<string, number> = {};
	#history: Step[] = [];
	#future: Step[] = [];
	#snacks = 0;
	#viewing = Promise.resolve();
	readonly #urls = new Map<string, string>();

	private constructor(core: Core, db: DB) {
		this.#core = core;
		this.#db = db;
	}

	static async open(core: Core, db: DB): Promise<Local> {
		const local = new Local(core, db);
		const ids: string[] = [];
		for (const stored of await db.getAll('projects')) {
			try {
				ids.push(core.open(stored.raw));
			} catch (error) {
				local.unreadable.push({ id: stored.id, error: String(error) });
			}
		}
		for (const v of await db.getAll('views')) {
			local.views[v.id] = { folds: v.folds, filter: fromFile(v.filter) };
		}
		local.#opened = ((await db.get('meta', 'opened')) as Record<string, number> | undefined) ?? {};
		local.current = ((await db.get('meta', 'current')) as string | undefined) ?? '';
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
			v.folds.includes(id) === folded ? undefined : { ...v, folds: folded ? [...v.folds, id] : v.folds.filter((f) => f !== id) }
		);
	}

	setFilter(pid: string, filter: Filter): Promise<void> {
		return this.#setView(pid, (v) => ({ ...v, filter }));
	}

	/** One change at a time, each written before it shows, so a reload never loses one already on screen. */
	#setView(pid: string, change: (view: View) => View | undefined): Promise<void> {
		const run = this.#viewing.then(async () => {
			const view = change(this.view(pid));
			if (!view) return;
			await this.#db.put('views', { id: pid, folds: [...view.folds].sort(), filter: toFile(view.filter) });
			this.views[pid] = view;
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
			await this.#keep(target);
		}
		const step = this.#record(target, answer);
		if (!quiet) {
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
		this.snack = { id: ++this.#snacks, msg, undo: step && (() => void this.#takeBack(step)) };
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
		let applied = false;
		for (const op of step.back) {
			applied = this.#core.perform(step.pid, op).status === 'applied' || applied;
		}
		if (!applied) {
			this.say("Can't undo that: it changed since");
			return false;
		}
		await this.#keep(step.pid);
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
		await this.#keep(step.pid);
		if (answer.queue && answer.undo?.length) this.#history.push({ pid: step.pid, forward: answer.queue, back: answer.undo });
		this.say('Redone');
		return true;
	}

	/** Adds a project file as a new project and answers its ID; throws on a file ordo can't import. */
	async import(raw: string): Promise<string> {
		const id = this.#core.import(raw);
		await this.#keep(id);
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
					await this.#db.put('images', { name, bytes, type: { jpg: 'image/jpeg' }[ext] ?? 'image/' + ext, pending: false });
				}
			} catch {
				return 'This browser can’t keep that image';
			}
			lines.push(`![${altOf(file.name)}](attachments/${name})`);
		}
		return this.perform(pid, { action: 'task.attach', ids: [tid], lines }, { said: lines.length === 1 ? 'Image added' : 'Images added' });
	}

	/** A URL for a kept image, undefined when this browser doesn't hold it. */
	async imageURL(name: string): Promise<string | undefined> {
		const known = this.#urls.get(name);
		if (known) return known;
		const image = await this.#db.get('images', name);
		if (!image) return undefined;
		const url = URL.createObjectURL(new Blob([image.bytes], { type: image.type }));
		this.#urls.set(name, url);
		return url;
	}

	async #keep(pid: string): Promise<void> {
		const raw = this.#core.project(pid);
		if (raw === null) {
			await this.#db.delete('projects', pid);
			await this.#db.delete('views', pid);
			delete this.views[pid];
		} else {
			const { name } = JSON.parse(raw) as Project;
			await this.#db.put('projects', { id: pid, name, stamp: '', raw });
		}
		this.#show([...this.projects.map((p) => p.id).filter((id) => id !== pid), ...(raw === null ? [] : [pid])]);
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
