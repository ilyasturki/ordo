import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { nodeCore } from '$lib/core/node';
import type { Category, Project, Status, Task } from '$lib/core/types';
import { lines as boardLines, type Block, type Line } from './board';
import { dropPlace, entered, placeOp, slotAllowed } from './drag';
import { active, noFilter, type Filter } from './filter';
import { isLine, isNote, isSeparator } from './project';

function task(id: string, o: { status?: Status; parent?: string; kind?: Task['kind'] } = {}): Task {
	return { id, parent_id: o.parent, title: id, status: o.status ?? 'todo', kind: o.kind, created_at: '', updated_at: '' };
}

function cat(id: string, name: string, tasks: Task[] = [], parent?: string): Category {
	return { id, name, parent_id: parent, created_at: '', tasks };
}

const statuses = (...s: string[]): Filter => ({ ...noFilter, statuses: s });

const at = (order: Line[], key: string) => order.findIndex((l) => l.key === key);

/** The list with the block lifted from it and set down at slot. */
function placed(lines: Line[], key: string, slot: number): Line[] {
	const others = lines.filter((l) => l.key !== key);
	expect(slotAllowed(others, slot), `${key} at ${slot}`).toBe(true);
	return [...others.slice(0, slot), lines[at(lines, key)], ...others.slice(slot)];
}

/** The block dragged onto a line: below it going down, above it going up, and above the lines that count a category's rows when that line is a header. */
function drag(lines: Line[], id: string, over: string): Line[] {
	const key = 't:' + id;
	const others = lines.filter((l) => l.key !== key);
	const t = at(others, over);
	if (at(lines, over) > at(lines, key)) return placed(lines, key, t + 1);
	let slot = t;
	if (others[t].kind === 'header') while (slot > 0 && (others[slot - 1].kind === 'hidden' || others[slot - 1].kind === 'empty')) slot--;
	return placed(lines, key, slot);
}

const block = (order: Line[], id: string) => order.find((l) => l.key === 't:' + id) as Block;

function drop(lines: Line[], id: string, over: string) {
	const d = dropPlace(drag(lines, id, over), block(lines, id), lines);
	return d && placeOp(d);
}

const place = (id: string, cat: string, before: string) => ({ action: 'task.place', ids: [id], cat, before });
const row = (id: string) => 't:' + id;
const header = (id: string) => 'c:' + id;

// Feature [a, b (b1), separator, c] with the sub-category UI [d], Fix [e, f done], an empty category, Docs [g done, h].
const project: Project = {
	id: 'p',
	name: 'P',
	created_at: '',
	updated_at: '',
	categories: [
		cat('feat', 'Feature', [task('a'), task('b'), task('b1', { parent: 'b' }), task('sep', { kind: 'separator' }), task('c')]),
		cat('ui', 'UI', [task('d')], 'feat'),
		cat('fix', 'Fix', [task('e'), task('f', { status: 'completed' })]),
		cat('empty', 'Empty'),
		cat('docs', 'Docs', [task('g', { status: 'completed' }), task('h')])
	]
};

const lines = (filter: Filter = noFilter, folds: string[] = []) => boardLines(project, filter, { folds: new Set(folds) });

describe('dropPlace', () => {
	it('lands above the next row under the header above it', () => {
		const all = lines();
		for (const [name, [id, over], want] of [
			['down by one', ['a', row('b')], place('a', 'feat', 'sep')],
			['up by one', ['f', row('e')], place('f', 'fix', 'e')],
			['first under its header', ['c', row('a')], place('c', 'feat', 'a')],
			['last above the next header', ['a', row('c')], place('a', 'feat', '')],
			['above a separator', ['c', row('sep')], place('c', 'feat', 'sep')],
			['a dragged separator', ['sep', row('a')], place('sep', 'feat', 'a')],
			['a row with its subtasks', ['b', row('a')], place('b', 'feat', 'a')],
			['past a header going down: first of the next category', ['c', header('ui')], place('c', 'ui', 'd')],
			['past a header going up: last of the previous one', ['e', header('fix')], place('e', 'ui', '')],
			['a parent category, above its first sub-category', ['d', header('ui')], place('d', 'feat', '')],
			['into an empty category from above', ['f', header('empty')], place('f', 'empty', '')],
			['into an empty category from below', ['g', header('docs')], place('g', 'empty', '')]
		] as const) {
			expect(drop(all, id, over), name).toEqual(want);
		}
	});

	it('never takes a slot above the first header nor below a line that counts rows', () => {
		const all = lines(statuses('todo'));
		expect(all.some((l) => l.kind === 'hidden') && all.some((l) => l.kind === 'empty')).toBe(true);
		const others = all.filter((l) => l.key !== row('e'));
		for (let s = 0; s <= others.length; s++) {
			const above = others[s - 1];
			expect(slotAllowed(others, s), String(s)).toBe(s > 0 && (above.kind === 'header' || above.kind === 'block'));
		}
	});

	it('sends nothing for a drop where the block was lifted', () => {
		const all = lines();
		expect(dropPlace(all, block(all, 'a'), all)).toBeUndefined();
		const there = drag(all, 'a', row('b'));
		const back = drag(there, 'a', row('b'));
		expect(back.map((l) => l.key)).toEqual(all.map((l) => l.key));
		expect(dropPlace(back, block(back, 'a'), all)).toBeUndefined();
		expect(dropPlace(there, block(there, 'a'), there), 'measured against what the lift showed').toBeUndefined();
	});

	it('under a filter keeps the hidden line last of its category', () => {
		const all = lines(statuses('todo'));
		const down = drag(all, 'e', header('empty'));
		expect(dropPlace(down, block(down, 'e'), all)).toEqual({ id: 'e', cat: 'empty', before: '', unfold: false });
		expect(down.slice(at(down, header('fix')), at(down, header('docs')) + 1).map((l) => l.key), 'No tasks keeps its place under the row').toEqual([
			header('fix'),
			'h:fix',
			header('empty'),
			row('e'),
			'n:empty',
			header('docs')
		]);
		expect(entered(down)).toEqual(new Set(['n:empty']));
		const up = drag(down, 'e', header('empty'));
		expect(up.slice(at(up, header('fix')), at(up, header('empty')) + 1).map((l) => l.key)).toEqual([header('fix'), row('e'), 'h:fix', header('empty')]);
		expect(dropPlace(up, block(up, 'e'), all)).toBeUndefined();
		expect(entered(up)).toEqual(new Set());
		const last = drag(all, 'h', header('docs'));
		expect(dropPlace(last, block(last, 'h'), all)).toMatchObject({ cat: 'empty', before: '' });
		expect(last.slice(-5).map((l) => l.key)).toEqual([header('empty'), row('h'), 'n:empty', header('docs'), 'h:docs']);
	});

	// Inbox [p, —done, b done, k done, —next, c, r], Work [w, v, —later, o done], Home [f], Loose [a, x done, y, z].
	const sections: Project = {
		...project,
		categories: [
			cat('inbox', 'Inbox', [
				task('p'),
				task('done', { kind: 'separator' }),
				task('b', { status: 'completed' }),
				task('k', { status: 'completed' }),
				task('next', { kind: 'separator' }),
				task('c'),
				task('r')
			]),
			cat('work', 'Work', [task('w'), task('v'), task('later', { kind: 'separator' }), task('o', { status: 'completed' })]),
			cat('home', 'Home', [task('f')]),
			cat('loose', 'Loose', [task('a'), task('x', { status: 'completed' }), task('y'), task('z')])
		]
	};

	it('under a filter stays in the section it shows in', () => {
		const all = boardLines(sections, statuses('todo'));
		expect(all.some((l) => l.key === row('done') || l.key === row('later'))).toBe(false);
		for (const [name, [id, over], want] of [
			['up onto a separator: above the section hidden whole', ['c', row('next')], place('c', 'inbox', 'done')],
			['down to the last row: above the hidden tail', ['w', row('v')], place('w', 'work', 'later')],
			['up past a header: above the hidden tail', ['f', header('home')], place('f', 'work', 'later')],
			['up past hidden rows: as K, right above the next row', ['z', row('y')], place('z', 'loose', 'y')],
			['down past hidden rows: as J, right below the last row', ['a', row('y')], place('a', 'loose', 'z')],
			['first under its header', ['r', row('p')], place('r', 'inbox', 'p')]
		] as const) {
			expect(drop(all, id, over), name).toEqual(want);
		}
	});

	it('below a folded header lands first there and unfolds it', () => {
		const all = lines(noFilter, ['fix']);
		const down = drag(all, 'c', header('fix'));
		expect(dropPlace(down, block(down, 'c'), all)).toEqual({ id: 'c', cat: 'fix', before: 'e', unfold: true });
		const up = drag(all, 'h', header('empty'));
		expect(dropPlace(up, block(up, 'h'), all)).toMatchObject({ cat: 'fix', before: 'e' });
	});

	it('below a folded parent lands in the parent', () => {
		const all = lines(noFilter, ['feat']);
		expect(all.slice(0, 2).map((l) => l.key)).toEqual([header('feat'), header('fix')]);
		expect(drop(all, 'e', header('fix'))).toEqual(place('e', 'feat', 'a'));
	});
});

type View = { name: string; filter?: Filter; peek?: string; folds?: string[] };

const todo = statuses('todo');

// Sections the filters hide whole: at a category's head, in its middle, at its tail; an empty category, a
// separator and a note shown by their own tag, a sub-category.
const fixtures: Record<string, View[]> = {
	'seed.json': [
		{ name: 'all' },
		{ name: 'todo', filter: todo },
		{ name: 'a peek', filter: todo, peek: 'cccccccc-0000-4000-8000-000000000019' },
		{ name: 'one category', filter: { ...noFilter, categories: ['cccccccc-0000-4000-8000-000000000008'] } },
		{ name: 'folded categories', folds: ['cccccccc-0000-4000-8000-000000000012', 'cccccccc-0000-4000-8000-000000000019'] },
		{ name: 'a folded row', folds: ['aaaaaaaa-0000-4000-8000-000000000001'] }
	],
	'drag.json': [
		{ name: 'all' },
		{ name: 'todo', filter: todo },
		{ name: 'open', filter: statuses('todo', 'in_progress') },
		{ name: 'green', filter: { ...noFilter, tags: ['green'] } },
		{ name: 'blue', filter: { ...noFilter, tags: ['blue'] } },
		{ name: 'a peek', filter: todo, peek: 'cccccccc-0000-4000-8000-000000000010' },
		{ name: 'one category', filter: { ...noFilter, categories: ['cccccccc-0000-4000-8000-000000000018'] } },
		{ name: 'folded categories', folds: ['cccccccc-0000-4000-8000-000000000031', 'cccccccc-0000-4000-8000-000000000010'] }
	]
};

const shape = (order: Line[]) => order.filter((l) => l.kind === 'header' || l.kind === 'block').map((l) => l.key);

const names = (order: Line[]) =>
	order.map((l) => (l.kind === 'header' ? `[${l.category.name}]` : l.kind === 'block' ? l.row.task.title : l.key[0]));

describe('a drop through the core', () => {
	it('lands every row on every slot of the fixtures where it was shown', async () => {
		const core = await nodeCore();
		const failures: string[] = [];
		let drops = 0;
		for (const [file, views] of Object.entries(fixtures)) {
			const raw = readFileSync(new URL(`../../../../testdata/native/${file}`, import.meta.url), 'utf8');
			const project = JSON.parse(raw) as Project;
			for (const v of views) {
				const filter = v.filter ?? noFilter;
				const folds = new Set(v.folds ?? []);
				const all = boardLines(project, filter, { peek: v.peek, folds });
				for (const b of all) {
					if (b.kind !== 'block') continue;
					const others = all.filter((l) => l.key !== b.key);
					for (let s = 0; s <= others.length; s++) {
						if (!slotAllowed(others, s)) continue;
						const order = [...others.slice(0, s), b, ...others.slice(s)];
						const what = `${file} ${v.name}: ${b.row.task.title} at ${s}`;
						const d = dropPlace(order, b, all);
						if (shape(order).join() === shape(all).join()) {
							if (d) failures.push(`${what}: sends ${JSON.stringify(placeOp(d))} for an order it didn't change`);
							continue;
						}
						if (!d) {
							failures.push(`${what}: sends nothing`);
							continue;
						}
						drops++;
						const pid = core.open(raw);
						const answer = core.perform(pid, placeOp(d));
						const after = JSON.parse(core.project(pid)!) as Project;
						core.close(pid);
						if (answer.status !== 'applied') {
							failures.push(`${what}: answered ${answer.status}`);
							continue;
						}
						const id = b.row.task.id;
						const held = new Set(active(filter) ? [id] : []);
						const edited = boardLines(after, filter, { peek: v.peek, held, folds: d.unfold ? new Set([...folds].filter((f) => f !== d.cat)) : folds });
						// A separator dragged off hands its rows to the section above, whose lines show with them.
						const left = project.categories.find((c) => c.tasks.some((t) => t.id === id))!.tasks;
						const k = left.findIndex((t) => t.id === id);
						const prevSep = left.slice(0, k).findLastIndex(isSeparator);
						const joined = new Set(isSeparator(b.row.task) ? left.slice(Math.max(0, prevSep), k).filter(isLine).map((t) => 't:' + t.id) : []);
						// A note shows with the section the row lands in, wherever the row goes in it.
						const into = after.categories.find((c) => c.id === d.cat)!.tasks;
						const x = into.findIndex((t) => t.id === id);
						const start = Math.max(0, into.slice(0, x).findLastIndex(isSeparator));
						const next = into.slice(x + 1).findIndex(isSeparator);
						const end = next < 0 ? into.length : x + 1 + next;
						const notes = new Set(into.slice(start, end).filter(isNote).map((t) => 't:' + t.id));
						const was = shape(order);
						const landedAll = shape(edited).filter((key) => (!joined.has(key) && !notes.has(key)) || was.includes(key));
						// A separator or a note whose section the drop emptied hides with it.
						const emptied = new Set(
							order.filter((l) => l.kind === 'block' && isLine(l.row.task) && l.key !== b.key && !landedAll.includes(l.key)).map((l) => l.key)
						);
						const shown = was.filter((key) => !emptied.has(key));
						const landed = d.unfold ? landedAll.filter((key) => shown.includes(key)) : landedAll;
						const above = (keys: string[]) => keys[keys.indexOf(b.key) - 1];
						if (landed.join() !== shown.join() || above(landedAll) !== above(shown)) {
							failures.push(`${what}: shown ${names(order.filter((l) => shown.includes(l.key)))}, landed ${names(edited.filter((l) => l.kind === 'header' || l.kind === 'block'))}`);
						}
					}
				}
			}
		}
		expect(failures.slice(0, 20), `${failures.length} of ${drops} drops`).toEqual([]);
		expect(drops).toBeGreaterThan(1000);
	}, 60_000);
});
