import { readFileSync } from 'node:fs';
import { Core } from '$lib/core';

/** The real ordo-core, built into static/ by `just web`, loaded the way Node runs Go wasm. */
export async function nodeCore(): Promise<Core> {
	if (globalThis.ordoCore !== undefined) {
		return new Core(globalThis.ordoCore);
	}
	await import(new URL('../../../static/wasm_exec.js', import.meta.url).href);
	const go = new Go();
	const ready = new Promise<void>((resolve) => {
		globalThis.ordoReady = resolve;
	});
	const bytes = readFileSync(new URL('../../../static/ordo-core.wasm', import.meta.url));
	const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
	void go.run(instance);
	await ready;
	return new Core(globalThis.ordoCore!);
}
