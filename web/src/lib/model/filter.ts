import type { Category, Project, Task } from '#lib/core/types.ts';
import { priorityOf, tagDot } from './project';

/**
 * The TUI's filter (internal/app/state.go): "" is the medium and the untagged bucket, and a
 * selected category also shows its sub-categories.
 */
export type Filter = {
	off: boolean;
	statuses: string[];
	priorities: string[];
	categories: string[];
	tags: string[];
};

export const noFilter: Filter = { off: false, statuses: [], priorities: [], categories: [], tags: [] };

export function hasValues(f: Filter): boolean {
	return f.statuses.length + f.priorities.length + f.categories.length + f.tags.length > 0;
}

export function active(f: Filter): boolean {
	return hasValues(f) && !f.off;
}

export function categoryVisible(f: Filter, cat: Category): boolean {
	return (
		f.off ||
		f.categories.length === 0 ||
		f.categories.includes(cat.id) ||
		(!!cat.parent_id && f.categories.includes(cat.parent_id))
	);
}

export function taskVisible(f: Filter, task: Task, cat: Category): boolean {
	if (!active(f)) return true;
	if (f.statuses.length && !f.statuses.includes(task.status)) return false;
	if (f.priorities.length && !f.priorities.includes(priorityOf(task))) return false;
	if (!categoryVisible(f, cat)) return false;
	if (f.tags.length && !f.tags.includes(tagDot(task.tag_color))) return false;
	return true;
}

/** A separator's or a note's own verdict: only its tag or a category list alone matches it. */
export function lineMatches(f: Filter, line: Task, cat: Category): boolean {
	if (!active(f)) return true;
	if (!categoryVisible(f, cat)) return false;
	if (!f.statuses.length && !f.priorities.length && !f.tags.length) return true;
	return !!line.tag_color && f.tags.includes(tagDot(line.tag_color));
}

/** The filter without the categories the project no longer holds. */
export function pruned(f: Filter, project: Project): Filter {
	const ids = new Set(project.categories.map((c) => c.id));
	return f.categories.every((c) => ids.has(c)) ? f : { ...f, categories: f.categories.filter((c) => ids.has(c)) };
}

export function flip(values: string[], v: string): string[] {
	return values.includes(v) ? values.filter((x) => x !== v) : [...values, v];
}

/** The `filter` of `<id>.view.json`: lists sorted, empty ones left out, "none" for "". */
export type FilterFile = { off?: boolean; statuses?: string[]; priorities?: string[]; categories?: string[]; tags?: string[] };

export function toFile(f: Filter): FilterFile {
	const out: FilterFile = {};
	if (f.off && hasValues(f)) out.off = true;
	const put = (key: 'statuses' | 'priorities' | 'categories' | 'tags', bucket: boolean) => {
		if (f[key].length) out[key] = f[key].map((v) => (bucket && v === '' ? 'none' : v)).sort();
	};
	put('statuses', false);
	put('priorities', true);
	put('categories', false);
	put('tags', true);
	return out;
}

/** Reads a view file's filter; older builds wrote "medium" and "" for the empty bucket. */
export function fromFile(file: FilterFile | undefined): Filter {
	const bucket = (vs: string[] | undefined) => [...new Set((vs ?? []).map((v) => (v === 'none' || v === 'medium' ? '' : v)))];
	return {
		off: file?.off === true,
		statuses: [...(file?.statuses ?? [])],
		priorities: bucket(file?.priorities),
		categories: [...(file?.categories ?? [])],
		tags: bucket(file?.tags)
	};
}
