import { expect, it } from 'vitest';
import { bump, KeyReader } from './bump';

it('bumps one part and zeroes the parts after it, as the TUI does', () => {
	expect(bump('v1.3.0', 2, 1)).toBe('v1.3.1');
	expect(bump('v1.3.1', 2, -1)).toBe('v1.3.0');
	expect(bump('v1.3.4', 1, 1)).toBe('v1.4.0');
	expect(bump('v1.3.4', 0, 1)).toBe('v2.0.0');
});

it('refuses to go below 0 with the TUI’s message', () => {
	expect(bump('v1.3.0', 2, -1)).toEqual(new Error('the version is already at 0'));
});

it('reads ctrl+a, ctrl+x, gm and gM, the shift of gM included', () => {
	const reader = new KeyReader();
	const press = (key: string, ctrlKey = false) => reader.read({ key, ctrlKey, altKey: false, metaKey: false });
	expect(press('a', true)).toEqual([2, 1]);
	expect(press('x', true)).toEqual([2, -1]);
	expect(press('g')).toBeUndefined();
	expect(press('m')).toEqual([1, 1]);
	expect(press('g')).toBeUndefined();
	expect(press('Shift')).toBeUndefined();
	expect(press('M')).toEqual([0, 1]);
	expect(press('m')).toBeUndefined();
	expect(press('a')).toBeUndefined();
	expect(press('g')).toBeUndefined();
	expect(press('j')).toBeUndefined();
	expect(press('m')).toBeUndefined();
});
