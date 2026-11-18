import { site } from '$lib/paths';
import { clearStore } from '$lib/store';
import { Api, pairingCode, parsePairing, type Session } from './api';

const key = 'ordo.session';

/** The store of the server's copy: the site build keeps its own projects apart, in `ordo`. */
export const syncedStore = site ? 'ordo-server' : 'ordo';

export function readSession(): Session | undefined {
	try {
		const raw = localStorage.getItem(key);
		const s = raw ? (JSON.parse(raw) as Partial<Session>) : undefined;
		if (!s || typeof s.server !== 'string' || typeof s.token !== 'string') return undefined;
		return { server: s.server, token: s.token, cookie: s.cookie === true, expired: s.expired === true };
	} catch {
		return undefined;
	}
}

export function saveSession(s: Session | undefined): void {
	try {
		if (s) localStorage.setItem(key, JSON.stringify(s));
		else localStorage.removeItem(key);
	} catch {
		// A browser that keeps no storage signs in again on the next start.
	}
}

/** The private build talks to the origin it was served from, with the cookie. */
function sessionFor(server: string): Session {
	return site ? { server, token: '', cookie: false } : { server: '', token: '', cookie: true };
}

/** Keeps the session a sign-in answered; a copy synced from another server goes first. */
async function signedIn(next: Session): Promise<void> {
	const previous = readSession();
	if (!previous || previous.server !== next.server) await clearStore(syncedStore);
	saveSession(next);
}

/** Pairs this browser with a code or a pairing link; answers why it failed, or undefined. */
export async function pair(server: string, text: string): Promise<string | undefined> {
	const code = pairingCode(text);
	if (!code) return 'Paste the whole pairing link, or its code';
	const s = sessionFor(site ? server || (parsePairing(text)?.server ?? '') : '');
	if (site && !s.server) return 'Type the address of your ordo serve';
	try {
		await signedIn({ ...s, token: await new Api(s).pair(code) });
		return undefined;
	} catch (error) {
		return error instanceof Error ? error.message : "Can't pair";
	}
}

export async function login(server: string, passphrase: string): Promise<string | undefined> {
	const s = sessionFor(server);
	try {
		await signedIn({ ...s, token: await new Api(s).login(passphrase) });
		return undefined;
	} catch (error) {
		return error instanceof Error ? error.message : "Can't sign in";
	}
}

/** Whether that server takes a passphrase: false for one that doesn't, or doesn't answer. */
export async function offersPassphrase(server: string): Promise<boolean> {
	try {
		return (await new Api(sessionFor(server)).hello()).has_passphrase;
	} catch {
		return false;
	}
}
