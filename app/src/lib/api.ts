import type { Session } from '$lib/db';
import type { Op, Project, Verdict } from '$lib/model';

export const PROTOCOL = 1;

/** The server answered: `status` is its HTTP status, the message is for the user. */
export class ServerError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.status = status;
	}
}

/** The request never got an answer. */
export class Offline extends Error {}

export type SyncOp = Op & { id: string; project: string };

export type SyncResult = { id: string; status: Verdict; msg?: string; theirs?: string };

export type ProjectState = {
	id: string;
	name?: string;
	stamp: string;
	data?: Project;
	error?: string;
};

export type SyncAnswer = { protocol: number; results: SyncResult[]; projects: ProjectState[] };

export type DeviceInfo = {
	id: string;
	name: string;
	created_at: string;
	last_seen: string;
	from?: string;
	also?: string[];
	self?: boolean;
};

const TIMEOUT_MS = 20_000;

export class Api {
	readonly session: Session;

	constructor(session: Session) {
		this.session = session;
	}

	async #fetch(path: string, init: RequestInit = {}): Promise<Response> {
		const headers = new Headers(init.headers);
		if (this.session.token !== '') {
			headers.set('Authorization', `Bearer ${this.session.token}`);
		}
		let res: Response;
		try {
			res = await fetch(this.session.server + path, {
				...init,
				headers,
				signal: init.signal ?? AbortSignal.timeout(TIMEOUT_MS)
			});
		} catch {
			throw new Offline("Can't reach ordo serve");
		}
		if (!res.ok) {
			let msg = `ordo serve answered ${res.status}`;
			try {
				const body = (await res.json()) as { error?: string };
				msg = body.error ?? msg;
			} catch {
				/* not JSON */
			}
			throw new ServerError(res.status, msg);
		}
		return res;
	}

	async #json<T>(
		path: string,
		body?: unknown,
		method = body === undefined ? 'GET' : 'POST'
	): Promise<T> {
		const res = await this.#fetch(path, {
			method,
			headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
			body: body === undefined ? undefined : JSON.stringify(body)
		});
		if (res.status === 204) {
			return undefined as T;
		}
		return (await res.json()) as T;
	}

	hello(): Promise<{ protocol: number; has_passphrase: boolean }> {
		return this.#json('/api/hello');
	}

	pair(code: string, name: string, cookie: boolean): Promise<{ token: string }> {
		return this.#json('/api/pair', { token: code, name, cookie });
	}

	login(passphrase: string, name: string, cookie: boolean): Promise<{ token: string }> {
		return this.#json('/api/login', { passphrase, name, cookie });
	}

	sync(ops: SyncOp[], have: Record<string, string>): Promise<SyncAnswer> {
		return this.#json('/api/sync', { protocol: PROTOCOL, ops, have });
	}

	async image(name: string): Promise<Blob> {
		return (await this.#fetch(`/api/attachments/${name}`)).blob();
	}

	async putImage(name: string, blob: Blob): Promise<void> {
		await this.#fetch(`/api/attachments/${name}`, {
			method: 'PUT',
			body: blob,
			signal: AbortSignal.timeout(120_000)
		});
	}

	devices(): Promise<{ devices: DeviceInfo[]; has_passphrase: boolean }> {
		return this.#json('/api/devices');
	}

	pairDevice(passphrase: string): Promise<{ link: string; qr: string; minutes: number }> {
		return this.#json('/api/devices/pair', { passphrase });
	}

	revoke(id: string): Promise<{ self: boolean }> {
		return this.#json(`/api/devices/${encodeURIComponent(id)}/revoke`, {});
	}

	logout(): Promise<void> {
		return this.#json('/api/logout', {});
	}
}

/** Reads the server and code out of a pairing link, `https://host/pair/<code>`, or a bare code. */
export function parsePairing(text: string): { server: string; code: string } | null {
	const raw = text.trim();
	if (raw === '') {
		return null;
	}
	try {
		const url = new URL(raw);
		const m = /\/pair\/([^/?#]+)\/?$/u.exec(url.pathname);
		if ((url.protocol === 'https:' || url.protocol === 'http:') && m) {
			const prefix = url.pathname.slice(0, m.index);
			return { server: url.origin + prefix, code: decodeURIComponent(m[1]) };
		}
		return null;
	} catch {
		return /^[\w-]+$/u.test(raw) ? { server: '', code: raw } : null;
	}
}
