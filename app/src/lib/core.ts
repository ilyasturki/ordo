import { base } from '$app/paths';
import type { Answer, Op } from '$lib/model';

type Exports = {
	open(raw: string): string;
	perform(pid: string, op: string): string;
	replay(pid: string, op: string): string;
	project(pid: string): string | null;
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

/** The edits ordo runs, compiled from internal/core: the same code `ordo serve` replays the queue with. */
export class Core {
	readonly #x: Exports;

	constructor(x: Exports) {
		this.#x = x;
	}

	open(raw: string): string {
		const out = JSON.parse(this.#x.open(raw)) as { id?: string; error?: string };
		if (out.error !== undefined) {
			throw new Error(out.error);
		}
		return out.id ?? '';
	}

	perform(pid: string, op: Op): Answer {
		return JSON.parse(this.#x.perform(pid, JSON.stringify(op))) as Answer;
	}

	replay(pid: string, op: Op): Answer {
		return JSON.parse(this.#x.replay(pid, JSON.stringify(op))) as Answer;
	}

	project(pid: string): string | null {
		return this.#x.project(pid);
	}

	close(pid: string): void {
		this.#x.close(pid);
	}
}

function script(src: string): Promise<void> {
	return new Promise((resolve, reject) => {
		const el = document.createElement('script');
		el.src = src;
		el.onload = () => resolve();
		el.onerror = () => reject(new Error(`${src} didn't load`));
		document.head.append(el);
	});
}

async function instantiate(go: InstanceType<typeof Go>): Promise<WebAssembly.Instance> {
	const url = `${base}/ordo-core.wasm`;
	try {
		return (await WebAssembly.instantiateStreaming(fetch(url), go.importObject)).instance;
	} catch {
		const bytes = await (await fetch(url)).arrayBuffer();
		return (await WebAssembly.instantiate(bytes, go.importObject)).instance;
	}
}

async function boot(): Promise<Core> {
	if (globalThis.Go === undefined) {
		await script(`${base}/wasm_exec.js`);
	}
	const go = new Go();
	const ready = new Promise<void>((resolve) => {
		globalThis.ordoReady = resolve;
	});
	void go.run(await instantiate(go));
	await ready;
	if (globalThis.ordoCore === undefined) {
		throw new Error('ordo-core started without its exports');
	}
	return new Core(globalThis.ordoCore);
}

let loading: Promise<Core> | undefined;

export function loadCore(): Promise<Core> {
	loading ??= boot();
	return loading;
}
