import { goto } from '$app/navigation';
import type { Local } from '#lib/local.svelte.ts';
import { app } from '#lib/paths.ts';
import { download } from './ui.svelte';

/** Asks for a project file and adds it as a new project, then shows it. */
export function importProject(local: Local): void {
	const input = document.createElement('input');
	input.type = 'file';
	input.accept = '.json,application/json';
	input.onchange = async () => {
		const file = input.files?.[0];
		if (!file) return;
		try {
			const id = await local.import(await file.text());
			local.say(`Imported ${local.project(id)?.name ?? file.name}`);
			await goto(app(`/p/${id}`));
		} catch (error) {
			local.say(`Can't import ${file.name}: ${error instanceof Error ? error.message : String(error)}`);
		}
	};
	input.click();
}

/** Downloads the project as ordo writes its file, under the name ordo gives it. */
export function exportProject(local: Local, pid: string): void {
	const raw = local.export(pid);
	if (raw) download(`${pid}.json`, raw);
}
