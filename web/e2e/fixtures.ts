import { test as base, expect, type Page } from '@playwright/test';
import { front } from './front';
import { passphrase, serve, type Served } from './serve';

export type Options = {
	build: 'site' | 'private';
	/** The private build starts signed in to its server, as after a sign-in; false shows the sign-in screen. */
	signedIn: boolean;
};

type Fixtures = {
	/** The URL of an app page in this build: `app('/p/x')`. */
	app: (path?: string) => string;
	/** A throwaway ordo serve, empty, of this test alone: the private build's own, behind its origin. */
	served: Served;
	/** Lets the clean check pass the console errors and failed requests a spec brings on itself. */
	noise: (pattern: RegExp) => void;
	/** Fails the test on any console error or failed request once it ends. */
	clean: void;
	stack: { served?: Served; front?: string; ignored: RegExp[] };
};

export const test = base.extend<Fixtures & Options>({
	build: ['site', { option: true }],
	signedIn: [true, { option: true }],
	stack: async ({}, use) => {
		const stack: Fixtures['stack'] = { ignored: [] };
		await use(stack);
		await stack.served?.close();
	},
	baseURL: async ({ baseURL, build, stack }, use) => {
		if (build !== 'private') return use(baseURL);
		stack.served = await serve({ seeded: false });
		const f = await front(stack.served.url);
		stack.front = f.url;
		await use(f.url);
		await f.close();
	},
	served: async ({ stack, baseURL }, use) => {
		void baseURL;
		stack.served ??= await serve({ seeded: false });
		await use(stack.served);
	},
	page: async ({ page, build, signedIn, stack }, use) => {
		if (build === 'private' && signedIn) {
			const res = await page.request.post(stack.front + '/api/login', { data: { passphrase, name: 'Playwright', cookie: true } });
			expect(res.ok()).toBe(true);
			await page.addInitScript(() => {
				if (sessionStorage.getItem('e2e-signed-in')) return;
				sessionStorage.setItem('e2e-signed-in', '1');
				localStorage.setItem('ordo.session', JSON.stringify({ server: '', token: '', cookie: true }));
			});
		}
		await use(page);
	},
	app: async ({ build }, use) => {
		await use((path = '/') => {
			const url = (build === 'site' ? '/app' : '') + path;
			return url.length > 1 && url.endsWith('/') ? url.slice(0, -1) : url;
		});
	},
	noise: async ({ stack }, use) => {
		await use((pattern) => stack.ignored.push(pattern));
	},
	clean: [
		async ({ page, stack }, use) => {
			const errors: string[] = [];
			page.on('console', (message) => {
				if (message.type() === 'error') errors.push(message.text());
			});
			page.on('pageerror', (error) => errors.push(error.message));
			page.on('requestfailed', (request) => {
				// A page leaving drops its live stream.
				if (request.url().endsWith('/api/events')) return;
				errors.push(`${request.url()}: ${request.failure()?.errorText}`);
			});
			await use();
			expect(errors.filter((e) => !stack.ignored.some((p) => p.test(e)))).toEqual([]);
		},
		{ auto: true }
	]
});

export { expect };

/** Light and dark screenshots under test-results/shots, for reading by eye. */
export async function shoot(page: Page, name: string, project: string): Promise<void> {
	for (const colorScheme of ['light', 'dark'] as const) {
		await page.emulateMedia({ colorScheme });
		await page.screenshot({ path: `test-results/shots/${project}/${name}-${colorScheme}.png`, fullPage: true });
	}
	await page.emulateMedia({ colorScheme: null });
}
