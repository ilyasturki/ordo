/** The part of a three-part version a bump changes: major, minor or patch. */
export type Part = 0 | 1 | 2;

/** The TUI's version bump on `v1.3.0`: one part moves by `delta` and the parts after it go to 0. */
export function bump(version: string, part: Part, delta: number): string | Error {
	const parts = version.slice(1).split('.').map(Number);
	if (parts[part] + delta < 0) {
		return new Error('the version is already at 0');
	}
	parts[part] += delta;
	for (let i = part + 1; i < parts.length; i++) {
		parts[i] = 0;
	}
	return 'v' + parts.join('.');
}

type Key = { key: string; ctrlKey: boolean; altKey: boolean; metaKey: boolean };

const modifiers = new Set(['Shift', 'Control', 'Alt', 'Meta', 'CapsLock']);

/** Reads `ctrl+a`, `ctrl+x`, `gm` and `gM` from keydowns, one at a time. */
export class KeyReader {
	private afterG = false;

	read(key: Key): [Part, number] | undefined {
		if (key.altKey || key.metaKey || modifiers.has(key.key)) return undefined;
		const afterG = this.afterG;
		this.afterG = !key.ctrlKey && key.key === 'g';
		if (key.ctrlKey && key.key === 'a') return [2, 1];
		if (key.ctrlKey && key.key === 'x') return [2, -1];
		if (afterG && !key.ctrlKey && key.key === 'm') return [1, 1];
		if (afterG && !key.ctrlKey && key.key === 'M') return [0, 1];
		return undefined;
	}
}
