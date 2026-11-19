import type { Category, Project, Status, Task } from '#lib/core/types.ts';

export const statuses: Status[] = ['todo', 'in_progress', 'completed', 'cancelled'];

export function statusLabel(status: string): string {
	return { todo: 'Todo', in_progress: 'In Progress', completed: 'Completed', cancelled: 'Cancelled' }[status] ?? status;
}

/** What status.cycle steps to. */
export function nextStatus(status: string): Status {
	return ({ todo: 'in_progress', in_progress: 'completed', completed: 'cancelled' } as Record<string, Status>)[status] ?? 'todo';
}

/** Medium is the missing field (docs/file-format.md), most urgent first. */
export const priorities = ['critical', 'high', '', 'low', 'trivial'];

export function priorityOf(task: Task): string {
	return task.priority === 'medium' ? '' : (task.priority ?? '');
}

export function priorityLabel(priority: string): string {
	return priority === '' ? 'Medium' : priority[0].toUpperCase() + priority.slice(1);
}

/** Red for up, teal for down: the phone's marks. */
export function priorityTone(priority: string): 'up' | 'down' | '' {
	if (priority === 'critical' || priority === 'high') return 'up';
	if (priority === 'low' || priority === 'trivial') return 'down';
	return '';
}

export const tags = ['green', 'blue', 'magenta', 'cyan'];

/** A diamond mark from a file written before the four dots reads as its colour's dot. */
export function tagDot(color: string | undefined): string {
	return (color ?? '').replace(/-diamond$/, '');
}

export function tagLabel(color: string): string {
	const dot = tagDot(color);
	return dot === '' ? 'None' : dot[0].toUpperCase() + dot.slice(1);
}

/** What `t` gives the row, as the TUI's tag.cycle does: a round starting at the first dot no other open task of the category wears, ending on none. */
export function nextTag(cat: Category, row: Task): string {
	const open = new Map<string, number>();
	for (const t of cat.tasks) {
		if (t.id === row.id || isLine(t) || !t.tag_color || finished(t)) continue;
		open.set(tagDot(t.tag_color), (open.get(tagDot(t.tag_color)) ?? 0) + 1);
	}
	let free = tags[0];
	for (const c of tags) {
		if ((open.get(c) ?? 0) < (open.get(free) ?? 0)) free = c;
		if (!open.get(c)) {
			free = c;
			break;
		}
	}
	const now = tagDot(row.tag_color);
	if (now === '') return free;
	const at = tags.indexOf(now);
	if (at < 0) return '';
	const next = tags[(at + 1) % tags.length];
	return next === free ? '' : next;
}

export const isLine = (t: Task) => (t.kind ?? '') !== '';
export const isNote = (t: Task) => t.kind === 'note';
export const isSeparator = (t: Task) => t.kind === 'separator';
export const finished = (t: Task) => t.status === 'completed' || t.status === 'cancelled';

export function category(project: Project, id: string): Category | undefined {
	return project.categories.find((c) => c.id === id);
}

/** The category's name, after its parent's for a sub-category. */
export function categoryPath(project: Project, cat: Category): string {
	const parent = cat.parent_id ? category(project, cat.parent_id) : undefined;
	return parent ? `${parent.name} › ${cat.name}` : cat.name;
}

export function find(project: Project, taskId: string): { cat: Category; task: Task } | undefined {
	for (const cat of project.categories) {
		const task = cat.tasks.find((t) => t.id === taskId);
		if (task) return { cat, task };
	}
	return undefined;
}

const versionPattern = /\bv?\d+\.\d+(?:\.\d+)?/g;

/** The name holds a version as internal/domain reads one: v1.2 or 1.2.3, not the start of 1.2.3.4. */
export function hasVersion(name: string): boolean {
	for (const m of name.matchAll(versionPattern)) {
		const rest = name.slice(m.index + m[0].length);
		if (!(rest.length >= 2 && rest[0] === '.' && rest[1] >= '0' && rest[1] <= '9')) return true;
	}
	return false;
}

/** Whether "Add next version" has a version to go from: the category's own name, else its parent's. */
export function versioned(project: Project, cat: Category): boolean {
	const parent = cat.parent_id ? category(project, cat.parent_id) : undefined;
	return hasVersion(cat.name) || (parent !== undefined && hasVersion(parent.name));
}

/** The TUI picker's grouping: groups match trimmed and case-insensitive, "" is no group. */
export function groupKey(project: Project): string {
	return (project.group ?? '').trim().toLowerCase();
}

const collator = new Intl.Collator(undefined, { sensitivity: 'base' });

/**
 * Most recently used first, a tie by name, gathered into groups: each group where its most recent
 * project stands, the projects without one last.
 */
export function rankProjects(projects: Project[], opened: Record<string, number>): Project[] {
	const recency = (p: Project) => Math.max(opened[p.id] ?? 0, Date.parse(p.updated_at) || 0);
	const ranked = [...projects].sort(
		(a, b) => recency(b) - recency(a) || collator.compare(a.name, b.name) || a.id.localeCompare(b.id)
	);
	const lead = new Map<string, number>();
	ranked.forEach((p, i) => {
		if (!lead.has(groupKey(p))) lead.set(groupKey(p), i);
	});
	lead.set('', ranked.length);
	return ranked
		.map((p, i) => ({ p, i }))
		.sort((a, b) => lead.get(groupKey(a.p))! - lead.get(groupKey(b.p))! || a.i - b.i)
		.map(({ p }) => p);
}

/** The groups in use, in the order of projects, each spelt as its first project has it. */
export function projectGroups(projects: Project[]): string[] {
	const seen = new Map<string, string>();
	for (const p of projects) {
		const key = groupKey(p);
		if (key !== '' && !seen.has(key)) seen.set(key, (p.group ?? '').trim());
	}
	return [...seen.values()];
}

/** The tasks a category's delete takes: its own and its sub-categories'. */
export function countTasks(project: Project, cat: Category): number {
	return project.categories
		.filter((c) => c.id === cat.id || c.parent_id === cat.id)
		.reduce((n, c) => n + c.tasks.filter((t) => !isLine(t)).length, 0);
}

/** What deleting a category takes with it, in the TUI's words. */
export function deletedWith(project: Project, cat: Category): string {
	const subs = project.categories.filter((c) => c.parent_id === cat.id).length;
	const n = countTasks(project, cat);
	const tasks = n === 1 ? '1 task' : `${n} tasks`;
	if (subs === 0) return `Its ${tasks} go too.`;
	if (subs === 1) return `Its sub-category and ${tasks} go too.`;
	return `Its ${subs} sub-categories and ${tasks} go too.`;
}
