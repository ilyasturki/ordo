import type { Project, Task } from '$lib/model';

export type Spot = { cat: string; parent: string; before: string };

const SLOP_PX = 6;
const INDENT_PX = 28;
const EDGE_PX = 96;

export function sameSpot(a: Spot, b: Spot): boolean {
	return a.cat === b.cat && a.parent === b.parent && a.before === b.before;
}

function blockEnd(tasks: Task[], i: number): number {
	let end = i + 1;
	if (!tasks[i].parent_id) {
		while (end < tasks.length && tasks[end].parent_id === tasks[i].id) {
			end++;
		}
	}
	return end;
}

/** Where a row sits now, written the way task.place names a spot. */
export function spotOf(p: Project, id: string): Spot | null {
	for (const c of p.categories) {
		const i = c.tasks.findIndex((t) => t.id === id);
		if (i < 0) {
			continue;
		}
		const parent = c.tasks[i].parent_id ?? '';
		const next = c.tasks[blockEnd(c.tasks, i)];
		const before = next && (next.parent_id ?? '') === parent ? next.id : '';
		return { cat: c.id, parent, before };
	}
	return null;
}

type RowInfo = { id: string; parent: string; separator: boolean; mid: number };

/**
 * A row lifted by a long press. The block keeps its place in the DOM, floating under the finger,
 * so the touch that lifted it keeps reaching the page; a gap in the list shows where it lands.
 */
export class Drag {
	readonly id: string;
	readonly origin: Spot;
	readonly hasChildren: boolean;
	readonly left: number;
	readonly width: number;
	readonly height: number;
	spot = $state<Spot>({ cat: '', parent: '', before: '' });
	top = $state(0);
	moved = $state(false);
	#x0: number;
	#y0: number;
	#x: number;
	#y: number;
	#grab: number;
	#frame = 0;

	constructor(id: string, origin: Spot, e: PointerEvent, block: HTMLElement, hasChildren: boolean) {
		this.id = id;
		this.origin = origin;
		this.spot = origin;
		this.hasChildren = hasChildren;
		const r = block.getBoundingClientRect();
		this.left = r.left;
		this.width = r.width;
		this.height = r.height;
		this.top = r.top;
		this.#x0 = this.#x = e.clientX;
		this.#y0 = this.#y = e.clientY;
		this.#grab = e.clientY - r.top;
	}

	move(x: number, y: number): void {
		this.#x = x;
		this.#y = y;
		if (!this.moved && Math.hypot(x - this.#x0, y - this.#y0) < SLOP_PX) {
			return;
		}
		this.moved = true;
		this.#settle();
		cancelAnimationFrame(this.#frame);
		this.#frame = requestAnimationFrame(() => this.#scroll());
	}

	#scroll(): void {
		const y = this.#y;
		const edge =
			y < EDGE_PX ? y - EDGE_PX : y > innerHeight - EDGE_PX ? y - innerHeight + EDGE_PX : 0;
		if (edge !== 0) {
			scrollBy(0, edge / 5);
			this.#settle();
			this.#frame = requestAnimationFrame(() => this.#scroll());
		}
	}

	#settle(): void {
		this.top = this.#y - this.#grab;
		const next = this.#target();
		if (next !== null && !sameSpot(next, this.spot)) {
			this.spot = next;
		}
	}

	#card(): HTMLElement | null {
		let best: HTMLElement | null = null;
		let bestDistance = Infinity;
		for (const c of document.querySelectorAll<HTMLElement>('[data-drop]')) {
			const r = c.getBoundingClientRect();
			const d = this.#y < r.top ? r.top - this.#y : this.#y > r.bottom ? this.#y - r.bottom : 0;
			if (d < bestDistance) {
				best = c;
				bestDistance = d;
			}
		}
		return best;
	}

	#target(): Spot | null {
		const card = this.#card();
		if (card === null) {
			return null;
		}
		const cat = card.dataset.drop ?? '';
		const rows: RowInfo[] = [...card.querySelectorAll<HTMLElement>('[data-open] [data-row]')]
			.filter((el) => el.closest(`[data-block="${CSS.escape(this.id)}"]`) === null)
			.map((el) => {
				const r = el.getBoundingClientRect();
				return {
					id: el.dataset.row ?? '',
					parent: el.dataset.parent ?? '',
					separator: el.dataset.sep !== undefined,
					mid: r.top + r.height / 2
				};
			});
		const k = rows.filter((r) => r.mid < this.#y).length;
		const prev = rows[k - 1];
		const next = rows[k];
		const dx = this.#x - this.#x0 + (this.origin.parent ? INDENT_PX : 0);
		if (next && next.parent !== '') {
			return this.hasChildren ? null : { cat, parent: next.parent, before: next.id };
		}
		const host = prev ? (prev.parent !== '' ? prev.parent : prev.separator ? '' : prev.id) : '';
		const nest =
			host !== '' && !this.hasChildren && (prev.parent !== '' ? dx > -INDENT_PX : dx > INDENT_PX);
		if (nest) {
			return { cat, parent: host, before: '' };
		}
		return { cat, parent: '', before: next?.id ?? '' };
	}

	stop(): void {
		cancelAnimationFrame(this.#frame);
	}
}
