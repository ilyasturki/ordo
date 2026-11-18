import type { Core } from '$lib/core/core';
import type { Answer, Op, Project } from '$lib/core/types';
import type { DB } from '$lib/store';

/** The projects kept in this browser: every edit runs through the core, then the project goes back to IndexedDB. */
export class Local {
	projects: Project[] = $state([]);
	/** Stored projects the core refused to open, kept untouched. */
	unreadable: { id: string; error: string }[] = $state([]);
	readonly #core: Core;
	readonly #db: DB;

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
		local.#show(ids);
		return local;
	}

	project(pid: string): Project | undefined {
		return this.projects.find((p) => p.id === pid);
	}

	/** Runs an edit and keeps what it changed. A project.create given no pid answers the new one as row. */
	async perform(pid: string, op: Op): Promise<Answer> {
		const answer = this.#core.perform(pid, op);
		if (answer.status === 'applied') {
			await this.#keep(op.action === 'project.create' && pid === '' ? (answer.row ?? '') : pid);
		}
		return answer;
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

	async #keep(pid: string): Promise<void> {
		const raw = this.#core.project(pid);
		if (raw === null) {
			await this.#db.delete('projects', pid);
		} else {
			const { name } = JSON.parse(raw) as Project;
			await this.#db.put('projects', { id: pid, name, stamp: '', raw });
		}
		this.#show([...this.projects.map((p) => p.id).filter((id) => id !== pid), ...(raw === null ? [] : [pid])]);
	}

	#show(ids: string[]): void {
		this.projects = ids
			.map((id) => JSON.parse(this.#core.project(id) ?? 'null') as Project | null)
			.filter((p) => p !== null)
			.sort((a, b) => a.name.localeCompare(b.name));
	}
}
