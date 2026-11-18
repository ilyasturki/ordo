import { expect, it } from 'vitest';
import { app, route } from './paths';

it('puts the site build’s app under /app and leaves the landing alone', () => {
	expect(app()).toBe('/app');
	expect(app('/p/x')).toBe('/app/p/x');
	expect(route('/')).toBeUndefined();
	expect(route('/app/p/x')).toBeUndefined();
});
