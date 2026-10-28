export type Task = {
	id: string;
	parent_id?: string;
	title: string;
	status: string;
	created_at: string;
	updated_at: string;
	priority?: string;
	completion_date?: string;
	estimate_minutes?: number;
	description?: string;
	kind?: string;
	tag_color?: string;
};

export type Category = {
	id: string;
	name: string;
	parent_id?: string;
	created_at: string;
	updated_at?: string;
	estimate_minutes?: number;
	tasks: Task[];
};

export type Project = {
	schema?: number;
	id: string;
	name: string;
	created_at: string;
	updated_at: string;
	categories: Category[];
};

export type Op = {
	action: string;
	ids?: string[];
	new?: string;
	cat?: string;
	parent?: string;
	before?: string;
	value?: string;
	title?: string;
	text?: string;
	base?: string;
	keep?: boolean;
	reverse?: boolean;
	up?: boolean;
	lines?: string[];
	rows?: Task[];
};

export type Verdict = 'applied' | 'unchanged' | 'conflict' | 'gone' | 'refused';

export type Answer = {
	status: Verdict;
	msg?: string;
	row?: string;
	saved?: boolean;
	theirs?: string;
	queue?: Op;
	undo?: Op[];
};

export const STATUSES = ['todo', 'in_progress', 'completed', 'cancelled'] as const;

export const STATUS_LABELS: Record<string, string> = {
	todo: 'To do',
	in_progress: 'Doing',
	completed: 'Done',
	cancelled: 'Cancelled'
};

export const PRIORITIES = ['', 'critical', 'high', 'medium', 'low', 'trivial'] as const;

export const PRIORITY_LABELS: Record<string, string> = {
	'': 'None',
	critical: 'Critical',
	high: 'High',
	medium: 'Medium',
	low: 'Low',
	trivial: 'Trivial'
};

export const TAGS = [
	'',
	'green',
	'blue',
	'magenta',
	'cyan',
	'green-diamond',
	'blue-diamond',
	'magenta-diamond',
	'cyan-diamond'
] as const;

export function tagName(color: string): string {
	if (color === '') {
		return 'None';
	}
	const diamond = color.endsWith('-diamond');
	const hue = diamond ? color.slice(0, -'-diamond'.length) : color;
	const name = hue.charAt(0).toUpperCase() + hue.slice(1);
	return diamond ? `${name} diamond` : name;
}

export const ESTIMATE_PRESETS = [0, 15, 30, 60, 120, 240, 480, 960, 1440, 2400];

export function formatEstimate(minutes = 0): string {
	if (minutes === 0) {
		return '';
	}
	if (minutes < 60) {
		return `${minutes}m`;
	}
	const hours = Math.floor(minutes / 60);
	if (hours < 8) {
		return `${hours}h`;
	}
	return `${Math.floor(hours / 8)}d`;
}

export function isSeparator(t: Task): boolean {
	return t.kind === 'separator';
}

export function finished(status: string): boolean {
	return status === 'completed' || status === 'cancelled';
}

export function openCount(p: Project): number {
	let n = 0;
	for (const c of p.categories) {
		for (const t of c.tasks) {
			if (!isSeparator(t) && (t.status === 'todo' || t.status === 'in_progress')) {
				n++;
			}
		}
	}
	return n;
}

const REF = /!\[([^[\]\n]*)\]\(([^()\n]+)\)/g;
const STORED = /^attachments\/([0-9a-f]{32}\.(?:png|jpg|gif|webp))$/;

export type ImageRef = { name: string; target: string; file?: string };

function refOf(alt: string, target: string): ImageRef | null {
	const ref: ImageRef = { name: alt.trim(), target: target.trim() };
	const stored = STORED.exec(ref.target);
	if (stored) {
		ref.file = stored[1];
		return ref;
	}
	return ref.target.startsWith('/') ? ref : null;
}

export function imagesOf(description = ''): ImageRef[] {
	if (!description.includes('![')) {
		return [];
	}
	const refs: ImageRef[] = [];
	for (const m of description.matchAll(REF)) {
		const ref = refOf(m[1], m[2]);
		if (ref && !refs.some((r) => r.target === ref.target)) {
			refs.push(ref);
		}
	}
	return refs;
}

function strip(text: string): string {
	return text.replace(REF, (match, alt: string, target: string) =>
		refOf(alt, target) ? ' ' : match
	);
}

export function preview(description = ''): string {
	const lines: string[] = [];
	for (const line of strip(description).split('\n')) {
		const flat = line.split(/\s+/u).filter(Boolean).join(' ');
		if (flat !== '') {
			lines.push(flat);
		}
		if (lines.length === 2) {
			break;
		}
	}
	return lines.join('\n');
}

function imageOnly(line: string): boolean {
	const matches = [...line.matchAll(REF)];
	return (
		matches.length === 1 && matches[0][0] === line && refOf(matches[0][1], matches[0][2]) !== null
	);
}

export function joinDescription(text: string, images: string[]): string {
	const body = text.replaceAll('\r\n', '\n').replace(/\n+$/u, '');
	const parts = body.trim() === '' ? [] : [body];
	return [...parts, ...images].join('\n');
}

/** The details field and the image lines under it, or the whole description when splitting would lose something. */
export function descriptionFields(description = ''): { text: string; images: string[] } {
	const lines = description.split('\n');
	let cut = lines.length;
	while (cut > 0) {
		const line = lines[cut - 1].trim();
		if (line === '') {
			cut--;
			continue;
		}
		if (!imageOnly(line)) {
			break;
		}
		cut--;
	}
	const images = lines
		.slice(cut)
		.map((l) => l.trim())
		.filter(Boolean);
	const text = lines.slice(0, cut).join('\n').replace(/\n+$/u, '');
	if (joinDescription(text, images) !== description.replace(/\n+$/u, '')) {
		return { text: description, images: [] };
	}
	return { text, images };
}

export type Row = {
	task: Task;
	children: Row[];
	childDone: number;
	childTotal: number;
	images: number;
	preview: string;
};

export type CategoryView = {
	cat: Category;
	parentName: string;
	open: Row[];
	completed: Row[];
};

export type Filter = {
	statuses: string[];
	priorities: string[];
	categories: string[];
	tags: string[];
};

export const NO_FILTER: Filter = { statuses: [], priorities: [], categories: [], tags: [] };

export function filtering(f: Filter): boolean {
	return f.statuses.length + f.priorities.length + f.categories.length + f.tags.length > 0;
}

function matches(t: Task, f: Filter, query: string): boolean {
	if (isSeparator(t)) {
		return !filtering(f) && query === '';
	}
	if (f.statuses.length > 0 && !f.statuses.includes(t.status)) {
		return false;
	}
	if (f.priorities.length > 0 && !f.priorities.includes(t.priority ?? '')) {
		return false;
	}
	if (f.tags.length > 0 && !f.tags.includes(t.tag_color ?? '')) {
		return false;
	}
	if (query !== '') {
		const hay = `${t.title}\n${t.description ?? ''}`.toLocaleLowerCase();
		return hay.includes(query);
	}
	return true;
}

function rowOf(t: Task): Row {
	return {
		task: t,
		children: [],
		childDone: 0,
		childTotal: 0,
		images: imagesOf(t.description).length,
		preview: preview(t.description)
	};
}

function blockEnd(tasks: Task[], i: number): number {
	let end = i + 1;
	if (tasks[i].parent_id || !tasks[i].id) {
		return end;
	}
	while (end < tasks.length && tasks[end].parent_id === tasks[i].id) {
		end++;
	}
	return end;
}

export function buildCategories(p: Project, f: Filter = NO_FILTER, search = ''): CategoryView[] {
	const query = search.trim().toLocaleLowerCase();
	const narrowed = filtering(f) || query !== '';
	const views: CategoryView[] = [];
	for (const cat of p.categories) {
		if (f.categories.length > 0 && !f.categories.includes(cat.id)) {
			continue;
		}
		const parent = cat.parent_id ? p.categories.find((c) => c.id === cat.parent_id) : undefined;
		const view: CategoryView = { cat, parentName: parent?.name ?? '', open: [], completed: [] };
		for (let i = 0; i < cat.tasks.length;) {
			const end = blockEnd(cat.tasks, i);
			const row = rowOf(cat.tasks[i]);
			let anyChild = false;
			for (const child of cat.tasks.slice(i + 1, end)) {
				row.childTotal++;
				if (finished(child.status)) {
					row.childDone++;
				}
				if (!narrowed || matches(child, f, query)) {
					row.children.push(rowOf(child));
					anyChild = true;
				}
			}
			i = end;
			if (narrowed && !matches(row.task, f, query) && !anyChild) {
				continue;
			}
			const done =
				!isSeparator(row.task) && finished(row.task.status) && row.childDone === row.childTotal;
			(done ? view.completed : view.open).push(row);
		}
		if (narrowed && view.open.length + view.completed.length === 0) {
			continue;
		}
		views.push(view);
	}
	return views;
}

/** The ID ordo mints: 16 random bytes as lowercase hex in 8-4-4-4-12 groups. */
export function newID(): string {
	const b = crypto.getRandomValues(new Uint8Array(16));
	const h = Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('');
	return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

export function findTask(
	p: Project,
	id: string
): { cat: Category; task: Task; index: number } | null {
	for (const cat of p.categories) {
		const index = cat.tasks.findIndex((t) => t.id === id);
		if (index >= 0) {
			return { cat, task: cat.tasks[index], index };
		}
	}
	return null;
}
