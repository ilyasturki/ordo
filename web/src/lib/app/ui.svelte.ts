import { goto } from '$app/navigation';
import { app } from '$lib/paths';

export type SheetState =
	| { kind: 'add'; pid: string; cat: string; after?: string }
	| { kind: 'filter'; pid: string }
	| { kind: 'project'; pid: string }
	| { kind: 'group'; pid: string }
	| { kind: 'newProject' }
	| { kind: 'category'; pid: string; cat: string }
	| { kind: 'newCategory'; pid: string; parent: string }
	| { kind: 'line'; pid: string; cat: string; after?: string; type: 'note' | 'separator' }
	| { kind: 'deleteCategory'; pid: string; cat: string }
	| { kind: 'deleteProject'; pid: string }
	| { kind: 'move'; pid: string; tid: string }
	| { kind: 'help' };

/**
 * What the screen shows beside the projects: the sheet open, the keyboard's cursor, the search, and
 * the rows a peek reveals or an edit holds in place under the filter, which end with the project.
 */
export const ui: {
	sheet?: SheetState;
	/** A line's key, as board.ts names them. */
	cursor: string;
	/** The category the bar names and the + adds to. */
	focus?: string;
	searching: boolean;
	query: string;
	peek?: string;
	held: string[];
	/** A category to bring into sight once the list draws it. */
	jump?: string;
	/** The task opening next starts with its title in edit. */
	editTitle: boolean;
} = $state({ cursor: '', searching: false, query: '', held: [], editTitle: false });

export function resetList(): void {
	ui.peek = undefined;
	ui.held = [];
}

export function hold(id: string): void {
	if (!ui.held.includes(id)) ui.held = [...ui.held, id];
}

export function openTask(pid: string, tid: string, replace = false): Promise<void> {
	ui.cursor = 't:' + tid;
	return goto(app(`/p/${pid}/t/${tid}`), { replaceState: replace, noScroll: true, keepFocus: replace });
}

/** A wide screen shows the list and the open task side by side; a phone shows one at a time. */
export function wide(): boolean {
	return matchMedia('(width >= 52.5rem)').matches;
}

export function download(name: string, text: string): void {
	const link = document.createElement('a');
	link.href = URL.createObjectURL(new Blob([text], { type: 'application/json' }));
	link.download = name;
	link.click();
	setTimeout(() => URL.revokeObjectURL(link.href), 0);
}
