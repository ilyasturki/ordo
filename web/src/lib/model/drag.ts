import type { Op } from '#lib/core/types.ts';
import { blockEnd, type Block, type Header, type Line, type Section } from './board';

/** A line a drag lifts and drops around: a block, or a hidden section's line. */
export type Moving = Block | Section;

const moving = (line: Line): line is Moving => line.kind === 'block' || line.kind === 'section';

/** A slot is the index, in the list without the lifted line, of the line it lands above. */
export function slotAllowed(others: Line[], slot: number): boolean {
	if (slot < 1 || slot > others.length) return false;
	const above = others[slot - 1];
	return above.kind === 'header' || moving(above);
}

/** The "No tasks" lines of categories a lifted line has entered, kept blank so the list holds still. */
export function entered(order: Line[]): Set<string> {
	const out = new Set<string>();
	let rows = false;
	for (const line of order) {
		if (line.kind === 'header') rows = false;
		else if (moving(line)) rows = true;
		else if (line.kind === 'empty' && rows) out.add(line.key);
	}
	return out;
}

export type Drop = { id: string; cat: string; before: string; unfold: boolean };

export function placeOp(drop: Drop): Op {
	return { action: 'task.place', ids: [drop.id], cat: drop.cat, before: drop.before };
}

function shape(order: Line[]): string[] {
	return order.filter((l) => l.kind === 'header' || moving(l)).map((l) => l.key);
}

/** Hidden rows around the drop stay below a line moved down, as J, and above one moved up, as K; a collapsed section's line stands for its run. */
export function dropPlace(order: Line[], lifted: Moving, before: Line[]): Drop | undefined {
	const now = shape(order);
	const was = shape(before);
	if (now.join('\n') === was.join('\n')) return undefined;
	const i = order.findIndex((l) => l.key === lifted.key);
	let h = i - 1;
	while (h >= 0 && order[h].kind !== 'header') h--;
	if (i < 0 || h < 0) return undefined;
	const header = order[h] as Header;
	const id = lifted.row.task.id;
	const all = header.category.tasks;
	const at = all.findIndex((t) => t.id === id);
	const tasks = at < 0 ? all : [...all.slice(0, at), ...all.slice(blockEnd(all, at))];
	if (header.folded) return { id, cat: header.cat, before: tasks.find((t) => !t.parent_id)?.id ?? '', unfold: true };
	let above: Moving | undefined;
	for (const l of order.slice(h + 1, i)) if (moving(l)) above = l;
	let below: Moving | undefined;
	for (const l of order.slice(i + 1)) {
		if (l.kind === 'header') break;
		if (moving(l)) {
			below = l;
			break;
		}
	}
	const index = (l: Moving) => tasks.findIndex((t) => t.id === l.row.task.id);
	const to = below && index(below) >= 0 ? index(below) : tasks.length;
	const past = (l: Moving) => {
		const k = index(l);
		if (k < 0) return to;
		return l.kind === 'block' ? blockEnd(tasks, k) : k + 1 + (l.revealed ? 0 : l.count);
	};
	const from = above ? Math.min(past(above), to) : 0;
	const slot = now.indexOf(lifted.key) < was.indexOf(lifted.key) ? to : from;
	return { id, cat: header.cat, before: tasks[slot]?.id ?? '', unfold: false };
}
