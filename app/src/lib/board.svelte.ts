import { SvelteSet } from 'svelte/reactivity';
import { ordo } from '$lib/state.svelte';
import type { Op } from '$lib/model';

/** What the list screen is in the middle of: a selection, an inline add. */
export class Board {
	readonly pid: () => string;
	selected = new SvelteSet<string>();
	adding = $state<{ cat: string; after: string } | null>(null);

	constructor(pid: () => string) {
		this.pid = pid;
	}

	get selecting(): boolean {
		return this.selected.size > 0;
	}

	toggle(id: string): void {
		if (this.selected.has(id)) {
			this.selected.delete(id);
		} else {
			this.selected.add(id);
		}
	}

	clear(): void {
		this.selected.clear();
	}

	edit(op: Op, opts?: { quiet?: boolean }) {
		return ordo.edit(this.pid(), op, opts);
	}

	onSelected(op: Omit<Op, 'ids'>, opts?: { keep?: boolean }) {
		const ids = [...this.selected];
		if (!opts?.keep) {
			this.clear();
		}
		return this.edit({ ...op, ids });
	}
}
