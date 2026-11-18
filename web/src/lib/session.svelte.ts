import { loadCore } from '$lib/core/core';
import { Local } from '$lib/local.svelte';
import { openStore } from '$lib/store';

/** What the app works on, set once the core and the store are open. */
export const session: { local?: Local; error?: string } = $state({});

let opening: Promise<void> | undefined;

export function openSession(): Promise<void> {
	opening ??= (async () => {
		try {
			const [core, db] = await Promise.all([loadCore(), openStore()]);
			session.local = await Local.open(core, db);
		} catch (error) {
			session.error = String(error);
		}
	})();
	return opening;
}
