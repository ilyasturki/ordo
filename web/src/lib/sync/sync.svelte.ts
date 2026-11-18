import type { Local, Outgoing } from '$lib/local.svelte';
import { clearStore } from '$lib/store';
import { Api, Offline, ServerError, type Devices, type PairCode, type Session, type SyncResult } from './api';
import { saveSession, syncedStore } from './session';

/** What keeps edits from reaching the server, once it is worth telling: never a single blip. */
export type Trouble =
	| { kind: 'failing'; since: number; message: string; offline: boolean }
	| { kind: 'refused'; message: string }
	| { kind: 'outdated'; message: string }
	| { kind: 'expired' };

export type Delays = {
	sync: number;
	poll: number;
	firstBackoff: number;
	maxBackoff: number;
	reconnect: number;
	maxReconnect: number;
	failuresBeforeTrouble: number;
	graceBeforeTrouble: number;
};

export const delays: Delays = {
	sync: 400,
	poll: 30_000,
	firstBackoff: 2_000,
	maxBackoff: 60_000,
	reconnect: 5_000,
	maxReconnect: 60_000,
	failuresBeforeTrouble: 3,
	graceBeforeTrouble: 3 * 60_000
};

/** The phone's SyncEngine: one round at a time, the live stream only while the page is in sight. */
export class Sync {
	session: Session = $state()!;
	trouble: Trouble | undefined = $state();
	lastSync = $state(0);
	readonly #local: Local;
	readonly #delays: Delays;
	#timer: ReturnType<typeof setTimeout> | undefined;
	#inflight: Promise<void> | undefined;
	#again = false;
	#backoff = 0;
	#failures = 0;
	#failingSince = 0;
	#stream: AbortController | undefined;
	#reconnectIn = 0;
	#stopped = false;

	constructor(local: Local, session: Session, d: Delays = delays) {
		this.#local = local;
		this.session = session;
		this.#delays = d;
		if (session.expired) this.trouble = { kind: 'expired' };
		local.onWaiting = () => this.#schedule(this.#delays.sync);
	}

	get api(): Api {
		return new Api(this.session);
	}

	get signedIn(): boolean {
		return !this.session.expired;
	}

	get waitingWords(): string {
		const l = this.#local;
		const count = (n: number, one: string, many: string) => (n === 0 ? undefined : n === 1 ? `1 ${one}` : `${n} ${many}`);
		return [count(l.pending, 'edit', 'edits'), count(l.uploads, 'image', 'images'), count(l.viewsWaiting, 'fold or filter', 'folds or filters')]
			.filter((w) => w !== undefined)
			.join(' and ');
	}

	get #inSight(): boolean {
		return typeof document === 'undefined' || document.visibilityState === 'visible';
	}

	start(): void {
		if (typeof document !== 'undefined') {
			document.addEventListener('visibilitychange', this.#onVisibility);
			addEventListener('online', this.#onOnline);
		}
		this.#listen(this.#inSight);
		void this.sync();
	}

	stop(): void {
		this.#stopped = true;
		clearTimeout(this.#timer);
		this.#listen(false);
		if (typeof document !== 'undefined') {
			document.removeEventListener('visibilitychange', this.#onVisibility);
			removeEventListener('online', this.#onOnline);
		}
	}

	#onVisibility = () => {
		this.#reconnectIn = 0;
		this.#listen(this.#inSight);
		if (this.#inSight) void this.sync();
	};

	#onOnline = () => {
		this.#reconnectIn = 0;
		this.#listen(this.#inSight);
		void this.sync();
	};

	#schedule(ms: number): void {
		clearTimeout(this.#timer);
		this.#timer = undefined;
		if (ms < 0 || !this.signedIn || this.#stopped) return;
		this.#timer = setTimeout(() => void this.sync(), ms);
	}

	/** One round at a time; a round asked for meanwhile runs once after it. */
	sync(): Promise<void> {
		if (this.#inflight) {
			this.#again = true;
			return this.#inflight;
		}
		this.#inflight = this.#run().finally(() => {
			this.#inflight = undefined;
			if (this.#again) {
				this.#again = false;
				void this.sync();
			}
		});
		return this.#inflight;
	}

	async #run(): Promise<void> {
		const s = this.session;
		if (s.expired || this.#stopped) return;
		clearTimeout(this.#timer);
		try {
			if (!(await this.#upload(s))) return;
			const out = this.#local.outgoing();
			const answer = await new Api(s).sync(out.ops, out.have, out.views);
			if (this.session !== s || this.#stopped) return;
			await Promise.all([this.#local.adopt(answer.projects, out), this.#local.adoptViews(answer.views, out)]);
			this.#report(out, answer.results);
			this.#backoff = 0;
			this.#failures = 0;
			this.#failingSince = 0;
			this.lastSync = Date.now();
			this.trouble = undefined;
			this.#schedule(this.#local.waiting ? this.#delays.sync : this.#inSight ? this.#delays.poll : -1);
		} catch (error) {
			if (this.session === s) this.#failed(error);
		}
	}

	/** Sends the images ahead of the ops naming them; false when the session changed under it. */
	async #upload(s: Session): Promise<boolean> {
		for (;;) {
			const next = await this.#local.nextUpload();
			if (!next) return true;
			try {
				await new Api(s).putImage(next.name, next.bytes);
			} catch (error) {
				// The same bytes get the same answer: one bad image can't hold back every edit behind it.
				if (!(error instanceof ServerError) || error.status === 401 || error.status === 408 || error.status === 429 || error.status >= 500) throw error;
				this.#local.say(`The server refused an image: ${error.message}`);
			}
			if (this.session !== s) return false;
			await this.#local.uploaded(next.name);
		}
	}

	#failed(error: unknown): void {
		if (error instanceof ServerError && error.status === 401) {
			this.#expire();
			return;
		}
		const now = Date.now();
		this.#failures++;
		if (this.#failingSince === 0) this.#failingSince = now;
		const message = error instanceof Error ? error.message : String(error);
		if (error instanceof ServerError && error.status === 409) {
			this.trouble = { kind: 'outdated', message };
			return;
		}
		if (error instanceof ServerError && error.status >= 400 && error.status < 500) {
			this.trouble = { kind: 'refused', message };
		} else if (
			this.#local.waiting &&
			(this.#failures >= this.#delays.failuresBeforeTrouble || now - this.#failingSince >= this.#delays.graceBeforeTrouble)
		) {
			this.trouble = { kind: 'failing', since: this.#failingSince, message, offline: error instanceof Offline };
		}
		this.#backoff = Math.min(this.#delays.maxBackoff, this.#backoff === 0 ? this.#delays.firstBackoff : this.#backoff * 2);
		this.#schedule(this.#backoff);
	}

	#report(out: Outgoing, results: SyncResult[]): void {
		for (const r of results) {
			const q = out.sent.find((s) => s.qid === r.id);
			if (!q || r.status === 'applied' || r.status === 'unchanged') continue;
			const theirs = r.theirs;
			if (r.status === 'conflict' && theirs !== undefined) {
				this.#local.offer(r.msg || 'Changed on another device', 'Keep mine', () => void this.#local.perform(q.project, { ...q.op, base: theirs }));
			} else if (r.msg) {
				this.#local.say(r.msg);
			}
		}
	}

	#listen(on: boolean): void {
		this.#stream?.abort();
		this.#stream = undefined;
		if (!on || !this.signedIn || this.#stopped) return;
		const ctl = new AbortController();
		this.#stream = ctl;
		const s = this.session;
		const closed = (error?: unknown) => {
			if (ctl.signal.aborted || this.#stream !== ctl) return;
			if (error instanceof ServerError && error.status === 401) {
				if (this.session === s) this.#expire();
				return;
			}
			this.#reconnectIn = Math.min(this.#delays.maxReconnect, this.#reconnectIn === 0 ? this.#delays.reconnect : this.#reconnectIn * 2);
			setTimeout(() => {
				if (this.#stream === ctl && this.#inSight) this.#listen(true);
			}, this.#reconnectIn);
		};
		new Api(s)
			.listen(
				() => {
					if (this.#stream === ctl) this.#schedule(this.#delays.sync);
				},
				() => {
					if (this.#stream === ctl) this.#reconnectIn = 0;
				},
				ctl.signal
			)
			.then(() => closed(), closed);
	}

	/** The server no longer takes the token: it goes, the copy and what waits stay for the next sign-in. */
	#expire(): void {
		this.#listen(false);
		clearTimeout(this.#timer);
		this.session = { ...this.session, token: '', expired: true };
		saveSession(this.session);
		this.trouble = { kind: 'expired' };
	}

	/** The device's own calls: a token the server stopped taking ends the session as a sync would. */
	async #asDevice<T>(call: (api: Api) => Promise<T>): Promise<T> {
		const s = this.session;
		if (s.expired) throw new ServerError(401, 'Sign in again');
		try {
			return await call(new Api(s));
		} catch (error) {
			if (error instanceof ServerError && error.status === 401 && this.session === s) this.#expire();
			throw error;
		}
	}

	devices(): Promise<Devices> {
		return this.#asDevice((api) => api.devices());
	}

	pairDevice(passphrase: string): Promise<PairCode> {
		return this.#asDevice((api) => api.pairDevice(passphrase));
	}

	/** Answers whether this browser was among the devices it signed out. */
	async revoke(id: string): Promise<boolean> {
		const self = await this.#asDevice((api) => api.revoke(id));
		if (self) await this.#forget();
		return self;
	}

	async signOut(): Promise<void> {
		this.stop();
		if (this.signedIn) await this.api.logout().catch(() => {});
		await this.#forget();
	}

	async #forget(): Promise<void> {
		this.stop();
		saveSession(undefined);
		await clearStore(syncedStore);
	}
}
