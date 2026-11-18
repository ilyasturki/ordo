import { test as base, expect, type Page } from '@playwright/test';

export type Options = { build: 'site' | 'private' };

type Fixtures = {
	/** The URL of an app page in this build: `app('/p/x')`. */
	app: (path?: string) => string;
	/** Fails the test on any console error or failed request once it ends. */
	clean: void;
};

export const test = base.extend<Fixtures & Options>({
	build: ['site', { option: true }],
	app: async ({ build }, use) => {
		await use((path = '/') => {
			const url = (build === 'site' ? '/app' : '') + path;
			return url.length > 1 && url.endsWith('/') ? url.slice(0, -1) : url;
		});
	},
	clean: [
		async ({ page }, use) => {
			const errors: string[] = [];
			page.on('console', (message) => {
				if (message.type() === 'error') errors.push(message.text());
			});
			page.on('pageerror', (error) => errors.push(error.message));
			page.on('requestfailed', (request) => errors.push(`${request.url()}: ${request.failure()?.errorText}`));
			await use();
			expect(errors).toEqual([]);
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
