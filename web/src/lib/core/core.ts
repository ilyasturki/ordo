import type { Answer, Op } from './types';

export type Exports = {
	open(raw: string): string;
	import(raw: string): string;
	perform(pid: string, op: string, hidden?: string): string;
	replay(pid: string, op: string): string;
	project(pid: string): string | null;
	export(pid: string): string | null;
	close(pid: string): void;
};

declare global {
	var Go: new () => {
		importObject: WebAssembly.Imports;
		run(i: WebAssembly.Instance): Promise<void>;
	};
	var ordoCore: Exports | undefined;
	var ordoReady: (() => void) | undefined;
}

/** The edits ordo runs, compiled from internal/core: the same code `ordo serve` replays a queue with. */
export class Core {
	readonly #x: Exports;

	constructor(x: Exports) {
		this.#x = x;
	}

	/** Holds a project as ordo reads its file; throws on one it can't. */
	open(raw: string): string {
		return held(this.#x.open(raw));
	}

	/** Holds a project file as a new project, the way `ordo import` adds one. */
	import(raw: string): string {
		return held(this.#x.import(raw));
	}

	/** Runs a user's edit, a shift stepping over the hidden rows. A project.create given no pid mints one and answers it as row. */
	perform(pid: string, op: Op, hidden?: string[]): Answer {
		const out = hidden ? this.#x.perform(pid, JSON.stringify(op), JSON.stringify(hidden)) : this.#x.perform(pid, JSON.stringify(op));
		return JSON.parse(out) as Answer;
	}

	/** Runs a queued or undo op again. */
	replay(pid: string, op: Op): Answer {
		return JSON.parse(this.#x.replay(pid, JSON.stringify(op))) as Answer;
	}

	/** The held project as compact JSON, null when it isn't held. */
	project(pid: string): string | null {
		return this.#x.project(pid);
	}

	/** The held project as ordo writes its file, null when it isn't held. */
	export(pid: string): string | null {
		return this.#x.export(pid);
	}

	close(pid: string): void {
		this.#x.close(pid);
	}
}

function held(out: string): string {
	const { id, error } = JSON.parse(out) as { id?: string; error?: string };
	if (error !== undefined) {
		throw new Error(error);
	}
	return id ?? '';
}

/** Runs the Go program in an instance and waits for it to publish its calls. */
export async function start(go: InstanceType<typeof Go>, instance: WebAssembly.Instance): Promise<Core> {
	const ready = new Promise<void>((resolve) => {
		globalThis.ordoReady = resolve;
	});
	void go.run(instance);
	await ready;
	if (globalThis.ordoCore === undefined) {
		throw new Error('ordo-wasm started without its calls');
	}
	return new Core(globalThis.ordoCore);
}

let loading: Promise<Core> | undefined;

/** The core, fetched and started on first use: the landing never loads it. */
export function loadCore(): Promise<Core> {
	loading ??= import('./boot').then((boot) => boot.boot());
	return loading;
}
