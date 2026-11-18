import type { Op, Project, Verdict } from '$lib/core/types';
import type { FilterFile } from '$lib/model/filter';

/** web.Protocol: docs/sync.md says when it goes up. */
export const protocol = 1;

/** The private build: the cookie, on its own origin (server empty). The site build: a bearer token. */
export type Session = {
	server: string;
	token: string;
	cookie: boolean;
	/** The server stopped taking this device: the token is gone, the copy and its queue stay. */
	expired?: boolean;
};

/** The server answered: `status` is its HTTP status, the message is for the user. */
export class ServerError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.status = status;
	}
}

export class Offline extends Error {}

export type SyncOp = Op & { id: string; project: string };

export type SyncResult = { id: string; status: Verdict; msg?: string; theirs?: string };

export type ProjectState = { id: string; name?: string; stamp: string; data?: Project; error?: string };

/** A view as `<id>.view.json` holds it, and a change to one: a field left out stays as it is. */
export type ViewFile = { folds?: string[]; filter?: FilterFile };

export type SyncAnswer = { protocol: number; results: SyncResult[]; projects: ProjectState[]; views?: Record<string, ViewFile> };

export type DeviceInfo = {
	id: string;
	name: string;
	created_at: string;
	last_seen: string;
	from?: string;
	also?: string[];
	self?: boolean;
};

export type Devices = { devices: DeviceInfo[]; has_passphrase: boolean };

export type PairCode = { link: string; qr: string; minutes: number };

const timeout = 20_000;

export class Api {
	readonly #session: Session;

	constructor(session: Session) {
		this.#session = session;
	}

	async #fetch(path: string, init: RequestInit = {}, wait = timeout): Promise<Response> {
		if (typeof navigator !== 'undefined' && navigator.onLine === false) throw new Offline("Can't reach ordo serve");
		const s = this.#session;
		const headers = new Headers(init.headers);
		if (!s.cookie && s.token !== '') headers.set('Authorization', `Bearer ${s.token}`);
		const signals = [init.signal, wait > 0 ? AbortSignal.timeout(wait) : undefined].filter((x) => x != null);
		let res: Response;
		try {
			res = await fetch(s.server + path, {
				...init,
				headers,
				// A bearer client sends no cookie anywhere, and a redirect can't carry its token off.
				credentials: s.cookie ? 'same-origin' : 'omit',
				redirect: 'error',
				cache: 'no-store',
				signal: signals.length > 1 ? AbortSignal.any(signals) : signals[0]
			});
		} catch (error) {
			if (init.signal?.aborted) throw error;
			throw new Offline("Can't reach ordo serve");
		}
		if (!res.ok) {
			let msg = `ordo serve answered ${res.status}`;
			try {
				msg = ((await res.json()) as { error?: string }).error ?? msg;
			} catch {
				// Not JSON: a proxy's page.
			}
			throw new ServerError(res.status, msg);
		}
		return res;
	}

	async #json<T>(path: string, body?: unknown): Promise<T> {
		const res = await this.#fetch(
			path,
			body === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
		);
		if (res.status === 204) return undefined as T;
		return (await res.json()) as T;
	}

	hello(): Promise<{ protocol: number; has_passphrase: boolean }> {
		return this.#json('/api/hello');
	}

	/** Answers the device token, empty when it came back as the cookie. */
	async pair(code: string): Promise<string> {
		return (await this.#json<{ token: string }>('/api/pair', { token: code, name: '', cookie: this.#session.cookie })).token;
	}

	async login(passphrase: string): Promise<string> {
		return (await this.#json<{ token: string }>('/api/login', { passphrase, name: '', cookie: this.#session.cookie })).token;
	}

	sync(ops: SyncOp[], have: Record<string, string>, views: Record<string, ViewFile>): Promise<SyncAnswer> {
		return this.#json('/api/sync', { protocol, ops, have, views });
	}

	/** A stored image, undefined when the server has none by that name. */
	async image(name: string): Promise<{ bytes: ArrayBuffer; type: string } | undefined> {
		try {
			const res = await this.#fetch(`/api/attachments/${encodeURIComponent(name)}`);
			return { bytes: await res.arrayBuffer(), type: res.headers.get('Content-Type') ?? '' };
		} catch (error) {
			if (error instanceof ServerError && error.status === 404) return undefined;
			throw error;
		}
	}

	async putImage(name: string, bytes: ArrayBuffer): Promise<void> {
		await this.#fetch(`/api/attachments/${encodeURIComponent(name)}`, { method: 'PUT', body: bytes }, 120_000);
	}

	devices(): Promise<Devices> {
		return this.#json('/api/devices');
	}

	pairDevice(passphrase: string): Promise<PairCode> {
		return this.#json('/api/devices/pair', { passphrase });
	}

	/** True when this device is one of those it signed out. */
	async revoke(id: string): Promise<boolean> {
		return (await this.#json<{ self: boolean }>(`/api/devices/${encodeURIComponent(id)}/revoke`, {})).self;
	}

	logout(): Promise<void> {
		return this.#json('/api/logout', {});
	}

	/** Read with fetch, which carries the bearer token where EventSource can't. */
	async listen(changed: (pid: string) => void, opened: () => void, signal: AbortSignal): Promise<void> {
		const res = await this.#fetch('/api/events', { signal, headers: { Accept: 'text/event-stream' } }, 0);
		const reader = res.body?.pipeThrough(new TextDecoderStream()).getReader();
		if (!reader) return;
		opened();
		let buffered = '';
		for (;;) {
			const { value, done } = await reader.read();
			if (done) return;
			buffered += value.replaceAll('\r\n', '\n');
			const events = buffered.split('\n\n');
			buffered = events.pop() ?? '';
			for (const ev of events) {
				const lines = ev.split('\n');
				if (!lines.includes('event: change')) continue;
				const data = lines.find((l) => l.startsWith('data:'));
				if (data !== undefined) changed(data.slice('data:'.length).trim());
			}
		}
	}
}

export function parsePairing(text: string): { server: string; code: string } | undefined {
	let url: URL;
	try {
		url = new URL(text.trim());
	} catch {
		return undefined;
	}
	const m = /\/pair\/([^/?#]+)\/?$/.exec(url.pathname);
	if ((url.protocol !== 'https:' && url.protocol !== 'http:') || !m) return undefined;
	try {
		return { server: url.origin + url.pathname.slice(0, m.index), code: decodeURIComponent(m[1]) };
	} catch {
		return undefined;
	}
}

export function pairingCode(text: string): string | undefined {
	const raw = text.trim();
	return parsePairing(raw)?.code ?? (/^[\w.~-]+$/.test(raw) ? raw : undefined);
}

/** The server a typed address names, a bare host taking https; undefined while it can't be one. */
export function parseServer(text: string): string | undefined {
	const raw = text.trim().replace(/\/+$/, '');
	if (raw === '') return undefined;
	const pairing = parsePairing(raw);
	if (pairing) return pairing.server;
	let url: URL;
	try {
		url = new URL(raw.includes('://') ? raw : `https://${raw}`);
	} catch {
		return undefined;
	}
	if (url.protocol !== 'https:' && url.protocol !== 'http:') return undefined;
	if (!url.hostname.includes('.') && url.hostname !== 'localhost' && !url.hostname.startsWith('[')) return undefined;
	if (url.username || url.password || url.search || url.hash) return undefined;
	return url.origin + url.pathname.replace(/\/+$/, '');
}
