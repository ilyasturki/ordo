const stored = /^attachments\/([0-9a-f]{32}\.(?:png|jpg|gif|webp))$/;
const ref = /!\[([^[\]\n]*)\]\(([^()\n]+)\)/g;

export type ImageRef = {
	name: string;
	target: string;
	/** The name the store keeps it under; undefined for a path only the machine that wrote it can open. */
	file?: string;
};

function isImage(target: string): boolean {
	const t = target.trim();
	return stored.test(t) || t.startsWith('/');
}

export function refs(description = ''): ImageRef[] {
	if (!description.includes('![')) return [];
	const out: ImageRef[] = [];
	for (const m of description.matchAll(ref)) {
		const target = m[2].trim();
		if (!isImage(target) || out.some((r) => r.target === target)) continue;
		out.push({ name: m[1].trim(), target, file: stored.exec(target)?.[1] });
	}
	return out;
}

/** As the TUI's search reads a description: each image reference turned into its label. */
export function display(description = ''): string {
	if (!description.includes('![')) return description;
	return description.replace(ref, (all, name: string, target: string) =>
		isImage(target) ? '▣ ' + (name.trim() || 'image') : all
	);
}

/** The first two lines with text, spaces folded and images left out. */
export function preview(description = ''): string {
	return description
		.replace(ref, (all, _name: string, target: string) => (isImage(target) ? ' ' : all))
		.split('\n')
		.map((line) => line.split(/\s+/).filter(Boolean).join(' '))
		.filter(Boolean)
		.slice(0, 2)
		.join('\n');
}

function imageOnly(line: string): boolean {
	const m = [...line.matchAll(ref)];
	return m.length === 1 && m[0][0] === line && isImage(m[0][2]);
}

export function join(text: string, images: string[]): string {
	const body = text.replace(/\r\n/g, '\n').replace(/\n+$/, '');
	return [...(body.trim() === '' ? [] : [body]), ...images].join('\n');
}

/** The details field and the image lines under it, or the whole description when splitting would lose something. */
export function fields(description = ''): { text: string; images: string[] } {
	const lines = description.split('\n');
	let cut = lines.length;
	while (cut > 0) {
		const line = lines[cut - 1].trim();
		if (line !== '' && !imageOnly(line)) break;
		cut--;
	}
	const images = lines
		.slice(cut)
		.map((l) => l.trim())
		.filter(Boolean);
	const text = lines.slice(0, cut).join('\n').replace(/\n+$/, '');
	if (join(text, images) !== description.replace(/\n+$/, '')) return { text: description, images: [] };
	return { text, images };
}
