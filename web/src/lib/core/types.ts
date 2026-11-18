/** A project file's shape, docs/file-format.md. */
export type Project = {
	schema?: number;
	id: string;
	name: string;
	group?: string;
	created_at: string;
	updated_at: string;
	categories: Category[];
};

export type Category = {
	id: string;
	name: string;
	parent_id?: string;
	created_at: string;
	updated_at?: string;
	pinned?: boolean;
	tasks: Task[];
};

export type Status = 'todo' | 'in_progress' | 'completed' | 'cancelled';

export type Task = {
	id: string;
	parent_id?: string;
	title: string;
	status: Status;
	created_at: string;
	updated_at: string;
	priority?: string;
	completion_date?: string;
	description?: string;
	kind?: '' | 'note' | 'separator';
	tag_color?: string;
};

/** operations.Op: one edit, by the IDs of the rows it touches. */
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
	cats?: Category[];
};

export type Verdict = 'applied' | 'unchanged' | 'conflict' | 'gone' | 'refused';

/** core.Answer. */
export type Answer = {
	status: Verdict;
	msg?: string;
	row?: string;
	saved?: boolean;
	theirs?: string;
	clash?: { taken: string; this: string; other: string };
	/** The op that carries the edit to a server, set when it applied. */
	queue?: Op;
	/** The ops that take the edit back, set when it applied. */
	undo?: Op[];
};
