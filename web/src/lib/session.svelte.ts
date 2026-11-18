import { loadCore } from '$lib/core/core';
import { Local } from '$lib/local.svelte';
import { site } from '$lib/paths';
import { openStore } from '$lib/store';
import { readSession, syncedStore } from '$lib/sync/session';
import { Sync } from '$lib/sync/sync.svelte';

/**
 * What the app works on, set once the core and the store are open: the server's projects while this
 * browser is signed in to one, else the site build's own. The private build signs in first.
 */
export const session: { local?: Local; sync?: Sync; signIn?: boolean; error?: string } = $state({});

let opening: Promise<void> | undefined;

export function openSession(): Promise<void> {
	opening ??= (async () => {
		try {
			const saved = readSession();
			if (!saved && !site) {
				session.signIn = true;
				return;
			}
			const [core, db] = await Promise.all([loadCore(), openStore(saved ? syncedStore : 'ordo')]);
			if (!saved) {
				session.local = await Local.open(core, db);
				return;
			}
			const local = await Local.open(core, db, { synced: true, fetchImage: (name) => sync.api.image(name) });
			const sync = new Sync(local, saved);
			session.local = local;
			session.sync = sync;
			sync.start();
		} catch (error) {
			session.error = String(error);
		}
	})();
	return opening;
}

/** Starts the app again on what changed: a sign-in, a sign-out. */
export function restart(to: string): void {
	location.replace(to);
}
