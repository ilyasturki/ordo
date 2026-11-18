import type { Category, Project, Task } from '$lib/core/types';
import { display, preview, refs } from './description';
import { categoryVisible, lineMatches, taskVisible, type Filter } from './filter';
import { finished, isLine, isSeparator } from './project';

export type Row = {
	task: Task;
	childDone: number;
	childTotal: number;
	images: number;
	preview: string;
	/** Shown only by a peek, or held where an edit left it though the filter no longer matches. */
	muted: boolean;
	folded: boolean;
};

/** A folded header also tells what the TUI's does of the rows it hides. */
export type Header = {
	kind: 'header';
	key: string;
	cat: string;
	category: Category;
	parentName: string;
	folded: boolean;
	status: string;
	done: number;
	total: number;
};

/** A top-level row with the subtasks the filter shows. */
export type Block = { kind: 'block'; key: string; cat: string; row: Row; children: Row[] };
export type Hidden = { kind: 'hidden'; key: string; cat: string; count: number; peeking: boolean };
export type NoTasks = { kind: 'empty'; key: string; cat: string };
export type Line = Header | Block | Hidden | NoTasks;

export function blockEnd(tasks: Task[], i: number): number {
	let end = i + 1;
	if (tasks[i].parent_id) return end;
	while (end < tasks.length && tasks[end].parent_id === tasks[i].id) end++;
	return end;
}

export function shownRows(cat: Category, filter: Filter, held: Set<string> = new Set()): boolean[] {
	const tasks = cat.tasks;
	const shown = tasks.map(
		(t) => held.has(t.id) || (isLine(t) ? lineMatches(filter, t, cat) : taskVisible(filter, t, cat))
	);
	// A line shows with a task of its section, from a separator to the next or the category's end.
	let start = 0;
	while (start < tasks.length) {
		let end = start + 1;
		while (end < tasks.length && !isSeparator(tasks[end])) end++;
		let any = false;
		for (let j = start; j < end; j++) any ||= !isLine(tasks[j]) && shown[j];
		if (any) for (let j = start; j < end; j++) if (isLine(tasks[j])) shown[j] = true;
		start = end;
	}
	for (let i = 0; i < tasks.length; ) {
		const end = blockEnd(tasks, i);
		for (let j = i + 1; j < end; j++) shown[i] ||= shown[j];
		i = end;
	}
	return shown;
}

/** Whether a separator or a note added first in cat would show under the filter. */
export function addedLineShows(cat: Category, filter: Filter, kind: 'note' | 'separator'): boolean {
	const added = { id: '+', title: '', status: 'todo', kind, created_at: '', updated_at: '' } as Task;
	return shownRows({ ...cat, tasks: [added, ...cat.tasks] }, filter)[0];
}

function row(task: Task, cat: Category, filter: Filter, held: Set<string>, hidden: boolean): Row {
	return {
		task,
		childDone: 0,
		childTotal: 0,
		images: refs(task.description).length,
		preview: preview(task.description),
		muted: !isLine(task) && (hidden || (held.has(task.id) && !taskVisible(filter, task, cat))),
		folded: false
	};
}

function aggregateStatus(tasks: Task[]): string {
	if (tasks.length === 0) return '';
	if (tasks.some((t) => t.status === 'in_progress')) return 'in_progress';
	if (tasks.every(finished)) return 'completed';
	if (tasks.some(finished)) return 'in_progress';
	return 'todo';
}

function header(cat: Category, parentName: string): Header {
	return { kind: 'header', key: 'c:' + cat.id, cat: cat.id, category: cat, parentName, folded: false, status: '', done: 0, total: 0 };
}

/** A parent speaks for its sub-categories too. */
function foldedHeader(project: Project, cat: Category, parentName: string): Header {
	const tree = cat.parent_id
		? cat.tasks
		: project.categories.filter((c) => c.id === cat.id || c.parent_id === cat.id).flatMap((c) => c.tasks);
	const tasks = tree.filter((t) => !isLine(t));
	const counted = !cat.parent_id;
	return {
		...header(cat, parentName),
		folded: true,
		status: aggregateStatus(tasks),
		done: counted ? tasks.filter(finished).length : 0,
		total: counted ? tasks.length : 0
	};
}

/**
 * The list the TUI draws: every category header, rows in file order with the finished ones in
 * place, and a line counting what the filter hides. A folded category shows its header alone and
 * hides its sub-categories; a folded row hides its subtasks.
 */
export function lines(
	project: Project,
	filter: Filter,
	{ peek, held = new Set(), folds = new Set() }: { peek?: string; held?: Set<string>; folds?: Set<string> } = {}
): Line[] {
	const names = new Map(project.categories.map((c) => [c.id, c.name]));
	const out: Line[] = [];
	let foldedParent = '';
	for (const cat of project.categories) {
		if (!cat.parent_id) foldedParent = '';
		else if (cat.parent_id === foldedParent) continue;
		const parentName = cat.parent_id ? (names.get(cat.parent_id) ?? '') : '';
		if (folds.has(cat.id)) {
			if (!cat.parent_id) foldedParent = cat.id;
			out.push(foldedHeader(project, cat, parentName));
			continue;
		}
		out.push(header(cat, parentName));
		const first = out.length;
		const shown = shownRows(cat, filter, held);
		const peeking = peek === cat.id;
		const tasks = cat.tasks;
		for (let i = 0; i < tasks.length; ) {
			const end = blockEnd(tasks, i);
			if (shown[i] || peeking) {
				const folded = folds.has(tasks[i].id);
				const children: Row[] = [];
				for (let j = i + 1; j < end; j++) {
					if (!folded && (shown[j] || peeking)) children.push(row(tasks[j], cat, filter, held, !shown[j]));
				}
				const kids = tasks.slice(i + 1, end);
				out.push({
					kind: 'block',
					key: 't:' + tasks[i].id,
					cat: cat.id,
					row: {
						...row(tasks[i], cat, filter, held, !shown[i]),
						childDone: kids.filter(finished).length,
						childTotal: kids.length,
						folded: folded && kids.length > 0
					},
					children
				});
			}
			i = end;
		}
		const hidden = tasks.filter((t, i) => !shown[i] && !isLine(t)).length;
		if (hidden > 0) out.push({ kind: 'hidden', key: 'h:' + cat.id, cat: cat.id, count: hidden, peeking });
		if (out.length === first && !project.categories.some((c) => c.parent_id === cat.id)) {
			out.push({ kind: 'empty', key: 'n:' + cat.id, cat: cat.id });
		}
	}
	return out;
}

/** The TUI's smartcase: an uppercase letter in the query makes it case-sensitive. */
function contains(text: string, query: string): boolean {
	return query === query.toLowerCase() ? text.toLowerCase().includes(query) : text.includes(query);
}

function matches(task: Task, query: string): boolean {
	return contains(task.title, query) || contains(display(task.description), query);
}

/** The TUI's search as a narrower list: a row stays when it or a subtask matches, every row under a matching category. */
export function search(all: Line[], query: string): Line[] {
	const q = query.trim();
	if (q === '') return all;
	const out: Line[] = [];
	let pending: Header | undefined;
	let named = false;
	for (const line of all) {
		if (line.kind === 'header') {
			pending = line;
			named = contains(line.category.name, q);
			if (named) out.push(line);
		} else if (line.kind === 'block') {
			const children = named ? line.children : line.children.filter((c) => matches(c.task, q));
			if (named || children.length > 0 || matches(line.row.task, q)) {
				if (!named && pending) out.push(pending);
				pending = undefined;
				out.push({ ...line, children });
			}
		} else if (line.kind === 'empty' && named) {
			out.push(line);
		}
	}
	return out;
}

/** The category the bar names and an add goes to: the focus, unless the filter hides it. */
export function target(project: Project, filter: Filter, focus: string | undefined): Category | undefined {
	return (
		project.categories.find((c) => c.id === focus && categoryVisible(filter, c)) ??
		project.categories.find((c) => categoryVisible(filter, c)) ??
		project.categories[0]
	);
}

/** The rows a cursor walks, in the order the list draws them: headers, tasks and subtasks. */
export function stops(shown: Line[]): string[] {
	const out: string[] = [];
	for (const line of shown) {
		if (line.kind === 'header') out.push(line.key);
		else if (line.kind === 'block') {
			out.push(line.key);
			for (const c of line.children) out.push('t:' + c.task.id);
		} else if (line.kind === 'hidden') out.push(line.key);
	}
	return out;
}
