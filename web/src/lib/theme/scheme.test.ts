import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';
import { colorsCSS } from './scheme';

it('colors.css is what `bun run theme` writes', () => {
	expect(readFileSync(new URL('./colors.css', import.meta.url), 'utf8')).toBe(colorsCSS());
});

it('keeps the phone marks and the logo indigo', () => {
	const css = colorsCSS();
	expect(css).toContain('--color-primary: light-dark(#3949ab, #bbc3ff);');
	expect(css).toContain('--color-mark-up: light-dark(#ba1a1a, #ffb4ab);');
});
