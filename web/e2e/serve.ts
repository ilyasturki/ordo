import { execFileSync, spawn, type ChildProcess } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const repo = fileURLToPath(new URL('../..', import.meta.url));

export const seed = join(repo, 'testdata/native/seed.json');
export const passphrase = 'throwaway-web-passphrase';

let binary: string | undefined;

/** This checkout's ordo, built once per process. */
export function ordo(): string {
	if (!binary) {
		const dir = mkdtempSync(join(tmpdir(), 'ordo-web-bin-'));
		binary = join(dir, 'ordo');
		execFileSync('go', ['build', '-o', binary, './cmd/ordo'], { cwd: repo, stdio: 'inherit' });
		process.once('exit', () => rmSync(dir, { recursive: true, force: true }));
	}
	return binary;
}

export function freePort(): Promise<number> {
	return new Promise((resolve, reject) => {
		const server = createServer();
		server.on('error', reject);
		server.listen(0, '127.0.0.1', () => {
			const { port } = server.address() as { port: number };
			server.close(() => resolve(port));
		});
	});
}

export type Served = {
	url: string;
	/** A device token signed in with the passphrase, for a client of the test's own. */
	token(): Promise<string>;
	/** Adds a project file, as `ordo import` on the server does. */
	add(file: string): void;
	/** A pairing link as `ordo serve pair` prints it. */
	pairLink(): string;
	/** Stops the server; start brings it back on the same port and data. */
	stop(): Promise<void>;
	start(): Promise<void>;
	close(): Promise<void>;
};

/** A throwaway ordo serve on its own data, with a passphrase set, and the seed's project when asked. */
export async function serve({ seeded = true } = {}): Promise<Served> {
	const bin = ordo();
	const dir = mkdtempSync(join(tmpdir(), 'ordo-web-serve-'));
	const env = { PATH: process.env.PATH ?? '', HOME: dir, ORDO_DATA_PATH: join(dir, 'data'), ORDO_CONFIG_PATH: join(dir, 'config') };
	if (seeded) execFileSync(bin, ['import', seed, '-q'], { env });
	execFileSync(bin, ['serve', 'passphrase'], { env, input: passphrase + '\n' });
	const port = await freePort();
	const url = `http://127.0.0.1:${port}`;
	let child: ChildProcess | undefined;
	const kill = () => child?.kill();
	process.once('exit', kill);

	async function start() {
		child = spawn(bin, ['serve', '--listen', `127.0.0.1:${port}`], { env, stdio: 'ignore' });
		for (let i = 0; i < 100; i++) {
			try {
				if ((await fetch(url + '/api/hello')).ok) return;
			} catch {
				// Not listening yet.
			}
			await new Promise((r) => setTimeout(r, 50));
		}
		throw new Error(`ordo serve didn't start on ${url}`);
	}

	async function stop() {
		const c = child;
		child = undefined;
		if (!c || c.exitCode !== null) return;
		await new Promise((r) => {
			c.once('exit', r);
			c.kill();
		});
	}

	await start();
	return {
		url,
		start,
		stop,
		async token() {
			const res = await fetch(url + '/api/login', { method: 'POST', body: JSON.stringify({ passphrase, name: 'Test client' }) });
			return ((await res.json()) as { token: string }).token;
		},
		add(file) {
			execFileSync(bin, ['import', file, '-q'], { env });
		},
		pairLink() {
			const out = execFileSync(bin, ['serve', 'pair', '--base-url', url], { env }).toString();
			const link = /https?:\/\/\S+\/pair\/\S+/.exec(out)?.[0];
			if (!link) throw new Error(`no link in: ${out}`);
			return link;
		},
		async close() {
			await stop();
			process.off('exit', kill);
			rmSync(dir, { recursive: true, force: true });
		}
	};
}
