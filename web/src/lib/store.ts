import { openDB, type DBSchema, type IDBPDatabase } from 'idb';
import type { Op } from '$lib/core/types';

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

interface Schema extends DBSchema {
	meta: { key: string; value: unknown };
	projects: { key: string; value: StoredProject };
	queue: { key: number; value: Queued };
	images: { key: string; value: StoredImage };
}

export type DB = IDBPDatabase<Schema>;

export function openStore(name = 'ordo'): Promise<DB> {
	return openDB<Schema>(name, 1, {
		upgrade(db) {
			db.createObjectStore('meta');
			db.createObjectStore('projects', { keyPath: 'id' });
			db.createObjectStore('queue', { keyPath: 'seq' });
			db.createObjectStore('images', { keyPath: 'name' });
		}
	});
}
