import { describe, expect, it } from 'vitest';
import type { Category, Project, Status, Task } from '#lib/core/types.ts';
import { addedLineShows, lines, search, shownRows, target, type Header, type Line } from './board';
import { fields, join, preview, refs } from './description';
import { active, categoryVisible, fromFile, hasValues, noFilter, pruned, toFile, type Filter } from './filter';
import { hasVersion, nextTag, priorityLabel, priorities, rankProjects } from './project';

function task(id: string, o: { status?: Status; parent?: string; priority?: string; tag?: string; kind?: Task['kind']; title?: string; description?: string } = {}): Task {
	return {
		id,
		parent_id: o.parent,
		title: o.title ?? id,
		status: o.status ?? 'todo',
		priority: o.priority,
		tag_color: o.tag,
		kind: o.kind,
		description: o.description,
		created_at: '',
		updated_at: ''
	};
}

function cat(id: string, name: string, tasks: Task[] = [], parent?: string): Category {
	return { id, name, parent_id: parent, created_at: '', tasks };
}

function project(categories: Category[]): Project {
	return { id: 'p', name: 'P', created_at: '', updated_at: '', categories };
}

const f = (o: Partial<Filter>): Filter => ({ ...noFilter, ...o });

const board = project([
	cat('feat', 'Feature', [
		task('a', { status: 'in_progress' }),
		task('a1', { status: 'completed', parent: 'a' }),
		task('a2', { parent: 'a', priority: 'high' }),
		task('sep', { kind: 'separator' }),
		task('b', { status: 'completed' }),
		task('c', { tag: 'green-diamond' })
	]),
	cat('ui', 'UI', [task('d', { status: 'cancelled' })], 'feat'),
	cat('fix', 'Fix', [task('e', { priority: 'high' })])
]);

function shape(ls: Line[]): string[] {
	return ls.map((l) => {
		switch (l.kind) {
			case 'header':
				return `# ${l.category.name}` + (l.parentName ? ` < ${l.parentName}` : '');
			case 'block':
				return [l.row.task.id + (l.row.muted ? '~' : ''), ...l.children.map((c) => '  ' + c.task.id + (c.muted ? '~' : ''))].join(',');
			case 'hidden':
				return `(${l.count} hidden${l.peeking ? ', peeking' : ''})`;
			case 'empty':
				return '(no tasks)';
		}
	});
}

const blocks = (ls: Line[]) => ls.flatMap((l) => (l.kind === 'block' ? [l] : []));

describe('board', () => {
	it('targets the focus unless the filter hides it', () => {
		expect(target(board, noFilter, 'fix')?.id).toBe('fix');
		expect(target(board, noFilter, undefined)?.id).toBe('feat');
		expect(target(board, noFilter, 'gone')?.id).toBe('feat');
		expect(target(board, f({ categories: ['fix'] }), 'feat')?.id).toBe('fix');
		expect(target(board, f({ categories: ['feat'] }), 'ui')?.id).toBe('ui');
		expect(target(project([]), noFilter, undefined)).toBeUndefined();
	});

	it('says so for an empty category without sub-categories', () => {
		const empty = project([cat('a', 'A'), cat('b', 'B'), cat('b1', 'B1', [], 'b'), cat('c', 'C', [task('s', { kind: 'separator' })])]);
		expect(shape(lines(empty, noFilter))).toEqual(['# A', '(no tasks)', '# B', '# B1 < B', '(no tasks)', '# C', 's']);
		expect(shape(lines(empty, f({ statuses: ['todo'] })))).toEqual(['# A', '(no tasks)', '# B', '# B1 < B', '(no tasks)', '# C', '(no tasks)']);
	});

	it('counts the rows a filter hides', () => {
		expect(shape(lines(board, f({ tags: ['blue'] })))).toEqual(['# Feature', '(5 hidden)', '# UI < Feature', '(1 hidden)', '# Fix', '(1 hidden)']);
	});

	it('keeps finished rows in place with no filter', () => {
		expect(shape(lines(board, noFilter))).toEqual(['# Feature', 'a,  a1,  a2', 'sep', 'b', 'c', '# UI < Feature', 'd', '# Fix', 'e']);
	});

	it('shows a parent when a child matches, and hides separators', () => {
		expect(shape(lines(board, f({ priorities: ['high'] })))).toEqual(['# Feature', 'a,  a2', '(3 hidden)', '# UI < Feature', '(1 hidden)', '# Fix', 'e']);
	});

	it('shows lines with their section under the filter', () => {
		const sections = (green: string[] = []) => {
			const line = (id: string, kind: 'note' | 'separator') => task(id, { kind, tag: green.includes(id) ? 'green' : undefined });
			return project([
				cat('c1', 'A', [task('t1'), line('s1', 'separator'), line('n1', 'note'), task('t2', { status: 'completed', tag: 'blue' }), line('s2', 'separator'), task('t3'), line('n2', 'note'), line('s4', 'separator')]),
				cat('c2', 'B', [line('s3', 'separator'), task('t4')])
			]);
		};
		const shown = (p: Project, filter: Filter) => p.categories.flatMap((c) => c.tasks.filter((_, i) => shownRows(c, filter)[i]).map((t) => t.id));
		expect(shown(sections(), f({ statuses: ['todo'] }))).toEqual(['t1', 's2', 't3', 'n2', 's3', 't4']);
		expect(shown(sections(), f({ statuses: ['completed'] }))).toEqual(['s1', 'n1', 't2']);
		expect(shown(sections(), f({ priorities: ['high'] }))).toEqual([]);
		expect(shown(sections(['n2', 's4']), f({ tags: ['green'] }))).toEqual(['n2', 's4']);
		expect(shown(sections(['s4']), f({ statuses: ['todo'], tags: ['green'] }))).toEqual(['s4']);
		expect(shown(sections(), f({ categories: ['c1'] }))).toEqual(['t1', 's1', 'n1', 't2', 's2', 't3', 'n2', 's4']);
		expect(shown(sections(['s3']), f({ categories: ['c1'], tags: ['green'] }))).toEqual([]);
		expect(shown(sections(), f({ statuses: ['todo'], off: true }))).toHaveLength(10);

		const p = sections();
		const todo = f({ statuses: ['todo'] });
		expect(addedLineShows(p.categories[0], todo, 'note')).toBe(true);
		expect(addedLineShows(p.categories[1], todo, 'separator')).toBe(false);
		expect(addedLineShows(p.categories[1], noFilter, 'separator')).toBe(true);
	});

	it('keeps sub-categories under a selected category', () => {
		const sel = f({ categories: ['feat'] });
		expect(categoryVisible(sel, board.categories[1])).toBe(true);
		expect(categoryVisible(sel, board.categories[2])).toBe(false);
		expect(shape(lines(board, sel))).toEqual(['# Feature', 'a,  a1,  a2', 'sep', 'b', 'c', '# UI < Feature', 'd', '# Fix', '(1 hidden)']);
	});

	it('reads medium and untagged as the empty buckets, and a diamond as its dot', () => {
		const ids = blocks(lines(board, f({ priorities: [''], tags: [''] }))).flatMap((b) => [b.row, ...b.children]).map((r) => r.task.id);
		expect(ids).toEqual(['a', 'a1', 'sep', 'b', 'd']);
		expect(blocks(lines(board, f({ tags: ['green'] }))).map((b) => b.row.task.id)).toEqual(['sep', 'c']);
	});

	it('mutes a peek and a held row', () => {
		const todo = f({ statuses: ['todo'] });
		expect(shape(lines(board, todo, { peek: 'feat' }))).toEqual(['# Feature', 'a,  a1~,  a2', 'sep', 'b~', 'c', '(2 hidden, peeking)', '# UI < Feature', '(1 hidden)', '# Fix', 'e']);
		expect(shape(lines(board, todo, { held: new Set(['b']) })).slice(0, 6)).toEqual(['# Feature', 'a,  a2', 'sep', 'b~', 'c', '(1 hidden)']);
	});

	it('folds categories and rows', () => {
		expect(shape(lines(board, noFilter, { folds: new Set(['fix']) }))).toEqual(['# Feature', 'a,  a1,  a2', 'sep', 'b', 'c', '# UI < Feature', 'd', '# Fix']);
		const parent = lines(board, noFilter, { folds: new Set(['feat']) });
		expect(shape(parent)).toEqual(['# Feature', '# Fix', 'e']);
		expect(parent[0]).toMatchObject({ folded: true, status: 'in_progress', done: 3, total: 6 });
		const sub = lines(board, noFilter, { folds: new Set(['ui']) }).find((l) => l.kind === 'header' && l.cat === 'ui') as Header;
		expect(sub).toMatchObject({ folded: true, status: 'completed', total: 0 });
		const a = blocks(lines(board, noFilter, { folds: new Set(['a']) }))[0];
		expect(a.children).toEqual([]);
		expect(a.row).toMatchObject({ folded: true, childDone: 1, childTotal: 2 });
		expect(blocks(lines(board, noFilter, { folds: new Set(['b']) })).find((b) => b.row.task.id === 'b')!.row.folded).toBe(false);
		expect(shape(lines(board, f({ statuses: ['todo'] }), { folds: new Set(['feat', 'fix']) }))).toEqual(['# Feature', '# Fix']);
		expect(shape(lines(board, noFilter, { folds: new Set(['gone']) }))).toEqual(shape(lines(board, noFilter)));
	});

	it('turns a filter off without losing it', () => {
		const off = f({ priorities: ['high'], categories: ['fix'], off: true });
		expect(active(off)).toBe(false);
		expect(hasValues(off)).toBe(true);
		expect(shape(lines(board, off))).toEqual(shape(lines(board, noFilter)));
		expect(pruned(f({ categories: ['feat', 'gone'] }), board).categories).toEqual(['feat']);
	});

	it('writes and reads the view file shape', () => {
		expect(toFile(f({ priorities: ['', 'high'], tags: [''] }))).toEqual({ priorities: ['high', 'none'], tags: ['none'] });
		expect(toFile(f({ statuses: ['todo'], off: true }))).toEqual({ off: true, statuses: ['todo'] });
		expect(toFile(f({ off: true }))).toEqual({});
		expect(fromFile({ priorities: ['medium', 'none', 'high'], tags: ['none'] })).toEqual(f({ priorities: ['', 'high'], tags: [''] }));
	});
});

describe('search', () => {
	const searched = project([
		cat('feat', 'Feature', [
			task('a', { title: 'Parent' }),
			task('a1', { title: 'Pair a phone', parent: 'a' }),
			task('a2', { title: 'Other child', parent: 'a' }),
			task('sep', { title: 'Later', kind: 'separator' }),
			task('b', { title: 'Done', description: 'Reach ordo\n![holiday](attachments/00000000000000000000000000000000.png)' })
		]),
		cat('docs', 'Docs', [task('c', { title: 'Guide' }), task('d', { title: 'Notes' })]),
		cat('idle', 'Idle')
	]);
	const found = (q: string, filter = noFilter) => shape(search(lines(searched, filter), q));

	it('keeps matching rows under their category, smartcase', () => {
		expect(found('pair')).toEqual(['# Feature', 'a,  a1']);
		expect(found('Pair')).toEqual(['# Feature', 'a,  a1']);
		expect(found('PAIR')).toEqual([]);
		expect(found('later')).toEqual(['# Feature', 'sep']);
		expect(found('reach')).toEqual(['# Feature', 'b']);
		expect(found('docs')).toEqual(['# Docs', 'c', 'd']);
		expect(found('idle')).toEqual(['# Idle', '(no tasks)']);
		expect(found('▣ holiday')).toEqual(['# Feature', 'b']);
		expect(found('attachments/')).toEqual([]);
		expect(found('pair', f({ statuses: ['completed'] }))).toEqual([]);
	});

	it('looks through folds', () => {
		expect(shape(search(lines(board, noFilter), 'a1'))).toEqual(['# Feature', 'a,  a1']);
	});
});

describe('description', () => {
	const img = '![shot](attachments/0123456789abcdef0123456789abcdef.png)';

	it('splits details from the image lines under them', () => {
		expect(fields(`Some text\n${img}\n`)).toEqual({ text: 'Some text', images: [img] });
		expect(fields(`Some text\n\n${img}`).images).toEqual([]);
		expect(fields(`${img}\nafter`)).toEqual({ text: `${img}\nafter`, images: [] });
		expect(join('Some text', [img])).toBe(`Some text\n${img}`);
		expect(join('  ', [img])).toBe(img);
	});

	it('previews text without images and reads refs', () => {
		expect(preview(`  one   two \n\n${img}\nthree\nfour`)).toBe('one two\nthree');
		expect(refs(`${img}\n${img}\n![x](https://e.com/a.png)`)).toEqual([
			{ name: 'shot', target: 'attachments/0123456789abcdef0123456789abcdef.png', file: '0123456789abcdef0123456789abcdef.png' }
		]);
	});
});

describe('project', () => {
	it('reads versions and priorities as the phone does', () => {
		expect(hasVersion('v1.2')).toBe(true);
		expect(hasVersion('Release 1.2.3')).toBe(true);
		expect(hasVersion('1.2.3.4')).toBe(false);
		expect(hasVersion('Next')).toBe(false);
		expect(priorities.map(priorityLabel)).toEqual(['Critical', 'High', 'Medium', 'Low', 'Trivial']);
	});

	it('ranks projects by recency inside groups, the ungrouped last', () => {
		const p = (id: string, updated: string, group?: string): Project => ({ id, name: id, group, created_at: '', updated_at: updated, categories: [] });
		const ranked = rankProjects(
			[p('a', '2026-01-01T00:00:00Z'), p('w1', '2026-01-03T00:00:00Z', 'Work'), p('h', '2026-01-04T00:00:00Z', 'home'), p('w2', '2026-01-05T00:00:00Z', ' work ')],
			{ h: Date.parse('2026-02-01T00:00:00Z') }
		);
		expect(ranked.map((x) => x.id)).toEqual(['h', 'w2', 'w1', 'a']);
	});
});

describe('nextTag', () => {
	const round = (others: Task[]) => {
		const row = task('r');
		const c = cat('c', 'C', [row, ...others]);
		const got: string[] = [];
		for (let i = 0; i <= 4; i++) {
			row.tag_color = nextTag(c, row);
			got.push(row.tag_color);
		}
		return got;
	};

	it('starts at the first dot no other open task wears, and reaches every dot', () => {
		expect(round([])).toEqual(['green', 'blue', 'magenta', 'cyan', '']);
		expect(round([task('a', { tag: 'green' }), task('b', { tag: 'blue-diamond' })])).toEqual(['magenta', 'cyan', 'green', 'blue', '']);
	});

	it('leaves out finished tasks and lines, and takes the least worn when all are', () => {
		expect(round([task('a', { tag: 'green', status: 'completed' }), task('n', { tag: 'green', kind: 'note' })])[0]).toBe('green');
		const worn = ['green', 'green', 'blue', 'magenta', 'cyan', 'cyan'].map((tag, i) => task('t' + i, { tag }));
		expect(round(worn)[0]).toBe('blue');
	});
});
