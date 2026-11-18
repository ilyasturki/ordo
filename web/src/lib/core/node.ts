import { readFileSync } from 'node:fs';
import './wasm/wasm_exec.js';
import { start, type Core } from './core';

let loading: Promise<Core> | undefined;

/** The real core for tests, run the way Node runs Go wasm: build it with `just wasm` first. */
export function nodeCore(): Promise<Core> {
	loading ??= (async () => {
		const go = new Go();
		const bytes = readFileSync(new URL('./wasm/ordo.wasm', import.meta.url));
		const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
		return start(go, instance);
	})();
	return loading;
}
