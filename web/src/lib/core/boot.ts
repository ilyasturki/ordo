import './wasm/wasm_exec.js';
import wasm from './wasm/ordo.wasm?url';
import { start, type Core } from './core';

async function instantiate(go: InstanceType<typeof Go>): Promise<WebAssembly.Instance> {
	try {
		return (await WebAssembly.instantiateStreaming(fetch(wasm), go.importObject)).instance;
	} catch {
		// A server that sends .wasm without application/wasm fails the streaming path.
		const bytes = await (await fetch(wasm)).arrayBuffer();
		return (await WebAssembly.instantiate(bytes, go.importObject)).instance;
	}
}

export async function boot(): Promise<Core> {
	const go = new Go();
	return start(go, await instantiate(go));
}
