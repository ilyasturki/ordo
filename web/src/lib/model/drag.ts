import type { Op } from '#lib/core/types.ts';
import { blockEnd, type Block, type Header, type Line } from './board';
import { isSeparator } from './project';

/** A slot is the index, in the list without the lifted block, of the line it lands above. */
export function slotAllowed(others: Line[], slot: number): boolean {
	if (slot < 1 || slot > others.length) return false;
	const above = others[slot - 1].kind;
	return above === 'header' || above === 'block';
}

/** The "No tasks" lines of categories a lifted block has entered, kept blank so the list holds still. */
export function entered(order: Line[]): Set<string> {
	const out = new Set<string>();
	let rows = false;
	for (const line of order) {
		if (line.kind === 'header') rows = false;
		else if (line.kind === 'block') rows = true;
		else if (line.kind === 'empty' && rows) out.add(line.key);
	}
	return out;
}

export type Drop = { id: string; cat: string; before: string; unfold: boolean };

export function placeOp(drop: Drop): Op {
	return { action: 'task.place', ids: [drop.id], cat: drop.cat, before: drop.before };
}

function shape(order: Line[]): string[] {
	return order.filter((l) => l.kind === 'header' || l.kind === 'block').map((l) => l.key);
}

/** Hidden rows around the drop stay below a block moved down, as J, and above one moved up, as K, unless a hidden separator is among them. */
export function dropPlace(order: Line[], block: Block, lifted: Line[]): Drop | undefined {
	const now = shape(order);
	const was = shape(lifted);
	if (now.join('\n') === was.join('\n')) return undefined;
	const i = order.findIndex((l) => l.key === block.key);
	let h = i - 1;
	while (h >= 0 && order[h].kind !== 'header') h--;
	if (i < 0 || h < 0) return undefined;
	const header = order[h] as Header;
	const id = block.row.task.id;
	const all = header.category.tasks;
	const at = all.findIndex((t) => t.id === id);
	const tasks = at < 0 ? all : [...all.slice(0, at), ...all.slice(blockEnd(all, at))];
	if (header.folded) return { id, cat: header.cat, before: tasks.find((t) => !t.parent_id)?.id ?? '', unfold: true };
	let above: Block | undefined;
	for (const l of order.slice(h + 1, i)) if (l.kind === 'block') above = l;
	let below: Block | undefined;
	for (const l of order.slice(i + 1)) {
		if (l.kind === 'header') break;
		if (l.kind === 'block') {
			below = l;
			break;
		}
	}
	const index = (b: Block) => tasks.findIndex((t) => t.id === b.row.task.id);
	const to = below && index(below) >= 0 ? index(below) : tasks.length;
	const from = !above ? 0 : index(above) < 0 ? to : Math.min(blockEnd(tasks, index(above)), to);
	let slot = from;
	if (now.indexOf(block.key) < was.indexOf(block.key)) {
		slot = to;
		for (let j = from; j < to; j++) {
			if (isSeparator(tasks[j])) {
				slot = j;
				break;
			}
		}
	}
	return { id, cat: header.cat, before: tasks[slot]?.id ?? '', unfold: false };
}
