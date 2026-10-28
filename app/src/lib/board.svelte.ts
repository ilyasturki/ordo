import { SvelteSet } from 'svelte/reactivity';
import { Drag, sameSpot, spotOf } from '$lib/drag.svelte';
import type { Op, Project } from '$lib/model';
import { ordo } from '$lib/state.svelte';

/** What the list screen is in the middle of: a selection, an inline add. */
export class Board {
	readonly pid: () => string;
	readonly project: () => Project | undefined;
	selected = new SvelteSet<string>();
	adding = $state<{ cat: string; after: string } | null>(null);
	drag = $state<Drag | null>(null);

	constructor(pid: () => string, project: () => Project | undefined) {
		this.pid = pid;
		this.project = project;
	}

	/** A long press: lifts the row to drag it, or selects it when the finger lets go where it was. */
	press(id: string, e: PointerEvent): void {
		const p = this.project();
		const block = document.querySelector<HTMLElement>(`[data-block="${CSS.escape(id)}"]`);
		const origin = p ? spotOf(p, id) : null;
		if (this.selecting || p === undefined || block === null || origin === null) {
			this.toggle(id);
			return;
		}
		const hasChildren = p.categories.some((c) => c.tasks.some((t) => t.parent_id === id));
		const drag = new Drag(id, origin, e, block, hasChildren);
		this.drag = drag;
		const listeners = new AbortController();
		const { signal } = listeners;
		const end = (drop: boolean) => {
			listeners.abort();
			drag.stop();
			this.drag = null;
			if (!drag.moved) {
				this.toggle(id);
				return;
			}
			const s = drag.spot;
			if (drop && !sameSpot(s, drag.origin)) {
				void this.edit(
					{ action: 'task.place', ids: [id], cat: s.cat, parent: s.parent, before: s.before },
					{ quiet: true }
				);
			}
		};
		window.addEventListener('pointermove', (ev) => drag.move(ev.clientX, ev.clientY), { signal });
		window.addEventListener('pointerup', () => end(true), { signal });
		window.addEventListener('pointercancel', () => end(false), { signal });
		window.addEventListener('touchmove', (ev) => ev.preventDefault(), { signal, passive: false });
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
