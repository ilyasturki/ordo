import { loadCore } from '$lib/core/core';
import { Local } from '$lib/local.svelte';
import { site } from '$lib/paths';
import { openStore } from '$lib/store';
import { readSession, syncedStore } from '$lib/sync/session';
import { Sync } from '$lib/sync/sync.svelte';

/** The server's copy once signed in, else the site build's own projects; the private build signs in first. */
export const session: { local?: Local; sync?: Sync; signIn?: boolean; elsewhere?: boolean; error?: string } = $state({});

let opening: Promise<void> | undefined;

/** One tab owns the synced copy: two would number their queued edits alike and overwrite each other's. */
function own(steal = false): Promise<boolean> {
	if (!navigator.locks) return Promise.resolve(true);
	return new Promise((owned) => {
		navigator.locks
			.request(`ordo:${syncedStore}`, steal ? { steal } : { ifAvailable: true }, (lock) => {
				owned(lock !== null);
				return lock && new Promise<void>(() => {});
			})
			.catch(() => {
				session.sync?.stop();
				session.elsewhere = true;
			});
	});
}

export async function useHere(): Promise<void> {
	await own(true);
	location.reload();
}

export function openSession(): Promise<void> {
	opening ??= (async () => {
		try {
			const saved = readSession();
			if (!saved && !site) {
				session.signIn = true;
				return;
			}
			if (saved && !(await own())) {
				session.elsewhere = true;
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

export function restart(to: string): void {
	location.replace(to);
}
