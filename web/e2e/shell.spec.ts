import { expect, shoot, test } from './fixtures';

test('the site build opens on the landing, which leads to the app without loading the core', async ({ page, build }, info) => {
	test.skip(build !== 'site', 'the landing is the site build’s');
	const wasm: string[] = [];
	page.on('request', (request) => {
		if (request.url().endsWith('.wasm')) wasm.push(request.url());
	});
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await shoot(page, 'landing', info.project.name);
	expect(wasm).toEqual([]);

	await page.getByRole('link', { name: 'Open Ordo in your browser' }).first().click();
	await expect(page).toHaveURL('/app');
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	expect(wasm).toHaveLength(1);

	await page.goto('/app/');
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
});

test('the private build opens on the app', async ({ page, build }) => {
	test.skip(build !== 'private', 'the private build has no landing');
	await page.goto('/');
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
});

test('the theme follows the system, in Google Sans Flex', async ({ page, app }) => {
	await page.goto(app());
	await expect(page.getByRole('heading', { name: 'No projects yet' })).toBeVisible();
	const look = () =>
		page.evaluate(async () => {
			await document.fonts.ready;
			return {
				background: getComputedStyle(document.body).backgroundColor,
				font: document.fonts.check('16px "Google Sans Flex"') && [...document.fonts].some((f) => f.family.includes('Google Sans Flex') && f.status === 'loaded')
			};
		});
	await page.emulateMedia({ colorScheme: 'light' });
	expect(await look()).toEqual({ background: 'rgb(251, 248, 255)', font: true });
	await page.emulateMedia({ colorScheme: 'dark' });
	expect((await look()).background).toBe('rgb(19, 19, 24)');
});
