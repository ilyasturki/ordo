import { openDB, type DBSchema, type IDBPDatabase } from 'idb';
import type { Op } from '$lib/core/types';
import type { FilterFile } from '$lib/model/filter';

export type StoredProject = {
	id: string;
	name: string;
	/** The stamp of the server copy this one derives from, empty for a project kept in the browser only. */
	stamp: string;
	/** The project as the core answers it. */
	raw: string;
};

export type Queued = {
	seq: number;
	/** Names the op in the sync answer. */
	qid: string;
	project: string;
	op: Op;
};

export type StoredImage = {
	name: string;
	blob: Blob;
	/** Not on the server yet: sync uploads it before the ops that reference it. */
	pending: boolean;
};

/** What every device shows of one project: `<id>.view.json` of docs/file-format.md. */
export type StoredView = {
	id: string;
	folds: string[];
	filter: FilterFile;
};

interface Schema extends DBSchema {
	meta: { key: string; value: unknown };
	projects: { key: string; value: StoredProject };
	queue: { key: number; value: Queued };
	images: { key: string; value: StoredImage };
	views: { key: string; value: StoredView };
}

export type DB = IDBPDatabase<Schema>;

export function openStore(name = 'ordo'): Promise<DB> {
	return openDB<Schema>(name, 2, {
		upgrade(db, from) {
			if (from < 1) {
				db.createObjectStore('meta');
				db.createObjectStore('projects', { keyPath: 'id' });
				db.createObjectStore('queue', { keyPath: 'seq' });
				db.createObjectStore('images', { keyPath: 'name' });
			}
			if (from < 2) {
				db.createObjectStore('views', { keyPath: 'id' });
			}
		}
	});
}
