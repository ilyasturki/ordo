import { goto } from '$app/navigation';
import type { Project } from '$lib/core/types';
import type { Local } from '$lib/local.svelte';
import { stops, target, type Line } from '$lib/model/board';
import { active } from '$lib/model/filter';
import { find, isLine, isNote, isSeparator, nextStatus, nextTag, priorities, priorityOf } from '$lib/model/project';
import { app } from '$lib/paths';
import { hold, openTask, ui, wide } from './ui.svelte';

/** The TUI's keys the web app takes, as `?` lists them. */
export const keyHelp: [string, [string, string][]][] = [
	[
		'Move',
		[
			['j / k, ↓ / ↑', 'next / previous row'],
			['g g / G', 'first / last row'],
			['z a', 'fold / unfold'],
			['/', 'search'],
			['f / F', 'filter / turn it off and on']
		]
	],
	[
		'Edit',
		[
			['enter / e', 'open and edit'],
			['a', 'add a task below'],
			['o / _', 'add a note'],
			['-', 'add a separator'],
			['A', 'add a category'],
			['space / x', 'cycle status'],
			['h / l', 'priority down / up'],
			['t', 'next colour'],
			['c', 'task ↔ note'],
			['d', 'delete']
		]
	],
	[
		'Organise',
		[
			['J / K', 'move down / up'],
			['> / <', 'nest under the row above / un-nest'],
			['m', 'move to a category']
		]
	],
	[
		'App',
		[
			['u / ctrl+r', 'undo / redo'],
			['esc', 'back to the list'],
			['?', 'these keys']
		]
	]
];

export type KeyContext = { local: Local; project: Project; lines: Line[]; taskOpen: boolean };

let prefix = '';

function typing(e: KeyboardEvent): boolean {
	const el = e.target as HTMLElement | null;
	if (!el) return false;
	if (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName)) return true;
	// A focused button or link answers enter and space itself.
	return (e.key === 'Enter' || e.key === ' ') && (el.tagName === 'BUTTON' || el.tagName === 'A');
}

/** Runs the key on the list; false when it isn't one of the list's. */
export function onKey(e: KeyboardEvent, ctx: KeyContext): boolean {
	if (e.defaultPrevented || typing(e) || e.altKey || e.metaKey) return false;
	if (document.querySelector('dialog[open], [popover]:popover-open')) return false;
	const key = e.ctrlKey ? (e.key === 'r' ? 'ctrl+r' : '') : e.key === ' ' && e.shiftKey ? 'shift+space' : e.key;
	if (key === '') return false;
	const was = prefix;
	prefix = '';
	if (was === 'g' && key === 'g') return move(ctx, -Infinity);
	if (was === 'z' && key === 'a') return fold(ctx);
	if ((key === 'g' || key === 'z') && was === '') {
		prefix = key;
		return true;
	}
	return run(key, ctx);
}

function run(key: string, ctx: KeyContext): boolean {
	const { local } = ctx;
	const pid = ctx.project.id;
	// A key pressed again before the last edit shows reads what that edit left.
	const project = local.held(pid) ?? ctx.project;
	const cursor = ui.cursor;
	const task = cursor.startsWith('t:') ? find(project, cursor.slice(2)) : undefined;
	const catOf = () =>
		cursor.startsWith('c:') ? cursor.slice(2) : (task?.cat.id ?? ctx.lines.find((l) => l.key === cursor)?.cat ?? target(project, local.view(pid).filter, ui.focus)?.id);
	const edit = (op: Parameters<Local['perform']>[1], quiet = true) => {
		if (task && active(local.view(pid).filter)) hold(task.task.id);
		void local.perform(pid, op, { quiet });
	};
	switch (key) {
		case 'j':
		case 'ArrowDown':
			return move(ctx, 1);
		case 'k':
		case 'ArrowUp':
			return move(ctx, -1);
		case 'G':
			return move(ctx, Infinity);
		case 'Enter':
		case 'e':
			if (task) {
				ui.editTitle = true;
				void openTask(pid, task.task.id);
			} else if (cursor.startsWith('c:')) ui.sheet = { kind: 'category', pid, cat: cursor.slice(2) };
			else if (cursor.startsWith('h:')) ui.peek = ui.peek === cursor.slice(2) ? undefined : cursor.slice(2);
			else return false;
			return true;
		case 'a': {
			const cat = catOf();
			if (!cat) ui.sheet = { kind: 'newCategory', pid, parent: '' };
			else ui.sheet = { kind: 'add', pid, cat, after: task?.task.id };
			return true;
		}
		case 'o':
		case '_':
		case '-': {
			const cat = catOf();
			if (!cat) return false;
			ui.sheet = { kind: 'line', pid, cat, after: task?.task.id, type: key === '-' ? 'separator' : 'note' };
			return true;
		}
		case 'A':
			ui.sheet = { kind: 'newCategory', pid, parent: '' };
			return true;
		case ' ':
		case 'x':
		case 'shift+space':
			if (!task || isLine(task.task)) return false;
			edit({ action: 'status.cycle', ids: [task.task.id], reverse: key === 'shift+space' }, key === 'shift+space' || nextStatus(task.task.status) !== 'completed');
			return true;
		case 'h':
		case 'l': {
			if (!task || isLine(task.task)) return false;
			const at = priorities.indexOf(priorityOf(task.task)) + (key === 'l' ? -1 : 1);
			if (at < 0 || at >= priorities.length) return true;
			edit({ action: 'priority.set', ids: [task.task.id], value: priorities[at] });
			return true;
		}
		case 't': {
			if (!task) return false;
			edit({ action: 'tag.set', ids: [task.task.id], value: nextTag(task.cat, task.task) });
			return true;
		}
		case 'c':
			if (!task || isSeparator(task.task)) return false;
			edit({ action: 'task.kind', ids: [task.task.id], value: isNote(task.task) ? '' : 'note' }, false);
			return true;
		case 'J':
		case 'K':
			if (!task) return false;
			edit({ action: 'task.shift', ids: [task.task.id], up: key === 'K' }, false);
			return true;
		case '>':
		case '<':
			if (!task) return false;
			edit({ action: key === '>' ? 'task.nest' : 'task.unnest', ids: [task.task.id] }, false);
			return true;
		case 'm':
			if (!task) return false;
			ui.sheet = { kind: 'move', pid, tid: task.task.id };
			return true;
		case 'd':
			if (task) {
				const next = neighbour(ctx);
				void local.perform(pid, { action: 'task.delete', ids: [task.task.id] }).then((a) => {
					if (a.status === 'applied') ui.cursor = next;
				});
			} else if (cursor.startsWith('c:')) ui.sheet = { kind: 'deleteCategory', pid, cat: cursor.slice(2) };
			else return false;
			return true;
		case 'u':
			void local.undo();
			return true;
		case 'ctrl+r':
			void local.redo();
			return true;
		case '/':
			ui.searching = true;
			return true;
		case 'f':
			ui.sheet = { kind: 'filter', pid };
			return true;
		case 'F': {
			const filter = local.view(pid).filter;
			void local.setFilter(pid, { ...filter, off: !filter.off });
			return true;
		}
		case '?':
			ui.sheet = { kind: 'help' };
			return true;
		case 'Escape':
			if (ui.searching) {
				ui.searching = false;
				ui.query = '';
			} else if (ctx.taskOpen) void goto(app(`/p/${pid}`), { noScroll: true });
			else return false;
			return true;
	}
	return false;
}

/** The row the cursor lands on once its own goes: the next, else the one above. */
function neighbour(ctx: KeyContext): string {
	const all = stops(ctx.lines);
	const at = all.indexOf(ui.cursor);
	return all[at + 1] ?? all[at - 1] ?? '';
}

function move(ctx: KeyContext, by: number): boolean {
	const all = stops(ctx.lines);
	if (all.length === 0) return true;
	const at = all.indexOf(ui.cursor);
	const next =
		by === -Infinity ? 0 : by === Infinity ? all.length - 1 : at < 0 ? (by > 0 ? 0 : all.length - 1) : Math.min(all.length - 1, Math.max(0, at + by));
	ui.cursor = all[next];
	document.querySelector(`[data-key="${ui.cursor}"]`)?.scrollIntoView({ block: 'nearest' });
	// The open task follows the cursor, as a mail client's reading pane does.
	if (ctx.taskOpen && wide() && ui.cursor.startsWith('t:')) void openTask(ctx.project.id, ui.cursor.slice(2), true);
	return true;
}

function fold(ctx: KeyContext): boolean {
	const pid = ctx.project.id;
	const line = ctx.lines.find((l) => l.key === ui.cursor);
	if (line?.kind === 'header') void ctx.local.setFolded(pid, line.cat, !line.folded);
	else if (line?.kind === 'block' && line.row.childTotal > 0) void ctx.local.setFolded(pid, line.row.task.id, !line.row.folded);
	return true;
}
